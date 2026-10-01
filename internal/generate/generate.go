// Package generate is the single place where public GitHub events become a
// commitling SVG. The command line and the WebAssembly build (the live
// generator of the website) both call it, so they draw the same bytes for
// the same events and the same date.
package generate

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/events"
	"github.com/BertMarti/commitling/internal/render"
	"github.com/BertMarti/commitling/internal/stats"
)

// Options say what to draw. The zero value means: the login found in the
// events, the moss sprout, the light theme and "now".
type Options struct {
	User    string    // GitHub login; empty takes the most common actor of the events
	Species string    // moss or mushroom (or an alias); empty is moss
	Theme   string    // light or dark; empty is light
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
		return fmt.Errorf("tema no válido %q (usa light o dark)", o.Theme)
	}
	if _, ok := creature.SpeciesByName(o.speciesName()); !ok {
		return fmt.Errorf("especie no válida %q (usa moss o mushroom)", o.Species)
	}
	if o.User != "" && !events.ValidLogin(o.User) {
		return fmt.Errorf("nombre de usuario no válido: %q", o.User)
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
	theme, _ := render.ThemeByName(o.themeName())
	species, _ := creature.SpeciesByName(o.speciesName())
	now := o.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	login := o.User
	if login == "" {
		login = events.Login(evs)
	}
	card := render.NewCard(login, stats.Compute(events.Activities(evs), now), theme)
	card.Creature.Species = species
	return Result{SVG: render.SVG(card), Description: render.Description(card), Login: login}, nil
}

// BaseWorkflow is the workflow for a profile repository; the README and the
// website show this same text (a test keeps them in sync).
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
      - name: Guardar el SVG si ha cambiado
        run: |
          git add commitling.svg
          if git diff --cached --quiet; then
            echo "Sin cambios."
            exit 0
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -m "chore: actualiza commitling"
          git push
`

// Workflow returns BaseWorkflow with the inputs of the Action filled in:
// user, and species and theme when they are not the defaults. Everything is
// validated first, so nothing that is not a plain login or a known value is
// ever pasted into the YAML.
func Workflow(o Options) (string, error) {
	if err := o.Check(); err != nil {
		return "", err
	}
	var inputs strings.Builder
	line := func(k, v string) { fmt.Fprintf(&inputs, "          %s: %s\n", k, v) }
	if o.User != "" {
		// Always a quoted string: YAML would read 007 as a number and null or
		// true as nothing or a boolean. A valid login only has letters, digits
		// and dashes, so Go quoting is also valid YAML quoting.
		line("user", strconv.Quote(o.User))
	}
	if sp, _ := creature.SpeciesByName(o.speciesName()); sp != creature.MossSprout {
		line("species", sp.Slug())
	}
	// The canonical name, not the raw text (spaces, capitals, line breaks).
	if theme, _ := render.ThemeByName(o.themeName()); theme.Name != render.Light.Name {
		line("theme", theme.Name)
	}
	return strings.Replace(BaseWorkflow, "        with:\n", "        with:\n"+inputs.String(), 1), nil
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
			mins := int((time.Unix(secs, 0).Sub(now) + time.Minute - 1) / time.Minute)
			if mins < 1 {
				mins = 1
			}
			unit := "minutos"
			if mins == 1 {
				unit = "minuto"
			}
			wait = fmt.Sprintf("Vuelve a probar en %d %s.", mins, unit)
		}
		return "GitHub deja hacer 60 peticiones por hora sin iniciar sesión y desde tu conexión ya se han gastado. " + wait +
			" La Action, con el token de tu workflow, no tiene este límite."
	case status >= 500:
		return fmt.Sprintf("GitHub no responde bien ahora mismo (%d). Inténtalo de nuevo en unos minutos.", status)
	}
	return fmt.Sprintf("GitHub respondió con el error %d. Inténtalo de nuevo más tarde.", status)
}
