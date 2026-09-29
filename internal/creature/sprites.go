package creature

// Size is the width and height of every pixel map.
const Size = 16

// The five colours of the creature ("papel y píxel" palette).
const (
	Ink    = "#2b2724"
	Paper  = "#f3efe6"
	Moss   = "#7fb069"
	Honey  = "#e6aa68"
	Accent = "#ca3c25"
)

// Pixel codes used in the ASCII maps. '.' is transparent.
const (
	Clear  byte = '.'
	PInk   byte = 'K'
	PPaper byte = 'P'
	PMoss  byte = 'G'
	PHoney byte = 'H'
	PRed   byte = 'R'
)

// PaletteOrder is the drawing order of the pixel codes.
var PaletteOrder = []byte{PMoss, PHoney, PPaper, PRed, PInk}

// Color returns the hex colour of a pixel code, or "" if transparent.
func Color(p byte) string {
	switch p {
	case PInk:
		return Ink
	case PPaper:
		return Paper
	case PMoss:
		return Moss
	case PHoney:
		return Honey
	case PRed:
		return Accent
	}
	return ""
}

// Grid is a Size×Size pixel map, indexed [y][x].
type Grid [Size][Size]byte

// Point is a pixel coordinate.
type Point struct{ X, Y int }

func (g *Grid) set(x, y int, p byte) {
	if x < 0 || y < 0 || x >= Size || y >= Size || p == Clear {
		return
	}
	g[y][x] = p
}

// stamp draws an ASCII sprite with its top-left corner at (x, y).
func (g *Grid) stamp(x, y int, rows []string) {
	for dy, row := range rows {
		for dx := 0; dx < len(row); dx++ {
			g.set(x+dx, y+dy, row[dx])
		}
	}
}

func parse(rows [Size]string) Grid {
	var g Grid
	for y, row := range rows {
		for x := 0; x < Size; x++ {
			g[y][x] = row[x]
		}
	}
	return g
}

// stageArt holds the pixel map of a stage and where things go on it.
type stageArt struct {
	body [Size]string
	// face is the top-left corner of the 8×4 face: two 3×2 eyes at
	// (x, y) and (x+5, y), blush at (x, y+2) and (x+7, y+2) and a 4×2
	// mouth at (x+2, y+2).
	face Point
	// hat is the top-left corner of the 6×5 hat.
	hat Point
	// scarfRow is the row wrapped by the scarf; scarfTail is where its
	// loose end hangs from.
	scarfRow  int
	scarfTail Point
	// flower is the top-left corner of the 3×3 flower.
	flower Point
}

// Pixel maps. K ink, P paper, G moss, H honey, R accent, . transparent.
var art = [...]stageArt{
	Seed: {
		body: [Size]string{
			"................",
			"................",
			"................",
			"................",
			"................",
			"........KKK.....",
			".......KGGGK....",
			".......KGKK.....",
			"....KKKKKKKK....",
			"...KPHHHHHHHK...",
			"..KHHHHHHHHHHK..",
			"..KHHHHHHHHHHK..",
			"..KHHHHHHHHHHK..",
			"..KHHHHHHHHHHK..",
			"...KHHHHHHHHK...",
			"....KKKKKKKK....",
		},
		face:      Point{4, 10},
		hat:       Point{4, 4},
		scarfRow:  14,
		scarfTail: Point{9, 15},
		flower:    Point{10, 3},
	},
	Sprout: {
		body: [Size]string{
			"................",
			"................",
			"................",
			"..KKK......KKK..",
			".KGGGKK..KKGGGK.",
			".KGGGGGKKGGGGGK.",
			"..KKKGGGGGGKKK..",
			".....KKGGKK.....",
			"....KKKGGKKK....",
			"..KKPGGGGGGGKK..",
			".KGGGGGGGGGGGGK.",
			".KGGGGGGGGGGGGK.",
			".KGGGGGGGGGGGGK.",
			".KGGGGGGGGGGGGK.",
			"..KGGGGGGGGGGK..",
			"...KKKKKKKKKK...",
		},
		face:      Point{4, 10},
		hat:       Point{5, 4},
		scarfRow:  14,
		scarfTail: Point{9, 15},
		flower:    Point{12, 0},
	},
	Sapling: {
		body: [Size]string{
			".......KK.......",
			"......KGGK......",
			"..KK..KGGK..KK..",
			".KGGK.KGGK.KGGK.",
			".KGGGKKGGKKGGGK.",
			"..KKGGGGGGGGKK..",
			"....KKKGGKKK....",
			"..KKKGGGGGGKKK..",
			".KGGGGGGGGGGGGK.",
			"KGPGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			".KGGGGGGGGGGGGK.",
			"..KKKKKKKKKKKK..",
			"...KHHK..KHHK...",
		},
		face:      Point{4, 8},
		hat:       Point{5, 0},
		scarfRow:  12,
		scarfTail: Point{10, 13},
		flower:    Point{12, 1},
	},
	Shrub: {
		body: [Size]string{
			"................",
			"..KKK.KKKK.KKK..",
			".KGGGKGGGGKGGGK.",
			".KGPGGGGGGGGGGK.",
			"KGGGGGGGGGGGRGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGRGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			"KGGGGGGGGGGGGGGK",
			".KGGGGGGGGGGGGK.",
			"..KKKKKKKKKKKK..",
			"...KHK....KHK...",
			"..KHHK....KHHK..",
		},
		face:      Point{4, 6},
		hat:       Point{5, 0},
		scarfRow:  10,
		scarfTail: Point{10, 11},
		flower:    Point{11, 0},
	},
	Ancient: {
		body: [Size]string{
			"....KKKKKKKK....",
			"..KKGGGGGGGGKK..",
			".KGPGGGGGGRGGGK.",
			"KGGGGGGGGGGGGGGK",
			"KGGGRGGGGGGGGGGK",
			"KGGGGGGGGGGGRGGK",
			".KGGGGGGGGGGGGK.",
			"..KKHHHHHHHHKK..",
			"...KHHHHHHHHK...",
			"...KHHHHHHHHK...",
			"...KHHHHHHHHK...",
			"...KHHHHHHHHK...",
			"...KHHHHHHHHK...",
			"..KHHHHHHHHHHK..",
			".KHHKHHHHHHKHHK.",
			"KKKK.KKKKKK.KKKK",
		},
		face:      Point{4, 8},
		hat:       Point{5, 0},
		scarfRow:  12,
		scarfTail: Point{10, 13},
		flower:    Point{11, 1},
	},
}

