package sitecrate

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// found is the result of walking a site directory
type found struct {
	paths   []string // files to pack
	invalid []string // names that are not valid UTF-8, which the manifest cannot carry
	outside []string // symlinks to files outside the site, when they are not followed
}

// walk finds the slash-separated relative paths of all non-hidden files
// under dir, except the file skip. Symlinks that do not resolve within root,
// the root at dir, are only packed when root is nil.
func walk(dir string, root *os.Root, skip fs.FileInfo) (found, error) {
	var f found
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
		outside := false
		if d.Type()&fs.ModeSymlink != 0 && root != nil {
			_, err := root.Stat(rel)
			outside = err != nil
		}
		rel = filepath.ToSlash(rel)
		switch {
		case outside:
			f.outside = append(f.outside, rel)
		case !utf8.ValidString(rel):
			f.invalid = append(f.invalid, rel)
		default:
			f.paths = append(f.paths, rel)
		}
		return nil
	})
	return f, err
}
