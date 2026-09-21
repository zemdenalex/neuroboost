package api

import (
	"strings"
	"testing"
)

// The exact body Denis's chat printed on 21.09, verbatim.
const occurrenceRefusal = `{"error":{"code":"NOT_AN_OCCURRENCE","message":"That day is not in the series"}}`

func TestTheEnvelopeBecomesACodeAndASentence(t *testing.T) {
	err := errorFromBody(400, []byte(occurrenceRefusal))

	if got := CodeOf(err); got != "NOT_AN_OCCURRENCE" {
		t.Errorf("code = %q, want NOT_AN_OCCURRENCE", got)
	}
	if got := err.Error(); got != "That day is not in the series" {
		t.Errorf("text = %q, want the API's sentence", got)
	}
	if strings.ContainsAny(err.Error(), "{}") {
		t.Errorf("JSON reached the user-facing text: %q", err.Error())
	}
}

// 🔴 A body that is NOT our envelope must not become the message. During an
// outage the thing on port 443 is a proxy, and it answers in HTML — a page of
// markup inside a Telegram bubble is a worse answer than the status alone.
func TestAPageOfHTMLNeverBecomesTheMessage(t *testing.T) {
	err := errorFromBody(502, []byte("<html><head><title>502 Bad Gateway</title></head></html>"))

	if CodeOf(err) != "" {
		t.Errorf("code = %q, want empty: that body carried no code of ours", CodeOf(err))
	}
	if strings.Contains(err.Error(), "<html>") {
		t.Errorf("markup reached the text: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("text = %q, want it to at least name the status", err.Error())
	}
}

func TestAnEmptyBodyStillSaysSomething(t *testing.T) {
	if got := errorFromBody(500, nil).Error(); got == "" {
		t.Error("an error with no text at all reaches the chat as «❌ » and nothing else")
	}
}

// CodeOf must answer "" for errors that never came from the API — a timeout,
// a DNS failure — or every handler branching on it would treat an outage as a
// validation problem.
func TestCodeOfAForeignErrorIsEmpty(t *testing.T) {
	if got := CodeOf(errNotOurs{}); got != "" {
		t.Errorf("code = %q, want empty", got)
	}
}

type errNotOurs struct{}

func (errNotOurs) Error() string { return "dial tcp 62.76.228.106:443: i/o timeout" }
