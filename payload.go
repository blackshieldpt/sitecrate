package sitecrate

import (
	"bufio"
	"compress/gzip"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"html"
	"io"
	"io/fs"
	"strings"
)

//go:embed shell.html
var shell string

//go:embed runtime/*.js
var runtimeFS embed.FS

// runtimeFiles are the parts of the runtime, in the order they are joined
var runtimeFiles = []string{"unpack.js", "url.js", "rewrite.js", "frame.js", "main.js"}

var (
	shellParts = splitShell()
	runtimeJS  = joinRuntime()
)

// splitShell splits shell around its {{title}}, {{data}} and {{runtime}}
// placeholders
func splitShell() [4]string {
	var parts [4]string
	rest := shell
	for i, mark := range []string{"{{title}}", "{{data}}", "{{runtime}}"} {
		var ok bool
		if parts[i], rest, ok = strings.Cut(rest, mark); !ok {
			panic("shell.html: missing " + mark)
		}
	}
	parts[3] = rest
	return parts
}

func joinRuntime() string {
	var b strings.Builder
	for _, name := range runtimeFiles {
		data, err := runtimeFS.ReadFile("runtime/" + name)
		if err != nil {
			panic(err)
		}
		b.Write(data)
	}
	return b.String()
}

type file struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

type manifest struct {
	Entry string `json:"entry"`
	Files []file `json:"files"`
}

// write writes the shell page with the files at paths in fsys as its
// payload to w, and returns warnings about unsupported features in use.
func write(w io.Writer, fsys fs.FS, title, entry string, paths []string) ([]string, error) {
	bw := bufio.NewWriter(w)
	bw.WriteString(shellParts[0] + html.EscapeString(title) + shellParts[1])

	// payload: gzip(file bytes... | manifest JSON | uint32 manifest length)
	b64 := base64.NewEncoder(base64.StdEncoding, bw)
	gz, _ := gzip.NewWriterLevel(b64, gzip.BestCompression)
	m := manifest{Entry: entry}
	s := newScanner()
	for _, p := range paths {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, err
		}
		typ := mimeType(p)
		s.scan(p, typ, data)
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

	bw.WriteString(shellParts[2] + runtimeJS + shellParts[3])
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	return s.warnings(), nil
}
