package render

import (
	"bytes"
	"encoding/xml"
	"html"
	"io"
	"regexp"
	"strconv"
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

// textBox is one <text> element with its estimated horizontal extent.
type textBox struct {
	content string
	y       int
	x0, x1  float64
}

var textRe = regexp.MustCompile(`<text class="t[^"]*" x="(\d+)" y="(\d+)" font-size="([\d.]+)" fill="[^"]*"( text-anchor="end")?>([^<]*)</text>`)

func textBoxes(t *testing.T, svg []byte) []textBox {
	t.Helper()
	var out []textBox
	for _, m := range textRe.FindAllStringSubmatch(string(svg), -1) {
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		size, _ := strconv.ParseFloat(m[3], 64)
		content := html.UnescapeString(m[5])
		w := textWidth(content, size)
		b := textBox{content: content, y: y, x0: float64(x), x1: float64(x) + w}
		if m[4] != "" {
			b.x0, b.x1 = float64(x)-w, float64(x)
		}
		out = append(out, b)
	}
	if len(out) < 9 {
		t.Fatalf("found only %d <text> elements, the regexp is out of date:\n%s", len(out), svg)
	}
	return out
}

// checkTextFits fails if any text leaves the right-hand panel or overlaps
// another text on the same line.
func checkTextFits(t *testing.T, name string, svg []byte) {
	t.Helper()
	boxes := textBoxes(t, svg)
	for i, a := range boxes {
		if a.x0 < panelX || a.x1 > panelEnd {
			t.Errorf("%s: %q spans %.0f-%.0f, outside the panel %d-%d", name, a.content, a.x0, a.x1, panelX, panelEnd)
		}
		for _, b := range boxes[i+1:] {
			if a.y == b.y && a.x0 < b.x1 && b.x0 < a.x1 {
				t.Errorf("%s: %q and %q overlap", name, a.content, b.content)
			}
		}
	}
}

func TestLongTextStaysInsideTheCard(t *testing.T) {
	user39 := strings.Repeat("a", 39)
	huge := stats.Stats{XP: 1234567, Streak: 90, ActiveDays30: 30, ActiveDays90: 90, Repos: 1234567, DaysSinceLast: 0}
	all := creature.Accessories{Hat: true, Scarf: true, Flower: true}
	for _, th := range []Theme{Light, Dark} {
		for _, st := range creature.Stages {
			for _, m := range creature.Moods {
				for _, user := range []string{"", "u", "octoexample", user39, strings.Repeat("W", 80)} {
					c := Card{User: user, Stats: huge, Theme: th,
						Creature: creature.Creature{Stage: st, Mood: m, Accessories: all}}
					checkTextFits(t, th.Name+"/"+st.Name()+"/"+m.Name()+"/"+user, SVG(c))
				}
			}
		}
	}
}

func TestLongUserIsShrunkNotCut(t *testing.T) {
	user39 := strings.Repeat("a", 39)
	svg := string(SVG(NewCard(user39, stats.Stats{XP: 1234567, DaysSinceLast: 0}, Light)))
	if !strings.Contains(svg, "@"+user39) {
		t.Errorf("a 39-character login should be shown in full:\n%s", svg)
	}
	if !strings.Contains(svg, "1.234.567 XP") {
		t.Error("7-digit XP should be shown in full")
	}
	// Absurd input is cut with an ellipsis instead of overflowing.
	long := string(SVG(NewCard(strings.Repeat("W", 80), stats.Stats{DaysSinceLast: 0}, Light)))
	if !strings.Contains(long, "…") {
		t.Error("an 80-character login should be truncated with an ellipsis")
	}
}

func TestFitAndTruncate(t *testing.T) {
	if s, sz := fit([]string{"hola"}, []float64{11, 9}, 100); s != "hola" || sz != 11 {
		t.Errorf("fit = %q, %v", s, sz)
	}
	// 10 chars at 11 px do not fit in 60 px, but do at 9 px.
	if s, sz := fit([]string{"0123456789"}, []float64{11, 9}, 60); s != "0123456789" || sz != 9 {
		t.Errorf("fit = %q, %v", s, sz)
	}
	// The second candidate is used when the first never fits.
	if s, _ := fit([]string{"muy largo texto", "corto"}, []float64{10}, 40); s != "corto" {
		t.Errorf("fit = %q", s)
	}
	if got := truncate("abcdefghij", 10, 31); got != "abcd…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abc", 10, 1); got != "…" {
		t.Errorf("truncate = %q", got)
	}
}

func TestFontStackHasFallbacks(t *testing.T) {
	svg := string(SVG(NewCard("u", sampleStats(), Light)))
	want := `ui-monospace,SFMono-Regular,Menlo,Consolas,"DejaVu Sans Mono","Liberation Mono",monospace`
	if !strings.Contains(svg, want) {
		t.Errorf("font stack not found: %s", want)
	}
}
