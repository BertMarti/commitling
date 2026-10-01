// Package gallery builds the static website of the project: every stage,
// mood and accessory drawn with synthetic stats, the rules, how to install
// the Action and a slot for a live creature.
package gallery

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/generate"
	"github.com/BertMarti/commitling/internal/og"
	"github.com/BertMarti/commitling/internal/render"
	"github.com/BertMarti/commitling/internal/stats"
)

// LiveUser is the user whose creature is shown live on the site.
const LiveUser = "BertMarti"

// SiteURL is where the site is published (GitHub Pages).
const SiteURL = "https://bertmarti.github.io/commitling/"

// OGImage is the preview image for social cards (Open Graph): a 1200×630 PNG,
// because social networks do not show SVG.
const OGImage = SiteURL + "og.png"

// Title and Description are used in <title>, the description meta tag and
// Open Graph.
const (
	Title       = "commitling · una mascota pixel-art para tu perfil de GitHub"
	Description = "Una mascota pixel-art original que vive en el README de tu perfil de GitHub y crece con tus commits."
)

// DemoUser is the fictitious user of the gallery cards.
const DemoUser = generate.DemoUser

// WorkflowSnippet is the workflow for a profile repository. The README
// shows the same text (a test keeps them in sync).
const WorkflowSnippet = generate.BaseWorkflow

// ReadmeSnippet is the line to paste in the profile README.
const ReadmeSnippet = `![commitling](./commitling.svg)`

//go:embed index.html.tmpl
var indexTmpl string

// generatorJS is the script of the live generator (see index.html.tmpl).
//
//go:embed generator.js
var generatorJS []byte

// Sample returns synthetic stats that produce exactly the given stage and
// mood with no accessories.
func Sample(st creature.Stage, m creature.Mood) stats.Stats {
	xp := 3140
	if next, ok := st.Next(); ok {
		xp = st.MinXP() + (next.MinXP()-st.MinXP())*45/100
	}
	s := stats.Stats{XP: xp, Repos: 3, ActiveDays90: 21}
	switch m {
	case creature.Sleeping:
		s.DaysSinceLast, s.Streak, s.ActiveDays30 = 9, 0, 4
	case creature.Bored:
		s.DaysSinceLast, s.Streak, s.ActiveDays30 = 4, 0, 9
	case creature.Happy:
		s.DaysSinceLast, s.Streak, s.ActiveDays30 = 0, 3, 16
	case creature.Radiant:
		s.DaysSinceLast, s.Streak, s.ActiveDays30 = 0, 7, 22
	}
	return s
}

// Figure is one card of the gallery.
type Figure struct {
	Light, Dark string // file paths relative to the site root
	Alt         string
	Caption     string
}

type row struct {
	Title   string
	Figures []Figure
}

// speciesSection is the part of the gallery of one species.
type speciesSection struct {
	ID     string
	Name   string
	Intro  template.HTML
	Rows   []row
	Extras []Figure
}

type rule struct{ Name, When string }

// choice is one option of a radio group of the generator.
type choice struct{ Value, Label string }

type page struct {
	Version     string
	Title       string
	Description string
	SiteURL     string
	OGImage     string
	LiveUser    string
	Favicon     string
	Hero        string
	Species     []speciesSection
	GenSpecies  []choice
	XP          []rule
	Stages      []rule
	Moods       []rule
	Acc         []rule
	Workflow    string
	Readme      string
	ReadmeDrk   string
}

