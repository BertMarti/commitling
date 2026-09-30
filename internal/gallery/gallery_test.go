package gallery

import (
	"bytes"
	"encoding/xml"
	"image/png"
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
	// Per species, 20 stage×mood cards and 4 accessory cards, light and
	// dark, plus favicon, hero, og.png and index.
	if want := 2*(20+4)*2 + 4; n != want {
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
		"Brote de musgo",
		"Espora",
		"Corro de setas",
		"species: mushroom",
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
	if len(refs) < 96 {
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

func buildPage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := Build(dir, "test"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// meta returns the content of <meta name|property="key" content="...">.
func meta(page, key string) string {
	re := regexp.MustCompile(`<meta (?:name|property)="` + regexp.QuoteMeta(key) + `" content="([^"]*)"`)
	if m := re.FindStringSubmatch(page); m != nil {
		return m[1]
	}
	return ""
}

func TestPageHeadForSharing(t *testing.T) {
	page := buildPage(t)
	if !strings.HasPrefix(page, "<!doctype html>\n<html lang=\"es\">") {
		t.Error(`page must start with <!doctype html> and <html lang="es">`)
	}
	if !strings.Contains(page, `<meta name="viewport" content="width=device-width, initial-scale=1">`) {
		t.Error("missing viewport meta")
	}
	title := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(page)
	if title == nil {
		t.Fatal("missing <title>")
	}
	desc := meta(page, "description")
	if len(desc) < 50 || len(desc) > 200 {
		t.Errorf("meta description has %d characters: %q", len(desc), desc)
	}
	for key, want := range map[string]string{
		"og:title":        title[1],
		"og:description":  desc,
		"og:image":        "https://bertmarti.github.io/commitling/og.png",
		"og:image:type":   "image/png",
		"og:image:width":  "1200",
		"og:image:height": "630",
		"og:url":          "https://bertmarti.github.io/commitling/",
		"og:type":         "website",
		"twitter:card":    "summary_large_image",
	} {
		if got := meta(page, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if meta(page, "og:image:alt") == "" {
		t.Error("og:image needs an alt text")
	}
}

func TestCopyButtonsAreAccessible(t *testing.T) {
	page := buildPage(t)
	buttons := regexp.MustCompile(`<button [^>]*class="copy"[^>]*>[^<]*</button>`).FindAllString(page, -1)
	if len(buttons) != 3 {
		t.Fatalf("found %d copy buttons, want 3", len(buttons))
	}
	labels := map[string]bool{}
	for _, b := range buttons {
		m := regexp.MustCompile(`aria-label="([^"]+)"`).FindStringSubmatch(b)
		if m == nil {
			t.Errorf("copy button without aria-label: %s", b)
			continue
		}
		// The accessible name must start with the visible text.
		if !strings.HasPrefix(m[1], "Copiar") {
			t.Errorf("aria-label %q should start with the visible text", m[1])
		}
		if labels[m[1]] {
			t.Errorf("two copy buttons share the label %q", m[1])
		}
		labels[m[1]] = true
		if !strings.Contains(b, `type="button"`) {
			t.Errorf("copy button must be type=button: %s", b)
		}
	}
	if got := strings.Count(page, `role="status" aria-live="polite"`); got != 3 {
		t.Errorf("found %d live regions for copy feedback, want 3", got)
	}
	// If the clipboard fails the user gets a message and the text is selected.
	for _, want := range []string{"No se pudo copiar", "navigator.clipboard", "execCommand", "selectNodeContents"} {
		if !strings.Contains(page, want) {
			t.Errorf("copy script does not handle %q", want)
		}
	}
}

func TestPageImagesHaveAltAndSize(t *testing.T) {
	page := buildPage(t)
	for _, img := range regexp.MustCompile(`<img [^>]*>`).FindAllString(page, -1) {
		if !strings.Contains(img, "alt=") {
			t.Errorf("image without alt: %s", img)
		}
		if !strings.Contains(img, "width=") || !strings.Contains(img, "height=") {
			t.Errorf("image without size (layout shift): %s", img)
		}
	}
}

func TestBuildWritesOpenGraphPNG(t *testing.T) {
	dir := t.TempDir()
	if _, err := Build(dir, "test"); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "og.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("og.png is not a valid PNG: %v", err)
	}
	if cfg.Width != 1200 || cfg.Height != 630 {
		t.Fatalf("og.png is %dx%d, want 1200x630", cfg.Width, cfg.Height)
	}
	if !strings.HasSuffix(OGImage, "/og.png") {
		t.Errorf("OGImage = %q, want the PNG", OGImage)
	}
}

func TestBuildShowsBothSpecies(t *testing.T) {
	dir := t.TempDir()
	if _, err := Build(dir, "test"); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)
	for _, sp := range creature.AllSpecies {
		if !strings.Contains(page, `id="especie-`+sp.Slug()+`"`) {
			t.Errorf("no section for %s", sp.Name())
		}
		prefix := ""
		if sp != creature.MossSprout {
			prefix = sp.Slug() + "-"
		}
		for _, st := range creature.Stages {
			for _, m := range creature.Moods {
				for _, suffix := range []string{".svg", "-dark.svg"} {
					name := "svg/" + prefix + st.Slug() + "-" + m.Slug() + suffix
					data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
					if err != nil {
						t.Fatalf("missing %s: %v", name, err)
					}
					if !strings.Contains(string(data), sp.StageName(st)) {
						t.Errorf("%s does not name the stage %q", name, sp.StageName(st))
					}
				}
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "svg", prefix+"acc-all.svg")); err != nil {
			t.Errorf("missing accessory card for %s: %v", sp.Name(), err)
		}
	}
	// The rules table lists both names of each stage.
	if !strings.Contains(page, "Semilla / Espora") || !strings.Contains(page, "Árbol ancestral / Corro de setas") {
		t.Error("the stages table does not show both species")
	}
}
