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
// under dir, except the file skip. Symlinks to files outside dir are only
// packed when followSymlinks is set.
func walk(dir string, skip fs.FileInfo, followSymlinks bool) (found, error) {
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
		rel = filepath.ToSlash(rel)
		switch {
		case d.Type()&fs.ModeSymlink != 0 && !followSymlinks && !inside(dir, p):
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

// inside reports whether the symlink p resolves to a file under dir
func inside(dir, p string) bool {
	target, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
