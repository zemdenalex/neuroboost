package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func signed(t *testing.T, secret, userID string, exp time.Time) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID:           userID,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)},
	})
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// OptionalJWTMiddleware is for endpoints anyone may call (feedback): a valid
// token attaches the user, anything else passes through anonymous. It never
// refuses a request: a feedback form must work for someone whose session died.
func TestOptionalJWTMiddleware(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"valid token attaches the user", "Bearer " + signed(t, testSecret, "u-1", time.Now().Add(time.Hour)), "u-1"},
		{"no header stays anonymous", "", ""},
		{"wrong secret stays anonymous", "Bearer " + signed(t, "other", "u-1", time.Now().Add(time.Hour)), ""},
		{"expired token stays anonymous", "Bearer " + signed(t, testSecret, "u-1", time.Now().Add(-time.Hour)), ""},
		{"not a bearer header stays anonymous", "Basic abc", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got string
			reached := false
			h := OptionalJWTMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
				got = UserIDFromContext(r.Context())
			}))
			req := httptest.NewRequest(http.MethodPost, "/api/feedback", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if !reached {
				t.Fatalf("request refused with %d; the optional middleware must always pass it on", rec.Code)
			}
			if got != c.want {
				t.Fatalf("user = %q, want %q", got, c.want)
			}
		})
	}
}

// The strict middleware keeps refusing what the optional one lets through.
func TestJWTMiddlewareStillRefusesABadToken(t *testing.T) {
	h := JWTMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("reached the handler with a bad token")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+signed(t, "other", "u-1", time.Now().Add(time.Hour)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
