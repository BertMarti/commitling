// Package render draws the commitling card as a self-contained animated SVG.
//
// The animation is plain CSS inside the SVG (GitHub does not run JavaScript
// in READMEs) and is disabled with prefers-reduced-motion. The output only
// depends on its input, so the same data always yields the same bytes.
package render

import (
	"strconv"
	"strings"
	"unicode/utf8"

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

// Size is the format of the card: Full (480×200, the zero value, so a Card
// that does not say anything keeps drawing what it always drew) or Compact
// (a 200×60 badge for signatures and sidebars).
type Size int

// Card sizes.
const (
	Full Size = iota
	Compact
)

// SizeByName returns the size called name ("full" or "compact"; empty is full).
func SizeByName(name string) (Size, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "full":
		return Full, true
	case "compact":
		return Compact, true
	}
	return Full, false
}

// Compact badge: the same 16×16 map with 3 px pixels on the left, then the
// stage, the mood and a progress bar in the 130 px that are left.
const (
	compactWidth   = 200
	compactHeight  = 60
	compactMargin  = 8
	compactPixel   = 3
	compactOrigin  = 6
	compactTextX   = 62
	compactCells   = 16
	compactCellW   = 6
	compactCellGap = 2
	compactCellH   = 4
	compactBarY    = 44
)

// layout is where the creature goes and how far it moves: the full card and
// the compact badge draw the same sprite at different scales.
type layout struct {
	pixel, originX, originY int
	hop, breathe, rest      int  // translateY of the animations, in px
	extras                  bool // zzz and sparkles (they do not fit in the badge)
}