// Build writes the site into dir and returns how many files it wrote.
func Build(dir, version string) (int, error) {
	n := 0
	write := func(rel string, data []byte) error {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		n++
		return os.WriteFile(path, data, 0o644)
	}
	card := func(name string, sp creature.Species, s stats.Stats) (Figure, error) {
		f := Figure{Light: "svg/" + name + ".svg", Dark: "svg/" + name + "-dark.svg"}
		lc := render.NewCard(DemoUser, s, render.Light)
		lc.Creature.Species = sp
		dc := render.NewCard(DemoUser, s, render.Dark)
		dc.Creature.Species = sp
		if err := write(f.Light, render.SVG(lc)); err != nil {
			return f, err
		}
		if err := write(f.Dark, render.SVG(dc)); err != nil {
			return f, err
		}
		f.Alt = "commitling: " + render.Description(lc)
		return f, nil
	}

	p := page{
		Version:     version,
		Title:       Title,
		Description: Description,
		SiteURL:     SiteURL,
		OGImage:     OGImage,
		LiveUser:    LiveUser,
		Workflow:    WorkflowSnippet,
		Readme:      ReadmeSnippet,
		ReadmeDrk: `<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./commitling-dark.svg">
  <img alt="commitling" src="./commitling.svg">
</picture>`,
	}

	extras := []struct {
		name, caption string
		s             stats.Stats
	}{
		{"acc-hat", "Gorro: más de 30 días activos en 90", stats.Stats{XP: 700, DaysSinceLast: 0, Streak: 3, ActiveDays30: 20, ActiveDays90: 42, Repos: 3}},
		{"acc-scarf", "Bufanda: racha de 10 días o más", stats.Stats{XP: 1500, DaysSinceLast: 0, Streak: 12, ActiveDays30: 21, ActiveDays90: 28, Repos: 4}},
		{"acc-flower", "Flor: 5 repositorios distintos o más", stats.Stats{XP: 260, DaysSinceLast: 1, Streak: 2, ActiveDays30: 12, ActiveDays90: 20, Repos: 6}},
		{"acc-all", "Todo desbloqueado", stats.Stats{XP: 3900, DaysSinceLast: 0, Streak: 14, ActiveDays30: 27, ActiveDays90: 64, Repos: 9}},
	}
	intros := map[creature.Species]string{
		creature.MossSprout: "La especie predeterminada: un brote de musgo con ojos que acaba siendo un árbol ancestral.",
		creature.Mushroom:   "Una seta pequeña con ojos que empieza siendo una espora y acaba en un corro de setas. Se elige con <code>species: mushroom</code>.",
	}
	for _, sp := range creature.AllSpecies {
		// The default species keeps the file names of the first release.
		prefix := ""
		if sp != creature.MossSprout {
			prefix = sp.Slug() + "-"
		}
		sec := speciesSection{ID: sp.Slug(), Name: sp.Name(), Intro: template.HTML(intros[sp])}
		for _, st := range creature.Stages {
			r := row{Title: fmt.Sprintf("%s · desde %s XP", sp.StageName(st), render.Thousands(st.MinXP()))}
			for _, m := range creature.Moods {
				f, err := card(prefix+st.Slug()+"-"+m.Slug(), sp, Sample(st, m))
				if err != nil {
					return n, err
				}
				f.Caption = m.Name()
				r.Figures = append(r.Figures, f)
			}
			sec.Rows = append(sec.Rows, r)
		}
		for _, e := range extras {
			f, err := card(prefix+e.name, sp, e.s)
			if err != nil {
				return n, err
			}
			f.Caption = e.caption
			sec.Extras = append(sec.Extras, f)
		}
		p.Species = append(p.Species, sec)
	}

	icon := render.SpriteSVG(creature.Creature{Stage: creature.Sprout, Mood: creature.Happy})
	if err := write("favicon.svg", icon); err != nil {
		return n, err
	}
	p.Favicon = "favicon.svg"
	for _, sp := range creature.AllSpecies {
		p.GenSpecies = append(p.GenSpecies, choice{sp.Slug(), sp.Name()})
	}
	if err := write("generator.js", generatorJS); err != nil {
		return n, err
	}
	hero := render.SpriteSVG(og.Hero)
	if err := write("hero.svg", hero); err != nil {
		return n, err
	}
	p.Hero = "hero.svg"
	card1200, err := og.PNG(og.Hero)
	if err != nil {
		return n, err
	}
	if err := write("og.png", card1200); err != nil {
		return n, err
	}

	p.XP = []rule{
		{"Commit", fmt.Sprintf("%d XP", stats.XPCommit)},
		{"Pull request abierta", fmt.Sprintf("%d XP", stats.XPPullRequest)},
		{"Issue abierta", fmt.Sprintf("%d XP", stats.XPIssue)},
		{"Otra actividad pública (estrellas, forks, comentarios…)", fmt.Sprintf("%d XP", stats.XPOther)},
	}
	for _, st := range creature.Stages {
		names := make([]string, len(creature.AllSpecies))
		for i, sp := range creature.AllSpecies {
			names[i] = sp.StageName(st)
		}
		p.Stages = append(p.Stages, rule{strings.Join(names, " / "), "desde " + render.Thousands(st.MinXP()) + " XP"})
	}
	p.Moods = []rule{
		{creature.Sleeping.Name(), fmt.Sprintf("%d días o más sin actividad (o ninguna)", creature.SleepingAfterDays)},
		{creature.Bored.Name(), fmt.Sprintf("de %d a %d días sin actividad", creature.BoredAfterDays, creature.SleepingAfterDays-1)},
		{creature.Happy.Name(), fmt.Sprintf("actividad en los últimos %d días", creature.BoredAfterDays)},
		{creature.Radiant.Name(), fmt.Sprintf("racha de %d días seguidos o más", creature.RadiantStreak)},
	}
	p.Acc = []rule{
		{"Gorro", fmt.Sprintf("más de %d días activos en los últimos 90", creature.HatActiveDays90)},
		{"Bufanda", fmt.Sprintf("racha de %d días o más", creature.ScarfStreak)},
		{"Flor", fmt.Sprintf("%d repositorios distintos o más con commits, PR o issues", creature.FlowerRepos)},
	}

	tmpl, err := template.New("index").Parse(indexTmpl)
	if err != nil {
		return n, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, p); err != nil {
		return n, err
	}
	html := strings.ReplaceAll(buf.String(), "\r\n", "\n")
	if err := write("index.html", []byte(html)); err != nil {
		return n, err
	}
	return n, nil
}
