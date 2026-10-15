package sitecrate

import (
	"io/fs"
	"slices"
	"testing"
)

func TestRuntimeFilesListed(t *testing.T) {
	names, err := fs.Glob(runtimeFS, "runtime/*.js")
	if err != nil {
		t.Fatal(err)
	}
	for i := range names {
		names[i] = names[i][len("runtime/"):]
	}
	listed := slices.Sorted(slices.Values(runtimeFiles))
	if !slices.Equal(names, listed) {
		t.Errorf("runtime files = %v, runtimeFiles lists %v", names, listed)
	}
}
