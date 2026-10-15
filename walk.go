package sitecrate

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// walk returns the slash-separated relative paths of all non-hidden files
// under dir, except the file skip. Paths that are not valid UTF-8, which the
// manifest cannot carry, are returned in invalid instead.
func walk(dir string, skip fs.FileInfo) (paths, invalid []string, err error) {
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// follow symlinks to files, as a web server would
		info, err := os.Stat(p)
		if err != nil || !info.Mode().IsRegular() || (skip != nil && os.SameFile(info, skip)) {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if rel = filepath.ToSlash(rel); utf8.ValidString(rel) {
			paths = append(paths, rel)
		} else {
			invalid = append(invalid, rel)
		}
		return nil
	})
	return paths, invalid, err
}
