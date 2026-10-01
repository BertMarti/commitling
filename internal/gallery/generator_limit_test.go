package gallery

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/events"
)

// What the generator does when GitHub's limit for anonymous requests (60 an
// hour) is spent, and what it does to avoid spending it. The behaviour is
// proved in Node (testdata/generator.test.js, a fake page around the real
// generator.js); the tests here pin the intent in the page and in the script.

// The Node test needs `node`: the CI runners have it. Elsewhere it is skipped
// (as wasm there is no way to start a process).
func TestGeneratorBehaviourInNode(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("cannot start node from a wasm test binary: run it by hand (see the file)")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("node is not installed, and the CI must run the generator test")
		}
		t.Skip("node is not installed")
	}
	dir := buildDir(t)
	out, err := exec.Command(node, "testdata/generator.test.js", dir+string(os.PathSeparator)+"generator.js").CombinedOutput()
	if err != nil {
		t.Fatalf("generator.test.js failed: %v\n%s", err, out)
	}
	if n := strings.Count(string(out), "\nok ") + strings.Count(string(out), "ok   "); n < 8 {
		t.Errorf("expected the Node test to run at least 8 scenarios, got %d:\n%s", n, out)
	}
}

// The name is judged in JavaScript with the rules of events.ValidLogin before
// the wasm is loaded or anything is requested; the two must never disagree.
func TestGeneratorValidatesTheLoginLikeGo(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	m := regexp.MustCompile(`return (/\^[^/]+\$/)\.test\(s\);`).FindStringSubmatch(js)
	if m == nil {
		t.Fatal("generator.js has no validLogin regular expression")
	}
	re := regexp.MustCompile(strings.Trim(m[1], "/"))
	names := []string{
		"", "a", "Z", "0", "007", "octoexample", "BertMarti", "a-b-c", "a-", "a--b", strings.Repeat("x", 39),
		"-a", "-", "a b", " a", "a ", "a\n", "\na", "a\r", "a\tb", "a/b", "../x", "a?b", "a_b", "a.b", "á", "aé", "日本", "a\x00",
		strings.Repeat("x", 40), strings.Repeat("x", 100),
	}
	for _, n := range names {
		if got, want := re.MatchString(n), events.ValidLogin(n); got != want {
			t.Errorf("validLogin(%q) = %v in JavaScript, events.ValidLogin = %v in Go", n, got, want)
		}
	}
	// Order inside draw(): empty, then invalid, and only then the wasm.
	draw := js[strings.Index(js, "async function draw()"):]
	iEmpty, iValid, iWasm := strings.Index(draw, "if (!user)"), strings.Index(draw, "!validLogin(user)"), strings.Index(draw, "await ensureWasm()")
	if iEmpty < 0 || iValid < iEmpty || iWasm < iValid {
		t.Errorf("draw() must check the name (empty %d, valid %d) before loading the wasm (%d)", iEmpty, iValid, iWasm)
	}
}

// The events are kept in the tab (not on disk, not shared) for ten minutes,
// per user, so that drawing again or changing size, species or theme never
// asks GitHub again.
func TestGeneratorCachesTheEventsInTheTab(t *testing.T) {
	js := readBuilt(t, buildDir(t), "generator.js")
	for _, want := range []string{
		"CACHE_MS = 10 * 60 * 1000",
		"'commitling:events:'",
		"toLowerCase()", // Octo and octo are one user
		"cachePut(user, events, current.at)",
		"cacheGet(user)",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js lacks %q", want)
		}
	}
	// Every use of the storage is inside a try: it may be missing, blocked or full.
	for _, fn := range []string{"function cacheGet", "function cachePut"} {
		i := strings.Index(js, fn)
		if i < 0 {
			t.Fatalf("generator.js lacks %s", fn)
		}
		body := js[i:]
		if end := strings.Index(body[1:], "\n  function "); end >= 0 {
			body = body[:end+1]
		}
		if !strings.Contains(body, "try {") || !strings.Contains(body, "catch (e)") {
			t.Errorf("%s uses sessionStorage without try/catch", fn)
		}
	}
	// A failed search must not be written: the only cachePut is after fetchEvents succeeds.
	if strings.Count(js, "cachePut(") != 2 { // the definition and the call
		t.Error("cachePut must be called once, after a successful search")
	}
	// Redrawing with another species, theme or size goes through redraw(), which
	// uses what is already in memory.
	redraw := js[strings.Index(js, "function redraw()"):]
	redraw = redraw[:strings.Index(redraw, "async function draw()")]
	if strings.Contains(redraw, "fetchEvents") || strings.Contains(redraw, "fetch(") {
		t.Error("redraw() must not ask GitHub for anything")
	}
}

// With the limit spent the person is told when it resets and can still see
// the product: the demo never touches the network.
func TestGeneratorOffersTheDemoWhenTheLimitIsSpent(t *testing.T) {
	dir := buildDir(t)
	page, js := readBuilt(t, dir, "index.html"), readBuilt(t, dir, "generator.js")
	if !regexp.MustCompile(`<button type="button" class="go" id="gen-demo-alt" hidden>Ver demo</button>`).MatchString(page) {
		t.Error("the page lacks a hidden «Ver demo» button (#gen-demo-alt) next to the error")
	}
	// It sits with the messages, after the alert, and is not inside a live region.
	i, j, k := strings.Index(page, `id="gen-error"`), strings.Index(page, `id="gen-demo-alt"`), strings.Index(page, `</form>`)
	if !(i > 0 && i < j && j < k) {
		t.Errorf("#gen-demo-alt must come after #gen-error and inside the form (%d %d %d)", i, j, k)
	}
	if !strings.Contains(page[i:j], "</p>") {
		t.Error("#gen-demo-alt must not be inside the alert paragraph")
	}
	for _, want := range []string{
		"demoAlt.addEventListener('click', startDemo)",              // the same demo as the other button
		"f.status === 403 || f.status === 429",                      // only for the limit
		"window.commitling.explain(f.status, f.remaining, f.reset)", // the wasm writes the message with the raw headers
		"'X-RateLimit-Reset'",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("generator.js lacks %q", want)
		}
	}
	// startDemo must not read the network (see also TestDemoNeverTouchesTheNetwork).
	start := js[strings.Index(js, "async function startDemo()"):]
	start = start[:strings.Index(start, "demoGo.addEventListener")]
	if strings.Contains(start, "fetchEvents") {
		t.Error("the demo must not ask GitHub for anything")
	}
}
