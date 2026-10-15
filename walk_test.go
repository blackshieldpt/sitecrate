package sitecrate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPackFollowsFileSymlinks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	target := filepath.Join(t.TempDir(), "lib.js")
	os.WriteFile(target, []byte("var lib"), 0o644)
	if err := os.Symlink(target, filepath.Join(dir, "lib.js")); err != nil {
		t.Skip(err)
	}
	os.Symlink(filepath.Dir(target), filepath.Join(dir, "linked-dir"))
	os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken.js"))
	out := filepath.Join(t.TempDir(), "out.html")
	if _, err := Pack(dir, "index.html", out); err != nil {
		t.Fatal(err)
	}
	m, files := unpack(t, out)
	if len(m.Files) != 2 || string(files["lib.js"]) != "var lib" {
		t.Errorf("files = %v, want index.html and the symlinked lib.js", m.Files)
	}
}

func TestPackInvalidUTF8Names(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	if err := os.WriteFile(filepath.Join(dir, "a\xff.txt"), []byte("x"), 0o644); err != nil {
		t.Skip(err)
	}
	out := filepath.Join(t.TempDir(), "out.html")
	warnings, err := Pack(dir, "index.html", out)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`skipped 1 file(s) whose name is not valid UTF-8, e.g. "a\xff.txt"`}
	if !slices.Equal(warnings, want) {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}
	if m, _ := unpack(t, out); len(m.Files) != 1 {
		t.Errorf("files = %v, want index.html", m.Files)
	}
}
