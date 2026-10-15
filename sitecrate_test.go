package sitecrate

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
		"badbase.html":      "text/html",
		"css/base.css":      "text/css",
		"css/style.css":     "text/css",
		"docs/page.html":    "text/html",
		"img/dot,copy.png":  "image/png",
		"img/dot.png":       "image/png",
		"index.html":        "text/html",
		"js/app.js":         "text/javascript",
		"js/search.js":      "text/javascript",
		"later.html":        "text/html",
		"old.html":          "text/html",
		"reload.html":       "text/html",
		"search-index.json": "application/json",
		"search.html":       "text/html",
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

func TestPackSkipsOutputThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Skip(err)
	}
	out := filepath.Join(alias, "out.html")
	for range 2 {
		if _, err := Pack(dir, "index.html", out); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := unpack(t, out)
	if len(m.Files) != 1 {
		t.Errorf("files = %v, want only index.html", m.Files)
	}
}

func TestPackSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	link := filepath.Join(t.TempDir(), "site")
	if err := os.Symlink(dir, link); err != nil {
		t.Skip(err)
	}
	out := filepath.Join(t.TempDir(), "out.html")
	if _, err := Pack(link, "index.html", out); err != nil {
		t.Fatal(err)
	}
	if m, _ := unpack(t, out); len(m.Files) != 1 {
		t.Errorf("files = %v, want index.html", m.Files)
	}
}

func TestPackErrorKeepsOutput(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>hi</p>"), 0o644)
	locked := filepath.Join(dir, "locked.txt")
	os.WriteFile(locked, []byte("secret"), 0o000)
	if _, err := os.ReadFile(locked); err == nil {
		t.Skip("cannot make a file unreadable here")
	}
	outDir := t.TempDir()
	out := filepath.Join(outDir, "out.html")
	os.WriteFile(out, []byte("old"), 0o644)
	if _, err := Pack(dir, "index.html", out); err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if data, _ := os.ReadFile(out); string(data) != "old" {
		t.Errorf("output = %q, want the previous contents", data)
	}
	if entries, _ := os.ReadDir(outDir); len(entries) != 1 {
		t.Errorf("temporary file left behind: %v", entries)
	}
}

func TestPackBadEntry(t *testing.T) {
	for _, entry := range []string{"nope.html", "css/style.css"} {
		out := filepath.Join(t.TempDir(), "out.html")
		if _, err := Pack("testdata/site", entry, out); err == nil {
			t.Errorf("%s: expected error", entry)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("%s: output created", entry)
		}
	}
}
