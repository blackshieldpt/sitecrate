package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blackshieldpt/sitecrate/pack"
)

func main() {
	out := flag.String("o", "out.html", "output file")
	entry := flag.String("entry", "index.html", "entry page, relative to site-dir")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sitecrate <site-dir> [-o out.html] [-entry index.html]")
		flag.PrintDefaults()
	}
	flag.Parse()
	dir := flag.Arg(0)
	if flag.NArg() > 1 {
		// allow flags after <site-dir>
		flag.CommandLine.Parse(flag.Args()[1:])
	}
	if dir == "" {
		flag.Usage()
		os.Exit(2)
	}

	warnings, err := pack.Pack(dir, filepath.ToSlash(filepath.Clean(*entry)), *out)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sitecrate:", err)
		os.Exit(1)
	}
}
