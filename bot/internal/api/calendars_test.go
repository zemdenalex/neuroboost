package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 🔴 Field names are checked against api-go, not guessed. The plan for this
// task named three fields that do not exist (`is_personal`, `member_count`,
// and a `member_count` on the list); the API returns `kind`, `role` and
// `status` instead (api-go/internal/calendars/types.go:21). A guessed name
// decodes to the zero value in Go and fails silently — a shared calendar would
// simply look personal forever.
func TestCalendarsFullReadsKindRoleAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"id": "c1", "name": "Личный", "color": nil, "kind": "personal", "role": "owner", "status": "active"},
			map[string]any{"id": "c2", "name": "Семья", "color": "#22c55e", "kind": "shared", "role": "viewer", "status": "active"},
		}})
	}))
	t.Cleanup(srv.Close)

	cals, err := NewClient(srv.URL).CalendarsFull("jwt")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(cals) != 2 {
		t.Fatalf("got %d calendars", len(cals))
	}
	if !cals[0].IsPersonal() || cals[0].Role != "owner" {
		t.Errorf("personal calendar misread: %+v", cals[0])
	}
	if cals[1].IsPersonal() || cals[1].Role != "viewer" {
		t.Errorf("shared calendar misread: %+v", cals[1])
	}
	// A null colour must not crash or become the string "null".
	if cals[0].ColourOrEmpty() != "" || cals[1].ColourOrEmpty() != "#22c55e" {
		t.Errorf("colour misread: %q / %q", cals[0].ColourOrEmpty(), cals[1].ColourOrEmpty())
	}
}

func TestInviteByEmailSendsEmailAndRole(t *testing.T) {
	var got map[string]any
	var path, method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	t.Cleanup(srv.Close)

	if err := NewClient(srv.URL).InviteByEmail("jwt", "cal-1", "nastya@example.com", "editor"); err != nil {
		t.Fatalf("invite failed: %v", err)
	}
	if path != "/api/calendars/cal-1/invites" || method != http.MethodPost {
		t.Errorf("wrong call: %s %s", method, path)
	}
	if got["email"] != "nastya@example.com" || got["role"] != "editor" {
		t.Errorf("wrong payload: %+v", got)
	}
}

func TestInviteLinkReturnsTheToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"token": "Zm9vYmFyYmF6", "role": "editor"}})
	}))
	t.Cleanup(srv.Close)

	tok, err := NewClient(srv.URL).CalendarInviteLink("jwt", "cal-1", "editor")
	if err != nil || tok != "Zm9vYmFyYmF6" {
		t.Fatalf("token = %q, err = %v", tok, err)
	}
}

// 🔴 A PATCH carrying `name` when only the colour changed would rename the
// calendar to the empty string: the API's updateRequest holds *string for both
// (api-go/internal/calendars/handlers.go:20), so an absent key means "leave it"
// and a present empty one means "make it empty".
func TestUpdateCalendarSendsOnlyWhatChanged(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	t.Cleanup(srv.Close)

	colour := "#7c3aed"
	if err := NewClient(srv.URL).UpdateCalendar("jwt", "cal-1", nil, &colour); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if _, present := got["name"]; present {
		t.Errorf("a colour change sent `name` and would blank the title: %+v", got)
	}
	if got["color"] != "#7c3aed" {
		t.Errorf("colour not sent under the key the API reads: %+v", got)
	}

	got = nil
	name := "Работа"
	if err := NewClient(srv.URL).UpdateCalendar("jwt", "cal-1", &name, nil); err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if _, present := got["color"]; present {
		t.Errorf("a rename sent `color` and would repaint the calendar: %+v", got)
	}
}

func TestMembersReadNullableEmailAndName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"user_id": "u1", "email": "d@example.com", "display_name": "Denis", "role": "owner", "status": "active"},
			// 🔴 A Telegram-only account has NO email — this is exactly the
			// person the invite link exists for, and rendering their row must
			// not print "<nil>".
			map[string]any{"user_id": "u2", "email": nil, "display_name": nil, "role": "editor", "status": "invited"},
		}})
	}))
	t.Cleanup(srv.Close)

	ms, err := NewClient(srv.URL).CalendarMembers("jwt", "cal-1")
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}
	if ms[0].Label() != "Denis" {
		t.Errorf("named member rendered as %q", ms[0].Label())
	}
	if ms[1].Label() == "" || ms[1].Label() == "<nil>" {
		t.Errorf("a member with no email and no name rendered as %q", ms[1].Label())
	}
}
