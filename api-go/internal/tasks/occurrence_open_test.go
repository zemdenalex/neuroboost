package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// Undo of a tick (docs/tasks-web-cleanup.md 4.11): the web Tasks page closes
// today's day of a series instead of the series, and its «Undo» must put the
// day back to «no answer». Overwriting with «skipped» would be a different
// answer, not an undo, so state "open" removes the day's row.
//
// ⚠ Needs DATABASE_URL and -count=1 (see callOccurrence).
func TestOpenPutsADoneDayBackToNoAnswer(t *testing.T) {
	d, ctx, user := repeatDB(t)

	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "зарядка", Rrule: str("FREQ=DAILY"),
	}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	today := userToday().Format("2006-01-02")
	if rec := callOccurrence(task.ID, user, map[string]any{"state": "done", "date": today}); rec.Code != http.StatusOK {
		t.Fatalf("done: status %d: %s", rec.Code, rec.Body.String())
	}
	// Positive control: the day is really marked before the undo.
	if n := occurrenceRows(t, task.ID); n != 1 {
		t.Fatalf("after done: %d rows, want 1", n)
	}

	rec := callOccurrence(task.ID, user, map[string]any{"state": "open", "date": today})
	if rec.Code != http.StatusOK {
		t.Fatalf("open: status %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data struct {
			Occurrence string `json:"occurrence"`
			State      string `json:"state"`
		} `json:"data"`
	}
	if uerr := json.Unmarshal(rec.Body.Bytes(), &got); uerr != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), uerr)
	}
	if got.Data.Occurrence != today || got.Data.State != "open" {
		t.Errorf("answer %+v, want {%s open}", got.Data, today)
	}
	if n := occurrenceRows(t, task.ID); n != 0 {
		t.Errorf("after open: %d rows, want 0: the day must have no answer again", n)
	}

	// The series itself was never touched.
	var status string
	if err := d.Pool.QueryRow(ctx, `SELECT status FROM task WHERE id = $1`, task.ID).Scan(&status); err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != "TODO" {
		t.Errorf("series status %q, want TODO", status)
	}
}

func occurrenceRows(t *testing.T, taskID string) int {
	t.Helper()
	var n int
	if err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM task_occurrence WHERE task_id = $1`, taskID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}
