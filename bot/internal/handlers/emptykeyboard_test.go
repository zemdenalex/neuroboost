package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 🔴 NewInlineKeyboardMarkup() with no rows is a keyboard Telegram refuses. Use
// keyboards.None(). Held by scan, because the fake Telegram only catches the
// screens a test happens to open.
func TestNoEmptyKeyboardConstructor(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	kb, _ := filepath.Glob("../keyboards/*.go")
	for _, f := range append(files, kb...) {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "NewInlineKeyboardMarkup()") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				t.Errorf("%s:%d builds an empty keyboard Telegram refuses — use keyboards.None()", f, i+1)
			}
		}
	}
}
