package daytasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/reminders"
	"neuroboost/api-go/internal/tasks"
)

// Findings of the final review of D1 (23.09), one test each. Every test was
// seen red before its fix.

// seedPromise writes a promise for a PAST day the way it would exist had it
// been made on that day: Add and Confirm refuse past days (I3), so the tests
// that need yesterday write the row directly.
func seedPromise(t *testing.T, d *database.DB, userID string, day time.Time, taskID string) {
	t.Helper()
	if _, err := d.Pool.Exec(context.Background(),
		`INSERT INTO day_commitment (user_id, day, task_id) VALUES ($1, $2, $3)`,
		userID, day.Format(dateFmt), taskID); err != nil {
		t.Fatalf("seed promise: %v", err)
	}
}

func setStatus(t *testing.T, userID, taskID, status string) {
	t.Helper()
	rec := callDay(t, tasks.UpdateHandler, userID, http.MethodPatch, "/api/tasks/"+taskID,
		map[string]any{"status": status}, map[string]string{"id": taskID})
	if rec.Code != http.StatusOK {
		t.Fatalf("set status %s: %d %s", status, rec.Code, rec.Body.String())
	}
}

// C1: the reminder's ✅ Done is a second door that closes a task. It must count
// for the day like PATCH does, and must not break the day screen.
func TestAReminderDoneCountsForTheDay(t *testing.T) {
	d, ctx, user := dayDB(t)
	reminders.InitDB(d)
	tgID := time.Now().UnixNano() % 1_000_000_000
	if _, err := d.Pool.Exec(ctx, `UPDATE "user" SET tg_id = $2 WHERE id = $1`, user, tgID); err != nil {
		t.Fatalf("set tg_id: %v", err)
	}
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "из напоминания"})
	if err := Confirm(ctx, user, today, []string{id}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	var reminderID string
	if err := d.Pool.QueryRow(ctx, `
		INSERT INTO reminder (user_id, source_kind, task_id, remind_at, status, channel, message)
		VALUES ($1, 'TASK', $2, NOW(), 'SENT', 'TELEGRAM', 'из напоминания') RETURNING id`,
		user, id).Scan(&reminderID); err != nil {
		t.Fatalf("seed reminder: %v", err)
	}
	b, _ := json.Marshal(map[string]any{"tg_id": tgID, "reminder_id": reminderID, "action": "done"})
	rec := httptest.NewRecorder()
	reminders.ActionHandler(rec, httptest.NewRequest(http.MethodPost, "/api/svc/reminders/action", bytes.NewReader(b)))
	if rec.Code != http.StatusOK {
		t.Fatalf("reminder done: %d %s", rec.Code, rec.Body.String())
	}

	days, err := List(ctx, user, today, today)
	if err != nil {
		t.Fatalf("list after a reminder ✅: %v", err)
	}
	if days[0].Done != 1 {
		t.Errorf("a task closed from its reminder is not counted: %+v", days[0])
	}
}

// I1: the target a day was taken with is the day's target. Changing the
// setting afterwards neither turns today green nor recolours the past.
func TestChangingTheTargetDoesNotRecolourATakenDay(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, newTask(t, user, map[string]any{"title": fmt.Sprintf("n%d", i)}))
	}
	if err := Confirm(ctx, user, today, ids); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	for _, id := range ids[:3] {
		closeTask(t, user, id)
	}
	if _, err := d.Pool.Exec(ctx,
		`UPDATE "user" SET settings = '{"day_tasks_target":3}' WHERE id = $1`, user); err != nil {
		t.Fatalf("set target: %v", err)
	}
	days, _ := List(ctx, user, today, today)
	if days[0].Target != 5 || days[0].Level != 3 {
		t.Errorf("after N 5→3 the taken day reads %+v, want target 5, level 3 (🟧)", days[0])
	}
}

// I2: «✅ Беру» on a proposal whose pinned task was done in the meantime takes
// the day; keeping a promised task is not adding a done one.
func TestConfirmKeepsAPinnedTaskThatIsAlreadyDone(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	pinned := newTask(t, user, map[string]any{"title": "заранее"})
	other := newTask(t, user, map[string]any{"title": "ещё"})
	if err := Add(ctx, user, today, pinned); err != nil {
		t.Fatal(err)
	}
	closeTask(t, user, pinned)
	if err := Confirm(ctx, user, today, []string{pinned, other}); err != nil {
		t.Fatalf("confirm with a pinned task done since: %v", err)
	}
	days, _ := List(ctx, user, today, today)
	if !days[0].Confirmed || days[0].Done != 1 || len(days[0].Items) != 2 {
		t.Errorf("day = %+v, want confirmed, 2 items, 1 done", days[0])
	}
}

