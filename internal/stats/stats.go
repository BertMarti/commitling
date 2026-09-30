// Package stats turns a list of public GitHub activities into the numbers
// that drive the creature: experience, active days, streak and so on.
//
// Everything here is pure: the reference time ("now") is always injected so
// the same input produces the same output.
package stats

import (
	"sort"
	"time"
)

// Kind classifies an activity for experience purposes.
type Kind int

const (
	// KindOther is any public event that is not a commit, PR or issue
	// (stars, forks, comments, reviews, releases...).
	KindOther Kind = iota
	// KindCommit is a single commit pushed to a public repository.
	KindCommit
	// KindPullRequest is a pull request opened.
	KindPullRequest
	// KindIssue is an issue opened.
	KindIssue
)

// Experience points per unit of each kind.
const (
	XPCommit      = 10
	XPPullRequest = 25
	XPIssue       = 5
	XPOther       = 2
)

// Window sizes in days.
const (
	// RecentWindowDays is the window used for "active days" on the card.
	RecentWindowDays = 30
	// LongWindowDays is the window the public events API covers.
	LongWindowDays = 90
)

// Activity is one public action by the user.
type Activity struct {
	At    time.Time
	Repo  string
	Kind  Kind
	Count int // number of units (commits in a push); values < 1 count as 1
}

// Stats is the summary used by the creature and the card.
type Stats struct {
	XP           int
	Commits      int
	PullRequests int
	Issues       int
	Others       int

	// ActiveDays30 is the number of distinct UTC days with activity in the
	// last 30 days (today included).
	ActiveDays30 int
	// ActiveDays90 is the same over the last 90 days.
	ActiveDays90 int
	// Streak is the number of consecutive UTC days with activity ending
	// today, or yesterday if there is nothing yet today.
	Streak int
	// DaysSinceLast is the number of UTC days since the last activity
	// (0 = today). It is -1 when there is no activity at all.
	DaysSinceLast int
	// Repos is the number of distinct repositories with commits, pull
	// requests or issues.
	Repos int
}

// HasActivity reports whether any activity was found.
func (s Stats) HasActivity() bool { return s.DaysSinceLast >= 0 }

// Day truncates t to its UTC calendar day.
func Day(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func daysBetween(from, to time.Time) int {
	return int(Day(to).Sub(Day(from)).Hours() / 24)
}

// Compute summarises the activities as seen at the instant now.
//
// Only activities inside the last 90 days (today included) and not after
// now are considered, which matches what the public events API returns.
func Compute(acts []Activity, now time.Time) Stats {
	s := Stats{DaysSinceLast: -1}
	today := Day(now)
	oldest := today.AddDate(0, 0, -(LongWindowDays - 1))

	days := map[time.Time]bool{}
	repos := map[string]bool{}

	for _, a := range acts {
		if a.At.After(now) || Day(a.At).Before(oldest) {
			continue
		}
		n := a.Count
		if n < 1 {
			n = 1
		}
		switch a.Kind {
		case KindCommit:
			s.Commits += n
			s.XP += n * XPCommit
		case KindPullRequest:
			s.PullRequests += n
			s.XP += n * XPPullRequest
		case KindIssue:
			s.Issues += n
			s.XP += n * XPIssue
		default:
			s.Others += n
			s.XP += n * XPOther
		}
		if a.Kind != KindOther && a.Repo != "" {
			repos[a.Repo] = true
		}
		days[Day(a.At)] = true
	}
	s.Repos = len(repos)
	if len(days) == 0 {
		return s
	}

	sorted := make([]time.Time, 0, len(days))
	for d := range days {
		sorted = append(sorted, d)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Before(sorted[j]) })

	last := sorted[len(sorted)-1]
	s.DaysSinceLast = daysBetween(last, today)

	for _, d := range sorted {
		ago := daysBetween(d, today)
		if ago < RecentWindowDays {
			s.ActiveDays30++
		}
		if ago < LongWindowDays {
			s.ActiveDays90++
		}
	}

	// The streak survives until the end of the day after the last activity,
	// so it does not reset in the morning before the first commit.
	if s.DaysSinceLast <= 1 {
		for d := last; days[d]; d = d.AddDate(0, 0, -1) {
			s.Streak++
		}
	}
	return s
}
