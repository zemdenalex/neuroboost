package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 🔴 The rule: no screen tells the user to go press a reply button. If an
// action is available from a screen, that screen carries a BUTTON for it.
//
// This is a source scan rather than a behavioural test on purpose. The prose
// lives in string literals scattered across handlers, there is no single place
// to intercept it, and the failure mode is a sentence — not a state. A future
// edit that reintroduces "Use ➕ New Task" is exactly what this must catch.
// AMENDED BY CONTROLLER RULING - see the ledger. The plan's pattern was
// English-only, and every string this bot sends is Russian: it could not have
// caught the violation it exists to catch, written the way it will be written.
// The Russian half is the reply-keyboard labels - naming one inside a SENT
// string is exactly the defect. Those labels appear as button text only in the
// keyboards package, which this scan does not read, so there is no false match.
var replyButtonProse = regexp.MustCompile(
	`(?i)(use\s+the\s+menu|use\s+/start|menu\s+below|` +
		`➕\s*New\s+Task|📅\s*New\s+Event|📝\s*Note\b|` +
		`🏠\s*Меню|🗓\s*Календарь|📅\s*События|📋\s*Задачи|➕\s*Создать|⚙️\s*Настройки)`)

// dispatchOnlyClause matches a line that IS a reply-keyboard dispatch clause
// and nothing else: optional leading whitespace, `case`, one or more quoted
// labels separated by commas, a colon, and nothing after it but whitespace.
//
// Anchored at both ends on purpose. A prefix check on `case "` would skip
// the whole physical line unconditionally, including a line that also
// carries a statement after the colon — `case "x": h.sendText(chatID, "жми
// ➕ Создать")` is valid Go and not flagged by gofmt (this repo's CI does not
// run `gofmt -l`), so a prefix check would silently exempt exactly the
// regression class this test exists to catch. Requiring the colon to be the
// last non-whitespace character closes that gap.
var dispatchOnlyClause = regexp.MustCompile(`^\s*case\s+"[^"]*"(?:\s*,\s*"[^"]*")*\s*:\s*$`)

// isDispatchOnlyLine tells a reply-keyboard dispatch clause apart from a
// screen that points the user at a reply button.
//
// A Go bot receives a reply-keyboard press by switching on the identical
// literal (`case "🏠 Меню":` in handler.go) — that literal has to live in
// internal/handlers, which this scan reads, so the amendment's claim that
// the labels "appear as button text only in the keyboards package" does
// not hold. This function is what tells the receiver apart from a sender.
func isDispatchOnlyLine(line string) bool {
	return dispatchOnlyClause.MatchString(line)
}

func TestNoScreenPointsAtAReplyButton(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("globbing handlers: %v", err)
	}
	if len(files) < 5 {
		// Positive control: if the glob silently matched nothing, every
		// assertion below would pass while checking no code at all.
		t.Fatalf("found %d handler files — the scan is not looking at the package", len(files))
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !strings.Contains(line, `"`) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if isDispatchOnlyLine(line) {
				continue
			}
			if m := replyButtonProse.FindString(line); m != "" {
				t.Errorf("%s:%d points the user at a reply button (%q). "+
					"Give the screen a button instead:\n\t%s",
					f, i+1, m, strings.TrimSpace(line))
			}
		}
	}
}

// TestDispatchLineFilterCatchesInlineStatement guards the shape of
// isDispatchOnlyLine itself, not just the files in this package today.
//
// A prefix check on "case \"" skips the WHOLE physical line unconditionally.
// That is correct for a pure dispatch clause, but a case clause that also
// carries a statement after the colon — `case "x": h.sendText(chatID, "жми
// ➕ Создать")` — is valid Go, is not caught by gofmt (this repo's CI does
// not run `gofmt -l`), and would be silently exempted from the scan by a
// prefix check alone. The filter must only skip a line that IS a dispatch
// clause and nothing else.
func TestDispatchLineFilterCatchesInlineStatement(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		wantSkip bool
	}{
		{
			name:     "pure dispatch, single label",
			line:     `	case "🏠 Меню":`,
			wantSkip: true,
		},
		{
			name:     "pure dispatch, multiple labels",
			line:     `	case "🏠 Меню", "🗓 Календарь":`,
			wantSkip: true,
		},
		{
			name:     "dispatch clause with a statement after the colon is a real violation",
			line:     `	case "x": h.sendText(chatID, "жми ➕ Создать")`,
			wantSkip: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isDispatchOnlyLine(c.line)
			if got != c.wantSkip {
				t.Errorf("isDispatchOnlyLine(%q) = %v, want %v", c.line, got, c.wantSkip)
			}
			if !c.wantSkip {
				// The line the filter must NOT skip is also a real violation
				// under the scan's own regex — confirm the regex still sees
				// it once the filter lets it through.
				if m := replyButtonProse.FindString(c.line); m == "" {
					t.Errorf("fixture line %q does not even match replyButtonProse — the fixture is not testing what it claims to", c.line)
				}
			}
		})
	}
}
