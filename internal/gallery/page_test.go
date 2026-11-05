package gallery

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
)

func TestPageBehaviourInNode(t *testing.T) {
	node := nodeOrSkip(t)
	dir := buildDir(t)
	out, err := exec.Command(node, "testdata/page.test.js", dir+string(os.PathSeparator)+"page.js").CombinedOutput()
	if err != nil {
		t.Fatalf("page.test.js failed: %v\n%s", err, out)
	}
	if n := strings.Count("\n"+string(out), "\nok "); n != 7 {
		t.Fatalf("expected 7 scenarios, got %d:\n%s", n, out)
	}
}

func TestGalleryHasSpeciesFilter(t *testing.T) {
	page := buildPage(t)
	if !strings.Contains(page, `<fieldset class="species-pick" id="species-pick" hidden>`) {
		t.Error("species filter must be hidden without JavaScript")
	}
	if !strings.Contains(page, `id="species-pick-msg" role="status"`) {
		t.Error("species filter lacks its status message")
	}
	if !strings.Contains(page, `name="gal-species" value="all" id="gal-sp-all" checked`) {
		t.Error("species filter lacks the default option")
	}
	for _, sp := range creature.AllSpecies {
		if got := strings.Count(page, `data-species="`+sp.Slug()+`"`); got != 2 {
			t.Errorf("%s has %d species blocks, want 2", sp.Name(), got)
		}
		if !strings.Contains(page, `name="gal-species" value="`+sp.Slug()+`" id="gal-sp-`+sp.Slug()+`"`) {
			t.Errorf("missing gallery radio for %s", sp.Name())
		}
	}
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "eval(", "new Function", "localStorage", "sessionStorage", "document.cookie"} {
		if strings.Contains(string(pageJS), bad) {
			t.Errorf("page.js uses %s", bad)
		}
	}
}

func TestPageExplainsAttestations(t *testing.T) {
	page := buildPageWithVersion(t, "1.2.0-test")
	for _, want := range []string{`id="verificar"`, `id="verificar-aviso"`, `gh attestation verify commitling_1.2.0-test_linux_amd64 --repo BertMarti/commitling`, `commitling_1.2.0-test_windows_amd64.exe`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

func buildPageWithVersion(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := Build(dir, version); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dir + string(os.PathSeparator) + "index.html")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
