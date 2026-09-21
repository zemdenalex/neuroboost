package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 🔴 The bot must read the day out of the envelope the API actually sends.
//
// `c.do` decodes the body exactly as handed to it and unwraps nothing — every
// other call in client.go declares its own `Data` field. MarkOccurrence's first
// version did not, so it would have decoded to a zero value on every success.
// Nothing would have failed: the caller would simply have found Occurrence == ""
// and printed «✅ Сделано на сегодня» — the sentence that started this whole
// change — on a task whose series starts tomorrow.
//
// A test against a dead port cannot see that. This one answers with the real
// shape `util.RespondJSON` produces.
func TestTheClosedDayIsReadOutOfTheDataEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"occurrence":"2026-09-22","state":"done"}}`))
	}))
	defer srv.Close()

	res, err := NewClient(srv.URL).MarkOccurrence("t0ken", "task-1", "done", 0)
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	if res.Occurrence != "2026-09-22" {
		t.Errorf("occurrence = %q, want 2026-09-22 — the envelope was not opened", res.Occurrence)
	}
	if res.State != "done" {
		t.Errorf("state = %q, want done", res.State)
	}
}

// Postponing answers with a different set of fields, and the bot reports the
// number of days it actually closed — so those must survive the envelope too.
func TestAPostponeAnswerSurvivesTheEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"postponed_days":7,"closed":2,"from":"2026-09-21"}}`))
	}))
	defer srv.Close()

	res, err := NewClient(srv.URL).MarkOccurrence("t0ken", "task-1", "", 7)
	if err != nil {
		t.Fatalf("postpone: %v", err)
	}
	if res.Closed != 2 || res.PostponedDays != 7 || res.From != "2026-09-21" {
		t.Errorf("got %+v, want closed=2 postponed=7 from=2026-09-21", res)
	}
}
