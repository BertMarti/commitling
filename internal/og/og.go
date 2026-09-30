// Package og draws the Open Graph preview image of the project: a 1200×630
// PNG with the creature and the name of the project.
//
// Social networks do not show SVG in link previews, so the image is drawn
// here with only the standard library (image, image/color and image/png).
// Everything is made of square pixels at an integer scale, with no
// anti-aliasing, and the result is deterministic: the same input gives the
// same bytes.
package og

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/BertMarti/commitling/internal/creature"
)

// Image size, the recommended one for Open Graph cards.
const (
	Width  = 1200
	Height = 630
)

// Text of the card.
const (
	Name     = "commitling"
	Tagline1 = "tu mascota pixel-art"
	Tagline2 = "que crece con tus commits"
	Site     = "bertmarti.github.io/commitling"
)

// Layout, in image pixels.
const (
	heroScale  = 20 // scale of the big creature
	heroX      = 96
	heroY      = 150
	textX      = 496 // left edge of the text column
	stageY     = 392 // top of the first row of small stages
	stageScale = 3
	stageGap   = 20
	stageRowH  = creature.Size*stageScale + 12 // distance between rows of stages
	siteY      = 530
)

// stageRowY is the top of the row of stages of the i-th species.
func stageRowY(i int) int { return stageY + i*stageRowH }

// Palette indexes.
const (
	iPaper = iota
	iInk
	iLine
	iMuted
	iMoss
	iHoney
	iAccent
)

// palette is the "papel y píxel" one: the five colours of the creature plus
// the two greys of the card (lines and muted text, as in the SVG card).
var palette = color.Palette{
	iPaper:  rgb(0xf3, 0xef, 0xe6),
	iInk:    rgb(0x2b, 0x27, 0x24),
	iLine:   rgb(0xdd, 0xd5, 0xc6),
	iMuted:  rgb(0x6f, 0x67, 0x5e),
	iMoss:   rgb(0x7f, 0xb0, 0x69),
	iHoney:  rgb(0xe6, 0xaa, 0x68),
	iAccent: rgb(0xca, 0x3c, 0x25),
}

func rgb(r, g, b uint8) color.Color { return color.RGBA{r, g, b, 0xff} }

// canvas is a paletted image with pixel-block drawing helpers.
type canvas struct{ img *image.Paletted }

func (c canvas) rect(x, y, w, h int, idx uint8) {
	b := c.img.Bounds()
	for py := max(y, b.Min.Y); py < min(y+h, b.Max.Y); py++ {
		for px := max(x, b.Min.X); px < min(x+w, b.Max.X); px++ {
			c.img.SetColorIndex(px, py, idx)
		}
	}
}

// text draws s with its top-left corner at (x, y); every font pixel is a
// scale×scale block.
func (c canvas) text(x, y int, s string, scale int, idx uint8) {
	for _, r := range s {
		for gy, row := range glyph(r) {
			for gx := 0; gx < glyphW; gx++ {
				if row[gx] == '#' {
					c.rect(x+gx*scale, y+gy*scale, scale, scale, idx)
				}
			}
		}
		x += advance * scale
	}
}

func colorIndex(p byte) (uint8, bool) {
	switch p {
	case creature.PInk:
		return iInk, true
	case creature.PPaper:
		return iPaper, true
	case creature.PMoss:
		return iMoss, true
	case creature.PHoney:
		return iHoney, true
	case creature.PRed:
		return iAccent, true
	}
	return 0, false
}

// sprite draws the creature with its top-left corner at (x, y), every
// sprite pixel being a scale×scale block.
func (c canvas) sprite(x, y int, cr creature.Creature, scale int) {
	sp := creature.Draw(cr)
	for _, g := range []*creature.Grid{&sp.Outline, &sp.Body, &sp.Eyes} {
		for gy := 0; gy < creature.Size; gy++ {
			for gx := 0; gx < creature.Size; gx++ {
				if idx, ok := colorIndex(g[gy][gx]); ok {
					c.rect(x+gx*scale, y+gy*scale, scale, scale, idx)
				}
			}
		}
	}
}

// Image draws the card. hero is the big creature on the left.
func Image(hero creature.Creature) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, Width, Height), palette)
	c := canvas{img}
	c.rect(0, 0, Width, Height, iPaper)

	// Frame, like the edge of a notebook page.
	const frame, inset = 6, 24
	c.rect(inset, inset, Width-2*inset, frame, iInk)
	c.rect(inset, Height-inset-frame, Width-2*inset, frame, iInk)
	c.rect(inset, inset, frame, Height-2*inset, iInk)
	c.rect(Width-inset-frame, inset, frame, Height-2*inset, iInk)

	// The creature, standing on a line.
	c.rect(heroX-20, heroY+creature.Size*heroScale, creature.Size*heroScale+40, 8, iLine)
	c.sprite(heroX, heroY, hero, heroScale)

	// Name and tagline.
	c.text(textX, 130, Name, 10, iInk)
	c.rect(textX, 234, TextWidth(Name, 10), 10, iMoss)
	c.text(textX, 282, Tagline1, 4, iInk)
	c.text(textX, 326, Tagline2, 4, iInk)

	// The five stages of each species, one row per species, from the
	// youngest to the oldest.
	for i, sp := range creature.AllSpecies {
		x := textX
		for _, st := range creature.Stages {
			c.sprite(x, stageRowY(i), creature.Creature{Species: sp, Stage: st, Mood: creature.Happy}, stageScale)
			x += creature.Size*stageScale + stageGap
		}
	}

	c.text(textX, siteY, Site, 3, iMuted)
	return img
}

// PNG returns the encoded card. The encoding is deterministic.
func PNG(hero creature.Creature) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, Image(hero)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Hero is the creature shown on the card: a radiant sapling with a flower,
// the same as the hero of the website.
var Hero = creature.Creature{
	Stage:       creature.Sapling,
	Mood:        creature.Radiant,
	Accessories: creature.Accessories{Flower: true},
}
