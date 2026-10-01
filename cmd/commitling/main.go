// Command commitling draws a pixel-art creature that grows with your public
// GitHub activity.
//
//	commitling render --user <login> [--theme light|dark] [--species moss|mushroom] [--keep-on-error] [--now RFC3339] --out commitling.svg
//	commitling render --fixture testdata/events.json --out commitling.svg
//	commitling gallery --out site/
//	commitling og --out og.png
//	commitling version
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BertMarti/commitling/internal/gallery"
	"github.com/BertMarti/commitling/internal/generate"
	"github.com/BertMarti/commitling/internal/github"
	"github.com/BertMarti/commitling/internal/og"
)

// version is the CLI version; it can be overridden with -ldflags "-X main.version=...".
var version = "0.5.0"

const usage = `commitling: una mascota pixel-art que crece con tus commits.

Uso:
  commitling render --user <usuario> [--theme light|dark] [--species moss|mushroom] [--keep-on-error] [--now RFC3339] --out <archivo.svg>
  commitling render --fixture <eventos.json> [--user <nombre>] [--theme light|dark] [--species moss|mushroom] [--now RFC3339] --out <archivo.svg>
  commitling gallery --out <directorio>
  commitling og --out <archivo.png>
  commitling version

Variables de entorno:
  GITHUB_TOKEN  token opcional para la API de GitHub (más límite de peticiones)
`

// newClient builds the API client; tests replace it to point at a fake server.
var newClient = github.NewClient

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
	keep := fs.Bool("keep-on-error", false, "si la API falla de forma pasajera (red, 5xx, límite de peticiones) y --out ya es un SVG, conservarlo y terminar con un aviso")
	if err := fs.Parse(args); err != nil {
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Sprintf("argumentos de más: %s", strings.Join(fs.Args(), " "))}
	}
	opts := generate.Options{User: *user, Species: *speciesFlag, Theme: *theme}
	if err := opts.Check(); err != nil {
		return usageError{err.Error()}
	}
	if *user == "" && *fixture == "" {
		return usageError{"indica --user o --fixture"}
	}

	var events []github.Event
	var err error
	if *fixture != "" {
		events, err = readFixture(*fixture)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		events, err = newClient(os.Getenv("GITHUB_TOKEN")).FetchEvents(ctx, *user)
		if err != nil && *keep && github.IsTransient(err) && hasSVG(*out) {
			warnKept(stdout, stderr, *out, err)
			return nil
		}
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

	opts.Now = now
	res, err := generate.RenderEvents(events, opts)
	if err != nil {
		return err
	}
	svg, login := res.SVG, res.Login

	if *out == "-" {
		_, err = stdout.Write(svg)
		return err
	}
	if err := writeFile(*out, svg); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "%s → %s (%s)\n", displayName(login), *out, res.Description)
	return nil
}

// hasSVG reports whether path already holds a finished SVG (one that ends in
// </svg>): the only thing worth keeping when the API is down.
func hasSVG(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && bytes.HasSuffix(bytes.TrimSpace(data), []byte("</svg>"))
}

// warnKept tells that the previous SVG was kept. Inside GitHub Actions it is
// also a workflow annotation (one line, on stdout, where the runner reads it).
func warnKept(stdout, stderr io.Writer, out string, cause error) {
	msg := fmt.Sprintf("la API de GitHub no responde (%v); se conserva %s sin cambios", cause, out)
	fmt.Fprintln(stderr, "commitling: aviso:", msg)
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fmt.Fprintln(stdout, "::warning title=commitling::"+strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(msg))
	}
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