// Eye shapes (3×2) per mood.
var eyes = [...][]string{
	Sleeping: {"...", "KKK"},
	Bored:    {"KKK", ".K."},
	Happy:    {".K.", ".K."},
	Radiant:  {".K.", "K.K"},
}

// Mouth shapes (4×2) per mood.
var mouths = [...][]string{
	Sleeping: {"....", ".KK."},
	Bored:    {"....", "KKKK"},
	Happy:    {"K..K", ".KK."},
	Radiant:  {"KRRK", ".KK."},
}

var hatArt = []string{
	"..KK..",
	".KHHK.",
	".KRRK.",
	"KRRRRK",
	"KPPPPK",
}

var flowerArt = []string{
	".R.",
	"RHR",
	".R.",
}

// Sprite is a creature ready to draw, split in layers:
//
//   - Outline: the silhouette of the body (always ink pixels). It is a
//     separate layer so dark themes can draw it in paper and keep the
//     creature readable, like a sticker.
//   - Body: fills, mouth, blush and accessories.
//   - Eyes: on their own so they can blink.
type Sprite struct {
	Outline Grid
	Body    Grid
	Eyes    Grid
}

// Blinks reports whether the eyes of this mood are open (and can blink).
func (m Mood) Blinks() bool { return m == Happy || m == Bored }

// Draw composes the sprite of a creature.
func Draw(c Creature) Sprite {
	a := art[c.Stage]
	var sp Sprite
	base := parse(a.body)
	sp.Body = base
	for y := range sp.Eyes {
		for x := range sp.Eyes[y] {
			sp.Eyes[y][x] = Clear
		}
	}

	f := a.face
	sp.Eyes.stamp(f.X, f.Y, eyes[c.Mood])
	sp.Eyes.stamp(f.X+5, f.Y, eyes[c.Mood])
	sp.Body.stamp(f.X+2, f.Y+2, mouths[c.Mood])
	if c.Mood == Happy || c.Mood == Radiant {
		sp.Body.set(f.X, f.Y+2, PRed)
		sp.Body.set(f.X+7, f.Y+2, PRed)
	}

	if c.Accessories.Scarf {
		drawScarf(&sp.Body, a.scarfRow, a.scarfTail)
	}
	if c.Accessories.Flower {
		sp.Body.stamp(a.flower.X, a.flower.Y, flowerArt)
	}
	if c.Accessories.Hat {
		sp.Body.stamp(a.hat.X, a.hat.Y, hatArt)
	}

	// Split the untouched silhouette from the rest.
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			sp.Outline[y][x] = Clear
			if base[y][x] == PInk && sp.Body[y][x] == PInk && !touched(a, c, x, y) {
				sp.Outline[y][x] = PInk
				sp.Body[y][x] = Clear
			}
		}
	}
	return sp
}

// touched reports whether an accessory covers the pixel (x, y), so an ink
// pixel there belongs to the accessory and not to the silhouette.
func touched(a stageArt, c Creature, x, y int) bool {
	in := func(p Point, rows []string) bool {
		dx, dy := x-p.X, y-p.Y
		return dy >= 0 && dy < len(rows) && dx >= 0 && dx < len(rows[dy]) && rows[dy][dx] != Clear
	}
	return (c.Accessories.Hat && in(a.hat, hatArt)) || (c.Accessories.Flower && in(a.flower, flowerArt))
}

// drawScarf wraps row y in a striped red scarf, keeping the outline at both
// ends, and lets a short tail hang from tail.
func drawScarf(g *Grid, y int, tail Point) {
	first, last := -1, -1
	for x := 0; x < Size; x++ {
		if g[y][x] != Clear {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	if first < 0 {
		return
	}
	for x := first + 1; x < last; x++ {
		if (x-first)%4 == 0 {
			g[y][x] = PPaper
		} else {
			g[y][x] = PRed
		}
	}
	g.stamp(tail.X, tail.Y, []string{"RR", "R."})
}
