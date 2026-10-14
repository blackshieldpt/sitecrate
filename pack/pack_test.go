package pack

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// unpack extracts the manifest and file contents from a packed HTML file
func unpack(t *testing.T, out string) (manifest, map[string][]byte) {
	t.Helper()
	page, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(page), `<script id="sitecrate-data" type="application/octet-stream">`)
	if !ok {
		t.Fatal("payload script not found")
	}
	b64, _, _ := strings.Cut(rest, "</script>")
	gz, err := gzip.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64)))
	if err != nil {
		t.Fatal(err)
	}
	buf, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	end := len(buf) - 4
	mlen := int(binary.BigEndian.Uint32(buf[end:]))
	var m manifest
	if err := json.Unmarshal(buf[end-mlen:end], &m); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	off := 0
	for _, f := range m.Files {
		files[f.Path] = buf[off : off+f.Size]
		off += f.Size
	}
	if off != end-mlen {
		t.Fatalf("file sizes sum to %d, want %d", off, end-mlen)
	}
	return m, files
}

func TestPackRoundTrip(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.html")
	warnings, err := Pack("testdata/site", "index.html", out)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	m, files := unpack(t, out)
	if m.Entry != "index.html" {
		t.Errorf("entry = %q", m.Entry)
	}
	types := map[string]string{
		"about/index.html":  "text/html",
		"css/base.css":      "text/css",
		"css/style.css":     "text/css",
		"docs/page.html":    "text/html",
		"img/dot.png":       "image/png",
		"index.html":        "text/html",
		"js/app.js":         "text/javascript",
		"js/search.js":      "text/javascript",
		"search-index.json": "application/json",
		"search.html":       "text/html",
		"old.html":          "text/html",
	}
	if len(m.Files) != len(types) {
		t.Errorf("packed %d files, want %d", len(m.Files), len(types))
	}
	for _, f := range m.Files {
		if f.Type != types[f.Path] {
			t.Errorf("%s: type = %q, want %q", f.Path, f.Type, types[f.Path])
		}
		want, err := os.ReadFile(filepath.Join("testdata/site", f.Path))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files[f.Path], want) {
			t.Errorf("%s: content differs from source", f.Path)
		}
	}
}

func TestPackSkipsOutputInsideSite(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	out := filepath.Join(dir, "out.html")
	for range 2 {
		if _, err := Pack(dir, "index.html", out); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := unpack(t, out)
	if len(m.Files) != 1 || m.Files[0].Path != "index.html" {
		t.Errorf("files = %v, want only index.html", m.Files)
	}
}

func TestPackWarnings(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<form action="/search"></form><form method="post" action="/login"></form><script src="app.js"></script>`), 0o644)
	os.WriteFile(filepath.Join(dir, "app.js"), []byte(`import x from "./x.js"; fetch("data.json"); new XMLHttpRequest()`), 0o644)
	os.WriteFile(filepath.Join(dir, "x.js"), []byte(`export default fetch`), 0o644)
	os.WriteFile(filepath.Join(dir, "y.js"), []byte(`fetch("y.json"); new XMLHttpRequest()`), 0o644)
	warnings, err := Pack(dir, "index.html", filepath.Join(t.TempDir(), "out.html"))
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

func TestPackMissingEntry(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.html")
	if _, err := Pack("testdata/site", "nope.html", out); err == nil {
		t.Error("expected error for missing entry")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("output created despite missing entry")
	}
}
