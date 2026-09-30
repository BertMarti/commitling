package og

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
)

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
	return img
}

func TestDimensions(t *testing.T) {
	data, err := PNG(Hero)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 1200 || cfg.Height != 630 {
		t.Fatalf("size = %dx%d, want 1200x630", cfg.Width, cfg.Height)
	}
	if len(data) > 200*1024 {
		t.Errorf("PNG is %d bytes; keep it small for link previews", len(data))
	}
}

func TestDeterministic(t *testing.T) {
	a, err := PNG(Hero)
	if err != nil {
		t.Fatal(err)
	}
	b, err := PNG(Hero)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("two renders give different bytes")
	}
	other, err := PNG(creature.Creature{Stage: creature.Seed, Mood: creature.Sleeping})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, other) {
		t.Fatal("a different creature gives the same image")
	}
}

// Only the colours of the palette appear: nothing is anti-aliased or blended.
func TestOnlyPaletteColours(t *testing.T) {
	data, _ := PNG(Hero)
	img := decode(t, data)
	allowed := map[color.RGBA]bool{}
	for _, c := range palette {
		allowed[c.(color.RGBA)] = true
	}
	seen := map[color.RGBA]int{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)}
			if !allowed[c] {
				t.Fatalf("pixel (%d,%d) has colour %v outside the palette", x, y, c)
			}
			seen[c]++
		}
	}
	// The creature's five colours are all used (hero has ink, paper, moss,
	// honey and accent) plus the two greys of the card.
	if len(seen) != len(palette) {
		t.Errorf("%d colours used, want all %d of the palette", len(seen), len(palette))
	}
}

// The big creature is drawn at an integer scale: every sprite pixel is a
// solid scale×scale block, so the edges are crisp.
func TestSpriteIsScaledInWholeBlocks(t *testing.T) {
	const scale, x0, y0 = heroScale, heroX, heroY
	img := Image(Hero)
	sp := creature.Draw(Hero)
	for gy := 0; gy < creature.Size; gy++ {
		for gx := 0; gx < creature.Size; gx++ {
			// Colour expected from the layers, top layer wins.
			want := uint8(iPaper)
			paint := false
			for _, g := range []*creature.Grid{&sp.Outline, &sp.Body, &sp.Eyes} {
				if idx, ok := colorIndex(g[gy][gx]); ok {
					want, paint = idx, true
				}
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					got := img.ColorIndexAt(x0+gx*scale+dx, y0+gy*scale+dy)
					if paint && got != want {
						t.Fatalf("sprite pixel (%d,%d) block offset (%d,%d): index %d, want %d", gx, gy, dx, dy, got, want)
					}
				}
			}
		}
	}
}

// Every stage of every species appears, small, in the rows under the tagline.
func TestStageRowsShowAllStages(t *testing.T) {
	img := Image(Hero)
	for i, sp := range creature.AllSpecies {
		x := textX
		for _, st := range creature.Stages {
			sprite := creature.Draw(creature.Creature{Species: sp, Stage: st, Mood: creature.Happy})
			ink := 0
			for gy := 0; gy < creature.Size; gy++ {
				for gx := 0; gx < creature.Size; gx++ {
					if sprite.Outline[gy][gx] == creature.PInk {
						if img.ColorIndexAt(x+gx*stageScale, stageRowY(i)+gy*stageScale) != iInk {
							t.Fatalf("%s/%s: outline pixel (%d,%d) is not ink", sp.Name(), sp.StageName(st), gx, gy)
						}
						ink++
					}
				}
			}
			if ink == 0 {
				t.Fatalf("%s/%s has no outline?", sp.Name(), sp.StageName(st))
			}
			x += creature.Size*stageScale + stageGap
		}
	}
	// The rows fit above the site name and inside the frame.
	last := stageRowY(len(creature.AllSpecies)-1) + creature.Size*stageScale
	if last >= siteY {
		t.Errorf("stage rows end at y=%d, overlapping the site name at y=%d", last, siteY)
	}
	if siteY+glyphH*3 > Height-24-6 {
		t.Error("the site name touches the frame")
	}
	if end := textX + 5*(creature.Size*stageScale+stageGap) - stageGap; end > Width-24-6-16 {
		t.Errorf("stage rows reach x=%d, too close to the frame", end)
	}
}

func TestTextIsDrawnInsideTheCard(t *testing.T) {
	img := Image(Hero)
	// The name is drawn in ink somewhere in its box.
	w := TextWidth(Name, 10)
	if w != (len(Name)*advance-1)*10 {
		t.Fatalf("TextWidth = %d", w)
	}
	found := false
	for y := 130; y < 130+glyphH*10 && !found; y++ {
		for x := textX; x < textX+w; x++ {
			if img.ColorIndexAt(x, y) == iInk {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("the name is not drawn")
	}
	// Nothing spills beyond the frame on the right.
	for _, line := range []struct {
		s     string
		scale int
	}{{Name, 10}, {Tagline1, 4}, {Tagline2, 4}, {Site, 3}} {
		if end := textX + TextWidth(line.s, line.scale); end > Width-24-6-16 {
			t.Errorf("%q ends at x=%d, too close to the frame", line.s, end)
		}
	}
}

func TestFontHasEveryCharacterOfTheCard(t *testing.T) {
	for _, s := range []string{Name, Tagline1, Tagline2, Site} {
		for _, r := range s {
			if _, ok := font[r]; !ok {
				t.Errorf("font lacks %q (in %q)", r, s)
			}
		}
	}
	for r, g := range font {
		if len(g) != 7 && len(g) != glyphH {
			t.Errorf("glyph %q has %d rows", r, len(g))
		}
		for i, row := range g {
			if len(row) != glyphW {
				t.Errorf("glyph %q row %d has width %d", r, i, len(row))
			}
		}
	}
	// Digits, capitals and lower case are all there.
	for _, set := range []string{"0123456789", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "abcdefghijklmnopqrstuvwxyz"} {
		for _, r := range set {
			if _, ok := font[r]; !ok {
				t.Errorf("font lacks %q", r)
			}
		}
	}
	// Unknown characters fall back to '?'.
	if glyph('€') != glyph('?') {
		t.Error("unknown rune should draw as '?'")
	}
}
