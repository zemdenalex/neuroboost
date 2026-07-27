package events

import (
	"testing"
	"time"
)

func occAt(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func nonRecurring(start time.Time) Event {
	return Event{
		ID: "11111111-1111-1111-1111-111111111111", Title: "one-off",
		StartsAt: start, EndsAt: start.Add(time.Hour), Timezone: "UTC",
	}
}

func TestOccurrencesInRangeNonRecurringDoesNotPanic(t *testing.T) {
	// expandRecurrence dereferences *event.Rrule, so a nil Rrule panics.
	// Every handler guards before calling it; the wrapper must guard too, or
	// the reminder worker takes the whole API process down with it.
	ev := nonRecurring(occAt(2026, 8, 1, 10))
	got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 2, 0), nil)
	if len(got) != 1 || !got[0].Equal(ev.StartsAt) {
		t.Fatalf("got %v, want [%v]", got, ev.StartsAt)
	}
}

func TestOccurrencesInRangeNonRecurringOutsideWindow(t *testing.T) {
	ev := nonRecurring(occAt(2026, 8, 5, 10))
	if got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 2, 0), nil); len(got) != 0 {
		t.Errorf("event outside the window should yield nothing, got %v", got)
	}
}

func TestOccurrencesInRangeIsHalfOpen(t *testing.T) {
	ev := nonRecurring(occAt(2026, 8, 1, 0))
	if got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 2, 0), nil); len(got) != 1 {
		t.Errorf("start of window is inclusive, got %v", got)
	}
	ev2 := nonRecurring(occAt(2026, 8, 2, 0))
	if got := OccurrencesInRange(ev2, occAt(2026, 8, 1, 0), occAt(2026, 8, 2, 0), nil); len(got) != 0 {
		t.Errorf("end of window is exclusive, got %v", got)
	}
}

func TestOccurrencesInRangeEmptyRruleStringIsNotRecurring(t *testing.T) {
	// The column is nullable but handlers also write "" — both mean "no
	// recurrence", and parseRRule("") would just fail into an empty list,
	// silently losing the event's own start.
	empty := ""
	ev := nonRecurring(occAt(2026, 8, 1, 10))
	ev.Rrule = &empty
	got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 2, 0), nil)
	if len(got) != 1 {
		t.Fatalf("empty rrule should behave as non-recurring, got %v", got)
	}
}

func TestOccurrencesInRangeDailySeries(t *testing.T) {
	rrule := "FREQ=DAILY"
	ev := nonRecurring(occAt(2026, 8, 1, 10))
	ev.Rrule = &rrule
	got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 4, 0), nil)
	if len(got) != 3 {
		t.Fatalf("want 3 daily occurrences, got %d: %v", len(got), got)
	}
	for i, want := range []time.Time{occAt(2026, 8, 1, 10), occAt(2026, 8, 2, 10), occAt(2026, 8, 3, 10)} {
		if !got[i].Equal(want) {
			t.Errorf("occurrence %d = %v, want %v", i, got[i], want)
		}
	}
}

func TestOccurrencesInRangeHonoursExceptions(t *testing.T) {
	rrule := "FREQ=DAILY"
	ev := nonRecurring(occAt(2026, 8, 1, 10))
	ev.Rrule = &rrule
	got := OccurrencesInRange(ev, occAt(2026, 8, 1, 0), occAt(2026, 8, 4, 0),
		[]time.Time{occAt(2026, 8, 2, 10)})
	if len(got) != 2 {
		t.Fatalf("skipped occurrence still present: %v", got)
	}
}
