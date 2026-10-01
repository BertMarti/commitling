package generate

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The WebAssembly build of the website is downloaded by every visitor of the
// generator, so what it links matters (v0.7.0 took it from 4.6 MB to 2.5 MB by
// dropping encoding/json and fmt). Packages that cmd/wasm pulls in must not
// bring the heavy ones back; the CLI and the gallery, which run natively, may
// use whatever they like. See docs/specs/v0.7.md for the measurements.
func TestWasmPackagesAvoidHeavyImports(t *testing.T) {
	heavy := map[string]string{
		"encoding/json": "reflect and a megabyte of decoder",
		"fmt":           "reflect, os and the printer",
		"reflect":       "type machinery",
		"regexp":        "half a megabyte",
		"net/http":      "five megabytes",
		"os":            "files and processes",
		"html/template": "reflect and the HTML parser",
		"text/template": "reflect",
	}
	for _, dir := range []string{"../events", "../generate", "../render", "../stats", "../creature", "../../cmd/wasm"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no Go files in %s: %v", dir, err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range src.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if why, bad := heavy[path]; bad {
					t.Errorf("%s imports %q (%s): it would make the browser build much bigger", f, path, why)
				}
			}
		}
	}
}
