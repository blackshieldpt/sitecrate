// Package sitecrate packs a static site directory into a single HTML file.
package sitecrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Pack writes the site rooted at dir to the single HTML file out, opening on
// the page entry. Symlinks that leave dir, including absolute ones, are
// skipped unless followSymlinks is set. It returns warnings about unsupported features in
// use and skipped files. On error, an existing out is left untouched.
func Pack(dir, entry, out string, followSymlinks bool) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	title := filepath.Base(dir)
	if dir, err = filepath.EvalSymlinks(dir); err != nil { // WalkDir does not follow a symlinked root
		return nil, err
	}
	// without followSymlinks, files are found and read through an os.Root,
	// which refuses symlinks that leave dir, even one swapped in after the walk
	var root *os.Root
	fsys := os.DirFS(dir)
	if !followSymlinks {
		if root, err = os.OpenRoot(dir); err != nil {
			return nil, err
		}
		defer root.Close()
		fsys = root.FS()
	}
	skip, _ := os.Stat(out) // the output is not packed when it lives inside dir
	found, err := walk(dir, root, skip)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(found.paths, entry) {
		return nil, fmt.Errorf("entry %q not found in %s", entry, dir)
	}
	if mimeType(entry) != "text/html" {
		return nil, fmt.Errorf("entry %q is not an HTML page", entry)
	}

	f, err := os.CreateTemp(filepath.Dir(out), "."+filepath.Base(out)+".*")
	if err != nil {
		return nil, err
	}
	warnings, err := write(f, fsys, title, entry, found.paths)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(f.Name(), out)
	}
	if err != nil {
		os.Remove(f.Name())
		return nil, err
	}
	if n := len(found.outside); n > 0 {
		warnings = append(warnings, fmt.Sprintf("skipped %d symlink(s) that leave the site or are absolute, e.g. %s; pack them with -follow-symlinks", n, found.outside[0]))
	}
	if n := len(found.invalid); n > 0 {
		warnings = append(warnings, fmt.Sprintf("skipped %d file(s) whose name is not valid UTF-8, e.g. %q", n, found.invalid[0]))
	}
	return warnings, nil
}
