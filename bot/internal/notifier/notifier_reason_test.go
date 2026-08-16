package notifier

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

// The reason string leaves this process: it is POSTed to
// /api/svc/notifications/{id}/ack and written to the API's structured log. The
// API module has no redaction of its own, so anything unredacted here becomes a
// secret in a second service's logs.
//
// 🔴 This is the test that would have caught the real defect. Between 14.08 and
// 16.08 the log line beside this value was redacted and the value itself was
// not — the fix looked done because the visible half of it was.
func TestDeliveryReasonRedactsTheToken(t *testing.T) {
	// Shaped like what the Telegram client actually returns on a transport
	// failure: a *url.Error whose message contains the full API URL.
	sendErr := &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/bot123456789:AAHfake-Token_Value/sendMessage",
		Err: errors.New("dial tcp 149.154.167.220:443: connect: connection refused"),
	}

	reason := deliveryReason(sendErr)

	if strings.Contains(reason, "AAHfake-Token_Value") {
		t.Fatalf("the bot token is in the string sent to the API: %q", reason)
	}
	if !strings.Contains(reason, "bot<REDACTED>") {
		t.Errorf("expected the token to be replaced by the redaction marker, got %q", reason)
	}
	// The reason must still be diagnosable — a redaction that threw the whole
	// message away would make every delivery failure read the same.
	if !strings.Contains(reason, "connection refused") {
		t.Errorf("redaction removed the actual cause; the message is now useless: %q", reason)
	}
}

func TestDeliveryReasonSurvivesNil(t *testing.T) {
	// Called on a logging path; a helper that panics is worse than the leak.
	if got := deliveryReason(nil); got == "" {
		t.Error("a nil error should still produce something printable")
	}
}