// I2: a refused task refuses the whole confirm — nothing half-written.
func TestConfirmWritesNothingWhenOneTaskIsRefused(t *testing.T) {
	_, ctx, user := dayDB(t)
	_, _, other := dayDB(t)
	today := Today(time.Now(), ny)
	mine := newTask(t, user, map[string]any{"title": "моё"})
	theirs := newTask(t, other, map[string]any{"title": "чужое"})
	if err := Confirm(ctx, user, today, []string{mine, theirs}); err != ErrTaskNotFound {
		t.Fatalf("confirm with a stranger's task: err = %v, want ErrTaskNotFound", err)
	}
	days, _ := List(ctx, user, today, today)
	if days[0].Confirmed || len(days[0].Items) != 0 {
		t.Errorf("a refused confirm left %+v behind", days[0])
	}
}

// I3: a past day is closed to every edit, not only to removal. Otherwise an
// untaken ⬛ day could be taken the next morning.
func TestAPastDayCannotBeTakenOrAddedTo(t *testing.T) {
	_, ctx, user := dayDB(t)
	yesterday := Today(time.Now(), ny).AddDate(0, 0, -1)
	id := newTask(t, user, map[string]any{"title": "задним числом"})
	if err := Add(ctx, user, yesterday, id); err != ErrTooLate {
		t.Errorf("add to yesterday: err = %v, want ErrTooLate", err)
	}
	if err := Confirm(ctx, user, yesterday, nil); err != ErrTooLate {
		t.Errorf("confirm yesterday: err = %v, want ErrTooLate", err)
	}
}

// I4: CANCELLED is closed, as everywhere else in the codebase.
func TestACancelledTaskIsNeitherProposedNorAddable(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "отменено"})
	setStatus(t, user, id, "CANCELLED")
	if err := Add(ctx, user, today, id); err != ErrNotOpen {
		t.Errorf("add cancelled: err = %v, want ErrNotOpen", err)
	}
	got, err := Propose(ctx, user, today)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range got {
		if it.TaskID == id {
			t.Errorf("a cancelled task is proposed")
		}
	}
}

// I5: a series enters only a day it occurs on — by Add or by the proposal.
func TestASeriesEntersOnlyItsOwnDays(t *testing.T) {
	_, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	weekly := newTask(t, user, map[string]any{"title": "раз в неделю", "rrule": "FREQ=WEEKLY",
		"due_date": today.Format(time.RFC3339)})
	if err := Add(ctx, user, today.AddDate(0, 0, 1), weekly); err != ErrNotAnOccurrence {
		t.Errorf("add weekly series on a day it does not occur: err = %v, want ErrNotAnOccurrence", err)
	}
	if err := Add(ctx, user, today.AddDate(0, 0, 7), weekly); err != nil {
		t.Errorf("add weekly series on its next day: %v", err)
	}
}

// I5: «yesterday's undone» means undone. A daily series done yesterday is
// today's task by its own rank (4, after the due one-off), not carried (2).
func TestASeriesDoneYesterdayIsNotCarried(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1)
	daily := newTask(t, user, map[string]any{"title": "каждый день", "rrule": "FREQ=DAILY",
		"due_date": yesterday.Format(time.RFC3339)})
	due := newTask(t, user, map[string]any{"title": "срок сегодня", "priority": 5,
		"due_date": today.Format(time.RFC3339)})
	seedPromise(t, d, user, yesterday, daily)
	if err := tasks.MarkOccurrence(ctx, user, daily, yesterday, tasks.StateDone); err != nil {
		t.Fatalf("mark yesterday: %v", err)
	}
	got, err := Propose(ctx, user, today)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, it := range got {
		ids = append(ids, it.TaskID)
	}
	if want := fmt.Sprint([]string{due, daily}); fmt.Sprint(ids) != want {
		t.Errorf("proposal = %v, want %v (due one-off, then the series by its own rank)", ids, want)
	}
}

// I5: a weekly series promised yesterday and not occurring today is not
// carried into today.
func TestASeriesThatDoesNotOccurTodayIsNotCarried(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	yesterday := today.AddDate(0, 0, -1)
	weekly := newTask(t, user, map[string]any{"title": "раз в неделю", "rrule": "FREQ=WEEKLY",
		"due_date": yesterday.Format(time.RFC3339)})
	seedPromise(t, d, user, yesterday, weekly)
	got, err := Propose(ctx, user, today)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range got {
		if it.TaskID == weekly {
			t.Errorf("a weekly series is carried into a day it does not occur on")
		}
	}
}

// C1, second half: whatever door forgets completed_at in future, a DONE row
// without it reads «not done» — never NULL, never a 500 for the whole day.
func TestADoneTaskWithoutAStampDoesNotBreakTheDay(t *testing.T) {
	d, ctx, user := dayDB(t)
	today := Today(time.Now(), ny)
	id := newTask(t, user, map[string]any{"title": "без отметки времени"})
	if err := Confirm(ctx, user, today, []string{id}); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`UPDATE task SET status = 'DONE', completed_at = NULL WHERE id = $1`, id); err != nil {
		t.Fatalf("close without a stamp: %v", err)
	}
	days, err := List(ctx, user, today, today)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if days[0].Done != 0 {
		t.Errorf("a DONE row without completed_at counted as done: %+v", days[0])
	}
}
