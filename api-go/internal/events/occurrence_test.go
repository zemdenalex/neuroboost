package events

import (
	"testing"
	"time"
)

func recurringParent(rrule string, start time.Time, durMin int) Event {
	r := rrule
	return Event{
		ID:       "11111111-1111-1111-1111-111111111111",
		Title:    "Standup",
		StartsAt: start,
		EndsAt:   start.Add(time.Duration(durMin) * time.Minute),
		Rrule:    &r,
		Timezone: "Europe/Moscow",
		Tags:     []string{},
	}
}

func TestOccurrenceWindowResolvesTheRequestedDate(t *testing.T) {
	parent := recurringParent("FREQ=DAILY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)

	start, end, ok := occurrenceWindow(parent, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC))
	if !ok {
		t.Fatal("expected the 2026-08-05 occurrence to resolve")
	}
	if want := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Errorf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 8, 5, 9, 30, 0, 0, time.UTC); !end.Equal(want) {
		t.Errorf("end = %v, want %v", end, want)
	}
}

// The ID's date is the basis expandRecurrence used, so a date the rule does not
// land on must not silently resolve to a neighbouring occurrence — that would
// detach the wrong day.
func TestOccurrenceWindowRejectsADateTheRuleSkips(t *testing.T) {
	parent := recurringParent("FREQ=WEEKLY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)

	if _, _, ok := occurrenceWindow(parent, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)); ok {
		t.Error("Wednesday resolved against a weekly Monday rule")
	}
}

// An occurrence that was already skipped must still resolve: otherwise editing a
// detached occurrence a second time would 404.
func TestOccurrenceWindowIgnoresExistingExceptions(t *testing.T) {
	parent := recurringParent("FREQ=DAILY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)

	if _, _, ok := occurrenceWindow(parent, time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)); !ok {
		t.Error("expected the occurrence to resolve regardless of exception rows")
	}
}

func TestApplySeriesDeltaShiftsTheParentByTheSameAmount(t *testing.T) {
	parentStart := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	parentEnd := parentStart.Add(30 * time.Minute)
	occStart := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	occEnd := occStart.Add(30 * time.Minute)

	newStart := occStart.Add(45 * time.Minute)
	newEnd := occEnd.Add(45 * time.Minute)

	start, end := applySeriesDelta(parentStart, parentEnd, occStart, occEnd, &newStart, &newEnd)

	// The parent keeps its own date and moves 45 minutes — it does NOT re-anchor
	// to the edited occurrence's date, which is what an absolute write would do.
	if want := time.Date(2026, 8, 3, 9, 45, 0, 0, time.UTC); !start.Equal(want) {
		t.Errorf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 8, 3, 10, 15, 0, 0, time.UTC); !end.Equal(want) {
		t.Errorf("end = %v, want %v", end, want)
	}
}

func TestApplySeriesDeltaLeavesUntouchedBoundsAlone(t *testing.T) {
	parentStart := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	parentEnd := parentStart.Add(30 * time.Minute)
	occStart := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	occEnd := occStart.Add(30 * time.Minute)

	newEnd := occEnd.Add(15 * time.Minute) // a resize: only the end moves

	start, end := applySeriesDelta(parentStart, parentEnd, occStart, occEnd, nil, &newEnd)

	if !start.Equal(parentStart) {
		t.Errorf("start = %v, want it unchanged at %v", start, parentStart)
	}
	if want := parentEnd.Add(15 * time.Minute); !end.Equal(want) {
		t.Errorf("end = %v, want %v", end, want)
	}
}

func TestMergeOccurrenceCopiesTheParentAndDropsTheRule(t *testing.T) {
	desc := "daily sync"
	loc := "room 1"
	parent := recurringParent("FREQ=DAILY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)
	parent.Description = &desc
	parent.Location = &loc
	parent.Tags = []string{"work"}
	parent.ReminderOffsets = []int{10, 60}
	parent.IsWorkEvent = true

	occStart := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	occEnd := occStart.Add(30 * time.Minute)

	got := mergeOccurrence(parent, occStart, occEnd, UpdateEventRequest{})

	if got.Rrule != nil {
		t.Error("the detached copy must not carry the recurrence rule")
	}
	if got.Title != parent.Title || got.Description == nil || *got.Description != desc {
		t.Error("expected the parent's content to be copied")
	}
	// Re-deriving the user's default preset here would silently change the
	// reminders on an occurrence the user only moved.
	if len(got.ReminderOffsets) != 2 || got.ReminderOffsets[0] != 10 {
		t.Errorf("reminder offsets = %v, want the parent's", got.ReminderOffsets)
	}
	if !got.StartsAt.Equal(occStart) || !got.EndsAt.Equal(occEnd) {
		t.Error("expected the occurrence's own times, not the parent's")
	}
}

func TestMergeOccurrenceAppliesTheRequestedEdits(t *testing.T) {
	parent := recurringParent("FREQ=DAILY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)
	parent.Tags = []string{"work"}

	occStart := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	title := "Standup (moved)"
	color := "#ff0000"
	offsets := []int{5}

	got := mergeOccurrence(parent, occStart, occStart.Add(30*time.Minute), UpdateEventRequest{
		Title:           &title,
		Color:           &color,
		ReminderOffsets: &offsets,
	})

	if got.Title != title {
		t.Errorf("title = %q, want %q", got.Title, title)
	}
	if got.Color == nil || *got.Color != color {
		t.Error("expected the requested colour")
	}
	if len(got.ReminderOffsets) != 1 || got.ReminderOffsets[0] != 5 {
		t.Errorf("reminder offsets = %v, want [5]", got.ReminderOffsets)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "work" {
		t.Errorf("tags = %v, want the parent's when the request omits them", got.Tags)
	}
}

// A request that sets a recurrence rule on a single occurrence would turn the
// detached copy into a second series. The occurrence path must ignore it.
func TestMergeOccurrenceNeverAcceptsARuleFromTheRequest(t *testing.T) {
	parent := recurringParent("FREQ=DAILY", time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC), 30)
	rogue := "FREQ=WEEKLY"

	got := mergeOccurrence(parent, parent.StartsAt, parent.EndsAt, UpdateEventRequest{Rrule: &rogue})

	if got.Rrule != nil {
		t.Errorf("rrule = %v, want nil", *got.Rrule)
	}
}