var (
	fullLayout    = layout{pixel: pixel, originX: originX, originY: originY, hop: 8, breathe: 3, rest: 2, extras: true}
	compactLayout = layout{pixel: compactPixel, originX: compactOrigin, originY: compactOrigin, hop: 3, breathe: 1, rest: 1}
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

// glyphWidth is the width of one monospace character as a fraction of the
// font size. The fonts in the stack range from 0.55 (Consolas) to 0.602
// (Menlo, DejaVu Sans Mono), so 0.62 is a safe upper bound.
const glyphWidth = 0.62

// panelWidth is the horizontal room of the right-hand panel.
const panelWidth = panelEnd - panelX

// textWidth estimates the rendered width of s at the given font size.
func textWidth(s string, size float64) float64 {
	return float64(utf8.RuneCountInString(s)) * size * glyphWidth
}

// truncate cuts s so that it fits in maxW at the given size, ending in "…".
func truncate(s string, size, maxW float64) string {
	n := int(maxW / (size * glyphWidth))
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// fit picks the first candidate and size (largest first) whose estimated
// width is at most maxW. If none fits, it truncates the last candidate at
// the smallest size. It is deterministic: the same input gives the same text.
func fit(candidates []string, sizes []float64, maxW float64) (string, float64) {
	for _, c := range candidates {
		for _, sz := range sizes {
			if textWidth(c, sz) <= maxW {
				return c, sz
			}
		}
	}
	sz := sizes[len(sizes)-1]
	return truncate(candidates[len(candidates)-1], sz, maxW), sz
}

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
	Light = Theme{Name: "light", Bg: "#f3efe6", Ink: "#2b2724", Muted: "#6f675e", Line: "#ddd5c6", Outline: creature.Ink}
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
	Size     Size // Full unless set
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
	s := cr.StageName() + ", " + strings.ToLower(cr.Mood.Name()) + ". " + Thousands(c.Stats.XP) + " XP, racha de " + plural(c.Stats.Streak, "día", "días") + ", " + strconv.Itoa(c.Stats.ActiveDays30) + " días activos en los últimos 30"
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
	if c.Size == Compact {
		return compactSVG(c, t)
	}
	cr := c.Creature
	sp := creature.Draw(cr)
	var b strings.Builder

	title := "commitling"
	if c.User != "" {
		title = "commitling de @" + c.User
	}

	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + strconv.Itoa(Width) + `" height="` + strconv.Itoa(Height) + `" viewBox="0 0 ` + strconv.Itoa(Width) + ` ` + strconv.Itoa(Height) + `" role="img" aria-labelledby="cl-title cl-desc">`)
	b.WriteString("\n")
	b.WriteString("<title id=\"cl-title\">" + Escape(title) + "</title>\n")
	b.WriteString("<desc id=\"cl-desc\">" + Escape(Description(c)) + "</desc>\n")
	writeStyle(&b, c, sp, fullLayout)

	// Paper and frame.
	b.WriteString("<rect x=\"0.5\" y=\"0.5\" width=\"" + strconv.Itoa(Width-1) + "\" height=\"" + strconv.Itoa(Height-1) + "\" rx=\"8\" fill=\"" + t.Bg + "\" stroke=\"" + t.Line + "\"/>\n")

	// Ground and divider.
	b.WriteString("<path d=\"M28 " + strconv.Itoa(groundY+1) + "h144\" stroke=\"" + t.Line + "\" stroke-width=\"2\" stroke-dasharray=\"8 4\" shape-rendering=\"crispEdges\"/>\n")
	b.WriteString("<path d=\"M202.5 28v144\" stroke=\"" + t.Line + "\" stroke-dasharray=\"2 4\" shape-rendering=\"crispEdges\"/>\n")

	// Creature.
	b.WriteString("<g class=\"bob\" shape-rendering=\"crispEdges\">\n")
	writeGrid(&b, &sp.Outline, "", t.Outline, fullLayout)
	writeGrid(&b, &sp.Body, "", "", fullLayout)
	writeGrid(&b, &sp.Eyes, "eyes", "", fullLayout)
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

// SpriteSVG draws only the creature, still and cropped to its pixels. The
// silhouette is ink, or paper when the viewer prefers a dark colour scheme.
// It is used for icons and headers.
func SpriteSVG(cr creature.Creature) []byte {
	sp := creature.Draw(cr)
	x0, y0, x1, y1 := creature.Size, creature.Size, -1, -1
	for _, g := range []*creature.Grid{&sp.Outline, &sp.Body, &sp.Eyes} {
		if a, b, c, d, ok := bbox(g); ok {
			x0, y0, x1, y1 = min(x0, a), min(y0, b), max(x1, c), max(y1, d)
		}
	}
	if x1 < 0 {
		x0, y0, x1, y1 = 0, 0, creature.Size-1, creature.Size-1
	}
	w, h := (x1-x0+1)*pixel, (y1-y0+1)*pixel
	side := max(w, h)
	vx := originX + x0*pixel - (side-w)/2
	vy := originY + y0*pixel - (side-h)/2
	var b strings.Builder
	b.WriteString("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"" + strconv.Itoa(vx) + " " + strconv.Itoa(vy) + " " + strconv.Itoa(side) + " " + strconv.Itoa(side) + "\" shape-rendering=\"crispEdges\">\n")
	b.WriteString("<style>.o path{fill:" + creature.Ink + "}@media (prefers-color-scheme:dark){.o path{fill:" + creature.Paper + "}}</style>\n")
	writeGrid(&b, &sp.Outline, "o", "", fullLayout)
	writeGrid(&b, &sp.Body, "", "", fullLayout)
	writeGrid(&b, &sp.Eyes, "", "", fullLayout)
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

func writeStyle(b *strings.Builder, c Card, sp creature.Sprite, l layout) {
	b.WriteString("<style>\n")
	b.WriteString(`.t{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"DejaVu Sans Mono","Liberation Mono",monospace}` + "\n")
	b.WriteString(".b{font-weight:700}\n")

	// Whole body: breathing or hopping, depending on the mood.
	switch c.Creature.Mood {
	case creature.Radiant:
		b.WriteString(".bob{animation:hop 1.6s ease-in-out infinite}\n")
		b.WriteString("@keyframes hop{0%,55%,100%{transform:translateY(0)}25%{transform:translateY(-" + strconv.Itoa(l.hop) + "px)}}\n")
	case creature.Happy:
		b.WriteString(".bob{animation:breathe 3.2s ease-in-out infinite}\n")
		b.WriteString("@keyframes breathe{0%,100%{transform:translateY(0)}50%{transform:translateY(-" + strconv.Itoa(l.breathe) + "px)}}\n")
	default:
		b.WriteString(".bob{animation:breathe 5.6s ease-in-out infinite}\n")
		b.WriteString("@keyframes breathe{0%,100%{transform:translateY(0)}50%{transform:translateY(-" + strconv.Itoa(l.rest) + "px)}}\n")
	}

	if c.Creature.Mood.Blinks() {
		if x0, y0, x1, y1, ok := bbox(&sp.Eyes); ok {
			cx := l.originX + (x0+x1+1)*l.pixel/2
			cy := l.originY + (y0+y1+1)*l.pixel/2
			b.WriteString(".eyes{transform-origin:" + strconv.Itoa(cx) + "px " + strconv.Itoa(cy) + "px;animation:blink 4.8s infinite}\n")
			b.WriteString("@keyframes blink{0%,91%,97%,100%{transform:scaleY(1)}94%{transform:scaleY(.1)}}\n")
		}
	}
	switch c.Creature.Mood {
	case creature.Sleeping:
		if !l.extras {
			break
		}
		b.WriteString(".z{animation:zz 3.6s ease-in-out infinite;opacity:.9}\n")
		b.WriteString(".z2{animation-delay:1.2s}.z3{animation-delay:2.4s}\n")
		b.WriteString("@keyframes zz{0%{opacity:0;transform:translate(0,6px)}35%{opacity:.9}100%{opacity:0;transform:translate(6px,-10px)}}\n")
	case creature.Radiant:
		if !l.extras {
			break
		}
		b.WriteString(".sp{animation:tw 1.8s ease-in-out infinite}.sp2{animation-delay:.9s}\n")
		b.WriteString("@keyframes tw{0%,100%{opacity:.2}50%{opacity:1}}\n")
	}
	b.WriteString("@media (prefers-reduced-motion:reduce){*{animation:none!important}}\n")
	b.WriteString("</style>\n")
}

// writeGrid emits one <path> per colour, merging horizontal runs. If ink is
// not empty it replaces the colour of ink pixels.
func writeGrid(b *strings.Builder, g *creature.Grid, class, ink string, l layout) {
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
				d.WriteString("M" + strconv.Itoa(l.originX+start*l.pixel) + " " + strconv.Itoa(l.originY+y*l.pixel) + "h" + strconv.Itoa((x-start)*l.pixel) + "v" + strconv.Itoa(l.pixel) + "h-" + strconv.Itoa((x-start)*l.pixel) + "z")
			}
		}
		if d.Len() > 0 {
			fill := creature.Color(p)
			if p == creature.PInk && ink != "" {
				fill = ink
			}
			paths = append(paths, "<path fill=\""+fill+"\" d=\""+d.String()+"\"/>")
		}
	}
	if len(paths) == 0 {
		return
	}
	if class != "" {
		b.WriteString("<g class=\"" + class + "\">")
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
				d.WriteString("M" + strconv.Itoa(x+dx*s) + " " + strconv.Itoa(y+dy*s) + "h" + strconv.Itoa(s) + "v" + strconv.Itoa(s) + "h-" + strconv.Itoa(s) + "z")
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
		b.WriteString("<path class=\"z z" + strconv.Itoa(i+1) + "\" fill=\"" + t.Ink + "\" d=\"" + glyph(zGlyph, x+z.dx, y+z.dy, z.s) + "\"/>")
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
	b.WriteString("<path class=\"sp\" fill=\"" + creature.Honey + "\" d=\"" + glyph(sparkle, right, top, 4) + "\"/>")
	b.WriteString("<path class=\"sp sp2\" fill=\"" + creature.Accent + "\" d=\"" + glyph(sparkle, left, top+30, 4) + "\"/>")
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
		a = " text-anchor=\"" + anchor + "\""
	}
	b.WriteString("<text class=\"t" + class + "\" x=\"" + strconv.Itoa(x) + "\" y=\"" + strconv.Itoa(y) + "\" font-size=\"" + strconv.FormatFloat(size, 'f', -1, 64) + "\" fill=\"" + fill + "\"" + a + ">" + Escape(s) + "</text>\n")
}

