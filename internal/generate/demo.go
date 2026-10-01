package generate

import (
	"strconv"
	"time"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/stats"
)

// DemoUser is the fictitious user of the demo and of the gallery cards.
const DemoUser = "octoexample"

// DemoDays is how many days the demo lasts: the whole window of the public
// events API, so every number is one commitling can really reach.
const DemoDays = 90

// demoStart is day 0 of the demo (a fixed date, so nothing depends on the clock).
var demoStart = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// Days without activity, so the creature gets bored and falls asleep before
// coming back. Day numbers start at 1; days 1 and 2 are quiet too, so the demo
// begins with a sleeping seed.
const (
	demoFirstDay = 3
	demoGapFrom  = 34
	demoGapTo    = 41
)

// demoNow is the instant the card of the given day is drawn at.
func demoNow(day int) time.Time { return demoStart.AddDate(0, 0, day).Add(12 * time.Hour) }

// demoActivities is what the fictitious user did on days 1..day: a few commits
// a day, a pull request every fourth day and a new repository every five days.
// Nothing here comes from a real person.
func demoActivities(day int) []stats.Activity {
	var acts []stats.Activity
	for d := demoFirstDay; d <= day; d++ {
		if d >= demoGapFrom && d <= demoGapTo {
			continue
		}
		at := demoStart.AddDate(0, 0, d).Add(9 * time.Hour)
		repo := DemoUser + "/repo-" + strconv.Itoa(1+d/5)
		acts = append(acts, stats.Activity{At: at, Repo: repo, Kind: stats.KindCommit, Count: 4 + d%3})
		if d%4 == 0 {
			acts = append(acts, stats.Activity{At: at, Repo: repo, Kind: stats.KindPullRequest, Count: 1})
		}
	}
	return acts
}

// DemoFrame is the card of one day of the demo.
type DemoFrame struct {
	Result
	Day   int    // the day drawn, 0..Days
	Days  int    // DemoDays
	Phase string // growth stage of the creature, in the language of the species
}

// Demo draws the card of the fictitious user on the given day of a
// DemoDays-day timelapse: a seed that grows through every stage, changes mood
// and unlocks every accessory. It is the same stats.Compute and render as the
// real cards. The day is clamped to 0..DemoDays; Options.User and Options.Now
// are ignored (the user is DemoUser, the clock is the demo's).
func Demo(day int, o Options) (DemoFrame, error) {
	if err := o.Check(); err != nil {
		return DemoFrame{}, err
	}
	day = min(max(day, 0), DemoDays)
	species, _ := creature.SpeciesByName(o.speciesName())
	res, c := draw(DemoUser, stats.Compute(demoActivities(day), demoNow(day)), species, o)
	return DemoFrame{Result: res, Day: day, Days: DemoDays, Phase: c.StageName()}, nil
}
