package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// A task can be created in a shared calendar, and only by someone allowed to
// write there.
//
// 🔴 The refusals matter more than the success. A viewer must be stopped by the
// ACCESS check — not incidentally by a NOT NULL constraint or a failed UUID
// parse, which would refuse today and stop refusing the moment the schema
// changes. Each case therefore asserts the specific sentinel error, not merely
// that something went wrong.
//
// Skips without DATABASE_URL, which CI sets.
func TestTaskCreationHonoursCalendarAccess(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()

	ownerID := seedTaskUser(t, ctx, d, "owner")
	editorID := seedTaskUser(t, ctx, d, "editor")
	viewerID := seedTaskUser(t, ctx, d, "viewer")
	strangerID := seedTaskUser(t, ctx, d, "stranger")

	shared, err := calendars.Create(ctx, ownerID, "Дом", nil)
	if err != nil {
		t.Fatalf("create shared calendar: %v", err)
	}
	for _, m := range []struct {
		user, role string
	}{{editorID, calendars.RoleEditor}, {viewerID, calendars.RoleViewer}} {
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, $4)`,
			shared.ID, m.user, m.role, calendars.StatusActive); err != nil {
			t.Fatalf("seed %s membership: %v", m.role, err)
		}
	}

	create := func(userID, calendarID, title string) (*Task, error) {
		req := CreateTaskRequest{Title: title, CalendarID: calendarID}
		return insertTask(ctx, userID, req, nil)
	}

	t.Run("an editor writes into the shared calendar", func(t *testing.T) {
		task, err := create(editorID, shared.ID, "Заливка")
		if err != nil {
			t.Fatalf("editor refused: %v", err)
		}
		t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

		if task.CalendarID != shared.ID {
			t.Errorf("task landed in %s, want the shared calendar %s", task.CalendarID, shared.ID)
		}
		// Authorship and access are different columns and mean different things.
		if task.UserID != editorID {
			t.Errorf("author = %s, want the editor %s", task.UserID, editorID)
		}
	})

	t.Run("no calendar means the personal one, as it always did", func(t *testing.T) {
		task, err := create(ownerID, "", "Созвон")
		if err != nil {
			t.Fatalf("unrouted create refused: %v", err)
		}
		t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

		personalID, err := calendars.PersonalIDFor(ctx, ownerID)
		if err != nil {
			t.Fatalf("personal calendar: %v", err)
		}
		if task.CalendarID != personalID {
			t.Errorf("task landed in %s, want the personal calendar %s", task.CalendarID, personalID)
		}
	})

	t.Run("a viewer is refused for lack of write access", func(t *testing.T) {
		task, err := create(viewerID, shared.ID, "Не должно появиться")
		if err == nil {
			_, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID)
			t.Fatal("a viewer created a task in a calendar they may only read")
		}
		if !errors.Is(err, calendars.ErrNotCalendarOwner) {
			t.Fatalf("refused, but for the wrong reason: %v", err)
		}
	})

	t.Run("a stranger is told the calendar does not exist", func(t *testing.T) {
		task, err := create(strangerID, shared.ID, "Не должно появиться")
		if err == nil {
			_, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID)
			t.Fatal("a non-member created a task in someone else's calendar")
		}
		// NotFound, not Forbidden: a permission error would confirm to a
		// stranger that the calendar exists.
		if !errors.Is(err, calendars.ErrCalendarNotFound) {
			t.Fatalf("refused, but for the wrong reason: %v", err)
		}
	})

	t.Run("a calendar id that is not a uuid reads as no such calendar", func(t *testing.T) {
		if _, err := create(ownerID, "not-a-uuid", "Не должно появиться"); !errors.Is(err, calendars.ErrCalendarNotFound) {
			t.Fatalf("want ErrCalendarNotFound for a malformed id, got %v", err)
		}
	})
}

func seedTaskUser(t *testing.T, ctx context.Context, d *database.DB, label string) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("taskshare-%s-%d@example.com", label, time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("seed %s: %v", label, err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id)
	})
	return id
}