// fitted draws the first candidate that fits maxW, shrinking through sizes
// and finally truncating with "…" (see fit).
func fitted(b *strings.Builder, x, y int, maxW float64, sizes []float64, fill, anchor, class string, candidates ...string) {
	s, size := fit(candidates, sizes, maxW)
	text(b, x, y, size, fill, anchor, class, s)
}

func writePanel(b *strings.Builder, c Card, t Theme) {
	cr := c.Creature
	s := c.Stats

	// Header: @user on the left and the project name on the right. A long
	// login shrinks (at most it is truncated) and then takes the whole
	// width, dropping the project name (the <title> still carries it).
	const tag = "commitling"
	if c.User == "" {
		text(b, panelX, 36, 11, t.Muted, "", "", tag)
	} else {
		user := "@" + c.User
		room := panelWidth - textWidth(tag, 11) - 12
		if textWidth(user, 11) <= room {
			text(b, panelX, 36, 11, t.Muted, "", "", user)
			text(b, panelEnd, 36, 11, t.Muted, "end", "", tag)
		} else {
			fitted(b, panelX, 36, panelWidth, []float64{11, 10, 9}, t.Muted, "", "", user)
		}
	}

	fitted(b, panelX, 64, panelWidth, []float64{22, 20, 18, 16}, t.Ink, "", " b", cr.StageName())

	b.WriteString("<rect x=\"" + strconv.Itoa(panelX) + "\" y=\"78\" width=\"8\" height=\"8\" fill=\"" + moodColor(cr.Mood, t) + "\" shape-rendering=\"crispEdges\"/>\n")
	text(b, panelX+14, 86, 12, t.Ink, "", "", cr.Mood.Name())
	text(b, panelEnd, 86, 11, t.Muted, "end", "", "fase "+strconv.Itoa(int(cr.Stage)+1)+" de "+strconv.Itoa(len(creature.Stages)))

	// XP row: the total goes first; the goal on the right gets what is left,
	// with a shorter wording if needed.
	xp, xpSize := fit([]string{Thousands(s.XP) + " XP"}, []float64{12, 11, 10, 9}, panelWidth*0.6)
	text(b, panelX, 110, xpSize, t.Ink, "", "", xp)
	goalRoom := panelWidth - textWidth(xp, xpSize) - 10
	goalSizes := []float64{11, 10, 9}
	if next, ok := cr.Stage.Next(); ok {
		fitted(b, panelEnd, 110, goalRoom, goalSizes, t.Muted, "end", "",
			strings.ToLower(cr.Species.StageName(next))+": "+Thousands(next.MinXP())+" XP",
			"meta: "+Thousands(next.MinXP())+" XP")
	} else {
		fitted(b, panelEnd, 110, goalRoom, goalSizes, t.Muted, "end", "", "fase máxima", "máxima")
	}

	filled := int(creature.Progress(s.XP) * barCells)
	if cr.Stage == creature.Ancient {
		filled = barCells
	}
	var on, off strings.Builder
	for i := 0; i < barCells; i++ {
		x := panelX + i*(barCellW+barGap)
		seg := "M" + strconv.Itoa(x) + " 118h" + strconv.Itoa(barCellW) + "v" + strconv.Itoa(barHeight) + "h-" + strconv.Itoa(barCellW) + "z"
		if i < filled {
			on.WriteString(seg)
		} else {
			off.WriteString(seg)
		}
	}
	b.WriteString("<g shape-rendering=\"crispEdges\">")
	if off.Len() > 0 {
		b.WriteString("<path fill=\"" + t.Line + "\" d=\"" + off.String() + "\"/>")
	}
	if on.Len() > 0 {
		b.WriteString("<path fill=\"" + creature.Moss + "\" d=\"" + on.String() + "\"/>")
	}
	b.WriteString("</g>\n")

	cols := []struct {
		x, w         int
		label, value string
	}{
		{panelX, 86, "racha", plural(s.Streak, "día", "días")},
		{panelX + 86, 94, "activo 30 d", (strconv.Itoa(s.ActiveDays30) + "/30")},
		{panelX + 180, panelEnd - (panelX + 180), "repos", strconv.Itoa(s.Repos)},
	}
	for _, col := range cols {
		room := float64(col.w - 4)
		fitted(b, col.x, 150, room, []float64{10, 9}, t.Muted, "", "", col.label)
		fitted(b, col.x, 168, room, []float64{15, 13, 11, 9}, t.Ink, "", " b", col.value)
	}

	acc := "accesorios: aún ninguno"
	if names := cr.Accessories.Names(); len(names) > 0 {
		acc = "accesorios: " + strings.Join(names, " · ")
	}
	fitted(b, panelX, 188, panelWidth, []float64{10, 9, 8}, t.Muted, "", "", acc)
}

