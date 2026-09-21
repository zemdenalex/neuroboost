package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The API wraps every answer in {"data": …} and c.do does not unwrap it. A
// client method that decodes the bare body reads zero values and reports
// success — exactly how OccurrenceResult failed on 21.09.
func TestEventToTaskReadsTheEnvelope(t *testing.T) {
	var path, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"task":{"title":"врач","due_date":"2026-10-20",` +
			`"estimated_minutes":70,"tags":[]},"lost":["start_time","color"],"event_id":null,"dry_run":true}}`))
	}))
	defer srv.Close()

	res, err := NewClient(srv.URL).EventToTask("tok", "e1:2026-10-20", ToTaskReq{Mode: "move", Repeat: "once", DryRun: true})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if path != "/api/events/e1:2026-10-20/to-task" {
		t.Errorf("path = %q", path)
	}
	if !strings.Contains(body, `"dry_run":true`) || !strings.Contains(body, `"repeat":"once"`) {
		t.Errorf("body = %s", body)
	}
	if res.Task.DueDate != "2026-10-20" || len(res.Lost) != 2 || res.Task.EstimatedMinutes == nil {
		t.Errorf("decoded %+v — the envelope was not read", res)
	}
}

func TestConvertTaskSendsTheChoiceAndReadsTheEvent(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"ev1","title":"банк","starts_at":"2026-10-20T09:00:00Z",` +
			`"ends_at":"2026-10-20T10:00:00Z","task_id":"t1"}}`))
	}))
	defer srv.Close()

	ev, err := NewClient(srv.URL).ConvertTask("tok", "t1", ConvertReq{
		Mode: "link", Repeat: "series", StartsAt: "2026-10-20T09:00:00Z", EndsAt: "2026-10-20T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(body, `"mode":"link"`) || !strings.Contains(body, `"repeat":"series"`) {
		t.Errorf("body = %s", body)
	}
	if ev.ID != "ev1" || ev.TaskID == nil || *ev.TaskID != "t1" {
		t.Errorf("decoded %+v", ev)
	}
}

// A repeating task sent without a choice must come back as the API's code, so
// the bot can ask — not as a generic failure.
func TestConvertTaskSurfacesRepeatChoiceRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"REPEAT_CHOICE_REQUIRED","message":"x"}}`))
	}))
	defer srv.Close()
	_, err := NewClient(srv.URL).ConvertTask("tok", "t1", ConvertReq{Mode: "link"})
	if CodeOf(err) != "REPEAT_CHOICE_REQUIRED" {
		t.Errorf("code = %q (err %v)", CodeOf(err), err)
	}
}
