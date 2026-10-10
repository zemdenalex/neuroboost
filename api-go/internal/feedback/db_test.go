package feedback

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

const secret = "feedback-test-secret"

// feedbackDB is a real database with one user, and the public route wired the
// way main.go wires it: optional JWT in front of Create.
func feedbackDB(t *testing.T) (*database.DB, context.Context, string, http.Handler) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	ctx := context.Background()
	var id string
	email := fmt.Sprintf("feedback-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx, `INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM feedback WHERE title LIKE 'fb-test %'`)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id)
	})
	r := chi.NewRouter()
	r.With(middleware.OptionalJWTMiddleware(secret)).Post("/api/feedback", NewHandler(d).Create)
	return d, ctx, id, r
}

func tokenFor(t *testing.T, userID string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &middleware.Claims{
		UserID:           userID,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func post(t *testing.T, h http.Handler, body, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// Denis, 25.09: feedback carries its sender when the sender is logged in.
// Until then every row had user_id NULL: the route is public and nothing read
// the token, even though the bot sent one.
func TestFeedbackKeepsTheSenderWhenLoggedIn(t *testing.T) {
	d, ctx, userID, h := feedbackDB(t)

	if code := post(t, h, `{"type":"bug","title":"fb-test logged in","description":"x","source":"bot"}`, tokenFor(t, userID)); code != http.StatusCreated {
		t.Fatalf("logged-in create: %d", code)
	}
	var got *string
	var source string
	if err := d.Pool.QueryRow(ctx, `SELECT user_id::text, source FROM feedback WHERE title = 'fb-test logged in'`).Scan(&got, &source); err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != userID {
		t.Fatalf("user_id = %v, want %s", got, userID)
	}
	if source != "bot" {
		t.Fatalf("source = %q, want bot", source)
	}
}

func TestFeedbackStaysOpenToAnonymousAndBadTokens(t *testing.T) {
	d, ctx, _, h := feedbackDB(t)

	if code := post(t, h, `{"type":"idea","title":"fb-test anonymous","description":"x"}`, ""); code != http.StatusCreated {
		t.Fatalf("anonymous create: %d", code)
	}
	// An expired session must not lose the feedback: stored, just anonymous.
	if code := post(t, h, `{"type":"idea","title":"fb-test bad token","description":"x","source":"admin"}`, "not-a-token"); code != http.StatusCreated {
		t.Fatalf("bad-token create: %d", code)
	}
	for _, title := range []string{"fb-test anonymous", "fb-test bad token"} {
		var got *string
		var source string
		if err := d.Pool.QueryRow(ctx, `SELECT user_id::text, source FROM feedback WHERE title = $1`, title).Scan(&got, &source); err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("%s: user_id = %s, want NULL", title, *got)
		}
		// Only web and bot are accepted; anything else is the old "user".
		if source != "user" {
			t.Fatalf("%s: source = %q, want user", title, source)
		}
	}
}
