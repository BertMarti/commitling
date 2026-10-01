package gallery

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/BertMarti/commitling/internal/creature"
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
	names := regexp.MustCompile(`(?m)^  ([a-z_-]+):\n`).FindAllStringSubmatch(inputs[1], -1)
	want := []string{"user", "out", "theme", "token", "species", "fixture", "keep-on-error"}
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
	for _, f := range []string{"README.md", "docs/USO.md", "internal/generate/generate.go"} {
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

// Descriptions are plain YAML scalars: ": " or " #" inside would break the
// file (GitHub would refuse the Action), and a tab or trailing space is
// almost always a mistake.
func TestActionYAMLPlainScalarsAreSafe(t *testing.T) {
	yml := readRepoFile(t, "action.yml")
	for i, line := range strings.Split(yml, "\n") {
		if strings.Contains(line, "\t") {
			t.Errorf("action.yml:%d has a tab", i+1)
		}
		m := regexp.MustCompile(`^\s*(?:description|name|author):[ \t]+([^"'\s].*)$`).FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.Contains(m[1], ": ") || strings.Contains(m[1], " #") {
			t.Errorf("action.yml:%d: plain scalar %q contains \": \" or \" #\"; quote it", i+1, m[1])
		}
	}
}

// The defaults and accepted values of the Action match the CLI.
func TestActionInputDefaultsMatchTheCLI(t *testing.T) {
	yml := readRepoFile(t, "action.yml")
	def := func(input string) string {
		m := regexp.MustCompile(`(?ms)^  ` + input + `:\n(.*?)(?:^  [\w-]+:\n|^outputs:)`).FindStringSubmatch(yml)
		if m == nil {
			t.Fatalf("input %q not found", input)
		}
		d := regexp.MustCompile(`(?m)^    default:[ \t]*(.*)$`).FindStringSubmatch(m[1])
		if d == nil {
			t.Fatalf("input %q has no default", input)
		}
		return d[1]
	}
	if got, want := def("species"), creature.AllSpecies[0].Slug(); got != want {
		t.Errorf("species default = %q, want the CLI default %q", got, want)
	}
	if got := def("theme"); got != "light" {
		t.Errorf("theme default = %q, want light", got)
	}
	if got := def("keep-on-error"); got != `"true"` {
		t.Errorf("keep-on-error default = %s, want \"true\" (on by default)", got)
	}
	if !strings.Contains(yml, "--keep-on-error") || !strings.Contains(yml, "CL_KEEP_ON_ERROR: ${{ inputs.keep-on-error }}") {
		t.Error("the keep-on-error input is not passed to the CLI")
	}
	if !strings.Contains(yml, "--species \"$CL_SPECIES\"") {
		t.Error("the species input is not passed to the CLI")
	}
	// The description names every species so nobody has to guess the slug.
	spec := regexp.MustCompile(`(?ms)^  species:\n(.*?)^  [\w-]+:\n`).FindStringSubmatch(yml)
	for _, sp := range creature.AllSpecies {
		if !strings.Contains(spec[1], sp.Slug()) {
			t.Errorf("the species description does not name %q", sp.Slug())
		}
	}
	// The user-facing docs list every input too.
	uso := readRepoFile(t, "docs/USO.md")
	for _, in := range []string{"user", "out", "theme", "token", "species", "fixture", "keep-on-error"} {
		if !strings.Contains(uso, "| `"+in+"` |") {
			t.Errorf("docs/USO.md does not document the %q input", in)
		}
	}
}
