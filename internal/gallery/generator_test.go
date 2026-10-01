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
	for _, sz := range []string{"full", "compact"} {
		if !strings.Contains(page, `<input type="radio" name="size" value="`+sz+`"`) {
			t.Errorf("no radio for size %s", sz)
		}
	}
	if !strings.Contains(page, `<legend>Tamaño</legend>`) || !strings.Contains(page, "Compacta</label>") || !strings.Contains(page, "Completa</label>") {
		t.Error("the size group needs its legend and visible labels")
	}
	if got := strings.Count(page, `checked>`); got != 3 {
		t.Errorf("%d options checked by default, want 3 (one species, one theme, one size)", got)
	}
	if !regexp.MustCompile(`<input type="radio" name="size" value="full"[^>]* checked>`).MatchString(page) {
		t.Error("the full card must be the default size")
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
	for _, want := range []string{"commitling.wasm", "wasm_exec.js", "'input'", "'change'", "'submit'", "WebAssembly", "instantiateStreaming"} {
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

// The status and alert regions are always in the accessibility tree: an empty
// one must not be display:none (a screen reader may miss the first message of
// a region that appears with its content).
func TestLiveRegionsAreNeverHidden(t *testing.T) {
	page := buildPage(t)
	style := regexp.MustCompile(`(?s)<style>(.*?)</style>`).FindStringSubmatch(page)[1]
	for _, m := range regexp.MustCompile(`([^{}]+)\{([^}]*)\}`).FindAllStringSubmatch(style, -1) {
		sel, body := m[1], m[2]
		if !regexp.MustCompile(`display:\s*none|visibility:\s*hidden`).MatchString(body) {
			continue
		}
		for _, bad := range []string{"gen-msgs", "gen-status", "gen-error", "[role"} {
			if strings.Contains(sel, bad) {
				t.Errorf("rule %q hides a live region", strings.TrimSpace(sel))
			}
		}
		if strings.Contains(sel, ":empty") {
			t.Errorf("rule %q hides empty elements, live regions included", strings.TrimSpace(sel))
		}
	}
	if !strings.Contains(page, `id="gen-status" class="note" role="status"`) || !strings.Contains(page, `id="gen-error" class="gen-error" role="alert"`) {
		t.Error("the live regions are missing from the page")
	}
}

// What the script promises to do (it runs in a browser, so these are
// static checks; the behaviour was verified in the browser pane).
func TestGeneratorScriptRobustness(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	for _, want := range []string{
		"TIMEOUT_MS = 15000",
		"AbortSignal.timeout(TIMEOUT_MS)", // a request that never answers must not hang the button
		"Array.isArray(",                  // a 200 that is not a list of events
		"60000",                           // events are reused for a minute for the same user
	} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js lacks %q", want)
		}
	}
	if strings.Contains(js, "focusin") {
		t.Error("the wasm must not start downloading on focus, only on input, submit or change")
	}
	// The alt is short; the description is already the visible caption.
	if !strings.Contains(js, "img.alt = 'commitling de @' + user;") {
		t.Error("the preview alt must be 'commitling de @user'")
	}
}

// The demo is a user action, never a surprise: nothing starts by itself, Pause
// and Stop are always on the page (disabled until there is a demo, not hidden)
// and there is a manual slider for people who ask for less motion.
func TestDemoControls(t *testing.T) {
	page := buildPage(t)
	for _, want := range []string{
		`<button type="button" class="go" id="demo-go">Ver demo</button>`,
		`<button type="button" class="go alt" id="demo-pause" disabled>Pausa</button>`,
		`<button type="button" class="go alt" id="demo-stop" disabled>Detener</button>`,
		`<input type="range" id="demo-range" min="0" max="90"`,
		`<label for="demo-range">`,
		`id="hero-demo" hidden>`, // the header link needs the script: hidden until it runs
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
	js := readBuilt(t, buildDir(t), "generator.js")
	if !strings.Contains(js, "demoGo.addEventListener('click', startDemo)") || !strings.Contains(js, "heroDemo.addEventListener('click', startDemo)") {
		t.Error("the demo must start from a click")
	}
	if strings.Contains(js, "startDemo();") || strings.Contains(js, "autoplay") {
		t.Error("the demo must never start by itself")
	}
	for _, want := range []string{"prefers-reduced-motion: reduce", "requestAnimationFrame", "cancelAnimationFrame", "commitling.demo("} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js lacks %q", want)
		}
	}
}

