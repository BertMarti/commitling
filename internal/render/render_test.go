package render

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/stats"
)

func sampleStats() stats.Stats {
	return stats.Stats{XP: 1234, Streak: 6, ActiveDays30: 18, ActiveDays90: 40, Repos: 5, DaysSinceLast: 0}
}

func mustParseXML(t *testing.T, svg []byte) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(svg))
	root := ""
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid XML: %v\n%s", err, svg)
		}
		if se, ok := tok.(xml.StartElement); ok && root == "" {
			root = se.Name.Local
		}
	}
	if root != "svg" {
		t.Fatalf("root element = %q, want svg", root)
	}
}

func TestSVGIsValidXMLForEveryState(t *testing.T) {
	all := []creature.Accessories{{}, {Hat: true, Scarf: true, Flower: true}}
	for _, th := range []Theme{Light, Dark} {
		for _, st := range creature.Stages {
			for _, m := range creature.Moods {
				for _, acc := range all {
					c := Card{User: "octoexample", Stats: sampleStats(), Theme: th,
						Creature: creature.Creature{Stage: st, Mood: m, Accessories: acc}}
					mustParseXML(t, SVG(c))
				}
			}
		}
	}
}

func TestSVGEscapesUserText(t *testing.T) {
	c := NewCard(`<script>&"'`, sampleStats(), Light)
	svg := SVG(c)
	mustParseXML(t, svg)
	if bytes.Contains(svg, []byte("<script>")) {
		t.Fatal("user text was not escaped")
	}
	if !bytes.Contains(svg, []byte("@&lt;script&gt;&amp;&quot;&apos;")) {
		t.Fatalf("escaped user not found in:\n%s", svg)
	}
}

func TestSVGIsDeterministic(t *testing.T) {
	a := SVG(NewCard("octoexample", sampleStats(), Light))
	b := SVG(NewCard("octoexample", sampleStats(), Light))
	if !bytes.Equal(a, b) {
		t.Fatal("same input produced different SVGs")
	}
	if bytes.Equal(a, SVG(NewCard("octoexample", sampleStats(), Dark))) {
		t.Fatal("themes produce the same SVG")
	}
}

func TestSVGContent(t *testing.T) {
	svg := string(SVG(NewCard("octoexample", sampleStats(), Light)))
	for _, want := range []string{
		`width="480" height="200"`,
		`shape-rendering="crispEdges"`,
		"@keyframes",
		"prefers-reduced-motion",
		"Arbusto",
		"Radiante",
		"1.234 XP",
		"6 días",
		"18/30",
		"gorro · flor",
		"#f3efe6",
		"<title",
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG does not contain %q", want)
		}
	}
	if strings.Contains(svg, "<script") {
		t.Error("SVG must not contain scripts")
	}

	dark := string(SVG(NewCard("octoexample", sampleStats(), Dark)))
	if !strings.Contains(dark, `fill="#2b2724" stroke`) {
		t.Error("dark theme background not applied")
	}
}

func TestSleepingAndRadiantExtras(t *testing.T) {
	sleep := string(SVG(NewCard("u", stats.Stats{DaysSinceLast: -1}, Light)))
	if !strings.Contains(sleep, `class="z z1"`) || !strings.Contains(sleep, "Durmiendo") {
		t.Error("sleeping creature should have zzz")
	}
	if strings.Contains(sleep, "blink") {
		t.Error("sleeping creature should not blink")
	}
	rad := string(SVG(NewCard("u", stats.Stats{DaysSinceLast: 0, Streak: 7}, Light)))
	if !strings.Contains(rad, `class="sp"`) || !strings.Contains(rad, "hop") {
		t.Error("radiant creature should sparkle and hop")
	}
	happy := string(SVG(NewCard("u", stats.Stats{DaysSinceLast: 0, Streak: 1}, Light)))
	if !strings.Contains(happy, "blink") {
		t.Error("happy creature should blink")
	}
}

func TestProgressBar(t *testing.T) {
	count := func(svg string, color string) int {
		i := strings.Index(svg, `<path fill="`+color+`" d="M222 118`)
		if i < 0 {
			return 0
		}
		end := strings.Index(svg[i:], `"/>`)
		return strings.Count(svg[i:i+end], "M")
	}
	// Halfway between Sapling (400) and Shrub (1000).
	svg := string(SVG(NewCard("u", stats.Stats{XP: 700, DaysSinceLast: 0}, Light)))
	if got := count(svg, creature.Moss); got != barCells/2 {
		t.Errorf("filled cells = %d, want %d", got, barCells/2)
	}
	svg = string(SVG(NewCard("u", stats.Stats{XP: 0, DaysSinceLast: -1}, Light)))
	if got := count(svg, Light.Line); got != barCells {
		t.Errorf("empty cells = %d, want %d", got, barCells)
	}
	svg = string(SVG(NewCard("u", stats.Stats{XP: 5000, DaysSinceLast: 0}, Light)))
	if !strings.Contains(svg, "fase máxima") || count(svg, creature.Moss) != barCells {
		t.Error("last stage should show a full bar")
	}
}

func TestThousands(t *testing.T) {
	cases := map[int]string{0: "0", 12: "12", 999: "999", 1000: "1.000", 1234567: "1.234.567", -2500: "-2.500"}
	for n, want := range cases {
		if got := Thousands(n); got != want {
			t.Errorf("Thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestThemeByName(t *testing.T) {
	for name, want := range map[string]string{"": "light", "light": "light", "DARK": "dark"} {
		th, ok := ThemeByName(name)
		if !ok || th.Name != want {
			t.Errorf("ThemeByName(%q) = %q, %v", name, th.Name, ok)
		}
	}
	if _, ok := ThemeByName("sepia"); ok {
		t.Error("unknown theme accepted")
	}
}

func TestEscapeDropsInvalidXMLChars(t *testing.T) {
	if got := Escape("a\x00b\x1fc"); got != "abc" {
		t.Errorf("Escape = %q", got)
	}
}
