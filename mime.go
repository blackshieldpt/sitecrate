package sitecrate

import (
	"mime"
	"path/filepath"
	"strings"
)

// types pins common web extensions, so that the output does not depend on the
// mime database of the system doing the packing
var types = map[string]string{
	".html":        "text/html",
	".htm":         "text/html",
	".css":         "text/css",
	".js":          "text/javascript",
	".mjs":         "text/javascript",
	".json":        "application/json",
	".xml":         "text/xml",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".wasm":        "application/wasm",
	".pdf":         "application/pdf",
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
