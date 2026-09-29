package creature

import (
	"testing"

	"github.com/BertMarti/commitling/internal/stats"
)

func TestStageThresholds(t *testing.T) {
	cases := []struct {
		xp   int
		want Stage
	}{
		{0, Seed}, {99, Seed},
		{100, Sprout}, {399, Sprout},
		{400, Sapling}, {999, Sapling},
		{1000, Shrub}, {2499, Shrub},
		{2500, Ancient}, {1_000_000, Ancient},
		{-5, Seed},
	}
	for _, c := range cases {
		if got := StageFor(c.xp); got != c.want {
			t.Errorf("StageFor(%d) = %s, want %s", c.xp, got.Name(), c.want.Name())
		}
	}
}

func TestProgress(t *testing.T) {
	cases := []struct {
		xp   int
		want float64
	}{
		{0, 0}, {50, 0.5}, {100, 0}, {250, 0.5}, {2500, 1}, {9999, 1},
	}
	for _, c := range cases {
		if got := Progress(c.xp); got != c.want {
			t.Errorf("Progress(%d) = %v, want %v", c.xp, got, c.want)
		}
	}
}

func TestStageNext(t *testing.T) {
	if n, ok := Seed.Next(); !ok || n != Sprout {
		t.Fatalf("Seed.Next() = %v, %v", n, ok)
	}
	if _, ok := Ancient.Next(); ok {
		t.Fatal("Ancient.Next() should not exist")
	}
}

func TestMoodThresholds(t *testing.T) {
	cases := []struct {
		name  string
		since int
		str   int
		want  Mood
	}{
		{"sin actividad", -1, 0, Sleeping},
		{"7 días sin actividad", 7, 0, Sleeping},
		{"6 días sin actividad", 6, 0, Bored},
		{"3 días sin actividad", 3, 0, Bored},
		{"2 días sin actividad", 2, 0, Happy},
		{"hoy, racha 4", 0, 4, Happy},
		{"hoy, racha 5", 0, 5, Radiant},
		{"ayer, racha 9", 1, 9, Radiant},
	}
	for _, c := range cases {
		s := stats.Stats{DaysSinceLast: c.since, Streak: c.str}
		if got := MoodFor(s); got != c.want {
			t.Errorf("%s: MoodFor = %s, want %s", c.name, got.Name(), c.want.Name())
		}
	}
}

func TestAccessories(t *testing.T) {
	cases := []struct {
		s    stats.Stats
		want Accessories
	}{
		{stats.Stats{}, Accessories{}},
		{stats.Stats{ActiveDays90: 30}, Accessories{}},
		{stats.Stats{ActiveDays90: 31}, Accessories{Hat: true}},
		{stats.Stats{Streak: 9}, Accessories{}},
		{stats.Stats{Streak: 10}, Accessories{Scarf: true}},
		{stats.Stats{Repos: 4}, Accessories{}},
		{stats.Stats{Repos: 5}, Accessories{Flower: true}},
		{stats.Stats{ActiveDays90: 60, Streak: 12, Repos: 8}, Accessories{true, true, true}},
	}
	for _, c := range cases {
		if got := AccessoriesFor(c.s); got != c.want {
			t.Errorf("AccessoriesFor(%+v) = %+v, want %+v", c.s, got, c.want)
		}
	}
	if got := (Accessories{Hat: true, Flower: true}).Names(); len(got) != 2 || got[0] != "gorro" || got[1] != "flor" {
		t.Errorf("Names() = %v", got)
	}
}

func TestFromStats(t *testing.T) {
	c := FromStats(stats.Stats{XP: 1200, DaysSinceLast: 0, Streak: 11, ActiveDays90: 40, Repos: 2})
	want := Creature{Stage: Shrub, Mood: Radiant, Accessories: Accessories{Hat: true, Scarf: true}}
	if c != want {
		t.Fatalf("FromStats = %+v, want %+v", c, want)
	}
}

func TestPixelMapsAreWellFormed(t *testing.T) {
	valid := map[byte]bool{Clear: true, PInk: true, PPaper: true, PMoss: true, PHoney: true, PRed: true}
	for _, st := range Stages {
		for y, row := range art[st].body {
			if len(row) != Size {
				t.Errorf("%s row %d has %d pixels, want %d", st.Name(), y, len(row), Size)
			}
			for x := 0; x < len(row); x++ {
				if !valid[row[x]] {
					t.Errorf("%s (%d,%d): invalid pixel %q", st.Name(), x, y, row[x])
				}
			}
		}
	}
}

// The face, scarf and accessories must land on the creature, not on air.
func TestAnchorsFitTheBody(t *testing.T) {
	for _, st := range Stages {
		a := art[st]
		g := parse(a.body)
		for dy := 0; dy < 4; dy++ {
			for dx := 0; dx < 8; dx++ {
				x, y := a.face.X+dx, a.face.Y+dy
				if p := g[y][x]; p == Clear || p == PInk {
					t.Errorf("%s: face pixel (%d,%d) is %q, want a fill colour", st.Name(), x, y, p)
				}
			}
		}
		filled := 0
		for x := 0; x < Size; x++ {
			if g[a.scarfRow][x] != Clear {
				filled++
			}
		}
		if filled < 6 {
			t.Errorf("%s: scarf row %d is too narrow (%d pixels)", st.Name(), a.scarfRow, filled)
		}
		if a.hat.X < 0 || a.hat.Y < 0 || a.hat.X+len(hatArt[0]) > Size || a.hat.Y+len(hatArt) > Size {
			t.Errorf("%s: hat out of bounds at %+v", st.Name(), a.hat)
		}
		if a.flower.X < 0 || a.flower.Y < 0 || a.flower.X+3 > Size || a.flower.Y+3 > Size {
			t.Errorf("%s: flower out of bounds at %+v", st.Name(), a.flower)
		}
	}
}

// Only the five colours of the palette may appear.
func TestDrawUsesPalette(t *testing.T) {
	all := Accessories{Hat: true, Scarf: true, Flower: true}
	for _, st := range Stages {
		for _, m := range Moods {
			sp := Draw(Creature{Stage: st, Mood: m, Accessories: all})
			for _, g := range []*Grid{&sp.Outline, &sp.Body, &sp.Eyes} {
				for y := range g {
					for x := range g[y] {
						if p := g[y][x]; p != Clear && Color(p) == "" {
							t.Fatalf("%s/%s: pixel %q at (%d,%d) is not in the palette", st.Name(), m.Name(), p, x, y)
						}
					}
				}
			}
		}
	}
}

func TestDrawChangesWithMoodAndAccessories(t *testing.T) {
	base := Draw(Creature{Stage: Sapling, Mood: Happy})
	if Draw(Creature{Stage: Sapling, Mood: Happy}) != base {
		t.Fatal("Draw is not deterministic")
	}
	for _, m := range []Mood{Sleeping, Bored, Radiant} {
		if Draw(Creature{Stage: Sapling, Mood: m}) == base {
			t.Errorf("mood %s draws the same as Happy", m.Name())
		}
	}
	for _, acc := range []Accessories{{Hat: true}, {Scarf: true}, {Flower: true}} {
		if Draw(Creature{Stage: Sapling, Mood: Happy, Accessories: acc}) == base {
			t.Errorf("accessory %+v is not drawn", acc)
		}
	}
}
