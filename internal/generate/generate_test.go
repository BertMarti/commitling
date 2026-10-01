package generate

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BertMarti/commitling/internal/creature"
	"github.com/BertMarti/commitling/internal/github"
	"github.com/BertMarti/commitling/internal/render"
	"github.com/BertMarti/commitling/internal/stats"
)

const fixture = "../../testdata/events.json"

func fixtureJSON(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fixtureNow(t *testing.T) time.Time {
	t.Helper()
	events, err := github.ParseEvents(bytes.NewReader(fixtureJSON(t)))
	if err != nil {
		t.Fatal(err)
	}
	return github.Latest(events)
}

// The reference drawing is built by hand with the same packages the CLI
// always used: Render must not change a single byte of it.
func TestRenderMatchesTheHandBuiltCard(t *testing.T) {
	now := fixtureNow(t)
	events, _ := github.ParseEvents(bytes.NewReader(fixtureJSON(t)))
	for _, sp := range creature.AllSpecies {
		for _, th := range []string{"light", "dark"} {
			theme, _ := render.ThemeByName(th)
			card := render.NewCard("octoexample", stats.Compute(github.Activities(events), now), theme)
			card.Creature.Species = sp
			want := render.SVG(card)

			got, err := Render(fixtureJSON(t), Options{User: "octoexample", Species: sp.Slug(), Theme: th, Now: now})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.SVG, want) {
				t.Errorf("%s/%s: Render differs from the hand-built card", sp.Slug(), th)
			}
			if got.Description != render.Description(card) {
				t.Errorf("%s/%s: description = %q, want %q", sp.Slug(), th, got.Description, render.Description(card))
			}
		}
	}
}

