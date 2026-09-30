package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := run([]string{"version"}, &stdout, &stderr); err != nil || stdout.String() != "commitling 0.2.0\n" {
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
