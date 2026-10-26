package gallery

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestThemeBehaviourInNode(t *testing.T) {
	node := nodeOrSkip(t)
	dir := buildDir(t)
	out, err := exec.Command(node, "testdata/theme.test.js", dir+string(os.PathSeparator)+"theme.js").CombinedOutput()
	if err != nil {
		t.Fatalf("theme.test.js failed: %v\n%s", err, out)
	}
	if n := strings.Count("\n"+string(out), "\nok "); n != 6 {
		t.Fatalf("expected 6 scenarios, got %d:\n%s", n, out)
	}
}

func TestThemeSectionAndScript(t *testing.T) {
	dir := buildDir(t)
	page := readBuilt(t, dir, "index.html")
	if strings.Index(page, `<script src="theme.js"></script>`) < 0 ||
		strings.Index(page, `<script src="theme.js"></script>`) > strings.Index(page, `<link rel="stylesheet" href="style.css">`) {
		t.Error("theme.js must be in head before style.css")
	}
	for _, value := range []string{"system", "light", "dark"} {
		if !strings.Contains(page, `name="site-theme" value="`+value+`"`) {
			t.Errorf("theme selector lacks %q", value)
		}
	}
	if !strings.Contains(page, `<fieldset class="theme-pick" id="theme-pick" hidden>`) {
		t.Error("theme selector must be hidden without JavaScript")
	}
	js := readBuilt(t, dir, "theme.js")
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "eval(", "new Function", "script.src"} {
		if strings.Contains(js, bad) {
			t.Errorf("theme.js uses %s", bad)
		}
	}
	if !strings.Contains(js, "commitling:theme") {
		t.Error("theme.js lacks the storage key")
	}
	if strings.Contains(page, `<script src="theme.js" defer`) || strings.Contains(page, `<script src="theme.js" async`) {
		t.Error("theme.js must run synchronously in head")
	}
}
