package events

import (
	"strings"
	"testing"
)

// ValidLogin is checked by hand (no regexp: it keeps the wasm smaller), so
// this table is the contract: the rules the regexp had.
func TestValidLogin(t *testing.T) {
	for _, ok := range []string{"a", "Z", "0", "007", "octoexample", "BertMarti", "a-b-c", "a-", "a--b", strings.Repeat("x", 39)} {
		if !ValidLogin(ok) {
			t.Errorf("ValidLogin(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{
		"", "-a", "-", "a b", " a", "a ", "a\n", "\na", "a\r", "a\tb", "a/b", "../x", "a?b", "a_b", "a.b",
		"á", "aé", "日本", "a\x00", strings.Repeat("x", 40), strings.Repeat("x", 100),
	} {
		if ValidLogin(bad) {
			t.Errorf("ValidLogin(%q) = true, want false", bad)
		}
	}
}

func TestParseEventsRejectsWhatIsNotAnArray(t *testing.T) {
	for _, in := range []string{``, `{nope`, `{"a":1}`, `"x"`} {
		if _, err := ParseEvents(strings.NewReader(in)); err == nil {
			t.Errorf("ParseEvents(%q) must fail", in)
		}
	}
	evs, err := ParseEvents(strings.NewReader(`[]`))
	if err != nil || len(evs) != 0 {
		t.Errorf("empty array: %v %v", evs, err)
	}
}

func TestDedupeAndLogin(t *testing.T) {
	evs, err := ParseEvents(strings.NewReader(`[
	 {"id":"1","type":"PushEvent","actor":{"login":"b"},"created_at":"2026-09-01T10:00:00Z"},
	 {"id":"1","type":"PushEvent","actor":{"login":"b"},"created_at":"2026-09-01T10:00:00Z"},
	 {"id":"2","type":"WatchEvent","actor":{"login":"a"},"created_at":"2026-09-02T10:00:00Z"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(Dedupe(evs)); got != 2 {
		t.Errorf("Dedupe left %d events, want 2", got)
	}
	if got := Login(evs); got != "b" {
		t.Errorf("Login = %q, want the most common actor b", got)
	}
	if got := Latest(evs); got.Day() != 2 {
		t.Errorf("Latest = %v", got)
	}
	if Login(nil) != "" || !Latest(nil).IsZero() {
		t.Error("no events: empty login and zero time")
	}
}
