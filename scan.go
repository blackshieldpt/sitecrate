package sitecrate

import (
	"fmt"
	"regexp"
)

// unsupported lists constructs that may not work in the packed output
var unsupported = []struct {
	re   *regexp.Regexp
	what string
}{
	{regexp.MustCompile(`\bXMLHttpRequest\b`), "XMLHttpRequest"},
	{regexp.MustCompile(`\b(?:import|from)\s*\(?\s*["']\.{0,2}/`), "relative module imports"},
	{regexp.MustCompile(`(?i)<form\b[^>]*\bmethod\s*=\s*["']?post`), "POST forms"},
}

// scanner records the files that use each construct of unsupported
type scanner [][]string

func newScanner() scanner {
	return make(scanner, len(unsupported))
}

// scan checks the file at path, of MIME type typ
func (s scanner) scan(path, typ string, data []byte) {
	if typ != "text/html" && typ != "text/javascript" {
		return
	}
	for i, u := range unsupported {
		if u.re.Match(data) {
			s[i] = append(s[i], path)
		}
	}
}

// warnings returns one warning per construct found
func (s scanner) warnings() []string {
	var warnings []string
	for i, u := range unsupported {
		if n := len(s[i]); n > 0 {
			warnings = append(warnings, fmt.Sprintf("%s in %d file(s), e.g. %s: may not work in packed output", u.what, n, s[i][0]))
		}
	}
	return warnings
}
