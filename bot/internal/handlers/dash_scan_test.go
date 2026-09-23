package handlers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// 🔴 No sentence the bot sends carries an em-dash (Denis's rule for client
// text). The D2 review found three on the day screens; a sweep on 23.09 found
// about a hundred more across the bot, so the rule is held here rather than by
// whoever next remembers it.
//
// Parsed, not grepped: only string literals count, so comments may keep their
// dashes. release/ is not scanned: past release notes are a record of what was
// sent.
func TestNoSentenceCarriesAnEmDash(t *testing.T) {
	// A lone «—» marks an empty value, «— — —» is the dash priority style
	// itself, and the trim set cuts a dash off a shortened title.
	allowed := map[string]bool{"—": true, " ,;:—-": true}
	literals := 0
	for _, dir := range []string{".", "../notifier", "../keyboards", "../format"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, f, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", f, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				literals++
				s, err := strconv.Unquote(lit.Value)
				if err != nil || allowed[s] || strings.Contains(s, "— — —") {
					return true
				}
				// A dash opening a line is the «dash» priority symbol in its
				// own preview (prio.go), not punctuation.
				lines := strings.Split(s, "\n")
				for i, l := range lines {
					lines[i] = strings.TrimPrefix(l, "— ")
				}
				if !strings.Contains(strings.Join(lines, "\n"), "—") {
					return true
				}
				t.Errorf("%s: an em-dash in %q; use a comma, a colon or a full stop",
					fset.Position(lit.Pos()), s)
				return true
			})
		}
	}
	// The floor: a glob that matched nothing would pass by checking nothing.
	if literals < 1000 {
		t.Fatalf("scanned only %d string literals; the scan is not looking at the bot", literals)
	}
}