// The demo draws locally: between its first and last line there is no network.
func TestDemoNeverTouchesTheNetwork(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	a, b := strings.Index(js, "function reducedMotion"), strings.Index(js, "form.addEventListener('submit'")
	if a < 0 || b < a {
		t.Fatal("cannot find the demo code")
	}
	for _, bad := range []string{"fetch(", "XMLHttpRequest", "API", "api.github.com", "fetchEvents"} {
		if strings.Contains(js[a:b], bad) {
			t.Errorf("the demo code uses %q", bad)
		}
	}
}

// Only the phase is announced. The day counter and the caption change every
// frame, so none of them may be a live region.
func TestDemoAnnouncesOnlyThePhase(t *testing.T) {
	page := buildPage(t)
	if !regexp.MustCompile(`id="demo-phase" role="status" aria-live="polite"`).MatchString(page) {
		t.Error("the demo phase must be a polite live region")
	}
	day := regexp.MustCompile(`<span class="demo-day" id="demo-day"[^>]*>`).FindString(page)
	if !strings.Contains(day, `aria-hidden="true"`) || strings.Contains(day, "aria-live") || strings.Contains(day, "role=") {
		t.Errorf("the day counter must not be live: %s", day)
	}
	for _, id := range []string{"gen-caption", "gen-stage", "demo-range"} {
		el := regexp.MustCompile(`<[^>]*` + id + `[^>]*>`).FindString(page)
		if strings.Contains(el, "aria-live") || strings.Contains(el, "role=\"status\"") || strings.Contains(el, "role=\"alert\"") {
			t.Errorf("%s must not be a live region: %s", id, el)
		}
	}
	// <output> is a live region by default.
	if strings.Contains(page, "<output") {
		t.Error("<output> announces every change: use a plain element")
	}
	js := readBuilt(t, buildDir(t), "generator.js")
	if !strings.Contains(js, "if (r.phase !== demo.phase)") {
		t.Error("the phase must only be written when it changes")
	}
}

// The size reaches every call into the wasm (render, workflow, check and the
// demo) as its last argument, and the preview takes the shape of the card.
func TestGeneratorPassesTheSizeToTheWasm(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	for _, call := range []string{"commitling.render(", "commitling.workflow(", "commitling.check(", "commitling.demo("} {
		i := strings.Index(js, call)
		if i < 0 {
			t.Errorf("generator.js never calls %s", call)
			continue
		}
		line := js[i : i+strings.Index(js[i:], "\n")]
		if !strings.Contains(line, "checked('size')") {
			t.Errorf("%s does not pass the size: %s", call, line)
		}
	}
	for _, want := range []string{"img.width", "img.height", "dataset.size"} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js does not adapt the preview (%q)", want)
		}
	}
	page := buildPage(t)
	if !strings.Contains(page, `id="gen-stage"`) {
		t.Error("the stage needs an id so the script can tell its size")
	}
	style := regexp.MustCompile(`(?s)<style>(.*?)</style>`).FindStringSubmatch(page)[1]
	rules := cssRules(style)
	if got := rules.prop(`.gen-stage[data-size=compact] img`, "max-width"); got != "200px" {
		t.Errorf("the compact preview must stop at 200px, got %q", got)
	}
	if got := rules.prop(`.gen-stage[data-size=compact] img`, "aspect-ratio"); got != "10/3" {
		t.Errorf("the compact preview keeps the 10:3 shape, got %q", got)
	}
}
