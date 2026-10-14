// Package pack bundles a static site directory into a single HTML file.
package pack

import (
	"bufio"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

//go:embed runtime.js
var runtimeJS string

const (
	shellHead = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>`
	shellData = `</title>
<style>html,body{height:100%;margin:0}iframe{display:block;width:100%;height:100%;border:0}</style>
</head>
<body>
<script id="sitecrate-data" type="application/octet-stream">`
	shellRuntime = `</script>
<script>
`
	shellTail = `</script>
</body>
</html>
`
)

type file struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

type manifest struct {
	Entry string `json:"entry"`
	Files []file `json:"files"`
}

// types covers common web extensions missing from Go's builtin mime table
var types = map[string]string{
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".eot":         "application/vnd.ms-fontobject",
	".ico":         "image/x-icon",
	".mp4":         "video/mp4",
	".webm":        "video/webm",
	".mp3":         "audio/mpeg",
	".ogg":         "audio/ogg",
	".wav":         "audio/wav",
	".txt":         "text/plain",
	".map":         "application/json",
	".webmanifest": "application/manifest+json",
}

// unsupported lists constructs that may not work in the packed output
var unsupported = []struct {
	re   *regexp.Regexp
	what string
}{
	{regexp.MustCompile(`\bXMLHttpRequest\b`), "XMLHttpRequest"},
	{regexp.MustCompile(`\b(?:import|from)\s*\(?\s*["']\.{0,2}/`), "relative module imports"},
	{regexp.MustCompile(`(?i)<form\b[^>]*\bmethod\s*=\s*["']?post`), "POST forms"},
}

// Pack writes the site rooted at dir to the single HTML file out, opening on
// the page entry. It returns warnings about unsupported features in use.
func Pack(dir, entry, out string) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	skip, err := filepath.Abs(out)
	if err != nil {
		return nil, err
	}
	paths, err := walk(dir, skip)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(paths, entry) {
		return nil, fmt.Errorf("entry %q not found in %s", entry, dir)
	}

	f, err := os.Create(out)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString(shellHead + html.EscapeString(filepath.Base(dir)) + shellData)

	// payload: gzip(file bytes... | manifest JSON | uint32 manifest length)
	b64 := base64.NewEncoder(base64.StdEncoding, w)
	gz, _ := gzip.NewWriterLevel(b64, gzip.BestCompression)
	m := manifest{Entry: entry}
	found := make([][]string, len(unsupported)) // files matching each unsupported construct
	for _, p := range paths {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		typ := mimeType(p)
		if typ == "text/html" || typ == "text/javascript" {
			for i, u := range unsupported {
				if u.re.Match(data) {
					found[i] = append(found[i], p)
				}
			}
		}
		gz.Write(data)
		m.Files = append(m.Files, file{p, typ, len(data)})
	}
	js, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	gz.Write(js)
	binary.Write(gz, binary.BigEndian, uint32(len(js)))
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if err := b64.Close(); err != nil {
		return nil, err
	}

	w.WriteString(shellRuntime + runtimeJS + shellTail)
	if err := w.Flush(); err != nil {
		return nil, err
	}
	var warnings []string
	for i, u := range unsupported {
		if n := len(found[i]); n > 0 {
			warnings = append(warnings, fmt.Sprintf("%s in %d file(s), e.g. %s: may not work in packed output", u.what, n, found[i][0]))
		}
	}
	return warnings, f.Close()
}

// walk returns the slash-separated relative paths of all non-hidden files
// under dir, except skip.
func walk(dir, skip string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		regular := d.Type().IsRegular()
		if d.Type()&fs.ModeSymlink != 0 {
			// follow symlinks to files, as a web server would
			info, err := os.Stat(p)
			regular = err == nil && info.Mode().IsRegular()
		}
		if !regular || p == skip {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	return paths, err
}

func mimeType(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	if t, ok := types[ext]; ok {
		return t
	}
	if t, _, err := mime.ParseMediaType(mime.TypeByExtension(ext)); err == nil {
		return t
	}
	return "application/octet-stream"
}
