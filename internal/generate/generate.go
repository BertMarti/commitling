// Package generate is the single place where public GitHub events become a
// commitling SVG. The command line and the WebAssembly build (the live
// generator of the website) both call it, so they draw the same bytes for
// the same events and the same date.
package generate

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/events"
	"github.com/BertMarti/commitling/internal/render"
	"github.com/BertMarti/commitling/internal/stats"
)

// Options say what to draw. The zero value means: the login found in the
// events, the moss sprout, the light theme, the full card and "now".
type Options struct {
	User    string    // GitHub login; empty takes the most common actor of the events
	Species string    // moss or mushroom (or an alias); empty is moss
	Theme   string    // light or dark; empty is light
	Size    string    // full or compact; empty is full
	Now     time.Time // reference date; zero is the current time
}

// Result is what Render draws.
type Result struct {
	SVG         []byte
	Description string // short text for alt attributes and screen readers
	Login       string // the login on the card ("" when the events have none)
}

// Check validates the options. Its messages are the ones the command line
// shows for a wrong --theme, --species or --user.
func (o Options) Check() error {
	if _, ok := render.ThemeByName(o.themeName()); !ok {
		return errors.New("tema no válido " + strconv.Quote(o.Theme) + " (usa light o dark)")
	}
	if _, ok := creature.SpeciesByName(o.speciesName()); !ok {
		return errors.New("especie no válida " + strconv.Quote(o.Species) + " (usa moss o mushroom)")
	}
	if _, ok := render.SizeByName(o.Size); !ok {
		return errors.New("tamaño no válido " + strconv.Quote(o.Size) + " (usa full o compact)")
	}
	if o.User != "" && !events.ValidLogin(o.User) {
		return errors.New("nombre de usuario no válido: " + strconv.Quote(o.User) + events.LoginHint(o.User))
	}
	return nil
}

func (o Options) themeName() string {
	if o.Theme == "" {
		return "light"
	}
	return o.Theme
}

func (o Options) speciesName() string {
	if o.Species == "" {
		return "moss"
	}
	return o.Species
}

// Render draws the creature of events, a JSON array like the one returned by
// GET /users/{user}/events/public (or several pages of it joined in one array).
func Render(eventsJSON []byte, o Options) (Result, error) {
	if err := o.Check(); err != nil {
		return Result{}, err
	}
	evs, err := events.ParseEvents(bytes.NewReader(eventsJSON))
	if err != nil {
		return Result{}, err
	}
	return RenderEvents(evs, o)
}

// RenderEvents is Render for events that are already decoded.
func RenderEvents(evs []events.Event, o Options) (Result, error) {
	if err := o.Check(); err != nil {
		return Result{}, err
	}
	species, _ := creature.SpeciesByName(o.speciesName())
	now := o.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	login := o.User
	if login == "" {
		login = events.Login(evs)
	}
	res, _ := draw(login, stats.Compute(events.Activities(evs), now), species, o)
	return res, nil
}

// draw is the one place where stats become a card; o is already checked.
func draw(login string, st stats.Stats, species creature.Species, o Options) (Result, creature.Creature) {
	theme, _ := render.ThemeByName(o.themeName())
	card := render.NewCard(login, st, theme)
	card.Size, _ = render.SizeByName(o.Size)
	card.Creature.Species = species
	return Result{SVG: render.SVG(card), Description: render.Description(card), Login: login}, card.Creature
}

// BaseWorkflow is the workflow for a profile repository; the README and the
// website show this same text (a test keeps them in sync). If the push fails
// (the usual cause is a workflow without write permission) it says what to do.
const BaseWorkflow = `name: commitling

on:
  schedule:
    - cron: "23 5 * * *" # cada día a las 05:23 UTC
  workflow_dispatch:

permissions:
  contents: write

jobs:
  commitling:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: BertMarti/commitling@v1
        with:
          out: commitling.svg
      - name: Guardar la criatura si ha cambiado
        run: |
          git add commitling.svg
          if git diff --cached --quiet; then
            echo "Sin cambios."
            exit 0
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -m "chore: actualiza commitling"
          if ! git push; then
            echo "::error title=commitling::No se pudo guardar la criatura en el repositorio. Comprueba que este archivo tiene «permissions: contents: write» y, si la rama está protegida, deja que github-actions[bot] suba cambios."
            exit 1
          fi
`

// baseStep is the step of BaseWorkflow that Install fills in or repeats.
const baseStep = "      - uses: BertMarti/commitling@v1\n        with:\n          out: commitling.svg\n"

// ReadmeSnippet is the line to paste in the profile README for one card.
const ReadmeSnippet = `![commitling](./commitling.svg)`

// ReadmePicture is the README fragment for the light and dark cards: GitHub
// shows the one that matches the theme of whoever visits the profile.
const ReadmePicture = `<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./commitling-dark.svg">
  <img alt="commitling" src="./commitling.svg">
</picture>`

// ThemeBoth asks Install for the light and the dark card at once.
const ThemeBoth = "both"

