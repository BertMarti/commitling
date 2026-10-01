package generate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/stats"
)

func demoStats(day int) stats.Stats {
	return stats.Compute(demoActivities(day), demoNow(day))
}

func TestDemoIsDeterministic(t *testing.T) {
	a, err := Demo(37, Options{Species: "mushroom", Theme: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Demo(37, Options{Species: "mushroom", Theme: "dark"})
	if !bytes.Equal(a.SVG, b.SVG) || a.Description != b.Description {
		t.Error("the same day and options must draw the same card")
	}
	if c, _ := Demo(38, Options{Species: "mushroom", Theme: "dark"}); bytes.Equal(a.SVG, c.SVG) && a.Phase == c.Phase {
		t.Error("two different days drew exactly the same card")
	}
}

func TestDemoDayIsClamped(t *testing.T) {
	first, _ := Demo(0, Options{})
	below, _ := Demo(-5, Options{})
	last, _ := Demo(DemoDays, Options{})
	above, _ := Demo(DemoDays+400, Options{})
	if !bytes.Equal(first.SVG, below.SVG) || below.Day != 0 {
		t.Error("a negative day must draw day 0")
	}
	if !bytes.Equal(last.SVG, above.SVG) || above.Day != DemoDays || above.Days != DemoDays {
		t.Errorf("a day past the end must draw the last one, got day %d of %d", above.Day, above.Days)
	}
}

func TestDemoValidatesOptions(t *testing.T) {
	for name, o := range map[string]Options{
		"species": {Species: "dragon"},
		"theme":   {Theme: "sepia"},
		"user":    {User: "no es válido"},
	} {
		if _, err := Demo(10, o); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// The user on the card is the fictitious one, whatever the options say: the
// demo never shows or claims data of a real person.
func TestDemoIsAboutAFictitiousUser(t *testing.T) {
	f, _ := Demo(60, Options{User: "BertMarti"})
	if f.Login != DemoUser || strings.Contains(string(f.SVG), "BertMarti") {
		t.Errorf("login %q: the demo must use %s", f.Login, DemoUser)
	}
}

// The point of the demo: in DemoDays days the creature is born, grows through
// every stage, goes through every mood and unlocks every accessory.
func TestDemoShowsEverything(t *testing.T) {
	stages := map[creature.Stage]int{}
	moods := map[creature.Mood]int{}
	var acc creature.Accessories
	prevXP, prevStage := -1, creature.Seed
	for d := 0; d <= DemoDays; d++ {
		st := demoStats(d)
		if st.XP < prevXP {
			t.Fatalf("day %d: XP went down from %d to %d", d, prevXP, st.XP)
		}
		prevXP = st.XP
		c := creature.FromStats(st)
		if c.Stage < prevStage {
			t.Fatalf("day %d: the creature shrank", d)
		}
		prevStage = c.Stage
		if _, seen := stages[c.Stage]; !seen {
			stages[c.Stage] = d // first day of the stage
		}
		moods[c.Mood]++
		acc.Hat = acc.Hat || c.Accessories.Hat
		acc.Scarf = acc.Scarf || c.Accessories.Scarf
		acc.Flower = acc.Flower || c.Accessories.Flower
	}
	if len(stages) != len(creature.Stages) {
		t.Errorf("stages reached: %v, want all %d", stages, len(creature.Stages))
	}
	if len(moods) != len(creature.Moods) {
		t.Errorf("moods seen: %v, want all %d", moods, len(creature.Moods))
	}
	if !acc.Hat || !acc.Scarf || !acc.Flower {
		t.Errorf("accessories unlocked: %+v, want all three", acc)
	}
	// Born as a seed on day 0 and a tree in the last third, so the timelapse
	// has time to show it.
	if first, ok := stages[creature.Seed]; !ok || first != 0 {
		t.Error("the creature must start as a seed")
	}
	t.Logf("first day of each stage: %v; days per mood: %v", stages, moods)
	if stages[creature.Ancient] > DemoDays*3/4 {
		t.Errorf("the ancient tree arrives on day %d of %d: too late to be seen", stages[creature.Ancient], DemoDays)
	}
}

// Phase names follow the species, and Description is the one of the card.
func TestDemoPhase(t *testing.T) {
	moss, _ := Demo(0, Options{})
	mush, _ := Demo(0, Options{Species: "mushroom"})
	if moss.Phase != creature.MossSprout.StageName(creature.Seed) || mush.Phase != creature.Mushroom.StageName(creature.Seed) {
		t.Errorf("phases %q and %q", moss.Phase, mush.Phase)
	}
	end, _ := Demo(DemoDays, Options{})
	if end.Phase != creature.MossSprout.StageName(creature.Ancient) {
		t.Errorf("last phase %q", end.Phase)
	}
	if end.Description == "" || !bytes.Contains(end.SVG, []byte("<svg")) {
		t.Error("the frame must carry an SVG and its description")
	}
}
