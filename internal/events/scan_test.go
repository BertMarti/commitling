package events

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// refEvent and refPush are the encoding/json decoding the hand-written reader
// replaced (the production code no longer links encoding/json: it weighed a
// megabyte in the WebAssembly build). The tests below compare the two.
type refEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Actor     struct {
		Login string `json:"login"`
	} `json:"actor"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload json.RawMessage `json:"payload"`
}

type refPush struct {
	Size    *int              `json:"size"`
	Commits []json.RawMessage `json:"commits"`
}

func refCount(payload []byte) int {
	var p refPush
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &p)
	}
	switch {
	case p.Size != nil && *p.Size > 0:
		return *p.Size
	case len(p.Commits) > 0:
		return len(p.Commits)
	default:
		return 1
	}
}

func refAction(payload []byte) string {
	var p struct {
		Action string `json:"action"`
	}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &p)
	}
	return p.Action
}

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Documents the reader must accept, or reject, exactly like encoding/json.
func parseCorpus(t *testing.T) []string {
	ev := func(rest string) string { return `[{"id":"1","type":"PushEvent"` + rest + `}]` }
	return []string{
		fixture(t),
		``, ` `, `[]`, ` [ ] `, `null`, `[null]`, `{}`, `{"a":1}`, `"x"`, `1`, `true`, `[1]`, `["x"]`, `[[]]`, `{nope`, `[{`, `[{"id"`, `[{"id":`, `[{"id":"1"`, `[{"id":"1",}]`, `[{"id":"1"} {"id":"2"}]`, `[,]`, `[{"id":"1"},]`,
		`[{}]`, `[{"id":"a","id":"b"}]`, `[{"ID":"a","Type":"T","CREATED_AT":"2026-09-01T10:00:00Z"}]`,
		`[{"id":null,"type":null,"created_at":null,"actor":null,"repo":null,"payload":null}]`,
		`[{"id":1}]`, `[{"type":true}]`, `[{"id":[]}]`, `[{"actor":[]}]`, `[{"actor":{"login":5}}]`, `[{"repo":"x"}]`, `[{"repo":{"name":null}}]`,
		`[{"created_at":"yesterday"}]`, `[{"created_at":5}]`, `[{"created_at":"2026-09-01T10:00:00+02:00"}]`, `[{"created_at":"2026-09-01T10:00:00.123456Z"}]`,
		`[{"actor":{"login":"a"},"actor":{"login":"b"}}]`, `[{"actor":{"login":"a"},"actor":{"id":2}}]`,
		`[{"extra":{"deep":[1,2.5e10,-0,"s",true,false,null,{"x":[[]]}]},"id":"z"}]`,
		ev(`,"payload":{"size":3}`), ev(`,"payload":[1,2]`), ev(`,"payload":"str"`), ev(`,"payload":  {"a" : [ ] }  `), ev(`,"payload":nul`),
		// strings: escapes, unicode, surrogates, control characters
		`[{"id":"a\"b\c\/d\b\f\n\r\t"}]`, `[{"id":"\u00e9\u20AC\ud83d\ude00"}]`, `[{"id":"\ud83d"}]`, `[{"id":"\ud83dx"}]`, `[{"id":"\ud83d\u0041"}]`, `[{"id":"\ude00"}]`, `[{"id":"\ude00\ud83d\ude00"}]`,
		`[{"id":"\x"}]`, `[{"id":"\u12"}]`, `[{"id":"\u12g4"}]`, "[{\"id\":\"a\nb\"}]", "[{\"id\":\"a\tb\"}]", `[{"id":"sin cerrar}]`, "[{\"id\":\"\xff\xfe ok \xc3\"}]", "[{\"id\":\"\u0041\xff\"}]",
		`[{"i\u0064":"escaped key"}]`,
		// numbers (skipped, but they must be valid)
		`[{"n":01}]`, `[{"n":-}]`, `[{"n":1.}]`, `[{"n":.5}]`, `[{"n":1e}]`, `[{"n":1e+}]`, `[{"n":+1}]`, `[{"n":0.5e-3}]`, `[{"n":-0}]`, `[{"n":NaN}]`,
		`[{"b":tru}]`, `[{"b":True}]`, `[{"b":falsey}]`,
		// what follows the array is ignored, like json.Decoder
		`[] trailing`, `[{"id":"1"}]]`, `[]{`,
		"\ufeff[]", "\t\r\n[\n]\n",
		strings.Repeat("[", 5) + strings.Repeat("]", 5), `[{"x":` + strings.Repeat("[", 400) + strings.Repeat("]", 400) + `}]`,
	}
}

func TestParseEventsMatchesEncodingJSON(t *testing.T) {
	for _, in := range parseCorpus(t) {
		name := in
		if len(name) > 70 {
			name = name[:70] + "..."
		}
		var want []refEvent
		wantErr := json.NewDecoder(strings.NewReader(in)).Decode(&want)
		got, err := ParseEvents(strings.NewReader(in))
		if (err != nil) != (wantErr != nil) {
			t.Errorf("%q: error = %v, encoding/json: %v", name, err, wantErr)
			continue
		}
		if err != nil {
			if !strings.Contains(err.Error(), "eventos") {
				t.Errorf("%q: the error should say that the events could not be read: %v", name, err)
			}
			continue
		}
		if (got == nil) != (want == nil) || len(got) != len(want) {
			t.Errorf("%q: %d events (nil %v), encoding/json: %d (nil %v)", name, len(got), got == nil, len(want), want == nil)
			continue
		}
		for i, g := range got {
			w := want[i]
			if g.ID != w.ID || g.Type != w.Type || !g.CreatedAt.Equal(w.CreatedAt) || g.Actor.Login != w.Actor.Login || g.Repo.Name != w.Repo.Name {
				t.Errorf("%q event %d: %+v, encoding/json: %+v", name, i, g, w)
			}
			if string(g.Payload) != string(w.Payload) && !(len(w.Payload) == 4 && string(w.Payload) == "null") {
				t.Errorf("%q event %d: payload %q, encoding/json: %q", name, i, g.Payload, w.Payload)
			}
		}
	}
}

// Intentional difference: nesting deeper than maxDepth is an error (it would
// overflow the stack in a browser), never a crash.
func TestParseEventsRejectsAbsurdNesting(t *testing.T) {
	for _, in := range []string{strings.Repeat("[", 100000), `[{"x":` + strings.Repeat("[", 2000) + strings.Repeat("]", 2000) + `}]`, `[{"payload":` + strings.Repeat(`{"a":`, 3000) + `1` + strings.Repeat("}", 3000) + `}]`} {
		if _, err := ParseEvents(strings.NewReader(in)); err == nil {
			t.Errorf("nesting of %d bytes must fail", len(in))
		}
	}
	if got := CommitCount(Event{Payload: []byte(`{"size": 2, "x": ` + strings.Repeat("[", 3000) + `}`)}); got != 1 {
		t.Errorf("a payload too deep (and not even closed) counts as one commit, got %d", got)
	}
}

func TestPayloadReadersMatchEncodingJSON(t *testing.T) {
	payloads := []string{
		``, `null`, `{}`, `[]`, `"not an object"`, `7`, `{"size": 4, "commits": [{}, {}]}`, `{"commits": [{}, {}, {}]}`, `{"size": 0, "commits": [{}]}`,
		`{"size": -3, "commits": [{}]}`, `{"size": 2.0}`, `{"size": 2.5, "commits": [{}]}`, `{"size": "7", "commits": [{}, {}]}`, `{"size": null, "commits": [1, 2]}`,
		`{"size": 1e2}`, `{"size": 99999999999999999999}`, `{"size": 5, "size": 6}`, `{"size": 5, "size": null}`, `{"size": 5, "size": "x"}`,
		`{"commits": [{}], "commits": [{}, {}]}`, `{"commits": [{}, {}], "commits": null}`, `{"commits": [{}, {}], "commits": {}}`, `{"commits": {}}`, `{"commits": "x"}`,
		`{"SIZE": 3}`, `{"Commits": [1]}`, `{"push_id": 1, "ref": "refs/heads/main"}`, `{"size": 2} x`, `{"size": 2,}`, `{"size": }`, `{"size": 2`, `{"commits": [{"a": [1, {"b": 2}]}, "x", null]}`,
		`{"action": "opened"}`, `{"action": "closed"}`, `{"action": 5}`, `{"action": null}`, `{"action": "a", "action": "b"}`, `{"Action": "opened"}`, `{"action": "\u006fpened"}`, `{"action": "opened"`, `{"action": ["x"]}`,
	}
	for _, p := range payloads {
		e := Event{Payload: []byte(p)}
		if got, want := CommitCount(e), refCount([]byte(p)); got != want {
			t.Errorf("CommitCount(%q) = %d, encoding/json: %d", p, got, want)
		}
		if got, want := action(e), refAction([]byte(p)); got != want {
			t.Errorf("action(%q) = %q, encoding/json: %q", p, got, want)
		}
	}
}