// Setup is everything a person pastes into a profile repository.
type Setup struct {
	Workflow string   // the content of .github/workflows/commitling.yml
	Readme   string   // what goes in README.md
	Files    []string // the SVG files that the workflow writes and commits
}

// Install fills in BaseWorkflow with the inputs of the Action: user, and
// species, theme and size when they are not the defaults. Theme may also be
// ThemeBoth: two steps (light and dark) and a <picture> for the README.
// Everything is validated first, so nothing that is not a plain login or a
// known value is ever pasted into the YAML. The website builds the same text
// in JavaScript (install.js) and a test keeps both identical.
func Install(o Options) (Setup, error) {
	both := strings.EqualFold(strings.TrimSpace(o.Theme), ThemeBoth)
	if both {
		o.Theme = render.Light.Name
	}
	if err := o.Check(); err != nil {
		if _, ok := render.ThemeByName(o.themeName()); !ok {
			return Setup{}, errors.New("tema no válido " + strconv.Quote(o.Theme) + " (usa light, dark o both)")
		}
		return Setup{}, err
	}
	species, _ := creature.SpeciesByName(o.speciesName())
	size, _ := render.SizeByName(o.Size)
	step := func(theme render.Theme, out string) string {
		var b strings.Builder
		line := func(k, v string) { b.WriteString("          " + k + ": " + v + "\n") }
		b.WriteString("      - uses: BertMarti/commitling@v1\n        with:\n")
		if o.User != "" {
			// Always a quoted string: YAML would read 007 as a number and null or
			// true as nothing or a boolean. A valid login only has letters, digits
			// and dashes, so Go quoting is also valid YAML quoting.
			line("user", strconv.Quote(o.User))
		}
		if species != creature.MossSprout {
			line("species", species.Slug())
		}
		// The canonical name, not the raw text (spaces, capitals, line breaks).
		if theme.Name != render.Light.Name {
			line("theme", theme.Name)
		}
		if size == render.Compact {
			line("size", "compact")
		}
		line("out", out)
		return b.String()
	}
	s := Setup{Readme: ReadmeSnippet, Files: []string{"commitling.svg"}}
	var steps string
	if both {
		s.Readme = ReadmePicture
		s.Files = append(s.Files, "commitling-dark.svg")
		steps = step(render.Light, s.Files[0]) + step(render.Dark, s.Files[1])
	} else {
		theme, _ := render.ThemeByName(o.themeName())
		steps = step(theme, s.Files[0])
	}
	s.Workflow = strings.Replace(BaseWorkflow, baseStep, steps, 1)
	s.Workflow = strings.Replace(s.Workflow, "git add commitling.svg\n", "git add "+strings.Join(s.Files, " ")+"\n", 1)
	return s, nil
}

// Workflow is the workflow of Install (the live generator of the website asks
// for one card at a time).
func Workflow(o Options) (string, error) {
	s, err := Install(o)
	return s.Workflow, err
}

// FetchError explains in Spanish why the browser could not read the events of
// a user. status is the HTTP status (0 when there was no answer at all);
// remaining and reset are the raw X-RateLimit-Remaining and X-RateLimit-Reset
// headers ("" when absent or hidden by CORS).
func FetchError(status int, remaining, reset string, now time.Time) string {
	switch {
	case status == 0:
		return "No se pudo contactar con GitHub. Revisa tu conexión e inténtalo de nuevo."
	case status == 404:
		return "Ese usuario no existe en GitHub (404). Revisa que esté bien escrito."
	case status == 403 || status == 429:
		wait := "Vuelve a probar dentro de un rato: el contador se reinicia como mucho en una hora."
		if secs, err := strconv.ParseInt(reset, 10, 64); err == nil {
			at := time.Unix(secs, 0)
			mins := int((at.Sub(now) + time.Minute - 1) / time.Minute)
			unit := "minutos"
			if mins <= 1 {
				unit = "minuto"
			}
			if mins < 1 {
				// The reset the server gave is already behind us (or our clock is off).
				wait = "Ya debería haberse restablecido: vuelve a probar en 1 minuto."
			} else {
				// The time goes in the zone of now (the browser's) and rounded up, so
				// nobody comes back a few seconds too early.
				at = at.In(now.Location()).Add(59 * time.Second).Truncate(time.Minute)
				wait = "Se restablece a las " + at.Format("15:04") + " (hora local, dentro de " + strconv.Itoa(mins) + " " + unit + ")."
			}
		}
		return "GitHub deja hacer 60 peticiones por hora sin iniciar sesión y desde tu conexión ya se han gastado. " + wait +
			" Mientras tanto, pulsa «Ver demo» para ver cómo crece una criatura de ejemplo, que no gasta peticiones." +
			" La Action, con el token de tu workflow, no tiene este límite."
	case status >= 500:
		return "GitHub no responde bien ahora mismo (" + strconv.Itoa(status) + "). Inténtalo de nuevo en unos minutos."
	}
	return "GitHub respondió con el error " + strconv.Itoa(status) + ". Inténtalo de nuevo más tarde."
}
