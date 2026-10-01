//go:build js && wasm

// Command wasm is the WebAssembly build of commitling: a thin layer that
// hands internal/generate to the JavaScript of the website. It has no logic of
// its own, so what the browser draws is what `commitling render` draws.
//
//	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o commitling.wasm ./cmd/wasm
//
// It registers one global object, `commitling`:
//
//	commitling.render(eventsJSON, user, species, theme, nowMs, size) -> {svg, description, login} | {error}
//	commitling.workflow(user, species, theme, size)                  -> {workflow} | {error}
//	commitling.explain(status, remaining, reset, nowMs)              -> string
//	commitling.check(user, species, theme, size)                     -> "" when valid, else the error
//	commitling.demo(day, species, theme, size)                       -> {svg, description, login, day, days, phase} | {error}
//
// size is "full" or "compact" and always the last argument, so a caller that
// predates it keeps working. Empty or undefined strings mean "the default"; an
// empty nowMs is the clock.
package main

import (
	"syscall/js"
	"time"

	"github.com/BertMarti/commitling/internal/generate"
)

func main() {
	register()
	select {} // keep the runtime alive: JavaScript calls into it
}

func register() {
	js.Global().Set("commitling", map[string]any{
		"render":   js.FuncOf(render),
		"workflow": js.FuncOf(workflow),
		"explain":  js.FuncOf(explain),
		"check":    js.FuncOf(check),
		"demo":     js.FuncOf(demo),
	})
}

// str reads an optional string argument.
func str(args []js.Value, i int) string {
	if i >= len(args) || args[i].IsUndefined() || args[i].IsNull() {
		return ""
	}
	return args[i].String()
}

// millis reads an optional time in milliseconds since the epoch (zero time
// when absent or zero).
func millis(args []js.Value, i int) time.Time {
	if i >= len(args) || args[i].Type() != js.TypeNumber || args[i].Float() == 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(args[i].Float())).UTC()
}

// options reads the user, species, theme and size from the given positions;
// -1 means that the call has no such argument.
func options(args []js.Value, user, species, theme, size int) generate.Options {
	o := generate.Options{Species: str(args, species), Theme: str(args, theme), Size: str(args, size)}
	if user >= 0 {
		o.User = str(args, user)
	}
	return o
}

func render(_ js.Value, args []js.Value) any {
	o := options(args, 1, 2, 3, 5)
	o.Now = millis(args, 4)
	res, err := generate.Render([]byte(str(args, 0)), o)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"svg": string(res.SVG), "description": res.Description, "login": res.Login}
}

func workflow(_ js.Value, args []js.Value) any {
	w, err := generate.Workflow(options(args, 0, 1, 2, 3))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"workflow": w}
}

// demo draws one day (0..days) of the fictitious timelapse of the website.
func demo(_ js.Value, args []js.Value) any {
	day := 0
	if len(args) > 0 && args[0].Type() == js.TypeNumber {
		day = args[0].Int()
	}
	f, err := generate.Demo(day, options(args, -1, 1, 2, 3))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"svg": string(f.SVG), "description": f.Description, "login": f.Login, "day": f.Day, "days": f.Days, "phase": f.Phase}
}

func check(_ js.Value, args []js.Value) any {
	if err := options(args, 0, 1, 2, 3).Check(); err != nil {
		return err.Error()
	}
	return ""
}

func explain(_ js.Value, args []js.Value) any {
	status := 0
	if len(args) > 0 && args[0].Type() == js.TypeNumber {
		status = args[0].Int()
	}
	now := millis(args, 3)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return generate.FetchError(status, str(args, 1), str(args, 2), now)
}
