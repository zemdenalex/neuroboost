package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/middleware"
)

// The press itself, through the HTTP handler, against a real database.
//
// 🔴 PressedDay's own tests prove the arithmetic; this one proves the wiring,
// and the wiring is where the 21.09 failure lived. The endpoint resolved the
// day from the clock alone and never consulted the series, so a task whose
// first day is tomorrow answered 400 to the only button on its card.
//
// ⚠ Needs DATABASE_URL and -count=1. Without the first it skips and still
// prints ok; without the second Go serves a cached result, because another
// process's database is not part of what it hashes.
func callOccurrence(taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/occurrences", bytes.NewReader(raw))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	rec := httptest.NewRecorder()
	MarkOccurrenceHandler(rec, req.WithContext(ctx))
	return rec
}

func TestPressingDoneOnASeriesThatStartsTomorrowClosesTomorrow(t *testing.T) {
	d, ctx, user := repeatDB(t)

	// 🔴 The due date goes in BOTH places, exactly as CreateTaskHandler does it:
	// the request carries the string, the parsed day is passed alongside, and
	// it is the parsed one that anchors the series. Passing only the string —
	// the first version of this test — anchored the series on today and made
	// the test green against the very bug it was written for.
	due := userToday().AddDate(0, 0, 1)
	tomorrow := due.Format("2006-01-02")
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "позвонить в банк", Rrule: str("FREQ=DAILY"), DueDate: &tomorrow,
	}, &due)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callOccurrence(task.ID, user, map[string]any{"state": "done"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 — this is the NOT_AN_OCCURRENCE Denis got:\n\t%s",
			rec.Code, rec.Body.String())
	}
	// ⚠ Asserted against the `data` envelope specifically, not «whichever field
	// happens to carry it». The first draft accepted either, and a test that
	// accepts either shape cannot notice a client reading the wrong one — which
	// is precisely the mistake the bot then made.
	var got struct {
		Data struct {
			Occurrence string `json:"occurrence"`
		} `json:"data"`
	}
	if uerr := json.Unmarshal(rec.Body.Bytes(), &got); uerr != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), uerr)
	}
	if day := got.Data.Occurrence; day != tomorrow {
		t.Errorf("closed %q, want %s — the first day the series actually has", day, tomorrow)
	}
}

// 🔴 And a date the caller NAMED is still refused. Sliding someone's chosen
// day to a neighbouring one would be a worse answer than «no», and this is the
// guard that keeps task_occurrence free of days nothing will ask about.
func TestANamedDayOutsideTheSeriesIsStillRefused(t *testing.T) {
	d, ctx, user := repeatDB(t)

	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "планёрка", Rrule: str("FREQ=WEEKLY"),
	}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	// Anchored today, weekly: tomorrow is not in the series.
	tomorrow := userToday().AddDate(0, 0, 1).Format("2006-01-02")
	rec := callOccurrence(task.ID, user, map[string]any{"state": "done", "date": tomorrow})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400: a named day outside the series must be refused, not moved", rec.Code)
	}
}
