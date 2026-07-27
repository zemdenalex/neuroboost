package reminders

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestServiceTokenAcceptsCorrectToken(t *testing.T) {
	h := ServiceTokenMiddleware("s3cr3t")(okHandler())
	req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
	req.Header.Set("X-Service-Token", "s3cr3t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("correct token rejected: %d", rec.Code)
	}
}

func TestServiceTokenRejectsWrongAndMissing(t *testing.T) {
	h := ServiceTokenMiddleware("s3cr3t")(okHandler())
	for _, tc := range []struct{ name, token string }{
		{"wrong", "nope"},
		{"missing", ""},
		{"prefix of the real token", "s3c"},
		{"real token plus suffix", "s3cr3tX"},
	} {
		req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
		if tc.token != "" {
			req.Header.Set("X-Service-Token", tc.token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", tc.name, rec.Code)
		}
	}
}

func TestServiceEndpointsDisabledWhenTokenUnset(t *testing.T) {
	// An empty configured token must not mean "anything matches", and must not
	// mean "an empty header matches" either. Unset == closed, loudly.
	h := ServiceTokenMiddleware("")(okHandler())
	for _, presented := range []string{"", "anything"} {
		req := httptest.NewRequest("GET", "/api/svc/notifications/pending", nil)
		if presented != "" {
			req.Header.Set("X-Service-Token", presented)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("unset token, presented %q: got %d, want 503", presented, rec.Code)
		}
	}
}

func TestRateLimiterFixedWindow(t *testing.T) {
	// The legitimate caller is one bot polling once a minute, so the limit can
	// be tight; anything near it is a bug or an attack.
	rl := newRateLimiter(3, time.Minute)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if !rl.allow(base) {
			t.Fatalf("request %d denied inside the limit", i+1)
		}
	}
	if rl.allow(base) {
		t.Error("4th request in the window should be denied")
	}
	// A new window resets the counter.
	if !rl.allow(base.Add(time.Minute)) {
		t.Error("next window should start fresh")
	}
}
