package main

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BertMarti/commitling/internal/generate"
	"github.com/BertMarti/commitling/internal/github"
)

const fixture = "../../testdata/events.json"

func TestRenderFixture(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sub", "c.svg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", "--fixture", fixture, "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("render: %v\n%s", err, stderr.String())
	}
	svg, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"@octoexample", "Retoño", "Radiante", "flor"} {
		if !bytes.Contains(svg, []byte(want)) {
			t.Errorf("SVG does not contain %q", want)
		}
	}
}

func TestRenderFixtureIsStable(t *testing.T) {
	render := func(extra ...string) string {
		var stdout, stderr bytes.Buffer
		args := append([]string{"render", "--fixture", fixture, "--out", "-"}, extra...)
		if err := run(args, &stdout, &stderr); err != nil {
			t.Fatalf("render %v: %v", extra, err)
		}
		return stdout.String()
	}
	a, b := render(), render()
	if a != b {
		t.Fatal("two renders of the fixture differ")
	}
	if !strings.HasPrefix(a, "<svg") {
		t.Fatalf("stdout does not start with <svg: %.40q", a)
	}
	// Looking at the fixture a month later, the creature is asleep.
	late := render("--now", "2026-11-01T00:00:00Z")
	if !strings.Contains(late, "Durmiendo") {
		t.Error("with --now a month later the creature should sleep")
	}
	if dark := render("--theme", "dark", "--user", "someone"); !strings.Contains(dark, "@someone") || !strings.Contains(dark, "#2b2724") {
		t.Error("--theme dark / --user not applied")
	}
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"nope"},
		{"render"},
		{"render", "--fixture", fixture, "--theme", "sepia"},
		{"render", "--fixture", fixture, "--now", "ayer"},
		{"render", "--user", "../../etc"},
		{"render", "--fixture", fixture, "extra"},
		{"gallery", "extra"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		var ue usageError
		if !errors.As(err, &ue) {
			t.Errorf("run(%q) = %v, want usage error", args, err)
		}
	}
}

func TestMissingFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", "--fixture", "no-existe.json", "--out", "-"}, &stdout, &stderr); err == nil {
		t.Fatal("expected error for missing fixture")
	}
}

func TestVersionAndGallery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, &stdout, &stderr); err != nil || stdout.String() != "commitling 0.5.0\n" {
		t.Fatalf("version: %v %q", err, stdout.String())
	}
	dir := t.TempDir()
	if err := run([]string{"gallery", "--out", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		t.Fatal(err)
	}
}

func TestOGCommandWritesPNG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sub", "og.png")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"og", "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("og: %v\n%s", err, stderr.String())
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 1200 || b.Dy() != 630 {
		t.Fatalf("og.png is %dx%d, want 1200x630", b.Dx(), b.Dy())
	}

	// To stdout it gives the same bytes, and extra arguments are rejected.
	stdout.Reset()
	if err := run([]string{"og", "--out", "-"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	disk, _ := os.ReadFile(out)
	if !bytes.Equal(stdout.Bytes(), disk) {
		t.Error("og --out - differs from the file")
	}
	var ue usageError
	if err := run([]string{"og", "sobra"}, &stdout, &stderr); !errors.As(err, &ue) {
		t.Errorf("og with extra args: err = %v, want a usage error", err)
	}
}

func TestRenderSpecies(t *testing.T) {
	render := func(args ...string) (string, error) {
		var stdout, stderr bytes.Buffer
		full := append([]string{"render", "--fixture", fixture, "--out", "-"}, args...)
		err := run(full, &stdout, &stderr)
		return stdout.String(), err
	}
	def, err := render()
	if err != nil {
		t.Fatal(err)
	}
	moss, err := render("--species", "moss")
	if err != nil {
		t.Fatal(err)
	}
	if def != moss {
		t.Error("moss must be the default species")
	}
	mush, err := render("--species", "mushroom")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mush, "Seta") || strings.Contains(mush, "Retoño") {
		t.Error("--species mushroom did not draw the mushroom")
	}
	if again, _ := render("--species", "mushroom"); again != mush {
		t.Error("the mushroom render is not deterministic")
	}
	// The Spanish name works too, and an unknown species is a usage error.
	if hongo, err := render("--species", "hongo"); err != nil || hongo != mush {
		t.Errorf("--species hongo: err=%v, same=%v", err, hongo == mush)
	}
	var ue usageError
	if _, err := render("--species", "dragon"); !errors.As(err, &ue) {
		t.Errorf("unknown species: err = %v, want a usage error", err)
	}
}

