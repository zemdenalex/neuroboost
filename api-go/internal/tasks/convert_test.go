package tasks

import (
	"context"
	"errors"
	"testing"
)

// 🔴 A task with no time cannot quietly become an event.
//
// Denis's shape for this («1 + 3») is that the bot SHOWS which field becomes
// which and asks for what is missing. That only works if the API refuses instead
// of inventing: a task carries a due DAY, and guessing 09:00–10:00 would put a
// thing in the calendar at an hour nobody chose and then remind about it.
//
// Checked before any database work, so this runs everywhere — the validation is
// the part that decides whether a user is asked or surprised.
func TestConvertRefusesWithoutATime(t *testing.T) {
	ctx := context.Background()

	for _, c := range []struct {
		name string
		req  ConvertRequest
	}{
		{"no times at all", ConvertRequest{Mode: ModeMove}},
		{"only a start", ConvertRequest{Mode: ModeMove, StartsAt: "2026-09-19T15:00:00Z"}},
		{"only an end", ConvertRequest{Mode: ModeLink, EndsAt: "2026-09-19T16:00:00Z"}},
		{"unparseable start", ConvertRequest{Mode: ModeMove, StartsAt: "tomorrow", EndsAt: "2026-09-19T16:00:00Z"}},
		{"end before start", ConvertRequest{Mode: ModeMove, StartsAt: "2026-09-19T16:00:00Z", EndsAt: "2026-09-19T15:00:00Z"}},
		{"zero length", ConvertRequest{Mode: ModeMove, StartsAt: "2026-09-19T15:00:00Z", EndsAt: "2026-09-19T15:00:00Z"}},
	} {
		_, err := Convert(ctx, "u1", "t1", c.req)
		if !errors.Is(err, ErrNeedsTime) {
			t.Errorf("%s: err = %v, want ErrNeedsTime", c.name, err)
		}
	}
}

// An unknown mode is refused before anything is touched. «move» and «link» mean
// opposite things — one deletes the task — so guessing is not available.
func TestConvertRefusesAnUnknownMode(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"", "delete", "MOVE ", "copy"} {
		_, err := Convert(ctx, "u1", "t1", ConvertRequest{
			Mode: mode, StartsAt: "2026-09-19T15:00:00Z", EndsAt: "2026-09-19T16:00:00Z",
		})
		if !errors.Is(err, ErrUnknownMode) {
			t.Errorf("mode %q: err = %v, want ErrUnknownMode", mode, err)
		}
	}
}

// 🔴 Validation must come BEFORE the mode check is satisfied by a valid mode but
// after nothing has been written. This asserts the ORDER: a bad mode with bad
// times reports the mode, because that is the first thing the caller can fix.
func TestConvertChecksTheModeFirst(t *testing.T) {
	_, err := Convert(context.Background(), "u1", "t1", ConvertRequest{Mode: "nonsense"})
	if !errors.Is(err, ErrUnknownMode) {
		t.Errorf("err = %v, want ErrUnknownMode before ErrNeedsTime", err)
	}
}

// ⚠ The hand-rolled requireWritable this file used to test is gone.
//
// calendars/writescoping_test.go caught Convert on its first run: an INSERT must
// check its destination with the SINGULAR resolver (WritableIDFor), which asks
// "may I write in this calendar", not with the plural list, which only scopes a
// WHERE. That is the precise hole scheduleTask fell through in August, and my
// version was the same shape.
//
// So the check now lives in the store, where it is tested once, and the guard
// enforces that every INSERT uses it. There is nothing left here to unit-test
// that would not be testing the store twice.