// compactSVG draws the 200×60 badge: the creature, its stage, its mood and
// phase number, and a progress bar. It is deliberately the same card in less
// room: same paper, ink and creature palette, same animation (at scale) and
// the same description in <desc> for whoever cannot see the pixels.
func compactSVG(c Card, t Theme) []byte {
	cr := c.Creature
	sp := creature.Draw(cr)
	l := compactLayout
	var b strings.Builder

	title := "commitling"
	if c.User != "" {
		title = "commitling de @" + c.User
	}

	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="` + strconv.Itoa(compactWidth) + `" height="` + strconv.Itoa(compactHeight) + `" viewBox="0 0 ` + strconv.Itoa(compactWidth) + ` ` + strconv.Itoa(compactHeight) + `" role="img" aria-labelledby="cl-title cl-desc">`)
	b.WriteString("\n")
	b.WriteString("<title id=\"cl-title\">" + Escape(title) + "</title>\n")
	b.WriteString("<desc id=\"cl-desc\">" + Escape(Description(c)) + "</desc>\n")
	writeStyle(&b, c, sp, l)

	b.WriteString("<rect x=\"0.5\" y=\"0.5\" width=\"" + strconv.Itoa(compactWidth-1) + "\" height=\"" + strconv.Itoa(compactHeight-1) + "\" rx=\"8\" fill=\"" + t.Bg + "\" stroke=\"" + t.Line + "\"/>\n")
	groundY := l.originY + creature.Size*l.pixel
	b.WriteString("<path d=\"M" + strconv.Itoa(l.originX) + " " + strconv.Itoa(groundY) + ".5h" + strconv.Itoa(creature.Size*l.pixel) + "\" stroke=\"" + t.Line + "\" stroke-dasharray=\"4 2\" shape-rendering=\"crispEdges\"/>\n")

	b.WriteString("<g class=\"bob\" shape-rendering=\"crispEdges\">\n")
	writeGrid(&b, &sp.Outline, "", t.Outline, l)
	writeGrid(&b, &sp.Body, "", "", l)
	writeGrid(&b, &sp.Eyes, "eyes", "", l)
	b.WriteString("</g>\n")

	room := float64(compactWidth - compactMargin - compactTextX)
	fitted(&b, compactTextX, 23, room, []float64{14, 13, 12, 11}, t.Ink, "", " b", cr.StageName())

	// The mood: a square of its colour, its name and the number of the phase.
	b.WriteString("<rect x=\"" + strconv.Itoa(compactTextX) + "\" y=\"31\" width=\"6\" height=\"6\" fill=\"" + moodColor(cr.Mood, t) + "\" shape-rendering=\"crispEdges\"/>\n")
	fitted(&b, compactTextX+10, 37, room-10, []float64{10, 9}, t.Muted, "", "",
		cr.Mood.Name()+" · fase "+strconv.Itoa(int(cr.Stage)+1)+"/"+strconv.Itoa(len(creature.Stages)))

	filled := int(creature.Progress(c.Stats.XP) * compactCells)
	if cr.Stage == creature.Ancient {
		filled = compactCells
	}
	var on, off strings.Builder
	for i := 0; i < compactCells; i++ {
		x := compactTextX + i*(compactCellW+compactCellGap)
		seg := "M" + strconv.Itoa(x) + " " + strconv.Itoa(compactBarY) + "h" + strconv.Itoa(compactCellW) + "v" + strconv.Itoa(compactCellH) + "h-" + strconv.Itoa(compactCellW) + "z"
		if i < filled {
			on.WriteString(seg)
		} else {
			off.WriteString(seg)
		}
	}
	b.WriteString("<g shape-rendering=\"crispEdges\">")
	if off.Len() > 0 {
		b.WriteString("<path fill=\"" + t.Line + "\" d=\"" + off.String() + "\"/>")
	}
	if on.Len() > 0 {
		b.WriteString("<path fill=\"" + creature.Moss + "\" d=\"" + on.String() + "\"/>")
	}
	b.WriteString("</g>\n")
	b.WriteString("</svg>\n")
	return []byte(b.String())
}
