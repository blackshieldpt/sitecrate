package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/blackshieldpt/sitecrate"
)

// version is the release version; the release workflow checks that the tag
// matches it
const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run runs the command line args and returns the exit status
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sitecrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("o", "out.html", "output file")
	entry := flags.String("entry", "index.html", "entry page, relative to site-dir")
	follow := flags.Bool("follow-symlinks", false, "also pack symlinks to files outside site-dir")
	showVersion := flags.Bool("version", false, "print the version and exit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: sitecrate <site-dir> [-o out.html] [-entry index.html] [-follow-symlinks]")
		fmt.Fprintln(stderr, "       sitecrate -version")
		flags.PrintDefaults()
	}
	failed := func(err error) int { // exit status for a parse error
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if err := flags.Parse(args); err != nil {
		return failed(err)
	}
	dir := flags.Arg(0)
	if flags.NArg() > 1 {
		// allow flags after <site-dir>
		if err := flags.Parse(flags.Args()[1:]); err != nil {
			return failed(err)
		}
		if flags.NArg() > 0 {
			fmt.Fprintf(stderr, "sitecrate: unexpected argument %q\n", flags.Arg(0))
			flags.Usage()
			return 2
		}
	}
	if *showVersion {
		fmt.Fprintln(stdout, "sitecrate", version)
		return 0
	}
	if dir == "" {
		flags.Usage()
		return 2
	}

	warnings, err := sitecrate.Pack(dir, filepath.ToSlash(filepath.Clean(*entry)), *out, *follow)
	for _, w := range warnings {
		fmt.Fprintln(stderr, "warning:", printable(w))
	}
	if err != nil {
		fmt.Fprintln(stderr, "sitecrate:", printable(err.Error()))
		return 1
	}
	return 0
}

// printable escapes control characters, which file names in warnings and
// errors can contain, so that the output cannot drive the terminal
func printable(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			fmt.Fprintf(&b, "\\x%02x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
