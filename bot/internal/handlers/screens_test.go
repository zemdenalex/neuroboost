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
			// The amendment's claim that the reply-keyboard labels "appear as
			// button text only in the keyboards package" is false: a Go bot
			// receives a reply-keyboard press by switching on the identical
			// literal (`case "🏠 Меню":` in handler.go), because that IS how
			// go-telegram-bot-api delivers the tap. That is a dispatch site,
			// not a screen pointing the user at a button — the two read
			// identically to a regex, so the line filter has to tell them
			// apart. Narrowing the regex instead would also blind it to a
			// `case` that names a button inside a *sent* string, which is
			// not the failure mode here.
			if strings.HasPrefix(strings.TrimSpace(line), `case "`) {
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
