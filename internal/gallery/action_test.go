package gallery

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile("../../" + rel)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// topLevel returns the value of a top-level `key: value` line of action.yml.
func topLevel(yml, key string) string {
	m := regexp.MustCompile(`(?m)^` + key + `:[ \t]*(.*)$`).FindStringSubmatch(yml)
	if m == nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(m[1]), `"'`)
}

// GitHub Marketplace only accepts these values for branding.
var (
	marketplaceColors = []string{"white", "yellow", "blue", "green", "orange", "red", "purple", "gray-dark"}
	// Feather icons accepted by the Marketplace (the ones plausible here;
	// the full list is in the GitHub docs).
	marketplaceIcons = []string{"feather", "sun", "moon", "star", "heart", "smile", "coffee", "zap", "award", "box", "package", "sunrise", "sunset", "leaf"}
)

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestActionMetadataIsMarketplaceReady(t *testing.T) {
	yml := readRepoFile(t, "action.yml")

	if got := topLevel(yml, "name"); got != "commitling" {
		t.Errorf("name = %q, want commitling", got)
	}
	desc := topLevel(yml, "description")
	if desc == "" || len(desc) > 125 {
		t.Errorf("description has %d characters; the Marketplace wants 1 to 125: %q", len(desc), desc)
	}
	if topLevel(yml, "author") == "" {
		t.Error("author is missing")
	}

	branding := regexp.MustCompile(`(?m)^branding:\n((?:[ \t]+.*\n)+)`).FindStringSubmatch(yml)
	if branding == nil {
		t.Fatal("branding block is missing")
	}
	icon := regexp.MustCompile(`icon:[ \t]*(\S+)`).FindStringSubmatch(branding[1])
	color := regexp.MustCompile(`color:[ \t]*(\S+)`).FindStringSubmatch(branding[1])
	if icon == nil || !contains(marketplaceIcons, icon[1]) {
		t.Errorf("branding icon = %v, want one of %v", icon, marketplaceIcons)
	}
	if color == nil || !contains(marketplaceColors, color[1]) {
		t.Errorf("branding color = %v, want one of %v", color, marketplaceColors)
	}

	if !strings.Contains(yml, "using: composite") {
		t.Error("the action should stay composite")
	}
}

func TestActionInputsAreDocumented(t *testing.T) {
	yml := readRepoFile(t, "action.yml")
	inputs := regexp.MustCompile(`(?ms)^inputs:\n(.*?)^outputs:`).FindStringSubmatch(yml)
	if inputs == nil {
		t.Fatal("no inputs block")
	}
	names := regexp.MustCompile(`(?m)^  ([a-z_]+):\n`).FindAllStringSubmatch(inputs[1], -1)
	want := []string{"user", "out", "theme", "token", "species", "fixture"}
	if len(names) != len(want) {
		t.Fatalf("inputs = %v, want %v", names, want)
	}
	readme := readRepoFile(t, "README.md")
	for i, n := range names {
		if n[1] != want[i] {
			t.Errorf("input %d is %q, want %q", i, n[1], want[i])
		}
		if !strings.Contains(readme, "| `"+n[1]+"` |") {
			t.Errorf("input %q is missing from the README table", n[1])
		}
	}
	if got := strings.Count(inputs[1], "    description:"); got != len(want) {
		t.Errorf("%d inputs have a description, want %d", got, len(want))
	}
}

// Every example that uses the Action points at the v1 tag, not at a branch.
func TestExamplesUseV1(t *testing.T) {
	for _, f := range []string{"README.md", "docs/USO.md", "internal/gallery/gallery.go"} {
		body := readRepoFile(t, f)
		if !strings.Contains(body, "BertMarti/commitling@v1") {
			t.Errorf("%s has no example with @v1", f)
		}
		if strings.Contains(body, "commitling@main") {
			t.Errorf("%s still uses @main in an example", f)
		}
	}
	if !strings.Contains(WorkflowSnippet, "uses: BertMarti/commitling@v1") {
		t.Error("the workflow of the gallery does not use @v1")
	}
}
