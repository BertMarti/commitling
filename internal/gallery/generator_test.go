package gallery

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
)

func buildDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := Build(dir, "test"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readBuilt(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The generator section is a real form: labelled fields, a radio per species
// and theme, and a hidden-until-JavaScript container with a fallback.
func TestGeneratorFormIsAccessible(t *testing.T) {
	page := buildPage(t)
	for _, want := range []string{
		`id="generador"`,
		`<form id="gen-form"`,
		`<label class="f" for="gen-user">`,
		`id="gen-user"`,
		`<legend>Especie</legend>`,
		`<legend>Tema</legend>`,
		`<button type="submit"`,
		`<noscript>`,
		`<script src="generator.js" defer></script>`,
		`href="#generador"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
	// Hidden until the script shows it: without JavaScript only the noscript
	// message is visible.
	if !regexp.MustCompile(`<div class="gen" id="gen" hidden>`).MatchString(page) {
		t.Error("the generator must start hidden")
	}
	for _, sp := range creature.AllSpecies {
		re := `<input type="radio" name="species" value="` + sp.Slug() + `"`
		if !strings.Contains(page, re) {
			t.Errorf("no radio for species %s", sp.Slug())
		}
		if !strings.Contains(page, sp.Name()+"</label>") {
			t.Errorf("no visible label %q", sp.Name())
		}
	}
	for _, th := range []string{"light", "dark"} {
		if !strings.Contains(page, `<input type="radio" name="theme" value="`+th+`"`) {
			t.Errorf("no radio for theme %s", th)
		}
	}
	if got := strings.Count(page, `checked>`); got != 2 {
		t.Errorf("%d options checked by default, want 2 (one species, one theme)", got)
	}
	// Loading and success go to a status region; failures to an alert.
	if !regexp.MustCompile(`id="gen-status"[^>]*role="status"|role="status"[^>]*id="gen-status"`).MatchString(page) {
		t.Error("no role=status region for loading and success")
	}
	if !regexp.MustCompile(`id="gen-error"[^>]*role="alert"|role="alert"[^>]*id="gen-error"`).MatchString(page) {
		t.Error("no role=alert region for errors")
	}
	if !strings.Contains(page, `aria-describedby="gen-hint gen-error"`) {
		t.Error("the user field is not tied to its hint and error")
	}
}

// The wasm is only downloaded when the person interacts with the generator:
// the page itself neither loads nor preloads it.
func TestGeneratorLoadsWasmLazily(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	for _, bad := range []string{"commitling.wasm", "wasm_exec.js", `rel="preload"`, `rel="modulepreload"`, `rel="prefetch"`} {
		if strings.Contains(page, bad) {
			t.Errorf("index.html must not reference %q: the wasm loads on interaction", bad)
		}
	}
	js := readBuilt(t, dir, "generator.js")
	for _, want := range []string{"commitling.wasm", "wasm_exec.js", "focusin", "submit", "WebAssembly", "instantiateStreaming"} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js does not mention %q", want)
		}
	}
}

// The script talks to api.github.com and nobody else, sends no credentials
// and never builds HTML from what comes from outside.
func TestGeneratorScriptIsSafe(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "Authorization", "localStorage", "sessionStorage", "document.cookie"} {
		if strings.Contains(js, bad) {
			t.Errorf("generator.js uses %q", bad)
		}
	}
	for _, u := range regexp.MustCompile(`https?://[^\s'"`+"`"+`)]+`).FindAllString(js, -1) {
		if !strings.HasPrefix(u, "https://api.github.com/") {
			t.Errorf("generator.js talks to %s", u)
		}
	}
	if !strings.Contains(js, "https://api.github.com/users/") {
		t.Error("generator.js does not read the public events of the user")
	}
}

// The page keeps loading nothing from third parties.
func TestPageLoadsNoThirdPartyResources(t *testing.T) {
	page := buildPage(t)
	for _, m := range regexp.MustCompile(`<(?:script|link|img|source|iframe)[^>]*(?:src|srcset|href)="(https?://[^"]+)"`).FindAllStringSubmatch(page, -1) {
		if !strings.Contains(m[0], "<link rel=\"canonical\"") {
			t.Errorf("external resource: %s", m[1])
		}
	}
}

// The button text is drawn in fixed ink on moss in both themes.
func TestGeneratorButtonContrast(t *testing.T) {
	if got := contrast("#2b2724", "#7fb069"); got < 4.5 {
		t.Errorf("button text contrast %.2f", got)
	}
}

// Next to the creature: a download link for the very SVG on screen and the
// workflow filled in for the person, with the same copy button as the rest of
// the page.
func TestGeneratorHasDownloadAndFilledWorkflow(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	for _, want := range []string{
		`<div class="gen-after" id="gen-after" hidden>`,
		`<a class="go" id="gen-dl" download="commitling.svg"`,
		`>Descargar SVG</a>`,
		`id="gen-wf"`,
		`data-copy="gen-wf"`,
		`aria-label="Copiar el workflow con tu usuario"`,
		`.github/workflows/commitling.yml`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
	js := readBuilt(t, dir, "generator.js")
	for _, want := range []string{"commitling.workflow(", "dl.download", "dl.href", "'commitling-' + "} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js does not contain %q", want)
		}
	}
}
