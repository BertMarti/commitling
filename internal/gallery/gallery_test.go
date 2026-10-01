package gallery

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	// dark, plus favicon, og.png, generator.js and index (the hero of the header is
	// one of the accessory cards), plus the compact badges: per species, one
	// per stage, light and dark.
	if want := 2*(20+4)*2 + 4 + 2*5*2; n != want {
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
		"BertMarti/commitling@v1",
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
	re := regexp.MustCompile(`(?:src|srcset|href)="((?:svg/|favicon)[^"]+)"`)
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
	// The PNG shows both species, so its alt text must say so.
	if alt := strings.ToLower(meta(page, "og:image:alt")); !strings.Contains(alt, "hongo") || !strings.Contains(alt, "musgo") {
		t.Errorf("og:image:alt %q should mention both species", alt)
	}
}

func TestCopyButtonsAreAccessible(t *testing.T) {
	page := buildPage(t)
	buttons := regexp.MustCompile(`<button [^>]*class="copy"[^>]*>[^<]*</button>`).FindAllString(page, -1)
	if len(buttons) != 4 {
		t.Fatalf("found %d copy buttons, want 4", len(buttons))
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
	if got := strings.Count(page, `class="copy-msg note" role="status" aria-live="polite"`); got != 4 {
		t.Errorf("found %d live regions for copy feedback, want 4", got)
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

func luminance(hex string) float64 {
	var c [3]float64
	for i := range c {
		v, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		f := float64(v) / 255
		if f <= 0.03928 {
			c[i] = f / 12.92
		} else {
			c[i] = math.Pow((f+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Text colours of the page, in both themes, reach WCAG AA (4.5:1) on the
// backgrounds they are drawn on; the focus outline needs 3:1.
func TestPageColoursReachAA(t *testing.T) {
	page := buildPage(t)
	vars := func(block string) map[string]string {
		m := map[string]string{}
		for _, kv := range regexp.MustCompile(`--([a-z]+):(#[0-9a-f]{6})`).FindAllStringSubmatch(block, -1) {
			m[kv[1]] = kv[2]
		}
		return m
	}
	light := vars(regexp.MustCompile(`(?s):root\{(.*?)\}`).FindStringSubmatch(page)[1])
	darkOver := vars(regexp.MustCompile(`(?s)prefers-color-scheme:dark\)\{\s*:root\{(.*?)\}`).FindStringSubmatch(page)[1])
	dark := map[string]string{}
	for k, v := range light {
		dark[k] = v
	}
	for k, v := range darkOver {
		dark[k] = v
	}
	for name, th := range map[string]map[string]string{"claro": light, "oscuro": dark} {
		for _, p := range []struct {
			fg, bg string
			min    float64
		}{
			{"ink", "paper", 4.5}, {"muted", "paper", 4.5}, {"muted", "code", 4.5},
			{"ink", "code", 4.5}, {"accent", "paper", 4.5}, // links on hover
		} {
			if got := contrast(th[p.fg], th[p.bg]); got < p.min {
				t.Errorf("tema %s: %s (%s) sobre %s (%s) tiene contraste %.2f, mínimo %.1f", name, p.fg, th[p.fg], p.bg, th[p.bg], got, p.min)
			}
		}
	}
}

// A keyboard user can skip the header and jump between the species.
func TestPageHasSkipLinkAndSpeciesNav(t *testing.T) {
	page := buildPage(t)
	if !strings.Contains(page, `href="#contenido"`) || !strings.Contains(page, `<main id="contenido"`) {
		t.Error("no skip link to the main content")
	}
	if !strings.Contains(page, `aria-label="Especies"`) {
		t.Error("no navigation between species")
	}
	for _, sp := range creature.AllSpecies {
		if !strings.Contains(page, `<a href="#especie-`+sp.Slug()+`">`+sp.Name()+`</a>`) {
			t.Errorf("no link to the %s section", sp.Name())
		}
	}
	// The skip link is hidden until it gets focus.
	if !strings.Contains(page, ".skip:focus") {
		t.Error("the skip link has no focus style")
	}
}

// Every image of the gallery has a text alternative, and no two images of
// the same section share it (each card says its stage, mood and accessories).
func TestGalleryAltsAreUnique(t *testing.T) {
	page := buildPage(t)
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`<img src="(svg/[^"]+)"[^>]*alt="([^"]*)"`).FindAllStringSubmatch(page, -1) {
		if m[2] == "" {
			t.Errorf("%s has an empty alt", m[1])
		}
		if strings.HasSuffix(m[1], "-dark.svg") {
			continue
		}
		key := m[2]
		if seen[key] {
			t.Errorf("duplicated alt %q", key)
		}
		seen[key] = true
	}
	if len(seen) < 40 {
		t.Errorf("only %d cards with alt, want 2 species x 24", len(seen))
	}
}

// The header shows the real card, animated, and respects reduced motion by
// itself: the SVG carries its own prefers-reduced-motion rule, which also
// works inside an <img>.
func TestHeaderShowsTheAnimatedCard(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	m := regexp.MustCompile(`(?s)<picture class="hero">.*?srcset="(svg/[^"]+-dark\.svg)".*?<img src="(svg/[^"]+\.svg)" width="480" height="200" alt="([^"]+)"`).FindStringSubmatch(page)
	if m == nil {
		t.Fatal("the header has no picture with the card (light and dark)")
	}
	for _, f := range []string{m[1], m[2]} {
		svg := readBuilt(t, dir, f)
		for _, want := range []string{"@keyframes", "prefers-reduced-motion:reduce"} {
			if !strings.Contains(svg, want) {
				t.Errorf("%s lacks %q", f, want)
			}
		}
	}
	if strings.Contains(page, `width="112"`) {
		t.Error("the old fixed 112 px sprite is still in the header")
	}
}

// pageCSS returns the CSS of the page as rules.
func pageCSS(t *testing.T) ruleSet {
	t.Helper()
	return cssRules(regexp.MustCompile(`(?s)<style>(.*?)</style>`).FindStringSubmatch(buildPage(t))[1])
}

// phone is the @media condition of the phone rules of the page.
const phone = "(max-width:720px)"

// effective is the value of a property on a phone: the phone rule if there is
// one, otherwise the general one.
func (rs ruleSet) effective(t *testing.T, sel, property string) string {
	t.Helper()
	if v := rs.propIn(t, phone, sel, property); v != "" {
		return v
	}
	return rs.prop(sel, property)
}

// withoutMinmax removes every minmax(...) (the parentheses may nest) from a
// grid-template-columns value.
func withoutMinmax(v string) string {
	for {
		i := strings.Index(v, "minmax(")
		if i < 0 {
			return v
		}
		depth, j := 0, i+len("minmax")
		for ; j < len(v); j++ {
			if v[j] == '(' {
				depth++
			} else if v[j] == ')' {
				if depth--; depth == 0 {
					break
				}
			}
		}
		v = v[:i] + v[min(j+1, len(v)):]
	}
}

// What made the page scroll sideways on a phone. It was measured twice:
// at 375 px the document was 730 px wide (the <pre> blocks of "Instálalo" in
// a grid track that grew with them); at 320 px it was 338 px (an unbreakable
// <code>.github/workflows/commitling.yml</code> in a list item, plus grids
// whose implicit column grew with its content and a table that could not
// wrap its first column). The intent, rule by rule, not the CSS text:
//
//  1. a grid track never grows with its content (no plain `fr`: minmax(0,1fr));
//  2. every grid says its columns (an implicit one is `auto`, which grows);
//  3. the one-column layouts of the phone can shrink;
//  4. inline code can wrap anywhere, and tables can wrap their first column;
//  5. images have no fixed width in CSS.
func TestPageDoesNotOverflowOnPhones(t *testing.T) {
	rules := pageCSS(t)

	for _, r := range rules {
		cols := r.props["grid-template-columns"]
		if left := withoutMinmax(cols); regexp.MustCompile(`\d*\.?\d+fr`).MatchString(left) {
			t.Errorf("%q has a plain fr track (%s): it grows with its content, use minmax(0,1fr)", r.sel, cols)
		}
		if r.props["display"] == "grid" && cols == "" && !strings.Contains(r.sel, "::") {
			t.Errorf("%q is a grid with no grid-template-columns: its implicit column is auto and grows with its content", r.sel)
		}
	}

	for _, sel := range []string{".steps", ".gen", ".gen-view", ".grid", ".tables"} {
		if got := rules.effective(t, sel, "grid-template-columns"); !strings.Contains(got, "minmax(0,1fr)") {
			t.Errorf("on a phone %q has columns %q, want minmax(0,1fr)", sel, got)
		}
	}
	if rules.prop(".steps li", "min-width") != "0" {
		t.Error("a list item of .steps must be able to shrink (min-width:0)")
	}

	for _, sel := range []string{"p code", "li code", "td code"} {
		if got := rules.prop(sel, "overflow-wrap"); got != "anywhere" {
			t.Errorf("%q: overflow-wrap = %q; an unbreakable path in code would push the page wider", sel, got)
		}
	}
	if got := rules.prop("pre", "overflow-x"); got != "auto" {
		t.Errorf("code blocks must scroll inside themselves (overflow-x:auto), got %q", got)
	}
	if got := rules.propIn(t, phone, "td:first-child", "white-space"); got != "normal" {
		t.Errorf("on a phone the first column of a table must be able to wrap, white-space = %q", got)
	}

	for _, r := range rules {
		last := r.sel[strings.LastIndex(r.sel, " ")+1:]
		if (last == "img" || strings.HasSuffix(last, " img")) && regexp.MustCompile(`^\d+px$`).MatchString(r.props["width"]) {
			t.Errorf("%q has a fixed width in px: %s", r.sel, r.props["width"])
		}
	}
}

// The preview box is a blank card (same ratio and rounded frame as the SVG),
// not a dashed box with dead space around the card.
func TestGeneratorStageMatchesTheCard(t *testing.T) {
	rules := pageCSS(t)
	for _, sel := range []string{".gen-stage", ".gen-stage img", ".gen-empty"} {
		if v := rules.prop(sel, "min-height"); v != "" {
			t.Errorf("%q has a minimum height (%s): the box is as tall as the card", sel, v)
		}
		for _, p := range []string{"border", "border-style"} {
			if strings.Contains(rules.prop(sel, p), "dashed") {
				t.Errorf("%q has a dashed %s: the frame of the card is a plain line", sel, p)
			}
		}
	}
	if got := rules.prop(".gen-stage img", "aspect-ratio"); got != "12/5" {
		t.Errorf("the preview image keeps the ratio of the card (12/5), got %q", got)
	}
	empty := func(p string) string { return rules.prop(".gen-empty", p) }
	if empty("aspect-ratio") != "12/5" || empty("border-radius") != "8px" || !strings.HasPrefix(empty("border"), "1px solid") {
		t.Errorf("the empty stage must look like the card: aspect-ratio 12/5, 1px solid frame, radius 8px (got %q, %q, %q)",
			empty("aspect-ratio"), empty("border"), empty("border-radius"))
	}
}

// Adding the compact card must not change a single byte of the full cards of
// v0.5.0. testdata/golden/full-sha256.txt holds the SHA-256 of each of them
// (sha256sum format), taken from the v0.5.0 gallery before render.go changed.
func TestFullCardsAreByteIdenticalToV050(t *testing.T) {
	list, err := os.ReadFile("../../testdata/golden/full-sha256.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := Build(dir, "test"); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(strings.TrimSpace(strings.ReplaceAll(string(list), "\r\n", "\n")), "\n") {
		sum, name, ok := strings.Cut(line, "  ")
		if !ok {
			t.Fatalf("bad line in the golden list: %q", line)
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != sum {
			t.Errorf("%s changed: sha256 %s, v0.5.0 had %s", name, got, sum)
		}
		n++
	}
	if want := 2 * (20 + 4) * 2; n != want {
		t.Errorf("the golden list has %d cards, want %d", n, want)
	}
}

// The gallery shows the compact badge for every stage of both species, light
// and dark, at its real size, with its own text alternative.
func TestGalleryShowsTheCompactBadge(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	if !strings.Contains(page, `id="compacta"`) || !strings.Contains(page, "size: compact") {
		t.Error("the gallery has no compact section that says how to ask for it")
	}
	if !strings.Contains(page, `href="#compacta"`) {
		t.Error("nothing links to the compact section")
	}
	for _, sp := range creature.AllSpecies {
		prefix := ""
		if sp != creature.MossSprout {
			prefix = sp.Slug() + "-"
		}
		for _, st := range creature.Stages {
			for _, suffix := range []string{".svg", "-dark.svg"} {
				name := "svg/compact-" + prefix + st.Slug() + suffix
				svg := readBuilt(t, dir, name)
				mustBeValidXML(t, name, svg)
				if !strings.Contains(svg, `width="200" height="60"`) || !strings.Contains(svg, sp.StageName(st)) {
					t.Errorf("%s is not the compact badge of %s", name, sp.StageName(st))
				}
			}
			img := regexp.MustCompile(`<img src="svg/compact-` + regexp.QuoteMeta(prefix+st.Slug()) + `\.svg" width="200" height="60" loading="lazy" alt="([^"]+)"`).FindStringSubmatch(page)
			if img == nil {
				t.Errorf("the page has no 200x60 image of the compact %s", sp.StageName(st))
			} else if !strings.Contains(img[1], sp.StageName(st)) || !strings.Contains(strings.ToLower(img[1]), "compacta") {
				t.Errorf("alt of the compact %s: %q", sp.StageName(st), img[1])
			}
		}
	}
}

func mustBeValidXML(t *testing.T, name string, data string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(data))
	for {
		if _, err := dec.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("%s is not valid XML: %v", name, err)
			return
		}
	}
}
