// Package render draws the commitling card as a self-contained animated SVG.
//
// The animation is plain CSS inside the SVG (GitHub does not run JavaScript
// in READMEs) and is disabled with prefers-reduced-motion. The output only
// depends on its input, so the same data always yields the same bytes.
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/stats"
)

// Card size in pixels.
const (
	Width  = 480
	Height = 200
)

// Creature placement: a 16×16 map drawn with 10 px pixels.
const (
	pixel   = 10
	originX = 20
	originY = 16
	groundY = originY + creature.Size*pixel
)

// Right-hand panel.
const (
	panelX    = 222
	panelEnd  = 460
	barCells  = 24
	barCellW  = 8
	barGap    = 2
	barHeight = 8
)

// Theme holds the card colours. The creature keeps its own palette.
type Theme struct {
	Name  string
	Bg    string
	Ink   string
	Muted string
	Line  string
	// Outline is the colour of the creature's silhouette (one of the five
	// creature colours).
	Outline string
}

// Themes.
var (
	Light = Theme{Name: "light", Bg: "#f3efe6", Ink: "#2b2724", Muted: "#8a8178", Line: "#ddd5c6", Outline: creature.Ink}
	Dark  = Theme{Name: "dark", Bg: "#2b2724", Ink: "#f3efe6", Muted: "#a39a8e", Line: "#4a433d", Outline: creature.Paper}
)

// ThemeByName returns the theme called name ("light" or "dark").
func ThemeByName(name string) (Theme, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "light":
		return Light, true
	case "dark":
		return Dark, true
	}
	return Theme{}, false
}

// Card is everything needed to draw.
type Card struct {
	User     string // GitHub login, shown as @user; may be empty
	Stats    stats.Stats
	Creature creature.Creature
	Theme    Theme
}

// NewCard derives the creature from the stats.
func NewCard(user string, s stats.Stats, t Theme) Card {
	return Card{User: user, Stats: s, Creature: creature.FromStats(s), Theme: t}
}

// Escape escapes text for XML content and attribute values.
func Escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			// Drop characters that are not allowed in XML 1.0.
			if r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r != 0xFFFE && r != 0xFFFF) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// Thousands formats n with a dot as thousands separator (Spanish style).
func Thousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// Description is a plain-text summary used for <desc> and alt texts.
func Description(c Card) string {
	cr := c.Creature
	s := fmt.Sprintf("%s, %s. %s XP, racha de %s, %d días activos en los últimos 30",
		cr.Stage.Name(), strings.ToLower(cr.Mood.Name()), Thousands(c.Stats.XP),
		plural(c.Stats.Streak, "día", "días"), c.Stats.ActiveDays30)
	if acc := cr.Accessories.Names(); len(acc) > 0 {
		s += ". Accesorios: " + strings.Join(acc, ", ")
	}
	return s + "."
}

