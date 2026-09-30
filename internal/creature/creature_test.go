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
	for _, sp := range AllSpecies {
		for _, st := range Stages {
			for y, row := range artFor(sp, st).body {
				if len(row) != Size {
					t.Errorf("%s/%s row %d has %d pixels, want %d", sp.Name(), st.Name(), y, len(row), Size)
				}
				for x := 0; x < len(row); x++ {
					if !valid[row[x]] {
						t.Errorf("%s/%s (%d,%d): invalid pixel %q", sp.Name(), st.Name(), x, y, row[x])
					}
				}
			}
		}
	}
}

// The face, scarf and accessories must land on the creature, not on air.
func TestAnchorsFitTheBody(t *testing.T) {
	for _, sp := range AllSpecies {
		for _, st := range Stages {
			name := sp.Name() + "/" + sp.StageName(st)
			a := artFor(sp, st)
			g := parse(a.body)
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 8; dx++ {
					x, y := a.face.X+dx, a.face.Y+dy
					if p := g[y][x]; p == Clear || p == PInk {
						t.Errorf("%s: face pixel (%d,%d) is %q, want a fill colour", name, x, y, p)
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
				t.Errorf("%s: scarf row %d is too narrow (%d pixels)", name, a.scarfRow, filled)
			}
			// The scarf row must be one solid run, or the scarf would
			// bridge gaps in the body.
			first, last := -1, -1
			for x := 0; x < Size; x++ {
				if g[a.scarfRow][x] != Clear {
					if first < 0 {
						first = x
					}
					last = x
				}
			}
			if last-first+1 != filled {
				t.Errorf("%s: scarf row %d has gaps", name, a.scarfRow)
			}
			if a.hat.X < 0 || a.hat.Y < 0 || a.hat.X+len(hatArt[0]) > Size || a.hat.Y+len(hatArt) > Size {
				t.Errorf("%s: hat out of bounds at %+v", name, a.hat)
			}
			if a.flower.X < 0 || a.flower.Y < 0 || a.flower.X+3 > Size || a.flower.Y+3 > Size {
				t.Errorf("%s: flower out of bounds at %+v", name, a.flower)
			}
			// The hat and the flower must not cover the face.
			if a.hat.Y+len(hatArt) > a.face.Y {
				t.Errorf("%s: hat (rows %d-%d) reaches the face (from row %d)", name, a.hat.Y, a.hat.Y+len(hatArt)-1, a.face.Y)
			}
			if a.flower.Y+3 > a.face.Y && a.flower.X < a.face.X+8 && a.flower.X+3 > a.face.X {
				t.Errorf("%s: flower at %+v covers the face", name, a.flower)
			}
		}
	}
}

// Only the five colours of the palette may appear.
func TestDrawUsesPalette(t *testing.T) {
	all := Accessories{Hat: true, Scarf: true, Flower: true}
	for _, sp := range AllSpecies {
		for _, st := range Stages {
			for _, m := range Moods {
				s := Draw(Creature{Species: sp, Stage: st, Mood: m, Accessories: all})
				for _, g := range []*Grid{&s.Outline, &s.Body, &s.Eyes} {
					for y := range g {
						for x := range g[y] {
							if p := g[y][x]; p != Clear && Color(p) == "" {
								t.Fatalf("%s/%s/%s: pixel %q at (%d,%d) is not in the palette", sp.Name(), st.Name(), m.Name(), p, x, y)
							}
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

func TestSpeciesNamesAndSlugs(t *testing.T) {
	if len(AllSpecies) != 2 || AllSpecies[0] != MossSprout {
		t.Fatalf("AllSpecies = %v, want the default first", AllSpecies)
	}
	if (Creature{}).Species != MossSprout {
		t.Error("the zero value must be the default species")
	}
	seen := map[string]bool{}
	for _, sp := range AllSpecies {
		if sp.Name() == "" || sp.Slug() == "" || seen[sp.Slug()] {
			t.Errorf("species %d has a bad or repeated name/slug: %q %q", sp, sp.Name(), sp.Slug())
		}
		seen[sp.Slug()] = true
		for _, st := range Stages {
			if sp.StageName(st) == "" {
				t.Errorf("%s has no name for stage %d", sp.Name(), st)
			}
		}
	}
	// The default species keeps the original stage names.
	for _, st := range Stages {
		if MossSprout.StageName(st) != st.Name() {
			t.Errorf("moss stage %d is %q, want %q", st, MossSprout.StageName(st), st.Name())
		}
	}
	if Mushroom.StageName(Seed) != "Espora" || Mushroom.StageName(Ancient) != "Corro de setas" {
		t.Error("unexpected mushroom stage names")
	}
}

func TestSpeciesByName(t *testing.T) {
	ok := map[string]Species{
		"": MossSprout, "  ": MossSprout, "moss": MossSprout, "MOSS": MossSprout, "musgo": MossSprout, "Brote de musgo": MossSprout,
		"mushroom": Mushroom, " Mushroom ": Mushroom, "hongo": Mushroom, "seta": Mushroom,
	}
	for in, want := range ok {
		got, found := SpeciesByName(in)
		if !found || got != want {
			t.Errorf("SpeciesByName(%q) = %v, %v; want %v", in, got, found, want)
		}
	}
	for _, bad := range []string{"dragon", "moss mushroom", "../x", "1"} {
		if _, found := SpeciesByName(bad); found {
			t.Errorf("SpeciesByName(%q) should not be found", bad)
		}
	}
}

func TestFromStatsAs(t *testing.T) {
	s := stats.Stats{XP: 450, DaysSinceLast: 0, Streak: 6}
	def := FromStats(s)
	if def.Species != MossSprout {
		t.Fatalf("FromStats gives species %v, want the default", def.Species)
	}
	m := FromStatsAs(Mushroom, s)
	if m.Species != Mushroom || m.Stage != def.Stage || m.Mood != def.Mood || m.Accessories != def.Accessories {
		t.Fatalf("FromStatsAs = %+v, want the same state as %+v with another species", m, def)
	}
	if m.StageName() != "Seta" || def.StageName() != "Retoño" {
		t.Errorf("stage names: %q and %q", m.StageName(), def.StageName())
	}
}

// Every stage, mood and accessory set draws differently for each species,
// deterministically, and the species look different from each other.
func TestSpeciesDrawDifferently(t *testing.T) {
	all := Accessories{Hat: true, Scarf: true, Flower: true}
	for _, st := range Stages {
		for _, m := range Moods {
			for _, acc := range []Accessories{{}, all} {
				moss := Draw(Creature{Species: MossSprout, Stage: st, Mood: m, Accessories: acc})
				mush := Draw(Creature{Species: Mushroom, Stage: st, Mood: m, Accessories: acc})
				if moss == mush {
					t.Errorf("stage %d mood %d: both species draw the same", st, m)
				}
				if again := Draw(Creature{Species: Mushroom, Stage: st, Mood: m, Accessories: acc}); again != mush {
					t.Errorf("stage %d mood %d: Draw is not deterministic", st, m)
				}
			}
		}
	}
	// Within a species, every stage is different from the others.
	for _, sp := range AllSpecies {
		seen := map[Sprite]Stage{}
		for _, st := range Stages {
			d := Draw(Creature{Species: sp, Stage: st, Mood: Happy})
			if prev, dup := seen[d]; dup {
				t.Errorf("%s: stages %d and %d look the same", sp.Name(), prev, st)
			}
			seen[d] = st
		}
	}
}

// The mushroom is its own design: it must not reuse the moss sprout's maps.
func TestMushroomMapsAreNotTheMossOnes(t *testing.T) {
	for _, st := range Stages {
		if mushroomArt[st].body == mossArt[st].body {
			t.Errorf("stage %d: mushroom body equals the moss body", st)
		}
	}
}
