package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Feedback from the bot carries the chat's token and says it came from the
// bot (Denis 25.09: attach the sender). The API keeps "bot" and "web" and
// stores anything else as "user".
func TestSubmitFeedbackSendsTheTokenAndSourceBot(t *testing.T) {
	var auth string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)

	if err := NewClient(srv.URL).SubmitFeedback("jwt", CreateFeedbackReq{Type: "bug", Title: "t", Description: "d"}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if auth != "Bearer jwt" {
		t.Errorf("Authorization = %q, want the chat's token", auth)
	}
	if body["source"] != "bot" {
		t.Errorf("source = %v, want bot", body["source"])
	}
}
