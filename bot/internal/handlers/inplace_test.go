package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// 🔴 The rule: a button press re-renders the screen it was pressed on. It never
// posts a new message.
//
// Denis, 23.08: «Только с меню, назад, или создать. Сегодня, планирование,
// статистика, настройки и тд создают новые сообщения». Seven handlers did not
// take a messageID at all, so `editOrSend` could not have been reached from
// them — the plan that introduced in-place rendering named the NEW screens by
// hand and the old ones were never on the list.
//
// That is why this is a source scan and not a behavioural test. The defect is
// not a wrong value at runtime; it is a call site that never passes the
// argument. Per-screen tests cannot see it, because each one passes whatever
// its own test hands it — the omission lives in the dispatcher, one level up.
// The list of screens grows, and prose does not hold it. This does.

// screenCall matches a call to a screen-rendering method inside HandleCallback,
// capturing the method name and its argument list.
//
// `handle`/`show` prefixes only: `start…Flow` methods open a TEXT prompt, the
// user answers with a message of their own, and the conversation moves down the
// chat regardless — there is nothing to edit in place there.
var screenCall = regexp.MustCompile(`h\.((?:handle|show)[A-Za-z]*)\(([^\n]*)\)`)

// notAScreen is the one call in HandleCallback that renders nothing.
//
// handleNotificationAction is handed cb.Message whole, because it edits the
// reminder's own keyboard away rather than drawing a screen. Every other entry
// here would be a hole in the rule, so the map carries the reason, not just the
// name.
var notAScreen = map[string]string{
	"handleNotificationAction": "takes cb.Message itself — it clears the reminder's buttons, it does not render a screen",
}

func TestEveryCallbackScreenRendersInPlace(t *testing.T) {
	src, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("reading handler.go: %v", err)
	}

	body, ok := funcBody(string(src), "func (h *Handler) HandleCallback(")
	if !ok {
		t.Fatalf("could not find HandleCallback in handler.go — the scan is looking at nothing")
	}

	matches := screenCall.FindAllStringSubmatch(body, -1)
	// Positive control. A regex that stopped matching, or a body extraction
	// that returned a stub, would make every assertion below pass while
	// checking no code at all. The dispatcher had 24 such calls when this was
	// written; the floor is deliberately well under that so ordinary edits do
	// not trip it, and well over zero so an empty scan cannot pass.
	if len(matches) < 15 {
		t.Fatalf("found only %d screen calls in HandleCallback — the scan is not reading the dispatcher", len(matches))
	}

	for _, m := range matches {
		name, args := m[1], m[2]
		if reason, skip := notAScreen[name]; skip {
			t.Logf("skipping %s: %s", name, reason)
			continue
		}
		if !strings.Contains(args, "cb.Message.MessageID") {
			t.Errorf("h.%s(%s) does not pass cb.Message.MessageID — "+
				"the screen will be posted as a new message instead of replacing the one "+
				"the button was pressed on", name, args)
		}
	}
}

// funcBody returns the text between a function's opening brace and the closing
// brace in column 0 — the gofmt-guaranteed end of a top-level declaration.
func funcBody(src, signature string) (string, bool) {
	i := strings.Index(src, signature)
	if i < 0 {
		return "", false
	}
	rest := src[i:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end], true
	}
	return "", false
}

// TestFuncBodyStopsAtTheFunction guards the extractor itself.
//
// A funcBody that ran past its closing brace would scan the whole file, which
// happens to still pass today — and would quietly keep passing if HandleCallback
// were emptied out. An extractor is exactly the kind of helper whose failure
// makes the test above vacuously green.
func TestFuncBodyStopsAtTheFunction(t *testing.T) {
	src := "package p\n\nfunc (h *Handler) Target(x int) {\n\th.handleA(chatID, cb.Message.MessageID)\n}\n\nfunc Other() {\n\th.handleB(chatID)\n}\n"
	body, ok := funcBody(src, "func (h *Handler) Target(")
	if !ok {
		t.Fatal("funcBody did not find the target function")
	}
	if strings.Contains(body, "handleB") {
		t.Errorf("funcBody ran past the closing brace and picked up the next function:\n%s", body)
	}
	if !strings.Contains(body, "handleA") {
		t.Errorf("funcBody did not return the target function's own body:\n%s", body)
	}
}
