package stats

import (
	"testing"
	"time"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func at(daysAgo int, hour int) time.Time {
	d := Day(now).AddDate(0, 0, -daysAgo)
	return d.Add(time.Duration(hour) * time.Hour)
}

func commit(daysAgo int, repo string, n int) Activity {
	return Activity{At: at(daysAgo, 10), Repo: repo, Kind: KindCommit, Count: n}
}

func TestComputeNoActivity(t *testing.T) {
	s := Compute(nil, now)
	if s.HasActivity() {
		t.Fatal("HasActivity() = true, want false")
	}
	if s.XP != 0 || s.Streak != 0 || s.ActiveDays30 != 0 || s.Repos != 0 {
		t.Fatalf("unexpected stats for empty input: %+v", s)
	}
	if s.DaysSinceLast != -1 {
		t.Fatalf("DaysSinceLast = %d, want -1", s.DaysSinceLast)
	}
}

func TestComputeXP(t *testing.T) {
	acts := []Activity{
		commit(0, "a/one", 3), // 30
		{At: at(1, 9), Repo: "a/two", Kind: KindPullRequest, Count: 1}, // 25
		{At: at(2, 9), Repo: "a/three", Kind: KindIssue},               // 5 (count 0 -> 1)
		{At: at(2, 9), Repo: "b/star", Kind: KindOther, Count: 1},      // 2
	}
	s := Compute(acts, now)
	if s.XP != 62 {
		t.Fatalf("XP = %d, want 62", s.XP)
	}
	if s.Commits != 3 || s.PullRequests != 1 || s.Issues != 1 || s.Others != 1 {
		t.Fatalf("counters wrong: %+v", s)
	}
	// Starred repos do not count as touched.
	if s.Repos != 3 {
		t.Fatalf("Repos = %d, want 3", s.Repos)
	}
}

func TestStreakWithGaps(t *testing.T) {
	acts := []Activity{
		commit(0, "r", 1), commit(1, "r", 1), commit(2, "r", 1),
		// gap on day 3
		commit(4, "r", 1), commit(5, "r", 1),
	}
	s := Compute(acts, now)
	if s.Streak != 3 {
		t.Fatalf("Streak = %d, want 3", s.Streak)
	}
	if s.ActiveDays30 != 5 {
		t.Fatalf("ActiveDays30 = %d, want 5", s.ActiveDays30)
	}
	if s.DaysSinceLast != 0 {
		t.Fatalf("DaysSinceLast = %d, want 0", s.DaysSinceLast)
	}
}

func TestStreakSurvivesUntilEndOfNextDay(t *testing.T) {
	acts := []Activity{commit(1, "r", 1), commit(2, "r", 1)}
	if s := Compute(acts, now); s.Streak != 2 {
		t.Fatalf("Streak = %d, want 2 (yesterday still counts)", s.Streak)
	}
	acts = []Activity{commit(2, "r", 1), commit(3, "r", 1)}
	if s := Compute(acts, now); s.Streak != 0 {
		t.Fatalf("Streak = %d, want 0 (broken streak)", s.Streak)
	}
}

func TestUTCDayBoundary(t *testing.T) {
	madrid := time.FixedZone("CEST", 2*60*60)
	acts := []Activity{
		// 23:59 UTC on the 28th and 00:01 UTC on the 29th: two different days.
		{At: time.Date(2026, 9, 28, 23, 59, 0, 0, time.UTC), Repo: "r", Kind: KindCommit, Count: 1},
		{At: time.Date(2026, 9, 29, 0, 1, 0, 0, time.UTC), Repo: "r", Kind: KindCommit, Count: 1},
		// 01:30 local on the 28th in UTC+2 is 23:30 UTC on the 27th.
		{At: time.Date(2026, 9, 28, 1, 30, 0, 0, madrid), Repo: "r", Kind: KindCommit, Count: 1},
	}
	s := Compute(acts, now)
	if s.ActiveDays30 != 3 {
		t.Fatalf("ActiveDays30 = %d, want 3", s.ActiveDays30)
	}
	if s.Streak != 3 {
		t.Fatalf("Streak = %d, want 3", s.Streak)
	}
	// Right before midnight UTC the 29th has not started yet.
	early := time.Date(2026, 9, 28, 23, 59, 30, 0, time.UTC)
	s = Compute(acts, early)
	if s.DaysSinceLast != 0 || s.Streak != 2 {
		t.Fatalf("at %s: DaysSinceLast=%d Streak=%d, want 0 and 2", early, s.DaysSinceLast, s.Streak)
	}
}

func TestIgnoresFutureAndOldActivity(t *testing.T) {
	acts := []Activity{
		{At: now.Add(time.Hour), Repo: "r", Kind: KindCommit, Count: 5},
		commit(90, "old", 5),
		commit(89, "r", 1),
	}
	s := Compute(acts, now)
	if s.XP != XPCommit {
		t.Fatalf("XP = %d, want %d", s.XP, XPCommit)
	}
	if s.ActiveDays90 != 1 || s.ActiveDays30 != 0 {
		t.Fatalf("ActiveDays90=%d ActiveDays30=%d, want 1 and 0", s.ActiveDays90, s.ActiveDays30)
	}
	if s.DaysSinceLast != 89 {
		t.Fatalf("DaysSinceLast = %d, want 89", s.DaysSinceLast)
	}
}

func TestActiveDaysWindowEdges(t *testing.T) {
	acts := []Activity{commit(29, "r", 1), commit(30, "r", 1)}
	s := Compute(acts, now)
	if s.ActiveDays30 != 1 {
		t.Fatalf("ActiveDays30 = %d, want 1 (day 29 in, day 30 out)", s.ActiveDays30)
	}
	if s.ActiveDays90 != 2 {
		t.Fatalf("ActiveDays90 = %d, want 2", s.ActiveDays90)
	}
}

func TestSameDayCountsOnce(t *testing.T) {
	acts := []Activity{commit(0, "r", 1), {At: at(0, 23), Repo: "r", Kind: KindIssue}}
	s := Compute(acts, now)
	if s.ActiveDays30 != 1 || s.Streak != 1 {
		t.Fatalf("ActiveDays30=%d Streak=%d, want 1 and 1", s.ActiveDays30, s.Streak)
	}
}
