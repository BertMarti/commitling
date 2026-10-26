package gallery

import (
	"os"
	"regexp"
	"strconv"
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
	want := []string{"user", "out", "theme", "token", "species", "size", "fixture", "keep-on-error"}
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
	if got := def("size"); got != "full" {
		t.Errorf("size default = %q, want full (the card of always)", got)
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
	if !strings.Contains(yml, "--size \"$CL_SIZE\"") || !strings.Contains(yml, "CL_SIZE: ${{ inputs.size }}") {
		t.Error("the size input is not passed to the CLI")
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
	for _, in := range []string{"user", "out", "theme", "token", "species", "size", "fixture", "keep-on-error"} {
		if !strings.Contains(uso, "| `"+in+"` |") {
			t.Errorf("docs/USO.md does not document the %q input", in)
		}
	}
}

// runBlocks returns the body of every `run: |` block of a workflow or action
// file (the indented lines that follow it).
func runBlocks(yml string) []string {
	var blocks []string
	lines := strings.Split(yml, "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		trim := strings.TrimLeft(l, " ")
		if !strings.HasPrefix(trim, "run:") && !strings.HasPrefix(trim, "- run:") {
			continue
		}
		indent := len(l) - len(trim)
		body := []string{strings.TrimSpace(strings.SplitN(trim, "run:", 2)[1])}
		for i+1 < len(lines) {
			next := lines[i+1]
			if strings.TrimSpace(next) != "" && len(next)-len(strings.TrimLeft(next, " ")) <= indent {
				break
			}
			body = append(body, next)
			i++
		}
		blocks = append(blocks, strings.Join(body, "\n"))
	}
	return blocks
}

// No `${{ }}` expression may be expanded inside a shell script: GitHub pastes
// it into the script before running it, so an input (or anything a user can
// influence) would become code. Inputs reach the scripts through `env:`.
func TestNoExpressionsInRunScripts(t *testing.T) {
	files := []string{"action.yml"}
	entries, err := os.ReadDir("../../.github/workflows")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
			files = append(files, ".github/workflows/"+e.Name())
		}
	}
	for _, f := range files {
		blocks := runBlocks(readRepoFile(t, f))
		if len(blocks) == 0 {
			t.Errorf("%s: no run blocks found (is the parser still right?)", f)
		}
		for _, b := range blocks {
			if strings.Contains(b, "${{") {
				t.Errorf("%s: a run script expands an expression; pass it through env: instead:\n%s", f, b)
			}
		}
	}
}

// Every workflow declares its permissions and no checkout keeps the token in
// .git/config (none of these jobs pushes).
func TestWorkflowsAreLeastPrivilege(t *testing.T) {
	entries, err := os.ReadDir("../../.github/workflows")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		f := ".github/workflows/" + e.Name()
		yml := readRepoFile(t, f)
		if !regexp.MustCompile(`(?m)^permissions:`).MatchString(yml) {
			t.Errorf("%s has no top-level permissions", f)
		}
		checkouts := strings.Count(yml, "uses: actions/checkout@")
		if got := strings.Count(yml, "persist-credentials: false"); got != checkouts {
			t.Errorf("%s: %d checkouts but %d with persist-credentials: false", f, checkouts, got)
		}
	}
}

// setup-go installs exactly the go line of go.mod for CI, Pages and the
// Action. It must name a patched release, and there must be no toolchain
// line: setup-go exports GOTOOLCHAIN=local, so the second use of the Action
// in one job (light and dark) would ignore it and install the go line.
func TestGoModPinsAPatchedGo(t *testing.T) {
	mod := readRepoFile(t, "go.mod")
	if regexp.MustCompile(`(?m)^toolchain\s`).MatchString(mod) {
		t.Error("go.mod has a toolchain line; raise the go line instead")
	}
	m := regexp.MustCompile(`(?m)^go (\d+)\.(\d+)\.(\d+)$`).FindStringSubmatch(mod)
	if m == nil {
		t.Fatal("the go line of go.mod must be a full version like 1.27.3")
	}
	got := [3]int{}
	for i := range got {
		got[i], _ = strconv.Atoi(m[i+1])
	}
	// The first release of 1.27 without the 11 vulnerabilities govulncheck
	// reported for 1.27.0 (net/http, crypto/tls, crypto/x509, html/template).
	minimum := [3]int{1, 27, 3}
	for i := range got {
		if got[i] != minimum[i] {
			if got[i] < minimum[i] {
				t.Errorf("go.mod says go %s.%s.%s, want at least 1.27.3", m[1], m[2], m[3])
			}
			break
		}
	}
}

// The token reaches the script only through the environment and is masked
// before anything runs, so even a token pasted as plain text never shows in
// the log; the script never prints the environment or traces commands.
func TestActionMasksTheToken(t *testing.T) {
	yml := readRepoFile(t, "action.yml")
	mask := strings.Index(yml, `printf '::add-mask::%s\n' "${GITHUB_TOKEN//[$'\r\n']/}"`)
	run := strings.Index(yml, "go run ./cmd/commitling")
	if mask < 0 || run < 0 || mask > run {
		t.Fatal("action.yml must mask $GITHUB_TOKEN, without line breaks, before go run")
	}
	if !strings.Contains(yml, "GITHUB_TOKEN: ${{ inputs.token }}") {
		t.Error("the token must arrive by env:")
	}
	for _, bad := range []string{"set -x", "set -o xtrace", "printenv", "env |", "echo \"$GITHUB_TOKEN", "echo $GITHUB_TOKEN"} {
		if strings.Contains(yml, bad) {
			t.Errorf("action.yml contains %q", bad)
		}
	}
}

