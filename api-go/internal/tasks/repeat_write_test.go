package tasks

import (
	"context"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// Can a task BECOME repeating through the same door a client uses?
//
// 🔴 On 20.09 Denis walked the v0.4.11.4 checklist and stopped at section 2:
// nothing could create a repeating task. He was right, and it was worse than the
// bot missing a button — CreateTaskRequest and UpdateTaskRequest had no rrule
// field, and no statement in this package wrote the column. The occurrence
// table, the «сделал на сегодня» endpoint, the postponement and the nagging were
// all built and all green, because every one of their tests put `rrule` into
// the database with a direct INSERT. An engine with no ignition key, and a test
// suite that started it with a screwdriver.
//
// So these tests go through insertTask and updateTask — the functions the HTTP
// handlers call — and never touch the column themselves.
func repeatDB(t *testing.T) (*database.DB, context.Context, string) {
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
	InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()
	return d, ctx, seedTaskUser(t, ctx, d, "repeat")
}

func str(s string) *string { return &s }
func num(n int) *int       { return &n }

func TestATaskCanBeCreatedRepeating(t *testing.T) {
	d, ctx, user := repeatDB(t)

	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "пить таблетки", Rrule: str("FREQ=DAILY"), NagMinutes: num(10),
	}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	if task.Rrule == nil || *task.Rrule != "FREQ=DAILY" {
		t.Errorf("the rule did not come back on the created task: %v", task.Rrule)
	}
	if task.NagMinutes == nil || *task.NagMinutes != 10 {
		t.Errorf("nag_minutes did not come back: %v", task.NagMinutes)
	}

	// 🔴 The anchor is what makes the rule mean anything: Occurs() counts steps
	// from it, and repeatOf() treats a missing anchor as «not repeating». A rule
	// saved without one is a task that says it repeats and never does.
	var anchor *time.Time
	if err := d.Pool.QueryRow(ctx, `SELECT repeat_anchor FROM task WHERE id = $1`, task.ID).
		Scan(&anchor); err != nil {
		t.Fatalf("read anchor: %v", err)
	}
	if anchor == nil {
		t.Fatal("rrule was saved with no repeat_anchor — the series would never occur")
	}

	// And the rest of the machinery must now accept it, which is the point.
	today := LocalDay(time.Now(), "Europe/Moscow")
	if err := MarkOccurrence(ctx, user, task.ID, today, "done"); err != nil {
		t.Errorf("a task created through the API cannot be ticked for today: %v", err)
	}
}

// With a due date, the series starts on that day — not on the day it was typed.
func TestTheAnchorFollowsTheDueDate(t *testing.T) {
	d, ctx, user := repeatDB(t)

	due := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC) // a Monday
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "планёрка", Rrule: str("FREQ=WEEKLY"),
	}, &due)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	var anchor time.Time
	if err := d.Pool.QueryRow(ctx, `SELECT repeat_anchor FROM task WHERE id = $1`, task.ID).
		Scan(&anchor); err != nil {
		t.Fatalf("read anchor: %v", err)
	}
	if got := anchor.Format("2006-01-02"); got != "2026-10-05" {
		t.Errorf("anchor = %s, want the due date 2026-10-05 — «каждый понедельник» must stay on Mondays", got)
	}
}

func TestAnInvalidRuleIsRefusedNotStored(t *testing.T) {
	_, ctx, user := repeatDB(t)

	for _, bad := range []string{"FREQ=YEARLY", "каждый день", "FREQ=DAILY;BYDAY=MO"} {
		if task, err := insertTask(ctx, user, CreateTaskRequest{Title: "x", Rrule: str(bad)}, nil); err == nil {
			t.Errorf("rule %q was accepted (task %s) — the scanner would choke on it every minute", bad, task.ID)
		}
	}
	for _, bad := range []int{-5, 3, 100000} {
		if task, err := insertTask(ctx, user, CreateTaskRequest{Title: "x", NagMinutes: num(bad)}, nil); err == nil {
			t.Errorf("nag_minutes=%d was accepted (task %s)", bad, task.ID)
		}
	}
}

// An existing task can be switched to repeating and back.
func TestRepeatCanBeTurnedOnAndOff(t *testing.T) {
	d, ctx, user := repeatDB(t)

	task, err := insertTask(ctx, user, CreateTaskRequest{Title: "зарядка"}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })
	if task.Rrule != nil && *task.Rrule != "" {
		t.Fatalf("a plain task came back repeating: %v", *task.Rrule)
	}

	on, err := updateTask(ctx, user, task.ID, UpdateTaskRequest{Rrule: str("FREQ=DAILY;INTERVAL=2")})
	if err != nil {
		t.Fatalf("turn on: %v", err)
	}
	if on.Rrule == nil || *on.Rrule != "FREQ=DAILY;INTERVAL=2" {
		t.Errorf("rule after update = %v", on.Rrule)
	}
	var anchor *time.Time
	_ = d.Pool.QueryRow(ctx, `SELECT repeat_anchor FROM task WHERE id = $1`, task.ID).Scan(&anchor)
	if anchor == nil {
		t.Error("turning repeat on did not set an anchor")
	}

	// ⚠ An empty string is «выключить повтор». nil means «не трогать» — the
	// difference every PATCH in this API already relies on.
	off, err := updateTask(ctx, user, task.ID, UpdateTaskRequest{Rrule: str("")})
	if err != nil {
		t.Fatalf("turn off: %v", err)
	}
	if off.Rrule != nil && *off.Rrule != "" {
		t.Errorf("rule survived being turned off: %v", *off.Rrule)
	}

	// Untouched by an unrelated edit.
	if _, err := updateTask(ctx, user, task.ID, UpdateTaskRequest{Rrule: str("FREQ=WEEKLY")}); err != nil {
		t.Fatalf("turn on again: %v", err)
	}
	renamed, err := updateTask(ctx, user, task.ID, UpdateTaskRequest{Title: str("зарядка утром")})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Rrule == nil || *renamed.Rrule != "FREQ=WEEKLY" {
		t.Errorf("renaming the task changed its repeat: %v", renamed.Rrule)
	}
}
