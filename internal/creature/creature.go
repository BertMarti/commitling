// Package creature decides what the commitling looks like: its growth stage,
// its mood and the accessories it has unlocked, plus the pixel maps used to
// draw it.
//
// The creature is an original design: a small moss sprout with eyes that
// grows from a honey-coloured seed into an ancient tree.
package creature

import "github.com/BertMarti/commitling/internal/stats"

// Stage is a growth stage, unlocked by experience.
type Stage int

// Growth stages, from youngest to oldest.
const (
	Seed Stage = iota
	Sprout
	Sapling
	Shrub
	Ancient
)

// Experience needed to reach each stage.
const (
	SeedXP    = 0
	SproutXP  = 100
	SaplingXP = 400
	ShrubXP   = 1000
	AncientXP = 2500
)

// Stages lists every stage in growth order.
var Stages = []Stage{Seed, Sprout, Sapling, Shrub, Ancient}

var stageXP = [...]int{SeedXP, SproutXP, SaplingXP, ShrubXP, AncientXP}

var stageNames = [...]string{"Semilla", "Brote", "Retoño", "Arbusto", "Árbol ancestral"}

var stageSlugs = [...]string{"seed", "sprout", "sapling", "shrub", "ancient"}

// Name is the Spanish name shown on the card.
func (s Stage) Name() string { return stageNames[s] }

// Slug is a stable ASCII identifier, used for file names.
func (s Stage) Slug() string { return stageSlugs[s] }

// MinXP is the experience needed to reach the stage.
func (s Stage) MinXP() int { return stageXP[s] }

// Next returns the following stage and whether there is one.
func (s Stage) Next() (Stage, bool) {
	if s >= Ancient {
		return s, false
	}
	return s + 1, true
}

// StageFor returns the stage reached with xp experience points.
func StageFor(xp int) Stage {
	st := Seed
	for _, s := range Stages {
		if xp >= s.MinXP() {
			st = s
		}
	}
	return st
}

// Progress returns how far xp is between the current stage and the next
// one, from 0 to 1. It is 1 at the last stage.
func Progress(xp int) float64 {
	st := StageFor(xp)
	next, ok := st.Next()
	if !ok {
		return 1
	}
	span := next.MinXP() - st.MinXP()
	p := float64(xp-st.MinXP()) / float64(span)
	if p < 0 {
		return 0
	}
	return p
}

// Mood depends on recent activity.
type Mood int

// Moods, from least to most active.
const (
	Sleeping Mood = iota
	Bored
	Happy
	Radiant
)

// Mood thresholds.
const (
	// SleepingAfterDays without activity the creature falls asleep.
	SleepingAfterDays = 7
	// BoredAfterDays without activity the creature gets bored.
	BoredAfterDays = 3
	// RadiantStreak consecutive days make the creature radiant.
	RadiantStreak = 5
)

// Moods lists every mood.
var Moods = []Mood{Sleeping, Bored, Happy, Radiant}

var moodNames = [...]string{"Durmiendo", "Aburrido", "Contento", "Radiante"}

var moodSlugs = [...]string{"sleeping", "bored", "happy", "radiant"}

// Name is the Spanish name shown on the card.
func (m Mood) Name() string { return moodNames[m] }

// Slug is a stable ASCII identifier, used for file names.
func (m Mood) Slug() string { return moodSlugs[m] }

// MoodFor derives the mood from the stats:
//
//   - Sleeping: no activity, or 7 or more days since the last one.
//   - Bored: 3 to 6 days since the last activity.
//   - Radiant: a streak of 5 or more days.
//   - Happy: anything else (activity in the last 2 days).
func MoodFor(s stats.Stats) Mood {
	switch {
	case !s.HasActivity() || s.DaysSinceLast >= SleepingAfterDays:
		return Sleeping
	case s.DaysSinceLast >= BoredAfterDays:
		return Bored
	case s.Streak >= RadiantStreak:
		return Radiant
	default:
		return Happy
	}
}

// Accessories unlocked by the user.
type Accessories struct {
	Hat    bool // more than 30 active days in the last 90
	Scarf  bool // streak of 10 days or more
	Flower bool // 5 or more distinct repositories
}

// Accessory thresholds.
const (
	HatActiveDays90 = 30 // strictly more than this
	ScarfStreak     = 10
	FlowerRepos     = 5
)

// AccessoriesFor derives the unlocked accessories from the stats.
func AccessoriesFor(s stats.Stats) Accessories {
	return Accessories{
		Hat:    s.ActiveDays90 > HatActiveDays90,
		Scarf:  s.Streak >= ScarfStreak,
		Flower: s.Repos >= FlowerRepos,
	}
}

// Names returns the Spanish names of the unlocked accessories, in a fixed
// order.
func (a Accessories) Names() []string {
	var out []string
	if a.Hat {
		out = append(out, "gorro")
	}
	if a.Scarf {
		out = append(out, "bufanda")
	}
	if a.Flower {
		out = append(out, "flor")
	}
	return out
}

// Creature is the full state to draw.
type Creature struct {
	Stage       Stage
	Mood        Mood
	Accessories Accessories
}

// FromStats derives the creature from the stats. It is deterministic.
func FromStats(s stats.Stats) Creature {
	return Creature{
		Stage:       StageFor(s.XP),
		Mood:        MoodFor(s),
		Accessories: AccessoriesFor(s),
	}
}