func TestRenderDefaultsAndLoginFromEvents(t *testing.T) {
	now := fixtureNow(t)
	got, err := Render(fixtureJSON(t), Options{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := Render(fixtureJSON(t), Options{User: "octoexample", Species: "moss", Theme: "light", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.SVG, explicit.SVG) {
		t.Error("empty options must mean octoexample (login of the events), moss and light")
	}
	if got.Login != "octoexample" {
		t.Errorf("Login = %q", got.Login)
	}
}

func TestRenderIsIndependentOfOrderAndDuplicates(t *testing.T) {
	now := fixtureNow(t)
	var events []json.RawMessage
	if err := json.Unmarshal(fixtureJSON(t), &events); err != nil {
		t.Fatal(err)
	}
	messy := make([]json.RawMessage, 0, 2*len(events))
	for i := len(events) - 1; i >= 0; i-- { // reversed, and every event twice
		messy = append(messy, events[i], events[i])
	}
	data, _ := json.Marshal(messy)
	a, err := Render(fixtureJSON(t), Options{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render(data, Options{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.SVG, b.SVG) {
		t.Error("the same events in another order, or repeated, must give the same SVG")
	}
}

func TestRenderEmptyEventsDrawsACreatureAnyway(t *testing.T) {
	got, err := Render([]byte(`[]`), Options{User: "newcomer", Now: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(bytes.TrimSpace(got.SVG), []byte("</svg>")) || !bytes.Contains(got.SVG, []byte("newcomer")) {
		t.Errorf("no valid SVG for newcomer:\n%s", got.SVG)
	}
}

func TestRenderWithoutNowUsesTheClock(t *testing.T) {
	if _, err := Render(fixtureJSON(t), Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestRenderErrors(t *testing.T) {
	now := fixtureNow(t)
	tests := []struct {
		name string
		data string
		o    Options
		want string
	}{
		{"broken json", `{nope`, Options{Now: now}, "eventos"},
		{"not an array", `{"a":1}`, Options{Now: now}, "eventos"},
		{"bad user", `[]`, Options{User: "no es válido", Now: now}, "usuario"},
		{"bad user dash", `[]`, Options{User: "-ana", Now: now}, "usuario"},
		{"bad theme", `[]`, Options{User: "ana", Theme: "sepia", Now: now}, "tema"},
		{"bad species", `[]`, Options{User: "ana", Species: "dragon", Now: now}, "especie"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render([]byte(tt.data), tt.o)
			if err == nil {
				t.Fatalf("want an error, got SVG of %d bytes", len(got.SVG))
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

func TestCheckMessagesAreTheOnesOfTheCLI(t *testing.T) {
	if err := (Options{Theme: "sepia"}).Check(); err == nil || err.Error() != `tema no válido "sepia" (usa light o dark)` {
		t.Errorf("theme: %v", err)
	}
	if err := (Options{Species: "dragon"}).Check(); err == nil || err.Error() != `especie no válida "dragon" (usa moss o mushroom)` {
		t.Errorf("species: %v", err)
	}
	if err := (Options{User: "a b"}).Check(); err == nil || err.Error() != `nombre de usuario no válido: "a b"` {
		t.Errorf("user: %v", err)
	}
	if err := (Options{User: "BertMarti", Species: "hongo", Theme: "dark"}).Check(); err != nil {
		t.Errorf("valid options: %v", err)
	}
}

func TestWorkflow(t *testing.T) {
	plain, err := Workflow(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if plain != BaseWorkflow {
		t.Error("without options the workflow is the base one")
	}

	got, err := Workflow(Options{User: "octocat"})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(BaseWorkflow, "        with:\n", "        with:\n          user: \"octocat\"\n", 1)
	if got != want {
		t.Errorf("workflow with user:\n%s", got)
	}

	got, err = Workflow(Options{User: "octocat", Species: "hongo", Theme: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"          user: \"octocat\"\n", "          species: mushroom\n", "          theme: dark\n", "          out: commitling.svg\n"} {
		if !strings.Contains(got, line) {
			t.Errorf("workflow lacks %q:\n%s", line, got)
		}
	}
	if strings.Index(got, "user:") > strings.Index(got, "out:") {
		t.Error("user must come before out")
	}

	moss, _ := Workflow(Options{User: "octocat", Species: "moss", Theme: "light"})
	if strings.Contains(moss, "species:") || strings.Contains(moss, "theme:") {
		t.Errorf("default species and theme are not written:\n%s", moss)
	}
}

func TestWorkflowRejectsWhatIsNotAYAMLSafeLogin(t *testing.T) {
	for _, u := range []string{"a\nb", "x: y", "octo cat", "ana\"", "-ana", strings.Repeat("a", 40)} {
		if _, err := Workflow(Options{User: u}); err == nil {
			t.Errorf("user %q must be rejected", u)
		}
	}
	if _, err := Workflow(Options{Species: "dragon"}); err == nil {
		t.Error("unknown species must be rejected")
	}
}

func TestFetchError(t *testing.T) {
	madrid := time.FixedZone("CEST", 2*3600)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, madrid)
	reset := func(d time.Duration) string { return strconv.FormatInt(now.Add(d).Unix(), 10) }
	tests := []struct {
		name              string
		status            int
		remaining, resets string
		want              []string
		wantNot           []string
	}{
		{"network", 0, "", "", []string{"conexión"}, []string{"Ver demo"}},
		{"404", 404, "", "", []string{"no existe"}, []string{"Ver demo"}},
		{"limit with reset", 403, "0", reset(17 * time.Minute), []string{"60 peticiones", "12:17", "hora local", "17 minutos", "«Ver demo»", "La Action"}, nil},
		{"limit with reset in seconds", 429, "0", reset(20 * time.Second), []string{"60 peticiones", "12:01", "1 minuto", "«Ver demo»"}, []string{"1 minutos"}},
		{"limit reset already past", 403, "0", reset(-time.Minute), []string{"60 peticiones", "Ya debería", "1 minuto", "«Ver demo»"}, []string{"11:59", "hora local"}},
		{"limit reset in an hour", 403, "0", reset(time.Hour), []string{"13:00", "60 minutos"}, nil},
		{"limit reset after midnight", 403, "0", strconv.FormatInt(time.Date(2026, 10, 1, 23, 50, 0, 0, madrid).Add(20*time.Minute).Unix(), 10), []string{"00:10"}, nil},
		{"limit without headers", 403, "", "", []string{"60 peticiones", "una hora", "«Ver demo»"}, []string{"hora local"}},
		{"limit bad reset header", 429, "0", "mañana", []string{"60 peticiones", "una hora", "«Ver demo»"}, []string{"hora local"}},
		{"5xx", 503, "", "", []string{"503", "minutos"}, []string{"Ver demo"}},
		{"other", 418, "", "", []string{"418"}, []string{"Ver demo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := FetchError(tt.status, tt.remaining, tt.resets, now)
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("FetchError = %q, want it to contain %q", msg, w)
				}
			}
			for _, w := range tt.wantNot {
				if strings.Contains(msg, w) {
					t.Errorf("FetchError = %q, must not contain %q", msg, w)
				}
			}
		})
	}
}

// The reset time is shown in the zone of the clock it is given (in the browser,
// the local one), not in UTC or in the zone of the machine that runs the test.
func TestFetchErrorShowsTheResetInTheZoneOfNow(t *testing.T) {
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC) // 12:00 in Madrid, 06:00 in New York
	reset := strconv.FormatInt(at.Add(25*time.Minute).Unix(), 10)
	for _, c := range []struct {
		zone *time.Location
		want string
	}{
		{time.UTC, "10:25"},
		{time.FixedZone("CEST", 2*3600), "12:25"},
		{time.FixedZone("EDT", -4*3600), "06:25"},
		{time.FixedZone("IST", 5*3600+1800), "15:55"},
	} {
		if msg := FetchError(403, "0", reset, at.In(c.zone)); !strings.Contains(msg, c.want) {
			t.Errorf("zone %v: %q must say %s", c.zone, msg, c.want)
		}
	}
}

// YAML would read user: 007 as a number, 1e3 as a float and null or true as
// nothing or a boolean: the login is always written as a quoted string.
func TestWorkflowQuotesTheUser(t *testing.T) {
	for _, u := range []string{"007", "1e3", "null", "true", "no", "0x1f", "octocat", "a-b"} {
		got, err := Workflow(Options{User: u})
		if err != nil {
			t.Fatalf("%q: %v", u, err)
		}
		if want := "          user: \"" + u + "\"\n"; !strings.Contains(got, want) {
			t.Errorf("user %q is not quoted:\n%s", u, got)
		}
	}
}

// The theme is written as its canonical name, never as the raw text (which
// may carry spaces, capitals or even a line break).
func TestWorkflowWritesTheNormalisedTheme(t *testing.T) {
	for _, th := range []string{"dark", "Dark", " dark ", "\ndark", "DARK\t"} {
		got, err := Workflow(Options{User: "octocat", Theme: th})
		if err != nil {
			t.Fatalf("%q: %v", th, err)
		}
		if !strings.Contains(got, "          theme: dark\n") {
			t.Errorf("theme %q is not written as dark:\n%s", th, got)
		}
		if strings.Contains(got, "theme: \n") || strings.Contains(got, "\ndark") {
			t.Errorf("theme %q leaked into the YAML:\n%s", th, got)
		}
	}
	got, _ := Workflow(Options{User: "octocat", Theme: " LIGHT "})
	if strings.Contains(got, "theme:") {
		t.Errorf("the default theme is not written:\n%s", got)
	}
}

func TestSizeOption(t *testing.T) {
	if err := (Options{Size: "grande"}).Check(); err == nil || err.Error() != `tamaño no válido "grande" (usa full o compact)` {
		t.Errorf("size: %v", err)
	}
	for _, ok := range []string{"", "full", "compact", " Compact "} {
		if err := (Options{Size: ok}).Check(); err != nil {
			t.Errorf("size %q: %v", ok, err)
		}
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	full, _ := Render(data, Options{Now: now})
	same, _ := Render(data, Options{Now: now, Size: "full"})
	small, err := Render(data, Options{Now: now, Size: "compact"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(full.SVG, same.SVG) {
		t.Error("size full must draw exactly what no size draws")
	}
	if !bytes.Contains(small.SVG, []byte(`width="200" height="60"`)) || bytes.Equal(small.SVG, full.SVG) {
		t.Errorf("size compact does not draw the compact card:\n%.200s", small.SVG)
	}
	if small.Description != full.Description {
		t.Error("both sizes describe the creature in the same words")
	}
}

func TestWorkflowWritesSizeOnlyWhenCompact(t *testing.T) {
	got, err := Workflow(Options{User: "octocat", Size: " COMPACT "})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "          size: compact\n") {
		t.Errorf("workflow lacks size: compact:\n%s", got)
	}
	for _, s := range []string{"", "full", " Full\n"} {
		got, _ := Workflow(Options{User: "octocat", Size: s})
		if strings.Contains(got, "size:") {
			t.Errorf("the default size %q is not written:\n%s", s, got)
		}
	}
	if _, err := Workflow(Options{Size: "xl\nrun: evil"}); err == nil {
		t.Error("an invalid size must be rejected before it reaches the YAML")
	}
}
