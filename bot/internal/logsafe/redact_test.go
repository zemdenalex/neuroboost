package logsafe

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// The token shape this must never let through. Not a real one.
const fakeToken = "123456789:AAHfakeTokenValue_forTests-only"

// The exact failure observed in the audit: the client library returns a
// *url.Error on a refused connection, and its message carries the whole API
// URL. This is the case that mattered, so it is tested through the real type
// rather than a hand-written string.
func TestRedactsATokenInsideAURLError(t *testing.T) {
	urlErr := &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot" + fakeToken + "/getMe",
		Err: errors.New("dial tcp 149.154.167.220:443: connect: connection refused"),
	}

	got := Redact(urlErr)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("token survived redaction: %q", got)
	}
	if !strings.Contains(got, "bot<REDACTED>") {
		t.Errorf("expected the redaction marker, got %q", got)
	}
	// The rest of the message must survive — a redactor that ate the diagnostic
	// would trade one blindness for another.
	if !strings.Contains(got, "connection refused") {
		t.Errorf("the diagnostic was lost: %q", got)
	}
}

// A wrapped error is the common shape in this codebase (`fmt.Errorf("...: %w")`),
// and wrapping must not smuggle the token past the filter.
func TestRedactsThroughWrapping(t *testing.T) {
	inner := fmt.Errorf("api: %w", &url.Error{
		Op:  "Get",
		URL: "https://api.telegram.org/bot" + fakeToken + "/getUpdates",
		Err: errors.New("i/o timeout"),
	})
	outer := fmt.Errorf("notifier: send failed: %w", inner)

	if got := Redact(outer); strings.Contains(got, fakeToken) {
		t.Fatalf("token survived through two wraps: %q", got)
	}
}

// Multiple occurrences: ReplaceAll, not Replace-once.
func TestRedactsEveryOccurrence(t *testing.T) {
	msg := "retrying https://api.telegram.org/bot" + fakeToken +
		"/getMe after https://api.telegram.org/bot" + fakeToken + "/getUpdates"

	got := String(msg)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("a second occurrence survived: %q", got)
	}
	if n := strings.Count(got, "bot<REDACTED>"); n != 2 {
		t.Errorf("markers = %d, want 2", n)
	}
}

// The negative control. Without it a Redact that blanked everything — or one
// whose pattern matched far too much — would satisfy every test above.
func TestLeavesOrdinaryMessagesAlone(t *testing.T) {
	for _, msg := range []string{
		"notifier: send to 12345 failed: user blocked the bot",
		"reminder 9f3a:0001 acked",
		"listening on :3002",
		"chat 42: flow=new_event step=line",
	} {
		if got := String(msg); got != msg {
			t.Errorf("message was altered:\n  in:  %q\n  out: %q", msg, got)
		}
	}
}

// A logging helper that panics is worse than the leak it prevents.
func TestNilErrorDoesNotPanic(t *testing.T) {
	if got := Redact(nil); got != "<nil>" {
		t.Errorf("Redact(nil) = %q, want \"<nil>\"", got)
	}
}
