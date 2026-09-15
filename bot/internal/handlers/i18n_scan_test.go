package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 🔴 The rule: no user-facing text is written in one language only.
//
// Denis chose, 15.09: «Язык интерфейса бота — все сообщения и кнопки ru/en».
// Every such string goes through i18n.T(lang, ru, en), whose SIGNATURE is the
// enforcement — leaving out the English does not compile. What a signature
// cannot catch is a string that never reached T at all, and that is this scan.
//
// It is a source scan for the same reason inplace_test.go is one: the defect is
// not a wrong value at runtime. A Russian-only button is perfectly correct Go
// that renders fine for the person who wrote it, and is invisible until someone
// switches the language — which, here, nobody ever will, because the one user
// reads Russian. A test is the only reader this defect has.

// cyrillicLiteral matches a double-quoted string containing Cyrillic.
//
// ⚠ Escaped quotes inside a literal would confuse it. There are none in this
// package, and a raw-string literal (backticks) is checked separately below.
var cyrillicLiteral = regexp.MustCompile(`"[^"]*[А-Яа-яЁё][^"]*"`)

// scannedDirs are the packages that render to a human.
//
// `parse` is deliberately absent: its Russian is VOCABULARY — the words the
// bot recognises in what the user types — not text it shows. Those words stay
// in both languages at once regardless of the interface setting, and running
// them through T would silently stop «среда» working for someone reading the
// English interface.
var scannedDirs = []string{".", "../keyboards", "../notifier"}

func TestNoUntranslatedUserFacingText(t *testing.T) {
	scanned := 0
	for _, dir := range scannedDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(dir, name)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			scanned++
			checkFile(t, path, string(src))
		}
	}
	if scanned == 0 {
		t.Fatal("no files were scanned — this test is checking nothing")
	}
}

func checkFile(t *testing.T, path, src string) {
	t.Helper()
	lines := strings.Split(src, "\n")
	inRawString := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// A raw string literal spanning lines — the long guides.
		//
		// ⚠ The rule here is per FUNCTION, not per line, and that is not a
		// loosening. A guide is two raw strings handed to one T call, and the
		// second one's opening backtick is twenty lines below that call — a
		// line-window rule would fail it forever, and a rule that fails
		// correct code is a rule people delete.
		if strings.Count(line, "`")%2 == 1 {
			inRawString = !inRawString
			if inRawString && !enclosingFuncHasT(lines, i) {
				t.Errorf("%s:%d: a multi-line text block sits in a function that never calls i18n.T", path, i+1)
			}
			continue
		}
		if inRawString || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !cyrillicLiteral.MatchString(line) {
			continue
		}
		if nearbyHasT(lines, i) {
			continue
		}
		if inPairedTable(lines, i) {
			continue
		}
		t.Errorf("%s:%d: Russian text outside i18n.T — %s", path, i+1, trimmed)
	}
}

// nearbyHasT looks for the T call on this line or the three above it.
//
// ⚠ An approximation, and a deliberate one: a precise answer means parsing Go,
// and the failure mode of the approximation is a false PASS on an oddly
// formatted call — never a false failure that would push someone to silence
// the scan. gofmt keeps the call and its arguments within this window.
func nearbyHasT(lines []string, i int) bool {
	// On the offending line itself, the call must come BEFORE the text.
	//
	// 🔴 Without this the check was far weaker than it looked: any line that
	// mentioned T anywhere passed, so
	//
	//	text += "Русский" + h.t(chatID, "Русский", "English")
	//
	// went through untouched. A sabotage run found it — the first version of
	// this function passed that line, which is exactly the kind of "control
	// that cannot fail" this repository keeps a rule about.
	if at := callAt(lines[i]); at >= 0 {
		return at < firstCyrillicLiteral(lines[i])
	}
	for j := i - 1; j >= 0 && j > i-4; j-- {
		if callAt(lines[j]) >= 0 {
			return true
		}
	}
	return false
}

func callAt(line string) int {
	if at := strings.Index(line, "i18n.T("); at >= 0 {
		return at
	}
	return strings.Index(line, "h.t(")
}

// firstCyrillicLiteral is where the first Russian-bearing literal starts.
func firstCyrillicLiteral(line string) int {
	loc := cyrillicLiteral.FindStringIndex(line)
	if loc == nil {
		return len(line)
	}
	return loc[0]
}

// enclosingFuncHasT reports whether the declaration containing line i mentions
// i18n.T anywhere.
//
// Top-level declarations in gofmt'd Go start at column zero, so the boundaries
// are unambiguous: walk back to the nearest `func ` / `var ` / `const ` at the
// left margin, forward to the next one, and look between them.
func enclosingFuncHasT(lines []string, i int) bool {
	atMargin := func(s string) bool {
		return strings.HasPrefix(s, "func ") || strings.HasPrefix(s, "var ") ||
			strings.HasPrefix(s, "const ")
	}

	start := 0
	for j := i; j >= 0; j-- {
		if atMargin(lines[j]) {
			start = j
			break
		}
	}
	end := len(lines)
	for j := i + 1; j < len(lines); j++ {
		if atMargin(lines[j]) {
			end = j
			break
		}
	}

	for j := start; j < end; j++ {
		if strings.Contains(lines[j], "i18n.T(") || strings.Contains(lines[j], "h.t(") {
			return true
		}
	}
	return false
}

// pairedTables are the three declarations that hold both languages as DATA —
// two parallel arrays, or a struct with an RU and an EN field — and hand them
// to i18n.T somewhere else in the same declaration.
//
// 🔴 A closed, named list, not a pattern. Each entry is here because a
// line-by-line scan cannot see a pairing that lives across lines; every one of
// them is checked below for still existing, so a rename fails this test rather
// than silently widening the exemption.
var pairedTables = []string{
	"func weekdayName(",     // ru[...] / en[...], joined by i18n.T
	"func monthGenitive(",   // the same shape, for months
	"var menuRows =",        // menuEntrance{RU, EN, Screen}; menuEntrance.label calls i18n.T
	"func monthNominative(", // ru[...] / en[...], joined by i18n.T
	"const bilingualTooOld", // answered before there is a session to read a language from
	"func Keyboard(",        // the notifier has no session to read a language from — see its comment
}

func inPairedTable(lines []string, i int) bool {
	for j := i; j >= 0; j-- {
		line := lines[j]
		if !strings.HasPrefix(line, "func ") && !strings.HasPrefix(line, "var ") &&
			!strings.HasPrefix(line, "const ") {
			continue
		}
		for _, allowed := range pairedTables {
			if strings.HasPrefix(line, allowed) {
				return true
			}
		}
		return false
	}
	return false
}

// A stale exemption is worse than none: it silently permits whatever moves
// into the name later.
func TestEveryPairedTableExemptionStillExists(t *testing.T) {
	found := map[string]bool{}
	for _, dir := range scannedDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			src, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, allowed := range pairedTables {
				if strings.Contains(string(src), allowed) {
					found[allowed] = true
				}
			}
		}
	}
	for _, allowed := range pairedTables {
		if !found[allowed] {
			t.Errorf("exemption %q names a declaration that no longer exists — "+
				"delete it, or it will quietly exempt whatever takes the name next", allowed)
		}
	}
}