// An invalid species is rejected before anything is fetched or written, and
// the message says which values are valid.
func TestRenderInvalidSpeciesFailsEarlyAndClearly(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.svg")
	for _, bad := range []string{"dragon", "moss,mushroom", "hongos", "Hóngo", "../etc"} {
		var stdout, stderr bytes.Buffer
		// --user without a fixture would go to the network: it must not get there.
		err := run([]string{"render", "--user", "octoexample", "--species", bad, "--out", out}, &stdout, &stderr)
		var ue usageError
		if !errors.As(err, &ue) {
			t.Errorf("%q: err = %v, want a usage error", bad, err)
			continue
		}
		for _, want := range []string{"especie no válida", bad, "moss", "mushroom"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%q: message %q should contain %q", bad, err, want)
			}
		}
		if _, statErr := os.Stat(out); statErr == nil {
			t.Errorf("%q: the output file was written", bad)
		}
	}
	// Case, spaces and the empty value (an unset Action input) are fine.
	for _, ok := range []string{"", "MUSHROOM", " Hongo ", "Seta", "MOSS", "Musgo"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{"render", "--fixture", fixture, "--species", ok, "--out", "-"}, &stdout, &stderr); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
}

// apiAnswering points the CLI at a server that always answers status (with
// headers) and returns how many requests it got; status 0 means a server that
// is not there (connection refused). Retries do not wait.
func apiAnswering(t *testing.T, status int, headers http.Header) *atomic.Int32 {
	t.Helper()
	hits := new(atomic.Int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		for k, v := range headers {
			w.Header()[k] = v
		}
		http.Error(w, "fallo de prueba", status)
	}))
	if status == 0 {
		srv.Close()
	} else {
		t.Cleanup(srv.Close)
	}
	old := newClient
	newClient = func(token string) *github.Client {
		c := github.NewClient(token)
		c.BaseURL = srv.URL
		c.Sleep = func(context.Context, time.Duration) error { return nil }
		return c
	}
	t.Cleanup(func() { newClient = old })
	return hits
}

// apiDown is a server with a 503.
func apiDown(t *testing.T) *atomic.Int32 { return apiAnswering(t, http.StatusServiceUnavailable, nil) }

func TestKeepOnErrorKeepsThePreviousSVG(t *testing.T) {
	hits := apiDown(t)
	t.Setenv("GITHUB_ACTIONS", "") // CI sets it to true
	out := filepath.Join(t.TempDir(), "commitling.svg")
	previous := []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"><title>ayer</title></svg>\n")
	if err := os.WriteFile(out, previous, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", "--user", "octoexample", "--keep-on-error", "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("with a previous SVG the run must end with a warning, got: %v", err)
	}
	if hits.Load() == 0 {
		t.Error("the API was never asked")
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, previous) {
		t.Errorf("the previous SVG changed: %q", got)
	}
	for _, want := range []string{"aviso", "conserva", out, "503"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr should mention %q:\n%s", want, stderr.String())
		}
	}
	if strings.Contains(stdout.String(), "::warning") {
		t.Error("the workflow annotation must only appear inside GitHub Actions")
	}

	// Inside GitHub Actions the warning is also an annotation.
	t.Setenv("GITHUB_ACTIONS", "true")
	stdout.Reset()
	if err := run([]string{"render", "--user", "octoexample", "--keep-on-error", "--out", out}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout.String(), "::warning title=commitling::") || strings.Count(stdout.String(), "\n") != 1 {
		t.Errorf("want a one-line ::warning annotation, got %q", stdout.String())
	}
}

func TestKeepOnErrorWithoutPreviousSVGStillFails(t *testing.T) {
	apiDown(t)
	dir := t.TempDir()
	cases := map[string]string{
		"no file":      filepath.Join(dir, "no-existe.svg"),
		"empty file":   filepath.Join(dir, "vacio.svg"),
		"not an SVG":   filepath.Join(dir, "roto.svg"),
		"standard out": "-",
	}
	writeTest(t, cases["empty file"], "")
	writeTest(t, cases["not an SVG"], "<svg><g>cortado")
	for name, out := range cases {
		var stdout, stderr bytes.Buffer
		err := run([]string{"render", "--user", "octoexample", "--keep-on-error", "--out", out}, &stdout, &stderr)
		if err == nil {
			t.Errorf("%s: the API error must be kept", name)
		} else if !strings.Contains(err.Error(), "503") {
			t.Errorf("%s: the error lost its cause: %v", name, err)
		}
	}
}

