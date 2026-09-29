package gallery

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
)

func TestSampleMatchesStageAndMood(t *testing.T) {
	for _, st := range creature.Stages {
		for _, m := range creature.Moods {
			got := creature.FromStats(Sample(st, m))
			want := creature.Creature{Stage: st, Mood: m}
			if got != want {
				t.Errorf("Sample(%s, %s) gives %+v", st.Name(), m.Name(), got)
			}
		}
	}
}

func TestBuild(t *testing.T) {
	dir := t.TempDir()
	n, err := Build(dir, "test")
	if err != nil {
		t.Fatal(err)
	}
	// 20 stage×mood cards and 4 accessory cards, light and dark, plus
	// favicon, hero and index.
	if want := (20+4)*2 + 3; n != want {
		t.Fatalf("Build wrote %d files, want %d", n, want)
	}

	html, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)
	for _, want := range []string{
		`<html lang="es">`,
		"live/BertMarti.svg",
		"BertMarti/commitling@main",
		"Árbol ancestral",
		"Radiante",
		"Privacidad",
		"contents: write",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}

	// Every local image referenced by the page exists and is valid XML.
	re := regexp.MustCompile(`(?:src|srcset|href)="((?:svg/|favicon|hero)[^"]+)"`)
	refs := re.FindAllStringSubmatch(page, -1)
	if len(refs) < 48 {
		t.Fatalf("only %d local images referenced", len(refs))
	}
	for _, m := range refs {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(m[1])))
		if err != nil {
			t.Errorf("missing %s: %v", m[1], err)
			continue
		}
		dec := xml.NewDecoder(bytes.NewReader(data))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Errorf("%s is not valid XML: %v", m[1], err)
				break
			}
		}
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if _, err := Build(a, "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(b, "x"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "svg/shrub-happy.svg", "svg/acc-all-dark.svg"} {
		x, _ := os.ReadFile(filepath.Join(a, name))
		y, _ := os.ReadFile(filepath.Join(b, name))
		if len(x) == 0 || !bytes.Equal(x, y) {
			t.Errorf("%s differs between builds", name)
		}
	}
}

// The README must show the same workflow as the site.
func TestReadmeHasSameSnippets(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(readme), "\r\n", "\n")
	if !strings.Contains(text, WorkflowSnippet) {
		t.Error("README.md does not contain gallery.WorkflowSnippet verbatim")
	}
	if !strings.Contains(text, ReadmeSnippet) {
		t.Error("README.md does not contain gallery.ReadmeSnippet")
	}
}
