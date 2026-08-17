package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeProbe stands in for the 25 handlers that decode a request body: it
// reports what the decoder saw, which is the only thing the limit can change.
func decodeProbe(t *testing.T, limit int64, bodyLen int) (decodeErr error, maxErr *http.MaxBytesError) {
	t.Helper()

	var seen error
	h := BodyLimit(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Text string `json:"text"`
		}
		seen = json.NewDecoder(r.Body).Decode(&payload)
	}))

	// A single JSON string long enough to cross the ceiling. The value must be
	// the large part, not the envelope, so that a body under the limit really
	// does parse.
	body := `{"text":"` + strings.Repeat("x", bodyLen) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	h.ServeHTTP(httptest.NewRecorder(), req)

	var m *http.MaxBytesError
	if errors.As(seen, &m) {
		return seen, m
	}
	return seen, nil
}

func TestBodyLimitRejectsOversizedBody(t *testing.T) {
	err, maxErr := decodeProbe(t, 1024, 4096)
	if err == nil {
		t.Fatal("a 4 KiB body under a 1 KiB limit must not decode cleanly")
	}
	if maxErr == nil {
		t.Fatalf("expected *http.MaxBytesError, got %T: %v", err, err)
	}
	if maxErr.Limit != 1024 {
		t.Errorf("reported limit = %d, want 1024", maxErr.Limit)
	}
}

// The other half of the claim. Without this, a BodyLimit of zero — or one that
// rejected everything — would pass the test above and look like a working
// control while breaking every request.
func TestBodyLimitPassesBodyUnderTheCeiling(t *testing.T) {
	err, _ := decodeProbe(t, 1024, 64)
	if err != nil {
		t.Fatalf("a 64-byte body under a 1 KiB limit must decode: %v", err)
	}
}

// GET carries http.NoBody; wrapping it would allocate for nothing, and the
// handler must still see an empty, readable body rather than a nil one.
func TestBodyLimitLeavesBodylessRequestsAlone(t *testing.T) {
	var panicked any
	h := BodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { panicked = recover() }()
		buf := make([]byte, 4)
		_, _ = r.Body.Read(buf)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if panicked != nil {
		t.Fatalf("reading the body of a GET panicked: %v", panicked)
	}
}

// The ceilings are ordered by exposure, and the ordering is the design: the
// public routes must be the tightest and import the widest. A later edit that
// makes the unauthenticated limit the largest would otherwise pass unnoticed.
func TestLimitsAreOrderedByExposure(t *testing.T) {
	if !(ServiceBodyLimit < PublicBodyLimit) {
		t.Errorf("service limit (%d) must be tighter than public (%d)", ServiceBodyLimit, PublicBodyLimit)
	}
	if !(PublicBodyLimit < AuthedBodyLimit) {
		t.Errorf("public limit (%d) must be tighter than authenticated (%d)", PublicBodyLimit, AuthedBodyLimit)
	}
	if !(AuthedBodyLimit < ImportBodyLimit) {
		t.Errorf("authenticated limit (%d) must be tighter than import (%d)", AuthedBodyLimit, ImportBodyLimit)
	}
}
