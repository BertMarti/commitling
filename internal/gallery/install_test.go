package gallery

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/generate"
)

// The workflow generator of «Instálalo en tu perfil» (install.js) builds in
// the browser, without wasm or network, the same text as generate.Install.
// Its behaviour is proved in Node (testdata/install.test.js); here it is
// compared, byte for byte, with Go for every option.

func nodeOrSkip(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "js" {
		t.Skip("cannot start node from a wasm test binary")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("node is not installed, and the CI must run the install test")
		}
		t.Skip("node is not installed")
	}
	return node
}

func TestInstallBehaviourInNode(t *testing.T) {
	node := nodeOrSkip(t)
	dir := buildDir(t)
	out, err := exec.Command(node, "testdata/install.test.js", dir+string(os.PathSeparator)+"install.js").CombinedOutput()
	if err != nil {
		t.Fatalf("install.test.js failed: %v\n%s", err, out)
	}
	if n := strings.Count("\n"+string(out), "\nok "); n < 10 {
		t.Errorf("expected at least 10 scenarios, got %d:\n%s", n, out)
	}
}

func TestInstallJSMatchesGo(t *testing.T) {
	node := nodeOrSkip(t)
	dir := buildDir(t)
	out, err := exec.Command(node, "testdata/install.test.js", dir+string(os.PathSeparator)+"install.js", "--dump").Output()
	if err != nil {
		t.Fatalf("install.test.js --dump failed: %v", err)
	}
	var cases []struct {
		User, Species, Theme, Size string
		Workflow, Readme           string
		Files                      []string
	}
	if err := json.Unmarshal(out, &cases); err != nil {
		t.Fatalf("--dump is not JSON: %v\n%.300s", err, out)
	}
	if len(cases) != 4*3*3*2 {
		t.Fatalf("%d cases, want 72", len(cases))
	}
	for _, c := range cases {
		want, err := generate.Install(generate.Options{User: c.User, Species: c.Species, Theme: c.Theme, Size: c.Size})
		if err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		name := c.User + "/" + c.Species + "/" + c.Theme + "/" + c.Size
		if c.Workflow != want.Workflow {
			t.Errorf("%s: the workflow of install.js differs from Go:\n%s", name, c.Workflow)
		}
		if c.Readme != want.Readme || strings.Join(c.Files, " ") != strings.Join(want.Files, " ") {
			t.Errorf("%s: readme %q files %v, want %q %v", name, c.Readme, c.Files, want.Readme, want.Files)
		}
	}
}

// The section is a real form (hidden until the script runs: without it the
// page shows the base workflow), with the three themes, and the script is
// loaded from a file, touches no network and keeps nothing in storage.
func TestInstallSectionAndScript(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	for _, want := range []string{
		`<form id="ins-form" class="gen-form ins-form" hidden novalidate>`,
		`<label class="f" for="ins-user">`,
		`aria-describedby="ins-hint ins-error"`,
		`<p id="ins-error" class="gen-error" role="alert">`,
		`<input type="radio" name="ins-theme" value="both"`,
		`<code id="wf">` + htmlEscaped(generate.BaseWorkflow),
		`<code id="rd">` + htmlEscaped(generate.ReadmeSnippet),
		`<script src="install.js" defer></script>`,
		`.github/workflows/commitling.yml`,
		`Run workflow`,
		`permissions: contents: write`,
		`id="ins-fallos"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html lacks %q", want)
		}
	}
	for _, a := range regexp.MustCompile(`<a id="ins-[a-z-]+"[^>]*>`).FindAllString(page, -1) {
		if !strings.Contains(a, `target="_blank" rel="noopener noreferrer"`) {
			t.Errorf("link %s must open apart with noopener", a)
		}
	}
	js := readBuilt(t, dir, "install.js")
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "Function(",
		"fetch(", "XMLHttpRequest", "localStorage", "sessionStorage", "document.cookie", "indexedDB"} {
		if strings.Contains(js, bad) {
			t.Errorf("install.js uses %s", bad)
		}
	}
	if !strings.Contains(js, "BertMarti/commitling@v1") || strings.Contains(js, "commitling@main") {
		t.Error("install.js must write @v1")
	}
}

// htmlEscaped escapes s as html/template does inside an element.
func htmlEscaped(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;", "'", "&#39;", "+", "&#43;").Replace(s)
}