// Without the flag nothing changes: the error stays and the file is not touched.
func TestAPIErrorWithoutKeepOnErrorFails(t *testing.T) {
	apiDown(t)
	out := filepath.Join(t.TempDir(), "commitling.svg")
	previous := []byte("<svg></svg>")
	writeTest(t, out, string(previous))
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", "--user", "octoexample", "--out", out}, &stdout, &stderr); err == nil {
		t.Fatal("expected the API error")
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, previous) {
		t.Error("a failed run must not touch the file")
	}
}

// --keep-on-error is only for API failures: a bad fixture is still an error.
func TestKeepOnErrorDoesNotHideOtherErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "commitling.svg")
	writeTest(t, out, "<svg></svg>")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", "--fixture", "no-existe.json", "--keep-on-error", "--out", out}, &stdout, &stderr); err == nil {
		t.Error("a missing fixture must stay an error")
	}
}

func writeTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Only passing failures keep the previous SVG; the ones that will fail the
// same way tomorrow (a wrong user or token) must stay visible as errors.
func TestKeepOnErrorOnlyForTransientFailures(t *testing.T) {
	limit := http.Header{"X-RateLimit-Remaining": {"0"}}
	cases := []struct {
		name      string
		status    int
		headers   http.Header
		transient bool
	}{
		{"network error", 0, nil, true},
		{"500", 500, nil, true},
		{"503", 503, nil, true},
		{"429", 429, nil, true},
		{"403 rate limit", 403, limit, true},
		{"404 unknown user", 404, nil, false},
		{"401 bad token", 401, nil, false},
		{"422", 422, nil, false},
		{"403 without rate-limit headers", 403, nil, false},
	}
	for _, c := range cases {
		apiAnswering(t, c.status, c.headers)
		out := filepath.Join(t.TempDir(), "commitling.svg")
		previous := "<svg>ayer</svg>\n"
		writeTest(t, out, previous)
		var stdout, stderr bytes.Buffer
		err := run([]string{"render", "--user", "octoexample", "--keep-on-error", "--out", out}, &stdout, &stderr)
		if c.transient && err != nil {
			t.Errorf("%s: should keep the SVG and warn, got %v", c.name, err)
		}
		if !c.transient && err == nil {
			t.Errorf("%s: a permanent error must fail even with --keep-on-error", c.name)
		}
		if got, _ := os.ReadFile(out); string(got) != previous {
			t.Errorf("%s: the file changed: %q", c.name, got)
		}
	}
}

// What counts as a finished SVG: trailing whitespace (CRLF included) and a
// BOM at the start do not matter; a cut-off or empty file does.
func TestHasSVG(t *testing.T) {
	cases := map[string]struct {
		content string
		want    bool
	}{
		"plain":        {"<svg></svg>", true},
		"final LF":     {"<svg></svg>\n", true},
		"final CRLF":   {"<svg></svg>\r\n", true},
		"BOM at start": {"\ufeff<svg></svg>\n", true},
		"empty":        {"", false},
		"only spaces":  {" \r\n", false},
		"cut off":      {"<svg><g>", false},
		"not an SVG":   {"<html></html>", false},
	}
	dir := t.TempDir()
	for name, c := range cases {
		p := filepath.Join(dir, strings.ReplaceAll(name, " ", "_"))
		writeTest(t, p, c.content)
		if got := hasSVG(p); got != c.want {
			t.Errorf("%s: hasSVG = %v, want %v", name, got, c.want)
		}
	}
	if hasSVG(filepath.Join(dir, "no-existe")) {
		t.Error("a missing file is not an SVG")
	}
}

// The website draws with generate.Render compiled to WebAssembly: the CLI must
// give the very same bytes for the same events and the same date.
func TestRenderMatchesGenerateRender(t *testing.T) {
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	for _, sp := range []string{"moss", "mushroom"} {
		for _, th := range []string{"light", "dark"} {
			var stdout, stderr bytes.Buffer
			args := []string{"render", "--fixture", fixture, "--user", "octoexample", "--species", sp, "--theme", th,
				"--now", now.Format(time.RFC3339), "--out", "-"}
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatalf("%s/%s: %v", sp, th, err)
			}
			want, err := generate.Render(data, generate.Options{User: "octoexample", Species: sp, Theme: th, Now: now})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(stdout.Bytes(), want.SVG) {
				t.Errorf("%s/%s: the CLI and generate.Render draw different SVGs", sp, th)
			}
		}
	}
}
