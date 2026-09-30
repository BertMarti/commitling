// Command commitling draws a pixel-art creature that grows with your public
// GitHub activity.
//
//	commitling render --user <login> [--theme light|dark] [--species moss|mushroom] [--now RFC3339] --out commitling.svg
//	commitling render --fixture testdata/events.json --out commitling.svg
//	commitling gallery --out site/
//	commitling og --out og.png
//	commitling version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/gallery"
	"github.com/BertMarti/commitling/internal/github"
	"github.com/BertMarti/commitling/internal/og"
	"github.com/BertMarti/commitling/internal/render"
	"github.com/BertMarti/commitling/internal/stats"
)

// version is the CLI version; it can be overridden with -ldflags "-X main.version=...".
var version = "0.2.0"

const usage = `commitling: una mascota pixel-art que crece con tus commits.

Uso:
  commitling render --user <usuario> [--theme light|dark] [--species moss|mushroom] [--now RFC3339] --out <archivo.svg>
  commitling render --fixture <eventos.json> [--user <nombre>] [--theme light|dark] [--species moss|mushroom] [--now RFC3339] --out <archivo.svg>
  commitling gallery --out <directorio>
  commitling og --out <archivo.png>
  commitling version

Variables de entorno:
  GITHUB_TOKEN  token opcional para la API de GitHub (más límite de peticiones)
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "commitling:", err)
		var ue usageError
		if errors.As(err, &ue) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return usageError{"falta el subcomando"}
	}
	switch args[0] {
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "gallery":
		return runGallery(args[1:], stdout, stderr)
	case "og":
		return runOG(args[1:], stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "commitling %s\n", version)
		return nil
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return nil
	}
	fmt.Fprint(stderr, usage)
	return usageError{fmt.Sprintf("subcomando desconocido: %q", args[0])}
}

func runRender(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	user := fs.String("user", "", "usuario de GitHub")
	fixture := fs.String("fixture", "", "archivo JSON de eventos para trabajar sin red")
	theme := fs.String("theme", "light", "tema: light o dark")
	speciesFlag := fs.String("species", "moss", "especie: moss (brote de musgo) o mushroom (hongo)")
	nowFlag := fs.String("now", "", "fecha de referencia en RFC3339 (por defecto, ahora; con --fixture, el último evento)")
	out := fs.String("out", "commitling.svg", "archivo SVG de salida (- para la salida estándar)")
	if err := fs.Parse(args); err != nil {
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("argumentos de más: %s", strings.Join(fs.Args(), " "))}
	}
	th, ok := render.ThemeByName(*theme)
	if !ok {
		return usageError{fmt.Sprintf("tema no válido %q (usa light o dark)", *theme)}
	}
	species, ok := creature.SpeciesByName(*speciesFlag)
	if !ok {
		return usageError{fmt.Sprintf("especie no válida %q (usa moss o mushroom)", *speciesFlag)}
	}
	if *user == "" && *fixture == "" {
		return usageError{"indica --user o --fixture"}
	}
	if *user != "" && !github.ValidLogin(*user) {
		return usageError{fmt.Sprintf("nombre de usuario no válido: %q", *user)}
	}

	var events []github.Event
	var err error
	if *fixture != "" {
		events, err = readFixture(*fixture)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		events, err = github.NewClient(os.Getenv("GITHUB_TOKEN")).FetchEvents(ctx, *user)
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if *fixture != "" {
		// Fixtures are frozen in time: look at them from their last event so
		// the result does not change from one day to the next.
		if latest := github.Latest(events); !latest.IsZero() {
			now = latest
		}
	}
	if *nowFlag != "" {
		now, err = time.Parse(time.RFC3339, *nowFlag)
		if err != nil {
			return usageError{fmt.Sprintf("--now no es una fecha RFC3339 válida: %v", err)}
		}
	}

	login := *user
	if login == "" {
		login = github.Login(events)
	}
	s := stats.Compute(github.Activities(events), now)
	card := render.NewCard(login, s, th)
	card.Creature.Species = species
	svg := render.SVG(card)

	if *out == "-" {
		_, err = stdout.Write(svg)
		return err
	}
	if err := writeFile(*out, svg); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "%s → %s (%s)\n", displayName(login), *out, render.Description(card))
	return nil
}

func displayName(login string) string {
	if login == "" {
		return "commitling"
	}
	return "@" + login
}

func readFixture(path string) ([]github.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return github.ParseEvents(f)
}

func writeFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func runGallery(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gallery", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "site", "directorio de salida")
	if err := fs.Parse(args); err != nil {
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("argumentos de más: %s", strings.Join(fs.Args(), " "))}
	}
	n, err := gallery.Build(*out, version)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "galería → %s (%d archivos)\n", *out, n)
	return nil
}

func runOG(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("og", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "og.png", "archivo PNG de salida (- para la salida estándar)")
	if err := fs.Parse(args); err != nil {
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("argumentos de más: %s", strings.Join(fs.Args(), " "))}
	}
	data, err := og.PNG(og.Hero)
	if err != nil {
		return err
	}
	if *out == "-" {
		_, err = stdout.Write(data)
		return err
	}
	if err := writeFile(*out, data); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "imagen Open Graph → %s (%dx%d)\n", *out, og.Width, og.Height)
	return nil
}
