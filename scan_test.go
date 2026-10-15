package sitecrate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPackWarnings(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<form action="/search"></form><form method="post" action="/login"></form><script src="app.js"></script>`), 0o644)
	os.WriteFile(filepath.Join(dir, "app.js"), []byte(`import x from "./x.js"; fetch("data.json"); new XMLHttpRequest()`), 0o644)
	os.WriteFile(filepath.Join(dir, "x.js"), []byte(`export default fetch`), 0o644)
	os.WriteFile(filepath.Join(dir, "y.js"), []byte(`fetch("y.json"); new XMLHttpRequest()`), 0o644)
	warnings, err := Pack(dir, "index.html", filepath.Join(t.TempDir(), "out.html"), false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"XMLHttpRequest in 2 file(s), e.g. app.js: may not work in packed output",
		"relative module imports in 1 file(s), e.g. app.js: may not work in packed output",
		"POST forms in 1 file(s), e.g. index.html: may not work in packed output",
	}
	if !slices.Equal(warnings, want) {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}
}