// SVG draws the card.
func SVG(c Card) []byte {
	t := c.Theme
	if t.Bg == "" {
		t = Light
	}
	cr := c.Creature
	sp := creature.Draw(cr)
	var b strings.Builder

	title := "commitling"
	if c.User != "" {
		title = "commitling de @" + c.User
	}

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="cl-title cl-desc">`, Width, Height, Width, Height)
	b.WriteString("\n")
	fmt.Fprintf(&b, "<title id=\"cl-title\">%s</title>\n", Escape(title))
	fmt.Fprintf(&b, "<desc id=\"cl-desc\">%s</desc>\n", Escape(Description(c)))
	writeStyle(&b, c, sp)

	// Paper and frame.
	fmt.Fprintf(&b, "<rect x=\"0.5\" y=\"0.5\" width=\"%d\" height=\"%d\" rx=\"8\" fill=\"%s\" stroke=\"%s\"/>\n", Width-1, Height-1, t.Bg, t.Line)

	// Ground and divider.
	fmt.Fprintf(&b, "<path d=\"M28 %dh144\" stroke=\"%s\" stroke-width=\"2\" stroke-dasharray=\"8 4\" shape-rendering=\"crispEdges\"/>\n", groundY+1, t.Line)
	fmt.Fprintf(&b, "<path d=\"M202.5 28v144\" stroke=\"%s\" stroke-dasharray=\"2 4\" shape-rendering=\"crispEdges\"/>\n", t.Line)

	// Creature.
	b.WriteString("<g class=\"bob\" shape-rendering=\"crispEdges\">\n")
	writeGrid(&b, &sp.Outline, "", t.Outline)
	writeGrid(&b, &sp.Body, "", "")
	writeGrid(&b, &sp.Eyes, "eyes", "")
	b.WriteString("</g>\n")

	switch cr.Mood {
	case creature.Sleeping:
		writeZzz(&b, sp, t)
	case creature.Radiant:
		writeSparkles(&b, sp)
	}

	writePanel(&b, c, t)
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

// bbox returns the bounding box (in pixels of the map) of a grid.
func bbox(g *creature.Grid) (minX, minY, maxX, maxY int, ok bool) {
	minX, minY = creature.Size, creature.Size
	maxX, maxY = -1, -1
	for y := 0; y < creature.Size; y++ {
		for x := 0; x < creature.Size; x++ {
			if g[y][x] == creature.Clear || g[y][x] == 0 {
				continue
			}
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
		}
	}
	return minX, minY, maxX, maxY, maxX >= 0
}

func writeStyle(b *strings.Builder, c Card, sp creature.Sprite) {
	b.WriteString("<style>\n")
	b.WriteString(`.t{font-family:ui-monospace,SFMono-Regular,"SF Mono",Menlo,Consolas,"Liberation Mono",monospace}` + "\n")
	b.WriteString(".b{font-weight:700}\n")

	// Whole body: breathing or hopping, depending on the mood.
	switch c.Creature.Mood {
	case creature.Radiant:
		b.WriteString(".bob{animation:hop 1.6s ease-in-out infinite}\n")
		b.WriteString("@keyframes hop{0%,55%,100%{transform:translateY(0)}25%{transform:translateY(-8px)}}\n")
	case creature.Happy:
		b.WriteString(".bob{animation:breathe 3.2s ease-in-out infinite}\n")
		b.WriteString("@keyframes breathe{0%,100%{transform:translateY(0)}50%{transform:translateY(-3px)}}\n")
	default:
		b.WriteString(".bob{animation:breathe 5.6s ease-in-out infinite}\n")
		b.WriteString("@keyframes breathe{0%,100%{transform:translateY(0)}50%{transform:translateY(-2px)}}\n")
	}

	if c.Creature.Mood.Blinks() {
		if x0, y0, x1, y1, ok := bbox(&sp.Eyes); ok {
			cx := originX + (x0+x1+1)*pixel/2
			cy := originY + (y0+y1+1)*pixel/2
			fmt.Fprintf(b, ".eyes{transform-origin:%dpx %dpx;animation:blink 4.8s infinite}\n", cx, cy)
			b.WriteString("@keyframes blink{0%,91%,97%,100%{transform:scaleY(1)}94%{transform:scaleY(.1)}}\n")
		}
	}
	switch c.Creature.Mood {
	case creature.Sleeping:
		b.WriteString(".z{animation:zz 3.6s ease-in-out infinite;opacity:.9}\n")
		b.WriteString(".z2{animation-delay:1.2s}.z3{animation-delay:2.4s}\n")
		b.WriteString("@keyframes zz{0%{opacity:0;transform:translate(0,6px)}35%{opacity:.9}100%{opacity:0;transform:translate(6px,-10px)}}\n")
	case creature.Radiant:
		b.WriteString(".sp{animation:tw 1.8s ease-in-out infinite}.sp2{animation-delay:.9s}\n")
		b.WriteString("@keyframes tw{0%,100%{opacity:.2}50%{opacity:1}}\n")
	}
	b.WriteString("@media (prefers-reduced-motion:reduce){*{animation:none!important}}\n")
	b.WriteString("</style>\n")
}

// writeGrid emits one <path> per colour, merging horizontal runs. If ink is
// not empty it replaces the colour of ink pixels.
func writeGrid(b *strings.Builder, g *creature.Grid, class, ink string) {
	var paths []string
	for _, p := range creature.PaletteOrder {
		var d strings.Builder
		for y := 0; y < creature.Size; y++ {
			for x := 0; x < creature.Size; {
				if g[y][x] != p {
					x++
					continue
				}
				start := x
				for x < creature.Size && g[y][x] == p {
					x++
				}
				fmt.Fprintf(&d, "M%d %dh%dv%dh-%dz", originX+start*pixel, originY+y*pixel, (x-start)*pixel, pixel, (x-start)*pixel)
			}
		}
		if d.Len() > 0 {
			fill := creature.Color(p)
			if p == creature.PInk && ink != "" {
				fill = ink
			}
			paths = append(paths, fmt.Sprintf("<path fill=\"%s\" d=\"%s\"/>", fill, d.String()))
		}
	}
	if len(paths) == 0 {
		return
	}
	if class != "" {
		fmt.Fprintf(b, "<g class=\"%s\">", class)
		b.WriteString(strings.Join(paths, ""))
		b.WriteString("</g>\n")
		return
	}
	b.WriteString(strings.Join(paths, "\n"))
	b.WriteString("\n")
}

// glyph draws a small pixel pattern as a single path at (x, y) with size s.
func glyph(rows []string, x, y, s int) string {
	var d strings.Builder
	for dy, row := range rows {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] != '.' {
				fmt.Fprintf(&d, "M%d %dh%dv%dh-%dz", x+dx*s, y+dy*s, s, s, s)
			}
		}
	}
	return d.String()
}

var zGlyph = []string{"KKKK", "..K.", ".K..", "KKKK"}

// writeZzz floats three pixel "z" above the head of a sleeping creature.
func writeZzz(b *strings.Builder, sp creature.Sprite, t Theme) {
	_, minY, maxX, _, ok := bbox(&sp.Body)
	if !ok {
		return
	}
	x := min(originX+(maxX+1)*pixel-4, 168)
	y := max(originY+minY*pixel+6, 52)
	zs := []struct{ dx, dy, s int }{{0, 0, 2}, {12, -14, 2}, {24, -30, 3}}
	b.WriteString("<g shape-rendering=\"crispEdges\">")
	for i, z := range zs {
		fmt.Fprintf(b, "<path class=\"z z%d\" fill=\"%s\" d=\"%s\"/>", i+1, t.Ink, glyph(zGlyph, x+z.dx, y+z.dy, z.s))
	}
	b.WriteString("</g>\n")
}

var sparkle = []string{".X.", "XXX", ".X."}

// writeSparkles twinkles two little stars around a radiant creature.
func writeSparkles(b *strings.Builder, sp creature.Sprite) {
	minX, minY, maxX, _, ok := bbox(&sp.Body)
	if !ok {
		return
	}
	left := max(originX+minX*pixel-16, 6)
	right := min(originX+(maxX+1)*pixel+4, 186)
	top := max(originY+minY*pixel+4, 10)
	b.WriteString("<g shape-rendering=\"crispEdges\">")
	fmt.Fprintf(b, "<path class=\"sp\" fill=\"%s\" d=\"%s\"/>", creature.Honey, glyph(sparkle, right, top, 4))
	fmt.Fprintf(b, "<path class=\"sp sp2\" fill=\"%s\" d=\"%s\"/>", creature.Accent, glyph(sparkle, left, top+30, 4))
	b.WriteString("</g>\n")
}

func moodColor(m creature.Mood, t Theme) string {
	switch m {
	case creature.Radiant:
		return creature.Accent
	case creature.Happy:
		return creature.Moss
	case creature.Bored:
		return creature.Honey
	}
	return t.Muted
}

func text(b *strings.Builder, x, y int, size float64, fill, anchor, class, s string) {
	a := ""
	if anchor != "" {
		a = fmt.Sprintf(" text-anchor=\"%s\"", anchor)
	}
	fmt.Fprintf(b, "<text class=\"t%s\" x=\"%d\" y=\"%d\" font-size=\"%s\" fill=\"%s\"%s>%s</text>\n",
		class, x, y, strconv.FormatFloat(size, 'f', -1, 64), fill, a, Escape(s))
}

func writePanel(b *strings.Builder, c Card, t Theme) {
	cr := c.Creature
	s := c.Stats

	user := "commitling"
	if c.User != "" {
		user = "@" + c.User
	}
	text(b, panelX, 36, 11, t.Muted, "", "", user)
	if c.User != "" {
		text(b, panelEnd, 36, 11, t.Muted, "end", "", "commitling")
	}

	text(b, panelX, 64, 22, t.Ink, "", " b", cr.Stage.Name())

	fmt.Fprintf(b, "<rect x=\"%d\" y=\"78\" width=\"8\" height=\"8\" fill=\"%s\" shape-rendering=\"crispEdges\"/>\n", panelX, moodColor(cr.Mood, t))
	text(b, panelX+14, 86, 12, t.Ink, "", "", cr.Mood.Name())
	text(b, panelEnd, 86, 11, t.Muted, "end", "", fmt.Sprintf("fase %d de %d", int(cr.Stage)+1, len(creature.Stages)))

	text(b, panelX, 110, 12, t.Ink, "", "", Thousands(s.XP)+" XP")
	if next, ok := cr.Stage.Next(); ok {
		text(b, panelEnd, 110, 11, t.Muted, "end", "", fmt.Sprintf("%s: %s XP", strings.ToLower(next.Name()), Thousands(next.MinXP())))
	} else {
		text(b, panelEnd, 110, 11, t.Muted, "end", "", "fase máxima")
	}

	filled := int(creature.Progress(s.XP) * barCells)
	if cr.Stage == creature.Ancient {
		filled = barCells
	}
	var on, off strings.Builder
	for i := 0; i < barCells; i++ {
		x := panelX + i*(barCellW+barGap)
		seg := fmt.Sprintf("M%d 118h%dv%dh-%dz", x, barCellW, barHeight, barCellW)
		if i < filled {
			on.WriteString(seg)
		} else {
			off.WriteString(seg)
		}
	}
	b.WriteString("<g shape-rendering=\"crispEdges\">")
	if off.Len() > 0 {
		fmt.Fprintf(b, "<path fill=\"%s\" d=\"%s\"/>", t.Line, off.String())
	}
	if on.Len() > 0 {
		fmt.Fprintf(b, "<path fill=\"%s\" d=\"%s\"/>", creature.Moss, on.String())
	}
	b.WriteString("</g>\n")

	cols := []struct {
		x            int
		label, value string
	}{
		{panelX, "racha", plural(s.Streak, "día", "días")},
		{panelX + 86, "activo 30 d", fmt.Sprintf("%d/30", s.ActiveDays30)},
		{panelX + 180, "repos", strconv.Itoa(s.Repos)},
	}
	for _, col := range cols {
		text(b, col.x, 150, 10, t.Muted, "", "", col.label)
		text(b, col.x, 168, 15, t.Ink, "", " b", col.value)
	}

	acc := "accesorios: aún ninguno"
	if names := cr.Accessories.Names(); len(names) > 0 {
		acc = "accesorios: " + strings.Join(names, " · ")
	}
	text(b, panelX, 188, 10, t.Muted, "", "", acc)
}
