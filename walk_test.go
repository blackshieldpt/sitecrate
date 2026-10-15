package sitecrate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPackSymlinks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	os.WriteFile(filepath.Join(dir, "local.js"), []byte("var local"), 0o644)
	target := filepath.Join(t.TempDir(), "lib.js")
	os.WriteFile(target, []byte("var lib"), 0o644)
	if err := os.Symlink(target, filepath.Join(dir, "lib.js")); err != nil {
		t.Skip(err)
	}
	os.Symlink(filepath.Join(dir, "local.js"), filepath.Join(dir, "alias.js"))
	os.Symlink(filepath.Dir(target), filepath.Join(dir, "linked-dir"))
	os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken.js"))

	for _, follow := range []bool{false, true} {
		out := filepath.Join(t.TempDir(), "out.html")
		warnings, err := Pack(dir, "index.html", out, follow)
		if err != nil {
			t.Fatal(err)
		}
		m, files := unpack(t, out)
		var paths []string
		for _, f := range m.Files {
			paths = append(paths, f.Path)
		}
		want := []string{"alias.js", "index.html", "local.js"}
		var wantWarnings []string
		if follow {
			want = []string{"alias.js", "index.html", "lib.js", "local.js"}
			if string(files["lib.js"]) != "var lib" {
				t.Errorf("lib.js = %q", files["lib.js"])
			}
		} else {
			wantWarnings = []string{"skipped 1 symlink(s) to files outside the site, e.g. lib.js; pack them with -follow-symlinks"}
		}
		if !slices.Equal(paths, want) {
			t.Errorf("follow=%v: files = %v, want %v", follow, paths, want)
		}
		if !slices.Equal(warnings, wantWarnings) {
			t.Errorf("follow=%v: warnings = %q, want %q", follow, warnings, wantWarnings)
		}
	}
}

func TestPackInvalidUTF8Names(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	if err := os.WriteFile(filepath.Join(dir, "a\xff.txt"), []byte("x"), 0o644); err != nil {
		t.Skip(err)
	}
	if entries, _ := os.ReadDir(dir); !slices.ContainsFunc(entries, func(e os.DirEntry) bool { return e.Name() == "a\xff.txt" }) {
		t.Skip("file system does not keep names that are not valid UTF-8")
	}
	out := filepath.Join(t.TempDir(), "out.html")
	warnings, err := Pack(dir, "index.html", out, false)
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
