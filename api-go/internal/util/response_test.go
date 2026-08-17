package util

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The whole point of RespondDecodeError: json.Decode reports "body was capped"
// and "body was malformed" identically at the call site, and answering both
// with 400 is how a too-large import gets reported as broken JSON.
func TestRespondDecodeErrorAnswers413ForAnOversizedBody(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondDecodeError(rec, &http.MaxBytesError{Limit: 1024})

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not the standard envelope: %v", err)
	}
	if body.Error == nil || body.Error.Code != "BODY_TOO_LARGE" {
		t.Fatalf("error code = %+v, want BODY_TOO_LARGE", body.Error)
	}
	// The ceiling belongs in the message: "too large" without a number leaves
	// the caller guessing what would fit.
	if !contains(body.Error.Message, "1024") {
		t.Errorf("message %q does not name the limit", body.Error.Message)
	}
}

// A wrapped MaxBytesError must still be recognised — handlers routinely wrap
// decode failures with fmt.Errorf before responding.
func TestRespondDecodeErrorUnwrapsBeforeDeciding(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondDecodeError(rec, errors.Join(errors.New("decoding request"), &http.MaxBytesError{Limit: 64}))

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 for a wrapped MaxBytesError", rec.Code)
	}
}

// The negative half: genuinely malformed JSON must keep its 400, or the new
// branch would relabel every bad request as too large.
func TestRespondDecodeErrorKeeps400ForMalformedJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondDecodeError(rec, errors.New("invalid character 'x' looking for beginning of value"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not the standard envelope: %v", err)
	}
	if body.Error == nil || body.Error.Code != "INVALID_BODY" {
		t.Fatalf("error code = %+v, want INVALID_BODY", body.Error)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(needle) == 0 ||
		indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
