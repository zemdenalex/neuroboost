package usersettings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

func TestParseWorkWeekDefaults(t *testing.T) {
	// An empty blob is the common case: settings starts as {} on registration.
	w := ParseWorkWeek([]byte(`{}`))
	if w.StartMinutes != 9*60 || w.EndMinutes != 17*60 {
		t.Errorf("hours = %d..%d, want 540..1020", w.StartMinutes, w.EndMinutes)
	}
	if len(w.Days) != 5 {
		t.Errorf("days = %v, want five", w.Days)
	}
	if got := w.Hours(); got != 40 {
		t.Errorf("Hours() = %v, want 40 — the number planning used to hardcode", got)
	}
}

func TestParseWorkWeekReadsWhatTheUserSet(t *testing.T) {
	w := ParseWorkWeek([]byte(`{"work_start":"07:30","work_end":"16:00","work_days":["Mon","Tue","Wed"]}`))
	if w.StartMinutes != 7*60+30 {
		t.Errorf("start = %d, want 450", w.StartMinutes)
	}
	if w.EndMinutes != 16*60 {
		t.Errorf("end = %d, want 960", w.EndMinutes)
	}
	// 8.5 hours × 3 days.
	if got := w.Hours(); got != 25.5 {
		t.Errorf("Hours() = %v, want 25.5", got)
	}
}

func TestEachFieldFallsBackOnItsOwn(t *testing.T) {
	// 🔴 Per field, not per blob. A profile with only an end time set must keep
	// the default start; falling back wholesale would silently discard a value
	// the user had just saved, which reads as "the save did not work".
	w := ParseWorkWeek([]byte(`{"work_end":"20:00"}`))
	if w.StartMinutes != 9*60 {
		t.Errorf("start = %d, want the default 540", w.StartMinutes)
	}
	if w.EndMinutes != 20*60 {
		t.Errorf("end = %d, want 1200", w.EndMinutes)
	}
	if got := w.Hours(); got != 55 {
		t.Errorf("Hours() = %v, want 55", got)
	}
}

func TestAnEmptyDayListIsKeptButAMissingOneIsNot(t *testing.T) {
	// Unticking every day is a statement; omitting the key is not.
	none := ParseWorkWeek([]byte(`{"work_days":[]}`))
	if len(none.Days) != 0 {
		t.Errorf("an explicit empty list became %v", none.Days)
	}
	if got := none.Hours(); got != 0 {
		t.Errorf("Hours() = %v with no working days, want 0", got)
	}

	missing := ParseWorkWeek([]byte(`{"work_start":"10:00"}`))
	if len(missing.Days) != 5 {
		t.Errorf("a missing key gave %v, want the five defaults", missing.Days)
	}
}

func TestRubbishFallsBackRatherThanPropagating(t *testing.T) {
	// The blob is user-controlled JSONB. Every one of these has reached a
	// settings column somewhere; none may produce a negative or absurd week.
	for _, raw := range []string{
		`{"work_start":"nonsense","work_end":"17:00"}`,
		`{"work_start":"25:00","work_end":"17:00"}`,
		`{"work_start":"09:61","work_end":"17:00"}`,
		`{"work_start":"0900","work_end":"17:00"}`,
		`{"work_start":9,"work_end":"17:00"}`,
		`not json at all`,
		``,
	} {
		w := ParseWorkWeek([]byte(raw))
		if w.StartMinutes != 9*60 {
			t.Errorf("%q: start = %d, want the default 540", raw, w.StartMinutes)
		}
		if w.Hours() < 0 {
			t.Errorf("%q: Hours() = %v, negative", raw, w.Hours())
		}
	}
}

func TestAnInvertedDayIsZeroNotNegative(t *testing.T) {
	// The web app accepts start > end — WorkHoursSection.tsx says so in its own
	// comment. Planning must not answer "you have -30 hours available".
	w := ParseWorkWeek([]byte(`{"work_start":"18:00","work_end":"09:00"}`))
	if got := w.Hours(); got != 0 {
		t.Errorf("Hours() = %v for an inverted day, want 0", got)
	}
}

func TestTheBlobTheBotWritesIsUnderstood(t *testing.T) {
	// 🔴 Round-trip against the bot's own format rather than a hand-typed
	// string: the bot writes "%02d:00" through MergeSettings, and a test using
	// its own spelling would agree with itself while the two drifted apart.
	botBlob, err := json.Marshal(map[string]any{
		"work_start":     "08:00",
		"work_end":       "18:00",
		"header_variant": "vertical", // an unrelated key the merge preserves
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	w := ParseWorkWeek(botBlob)
	if w.StartMinutes != 8*60 || w.EndMinutes != 18*60 {
		t.Errorf("bot blob parsed as %d..%d", w.StartMinutes, w.EndMinutes)
	}
	if got := w.Hours(); got != 50 {
		t.Errorf("Hours() = %v, want 50", got)
	}
}

// The SQL, against a real row.
//
// 🔴 ParseWorkWeek being right proves nothing about LoadWorkWeek: that function
// defaults on ANY failure, so a wrong column name, a wrong table, or a nil pool
// all produce a perfectly plausible 40-hour week. It would have looked correct
// forever. This asks the database for a user whose settings it just wrote.
func TestLoadWorkWeekReadsTheRowRatherThanDefaulting(t *testing.T) {
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
	ctx := context.Background()

	var userID string
	email := fmt.Sprintf("workhours-test-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, settings) VALUES ($1, $2::jsonb) RETURNING id`,
		email, `{"work_start":"07:00","work_end":"15:00","work_days":["Mon","Tue","Wed","Thu"]}`,
	).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID) })

	w := LoadWorkWeek(ctx, userID)
	// 8 hours × 4 days. Deliberately NOT 40: a default would also be a round
	// number, and the assertion has to be able to tell the two apart.
	if got := w.Hours(); got != 32 {
		t.Errorf("Hours() = %v, want 32 — this is the default 40 leaking through", got)
	}
	if w.StartMinutes != 7*60 {
		t.Errorf("start = %d, want 420", w.StartMinutes)
	}

	// A user who has never touched settings still gets the old number, so this
	// change moves nobody's planning page on its own.
	var freshID string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`,
		fmt.Sprintf("workhours-fresh-%d@example.com", time.Now().UnixNano()),
	).Scan(&freshID); err != nil {
		t.Fatalf("seed fresh user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, freshID) })

	if got := LoadWorkWeek(ctx, freshID).Hours(); got != 40 {
		t.Errorf("a user with no settings got %v hours, want the unchanged 40", got)
	}
}
