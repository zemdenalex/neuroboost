package events

import (
	"context"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// The badge and the author name are per-viewer, so the only honest test loads
// the SAME events twice — once as each member — and asserts the two answers
// differ in exactly the right place.
//
// Skips without DATABASE_URL, which CI sets. See CLAUDE.md for the throwaway
// Postgres recipe; a green run without a database proves nothing here, and this
// repository has been bitten by exactly that.
func TestSharingIsAnsweredPerViewer(t *testing.T) {
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

	ownerID := seedUser(t, ctx, d, "owner")
	editorID := seedUser(t, ctx, d, "editor")

	// A display name on one side only: the other must still get a label.
	if _, err := d.Pool.Exec(ctx,
		`UPDATE "user" SET display_name = 'Настя' WHERE id = $1`, editorID); err != nil {
		t.Fatalf("name the editor: %v", err)
	}

	shared, err := calendars.Create(ctx, ownerID, "Дом", nil)
	if err != nil {
		t.Fatalf("create shared calendar: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, $4)`,
		shared.ID, editorID, calendars.RoleEditor, calendars.StatusActive); err != nil {
		t.Fatalf("seed editor membership: %v", err)
	}

	personalID, err := calendars.PersonalIDFor(ctx, ownerID)
	if err != nil {
		t.Fatalf("personal calendar: %v", err)
	}

	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	byOwner := seedEvent(t, ctx, d, ownerID, shared.ID, "Ужин", start)
	byEditor := seedEvent(t, ctx, d, editorID, shared.ID, "Заливка", start)
	private := seedEvent(t, ctx, d, ownerID, personalID, "Созвон", start)

	load := func(viewerID string) map[string]Event {
		t.Helper()
		list, err := listEvents(ctx, viewerID, start.Add(-time.Hour), start.Add(2*time.Hour))
		if err != nil {
			t.Fatalf("listEvents for %s: %v", viewerID, err)
		}
		byID := map[string]Event{}
		for _, e := range list {
			byID[e.ID] = e
		}
		return byID
	}

	asOwner := load(ownerID)
	asEditor := load(editorID)

	// A control that could not fail is not a control: if the fixture never
	// produced the rows, every assertion below would pass vacuously.
	for _, id := range []string{byOwner, byEditor, private} {
		if _, ok := asOwner[id]; !ok {
			t.Fatalf("fixture did not reach the owner's list: %s", id)
		}
	}
	if _, ok := asEditor[private]; ok {
		t.Fatal("the editor can see the owner's personal event — access, not decoration, is broken")
	}

	t.Run("the shared badge does not depend on who authored the event", func(t *testing.T) {
		if !asOwner[byOwner].IsShared {
			t.Error("the owner's own event in a shared calendar is not marked shared")
		}
		if !asOwner[byEditor].IsShared {
			t.Error("the editor's event is not marked shared")
		}
	})

	t.Run("a personal calendar is never shared", func(t *testing.T) {
		if asOwner[private].IsShared {
			t.Error("an event in the personal calendar is marked shared")
		}
		if asOwner[private].AuthorName != nil {
			t.Errorf("a personal event named an author: %q", *asOwner[private].AuthorName)
		}
	})

	t.Run("the author is named only to the other person", func(t *testing.T) {
		if got := asOwner[byOwner].AuthorName; got != nil {
			t.Errorf("the owner was told the author of their own event: %q", *got)
		}
		if got := asOwner[byEditor].AuthorName; got == nil {
			t.Error("the owner was not told who created the editor's event")
		} else if *got != "Настя" {
			t.Errorf("author name = %q, want %q", *got, "Настя")
		}

		// The same row, the other way round. This is the assertion a stored
		// column could never satisfy.
		if got := asEditor[byEditor].AuthorName; got != nil {
			t.Errorf("the editor was told the author of their own event: %q", *got)
		}
		if got := asEditor[byOwner].AuthorName; got == nil {
			t.Error("the editor was not told who created the owner's event")
		} else if *got == "Настя" {
			t.Errorf("the editor was shown their own name as the author: %q", *got)
		}
	})

	t.Run("the badge follows membership, not the calendar's kind", func(t *testing.T) {
		// Remove the second member and re-read. Nothing about the event row
		// changes; the answer must change anyway.
		if _, err := d.Pool.Exec(ctx,
			`DELETE FROM calendar_member WHERE calendar_id = $1 AND user_id = $2`,
			shared.ID, editorID); err != nil {
			t.Fatalf("remove the editor: %v", err)
		}
		if load(ownerID)[byOwner].IsShared {
			t.Error("still marked shared after the only other member left — the flag is stale")
		}
	})
}

func seedEvent(t *testing.T, ctx context.Context, d *database.DB, userID, calendarID, title string, start time.Time) string {
	t.Helper()
	var id string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, all_day, timezone)
		 VALUES ($1, $2, $3, $4, $5, false, 'UTC') RETURNING id`,
		userID, calendarID, title, start, start.Add(time.Hour)).Scan(&id); err != nil {
		t.Fatalf("seed event %q: %v", title, err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, id)
	})
	return id
}
