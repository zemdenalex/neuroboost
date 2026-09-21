package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// 🔴 No handler puts a raw error into a chat message.
//
// This scan exists because the fix could not live at one call site: twenty
// handlers each did `h.t(chatID, "❌ Не получилось: ", …) + err.Error()`, and on
// 21.09 that printed Denis a JSON envelope, and an hour later the full text of
// a Go http timeout with a URL in it. Fixing the screen he happened to be on
// would have left nineteen loaded.
//
// Errors reach the reader through errorText, which maps the API's code to one
// sentence in their language and keeps the detail in the log.
func TestNoHandlerPrintsARawError(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "errors.go" {
			continue
		}
		src, rerr := os.ReadFile(f)
		if rerr != nil {
			t.Fatalf("read %s: %v", f, rerr)
		}
		scanned++
		for i, line := range strings.Split(string(src), "\n") {
			if !strings.Contains(line, ".Error()") {
				continue
			}
			// ⚠ Reading an error's text is fine — the bot inspects Telegram's
			// own wording to tell «message is not modified» from a real
			// failure. What is forbidden is that text ending up in a message,
			// which in this codebase always looks like concatenation or a
			// send. Narrowed after the first run flagged that inspection.
			userFacing := false
			for _, tell := range []string{"+", "Escape(", "sendText", "sendHTML", "editOrSend", "answerCallback"} {
				if strings.Contains(line, tell) {
					userFacing = true
				}
			}
			if userFacing {
				t.Errorf("%s:%d puts a raw error in front of a person:\n\t%s\n"+
					"use h.errorText(chatID, err)", f, i+1, strings.TrimSpace(line))
			}
		}
	}

	// 🔴 The floor. A glob that matched nothing, or a rename that emptied this
	// directory, would otherwise make the scan pass by checking nothing —
	// the exact shape of control this project has been caught by three times.
	if scanned < 10 {
		t.Fatalf("scanned only %d handler files; this scan is not looking at the package", scanned)
	}
}

// And the mapping itself answers in the reader's language, with no machinery in
// it. Without this, errorText could return "" for an unknown code and the chat
// would show a bare «❌».
func TestEveryErrorAnswerIsASentence(t *testing.T) {
	h, _ := newTestHandler(t)
	seen := map[string]bool{}
	for _, err := range []error{
		&api.Error{Status: 400, Code: "NOT_AN_OCCURRENCE", Message: "That day is not in the series"},
		&api.Error{Status: 400, Code: "NOT_RECURRING"},
		&api.Error{Status: 404, Code: "NOT_FOUND"},
		&api.Error{Status: 401, Code: "NOT_AUTHENTICATED"},
		&api.Error{Status: 400, Code: "VALIDATION_ERROR"},
		&api.Error{Status: 403, Code: "FORBIDDEN"},
		&api.Error{Status: 500, Code: "SOMETHING_WE_NEVER_HEARD_OF"},
		errCoded{code: "dial tcp 62.76.228.106:443: i/o timeout"},
	} {
		got := h.errorText(0, err)
		if got == "" {
			t.Errorf("%v produced no text at all", err)
			continue
		}
		if strings.ContainsAny(got, "{}<>") || strings.Contains(got, "http") {
			t.Errorf("%v produced machinery, not a sentence: %q", err, got)
		}
		seen[got] = true
	}

	// 🔴 Could this test have failed? Only this line makes it so. A mapping
	// that returned the same sentence for everything would satisfy every
	// assertion above — and would be exactly the bug of printing one useless
	// answer, just politely. Six named codes and a foreign error are eight
	// inputs; they must not collapse into one reply.
	if len(seen) < 5 {
		t.Errorf("errorText gave only %d distinct answers for 8 inputs — "+
			"the codes are not being read", len(seen))
	}

	if h.errorText(0, nil) != "" {
		t.Error("no error must produce no text, or handlers would print «❌ » on success")
	}
}

type errCoded struct{ code string }

func (e errCoded) Error() string { return e.code }
