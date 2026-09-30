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

func TestComputeIgnoresOrder(t *testing.T) {
	acts := []Activity{
		commit(0, "a/one", 2), commit(1, "a/two", 1), commit(2, "a/one", 1),
		commit(5, "a/three", 3), commit(40, "a/one", 1),
		{At: at(3, 8), Repo: "a/two", Kind: KindPullRequest, Count: 1},
	}
	want := Compute(acts, now)
	reversed := make([]Activity, len(acts))
	for i, a := range acts {
		reversed[len(acts)-1-i] = a
	}
	shuffled := []Activity{acts[3], acts[0], acts[5], acts[4], acts[2], acts[1]}
	for name, in := range map[string][]Activity{"reversed": reversed, "shuffled": shuffled} {
		if got := Compute(in, now); got != want {
			t.Errorf("%s: got %+v, want %+v", name, got, want)
		}
	}
	if want.Streak != 4 || want.DaysSinceLast != 0 {
		t.Fatalf("unexpected reference stats: %+v", want)
	}
}

func TestComputeIgnoresFutureEvents(t *testing.T) {
	base := []Activity{commit(1, "r", 1)}
	withFuture := append([]Activity{
		{At: now.Add(time.Minute), Repo: "r", Kind: KindCommit, Count: 50}, // later today
		{At: at(-1, 10), Repo: "r", Kind: KindCommit, Count: 50},           // tomorrow
		{At: now.AddDate(1, 0, 0), Repo: "x", Kind: KindPullRequest},       // next year
	}, base...)
	got, want := Compute(withFuture, now), Compute(base, now)
	if got != want {
		t.Fatalf("future events changed the stats: got %+v, want %+v", got, want)
	}
	if got.DaysSinceLast != 1 || got.XP != 10 {
		t.Fatalf("stats = %+v", got)
	}
	// An event exactly at "now" is not in the future.
	if s := Compute([]Activity{{At: now, Repo: "r", Kind: KindCommit}}, now); s.XP != 10 || s.DaysSinceLast != 0 {
		t.Fatalf("event at now: %+v", s)
	}
}

func TestComputeUsesUTCDays(t *testing.T) {
	// 23:30 in UTC-5 on the 28th is 04:30 UTC on the 29th: today.
	west := time.FixedZone("west", -5*3600)
	a := Activity{At: time.Date(2026, 9, 28, 23, 30, 0, 0, west), Repo: "r", Kind: KindCommit}
	if s := Compute([]Activity{a}, now); s.DaysSinceLast != 0 || s.Streak != 1 {
		t.Fatalf("stats = %+v, want today", s)
	}
}

func TestComputeWindowEdgesAndZeroTime(t *testing.T) {
	acts := []Activity{
		commit(89, "r", 1),                      // oldest day still counted
		commit(90, "r", 1),                      // one day too old
		{Repo: "r", Kind: KindCommit, Count: 9}, // missing date (zero time)
	}
	s := Compute(acts, now)
	if s.Commits != 1 || s.ActiveDays90 != 1 || s.ActiveDays30 != 0 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestComputeDuplicateActivitiesInSameDay(t *testing.T) {
	// Many activities on one day are one active day and one streak day.
	acts := []Activity{commit(0, "r", 1), commit(0, "r", 1), commit(0, "s", 1)}
	s := Compute(acts, now)
	if s.ActiveDays30 != 1 || s.Streak != 1 || s.Repos != 2 {
		t.Fatalf("stats = %+v", s)
	}
}