// Third-party actions in the workflows and in action.yml are pinned to a full
// commit SHA with the version as a comment (dependabot keeps both current): a
// moved tag upstream cannot change what runs here or in the profiles.
func TestThirdPartyActionsArePinnedBySHA(t *testing.T) {
	files := []string{"action.yml"}
	entries, err := os.ReadDir("../../.github/workflows")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		files = append(files, ".github/workflows/"+e.Name())
	}
	uses := regexp.MustCompile(`(?m)^\s*(?:-\s+)?uses:\s*(\S+)(.*)$`)
	pinned := regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._/-]+@[0-9a-f]{40}$`)
	version := regexp.MustCompile(`^\s+# v\d+\.\d+\.\d+$`)
	total := 0
	for _, f := range files {
		for _, m := range uses.FindAllStringSubmatch(readRepoFile(t, f), -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "./") {
				continue // this repository's own Action
			}
			total++
			if !pinned.MatchString(ref) || !version.MatchString(m[2]) {
				t.Errorf("%s: %q must be owner/repo@<40-hex SHA> # vX.Y.Z", f, strings.TrimSpace(m[0]))
			}
		}
	}
	if total < 5 {
		t.Errorf("only %d third-party uses found: is the parser still right?", total)
	}
}

// The precompiled binary is opt-in per tree: the Action only looks for one if
// its own release/checksums.txt lists it, checks the SHA-256 before using it,
// never fails because of it, and keeps setup-go and go run as the fallback.
// The token is masked before either way of running the CLI.
func TestActionPrebuiltBinaryIsSafeAndOptional(t *testing.T) {
	yml := readRepoFile(t, "action.yml")
	start := strings.Index(yml, "- id: bin\n")
	end := strings.Index(yml, "uses: actions/setup-go@")
	if start < 0 || end < start {
		t.Fatal("action.yml must have the bin step before setup-go")
	}
	bin := yml[start:end]
	for _, want := range []string{
		`sums="$GITHUB_ACTION_PATH/release/checksums.txt"`,
		`[ -f "$sums" ] || exit 0`,
		`if [ "$got" != "$want" ]; then`,
		`https://github.com/BertMarti/commitling/releases/download`,
	} {
		if !strings.Contains(bin, want) {
			t.Errorf("bin step lacks %q", want)
		}
	}
	if strings.Contains(bin, "exit 1") || strings.Contains(bin, "set -e") {
		t.Error("the bin step must never fail the job: on any problem it falls back to go run")
	}
	// The default shell of a composite step is bash -e -o pipefail, so every
	// command that can fail on its own must be guarded.
	for _, want := range []string{
		`mkdir -p "$dir" || exit 0`,
		`got="$(sha256sum "$dir/$name" 2>/dev/null || true)"`,
		`got="$(shasum -a 256 "$dir/$name" 2>/dev/null || true)"`,
		`chmod +x "$dir/$name" || exit 0`,
		`| head -n 1 || true)"`,
		`if ! curl `,
	} {
		if !strings.Contains(bin, want) {
			t.Errorf("bin step: %q is missing (a failure there would fail the job)", want)
		}
	}
	if check, out := strings.Index(bin, `"$got" != "$want"`), strings.Index(bin, `echo "path=`); check < 0 || out < check {
		t.Error("the binary path must only be published after its SHA-256 matched")
	}
	if !strings.Contains(yml, "- if: steps.bin.outputs.path == ''\n      uses: actions/setup-go@") {
		t.Error("setup-go must run whenever there is no verified binary")
	}
	mask := strings.Index(yml, `printf '::add-mask::%s\n'`)
	for _, run := range []string{`"$CL_BIN" "${args[@]}"`, "go run ./cmd/commitling"} {
		if i := strings.Index(yml, run); i < 0 || mask < 0 || i < mask {
			t.Errorf("%s must exist and come after the token is masked", run)
		}
	}
}

// If release/checksums.txt is committed, it describes this version of the CLI
// on the six platforms (the CI also rebuilds them and compares the bytes).
func TestReleaseChecksumsMatchTheVersion(t *testing.T) {
	data, err := os.ReadFile("../../release/checksums.txt")
	if os.IsNotExist(err) {
		t.Log("no release/checksums.txt: the Action compiles with go run")
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^var version = "([^"]+)"$`).FindStringSubmatch(readRepoFile(t, "cmd/commitling/main.go"))
	if m == nil {
		t.Fatal("no version in cmd/commitling/main.go")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 6 {
		t.Errorf("release/checksums.txt has %d lines, want 6", len(lines))
	}
	line := regexp.MustCompile(`^[0-9a-f]{64}  commitling_` + regexp.QuoteMeta(m[1]) + `_(linux|darwin|windows)_(amd64|arm64)(\.exe)?$`)
	for _, l := range lines {
		if !line.MatchString(l) {
			t.Errorf("release/checksums.txt: %q is not a binary of v%s", l, m[1])
		}
	}
}

// The release binaries must not depend on the environment of whoever builds
// them: build-release.sh resets GOFLAGS and GOEXPERIMENT before building (the
// CI also builds with GOFLAGS=-tags=zzz and compares).
func TestBuildReleaseResetsTheEnvironment(t *testing.T) {
	sh := readRepoFile(t, "scripts/build-release.sh")
	set := strings.Index(sh, "set -euo pipefail\n")
	reset := strings.Index(sh, "export GOFLAGS=-p=2 GOEXPERIMENT= GOFIPS140=off\n")
	build := strings.Index(sh, "go build ")
	if set < 0 || reset < set || build < reset {
		t.Error("build-release.sh must export GOFLAGS=-p=2 GOEXPERIMENT= GOFIPS140=off right after set -euo pipefail and before go build")
	}
}
