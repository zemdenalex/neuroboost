package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The notifier's two calls are the only ones on this client that use a service
// token instead of a user JWT, and the only ones whose path was written by
// hand rather than copied from a neighbour. API_BASE is "http://api:8080"
// with no /api suffix — every other method spells the prefix out, and these
// must too or they 404 against a live API that looks perfectly healthy.

func TestPendingNotificationsHitsTheRightPathWithTheToken(t *testing.T) {
	var gotPath, gotToken, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Service-Token")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":[{"id":"abc","tg_id":42,"text":"hi","source_kind":"EVENT","source_id":"e1"}]}`)
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL).PendingNotifications("tok")
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if gotPath != "/api/svc/notifications/pending" {
		t.Errorf("path = %q, want /api/svc/notifications/pending", gotPath)
	}
	if gotToken != "tok" {
		t.Errorf("X-Service-Token = %q, want tok", gotToken)
	}
	if gotAuth != "" {
		t.Errorf("a user JWT leaked onto a service call: %q", gotAuth)
	}
	if len(got) != 1 || got[0].TgID != 42 || got[0].Text != "hi" {
		t.Errorf("payload not unwrapped from the data envelope: %+v", got)
	}
}

func TestPendingNotificationsEmptyEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[]}`)
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL).PendingNotifications("tok")
	if err != nil {
		t.Fatalf("empty pull errored: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want nothing, got %+v", got)
	}
}

func TestPendingNotificationsSurfacesAuthFailure(t *testing.T) {
	// A 401 must be an error, not an empty list — otherwise a wrong token
	// looks exactly like "no reminders due" and nothing is ever delivered.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"code":"UNAUTHORIZED"}}`)
	}))
	defer srv.Close()

	if _, err := NewClient(srv.URL).PendingNotifications("bad"); err == nil {
		t.Error("401 was swallowed")
	}
}

func TestAckNotificationPostsDeliveryOutcome(t *testing.T) {
	var gotPath, gotToken string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Service-Token")
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"data":{"ok":true}}`)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).AckNotification("tok", "abc", false, "blocked"); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
	if gotPath != "/api/svc/notifications/abc/ack" {
		t.Errorf("path = %q", gotPath)
	}
	if gotToken != "tok" {
		t.Errorf("X-Service-Token = %q", gotToken)
	}
	if body["delivered"] != false {
		t.Errorf("delivered = %v, want false", body["delivered"])
	}
	if body["error"] != "blocked" {
		t.Errorf("error = %v, want blocked", body["error"])
	}
}

func TestAckNotificationHandlesEmptyResponseBody(t *testing.T) {
	// do() is called with a nil result here; a 204 or empty body must not be
	// treated as a decode failure.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).AckNotification("tok", "abc", true, ""); err != nil {
		t.Errorf("empty ack response errored: %v", err)
	}
}
