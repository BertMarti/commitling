//go:build js && wasm

package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"syscall/js"

	"github.com/BertMarti/commitling/internal/generate"
)

// These tests run inside a real WebAssembly runtime (Node, through
// go_js_wasm_exec): GOOS=js GOARCH=wasm go test -exec=$(go env GOROOT)/lib/wasm/go_js_wasm_exec ./cmd/wasm

func api(t *testing.T) js.Value {
	t.Helper()
	register()
	c := js.Global().Get("commitling")
	if c.IsUndefined() {
		t.Fatal("commitling is not registered on the global object")
	}
	return c
}

func TestRenderIsGenerateRender(t *testing.T) {
	data, err := os.ReadFile("../../testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	want, err := generate.Render(data, generate.Options{User: "octoexample", Species: "mushroom", Theme: "dark", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	got := api(t).Call("render", string(data), "octoexample", "mushroom", "dark", now.UnixMilli())
	if e := got.Get("error"); !e.IsUndefined() {
		t.Fatalf("error: %s", e.String())
	}
	if !bytes.Equal([]byte(got.Get("svg").String()), want.SVG) {
		t.Error("commitling.render and generate.Render draw different SVGs")
	}
	if got.Get("description").String() != want.Description || got.Get("login").String() != "octoexample" {
		t.Errorf("description %q, login %q", got.Get("description"), got.Get("login"))
	}
}

func TestRenderWithoutDateUsesTheClock(t *testing.T) {
	got := api(t).Call("render", `[]`, "newcomer", "", "", js.Undefined())
	if e := got.Get("error"); !e.IsUndefined() {
		t.Fatalf("error: %s", e.String())
	}
}

func TestRenderErrorsComeBackAsAnErrorField(t *testing.T) {
	c := api(t)
	for name, args := range map[string][]any{
		"broken json": {`{nope`, "ana", "moss", "light", 0},
		"bad user":    {`[]`, "no es válido", "moss", "light", 0},
		"bad species": {`[]`, "ana", "dragon", "light", 0},
		"bad theme":   {`[]`, "ana", "moss", "sepia", 0},
	} {
		got := c.Call("render", args...)
		if got.Get("error").IsUndefined() || got.Get("error").String() == "" {
			t.Errorf("%s: want an error, got %v", name, got)
		}
		if !got.Get("svg").IsUndefined() {
			t.Errorf("%s: an error must not carry an svg", name)
		}
	}
}

func TestWorkflow(t *testing.T) {
	want, _ := generate.Workflow(generate.Options{User: "octocat", Species: "mushroom", Theme: "dark"})
	got := api(t).Call("workflow", "octocat", "mushroom", "dark")
	if got.Get("workflow").String() != want {
		t.Errorf("workflow differs:\n%s", got.Get("workflow").String())
	}
	if bad := api(t).Call("workflow", "a b", "moss", "light"); bad.Get("error").IsUndefined() {
		t.Error("a login with a space must be rejected")
	}
}

func TestExplain(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	want := generate.FetchError(403, "0", "1790856600", now)
	got := api(t).Call("explain", 403, "0", "1790856600", now.UnixMilli()).String()
	if got != want {
		t.Errorf("explain = %q, want %q", got, want)
	}
	if api(t).Call("explain", 404, js.Undefined(), js.Undefined(), js.Undefined()).String() == "" {
		t.Error("explain must say something for a 404")
	}
}

func TestCheck(t *testing.T) {
	c := api(t)
	if got := c.Call("check", "BertMarti", "mushroom", "dark").String(); got != "" {
		t.Errorf("valid options: %q", got)
	}
	if got := c.Call("check", "BertMarti", js.Undefined(), js.Undefined()).String(); got != "" {
		t.Errorf("defaults: %q", got)
	}
	want := generate.Options{User: "a b"}.Check().Error()
	if got := c.Call("check", "a b", "moss", "light").String(); got != want {
		t.Errorf("check = %q, want %q", got, want)
	}
}
