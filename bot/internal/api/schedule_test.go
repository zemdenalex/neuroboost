package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ScheduleTask existed on this client before anything called it, and it sent
// the wrong body: {"start_time", "estimated_minutes"} against an endpoint that
// decodes ScheduleTaskRequest{starts_at, ends_at, all_day}. Nothing noticed,
// because a method with no callers has no failing user — the same shape as the
// web task wrappers that read a `.task` key the API never sends (fixed 14.08).
//
// The endpoint would not have errored either: encoding/json ignores unknown
// keys, so starts_at would parse as the zero string, and the handler's own
// time parse is the only thing between that and a task scheduled at year zero.
func TestScheduleTaskSendsTheBodyTheAPIActuallyDecodes(t *testing.T) {
	var gotPath, gotAuth string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body was not JSON: %v", err)
		}
		io.WriteString(w, `{"data":{"id":"ev1"}}`)
	}))
	defer srv.Close()

	err := NewClient(srv.URL).ScheduleTask("jwt", "task-1",
		"2026-08-19T06:00:00Z", "2026-08-19T06:30:00Z")
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}

	if gotPath != "/api/tasks/task-1/schedule" {
		t.Errorf("path = %q, want /api/tasks/task-1/schedule", gotPath)
	}
	if gotAuth != "Bearer jwt" {
		t.Errorf("Authorization = %q, want Bearer jwt", gotAuth)
	}

	// The three fields ScheduleTaskRequest declares (api-go/internal/tasks/
	// types.go:128). all_day must be present and false: omitting it is harmless
	// today only because Go's zero value agrees, and that is not a contract.
	if body["starts_at"] != "2026-08-19T06:00:00Z" {
		t.Errorf("starts_at = %v", body["starts_at"])
	}
	if body["ends_at"] != "2026-08-19T06:30:00Z" {
		t.Errorf("ends_at = %v", body["ends_at"])
	}
	if v, ok := body["all_day"]; !ok || v != false {
		t.Errorf("all_day = %v (present: %v), want false", v, ok)
	}

	// The old keys, named so a regression says what it is rather than only
	// which assertion moved.
	for _, dead := range []string{"start_time", "estimated_minutes"} {
		if _, ok := body[dead]; ok {
			t.Errorf("%q is still on the wire — the API does not decode it", dead)
		}
	}
}

func TestScheduleTaskSurfacesRejection(t *testing.T) {
	// A viewer on a shared calendar gets 403 here. Swallowing it would tell the
	// user their task was scheduled while the calendar stayed empty.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":{"code":"FORBIDDEN","message":"read-only"}}`)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).ScheduleTask("jwt", "t", "a", "b"); err == nil {
		t.Error("403 was swallowed")
	}
}
