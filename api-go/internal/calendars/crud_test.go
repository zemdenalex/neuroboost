package calendars

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

// setupTestDB connects to DATABASE_URL and seeds a user, returning the user ID
// and a cleanup func. Skips when DATABASE_URL is unset, matching the export and
// tasks packages: these run in CI, not on a bare local run.
func setupTestDB(t *testing.T) (userID string, cleanup func()) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	InitDB(d)
	ctx := context.Background()

	email := fmt.Sprintf("calendars-test-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	cleanup = func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE user_id = $1`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
		d.Close()
	}
	return userID, cleanup
}

// TestListForPutsPersonalFirst also covers the self-healing call: the seeded
// user has no calendar until ListFor creates one.
func TestListForPutsPersonalFirst(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := Create(ctx, userID, "Общий", nil); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := ListFor(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 calendars, got %d", len(list))
	}
	if list[0].Kind != KindPersonal {
		t.Errorf("personal calendar must sort first, got %q", list[0].Kind)
	}
	if list[0].Role != RoleOwner || list[0].Status != StatusActive {
		t.Errorf("role/status must come from my own membership row, got %q/%q",
			list[0].Role, list[0].Status)
	}
}

// TestCreateMakesCreatorAMember is the invariant that makes a calendar usable:
// without the membership row CalendarIDsFor never returns it and the creator
// cannot see their own calendar.
func TestCreateMakesCreatorAMember(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "  Отпуск  ", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.Name != "Отпуск" {
		t.Errorf("name must be trimmed, got %q", c.Name)
	}
	if c.Kind != KindShared {
		t.Errorf("created calendars are shared, got %q", c.Kind)
	}

	ids, err := CalendarIDsFor(ctx, userID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == c.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("created calendar is not reachable through CalendarIDsFor")
	}
}

// TestDeleteRefusesNonEmpty is the spec §5.1 guard: deleting a calendar with
// content would take events belonging to every member, not just the deleter.
func TestDeleteRefusesNonEmpty(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "Проект", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at)
		 VALUES ($1, $2, 'x', NOW(), NOW() + interval '1 hour')`,
		userID, c.ID); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	err = Delete(ctx, userID, c.ID)
	var ne *NotEmptyError
	if !errors.As(err, &ne) {
		t.Fatalf("want NotEmptyError, got %v", err)
	}
	if ne.Events != 1 || ne.Tasks != 0 {
		t.Errorf("counts must report contents, got %d events / %d tasks", ne.Events, ne.Tasks)
	}
}

// TestDeleteRefusesPersonal: event.calendar_id references calendar(id) with no
// ON DELETE, so deleting the personal calendar is either an FK violation or a
// silent loss of the user's own container.
func TestDeleteRefusesPersonal(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	personalID, err := PersonalIDFor(ctx, userID)
	if err != nil {
		t.Fatalf("personal: %v", err)
	}
	if err := Delete(ctx, userID, personalID); !errors.Is(err, ErrCalendarIsPersonal) {
		t.Fatalf("want ErrCalendarIsPersonal, got %v", err)
	}
}

// TestDeleteEmptySharedSucceeds is the positive control: without it the three
// refusals above would pass on an implementation that refuses everything.
func TestDeleteEmptySharedSucceeds(t *testing.T) {
	userID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, userID, "Временный", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Delete(ctx, userID, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err := ListFor(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, x := range list {
		if x.ID == c.ID {
			t.Fatal("calendar still listed after delete")
		}
	}
}

// TestNonMemberGetsNotFound: a stranger must not learn that the calendar
// exists — 403 would confirm it.
func TestNonMemberGetsNotFound(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Чужой", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var strangerID string
	email := fmt.Sprintf("stranger-%d@example.com", time.Now().UnixNano())
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&strangerID); err != nil {
		t.Fatalf("seed stranger: %v", err)
	}
	defer func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, strangerID) }()

	if _, err := Update(ctx, strangerID, c.ID, UpdateFields{}); !errors.Is(err, ErrCalendarNotFound) {
		t.Errorf("update by stranger: want ErrCalendarNotFound, got %v", err)
	}
	if err := Delete(ctx, strangerID, c.ID); !errors.Is(err, ErrCalendarNotFound) {
		t.Errorf("delete by stranger: want ErrCalendarNotFound, got %v", err)
	}
}
