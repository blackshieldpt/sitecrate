package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		args func(out string) []string
		code int
		out  bool // whether the output file is written
	}{
		{func(out string) []string { return nil }, 2, false},
		{func(out string) []string { return []string{"-x"} }, 2, false},
		{func(out string) []string { return []string{"-h"} }, 0, false},
		{func(out string) []string { return []string{"../../testdata/site", "-o", out} }, 0, true},
		{func(out string) []string { return []string{"-o", out, "../../testdata/site"} }, 0, true},
		{func(out string) []string {
			return []string{"../../testdata/site", "-o", out, "-entry", "./about/index.html"}
		}, 0, true},
		{func(out string) []string { return []string{"../../testdata/site", "-o", out, "extra"} }, 2, false},
		{func(out string) []string { return []string{"../../testdata/site", "extra", "-o", out} }, 2, false},
		{func(out string) []string { return []string{"../../testdata/site", "-o", out, "-entry", "nope.html"} }, 1, false},
		{func(out string) []string { return []string{"no/such/dir", "-o", out} }, 1, false},
		{func(out string) []string { return []string{"../../testdata/site", "-follow-symlinks", "-o", out} }, 0, true},
	}
	for _, tt := range tests {
		out := filepath.Join(t.TempDir(), "out.html")
		args := tt.args(out)
		if code := run(args, io.Discard, io.Discard); code != tt.code {
			t.Errorf("%q: exit status %d, want %d", args, code, tt.code)
		}
		if _, err := os.Stat(out); (err == nil) != tt.out {
			t.Errorf("%q: output written = %v, want %v", args, err == nil, tt.out)
		}
	}
}

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	if code := run([]string{"-version"}, &stdout, io.Discard); code != 0 || stdout.String() != "sitecrate "+version+"\n" {
		t.Errorf("exit status %d, output %q", code, stdout.String())
	}
}
