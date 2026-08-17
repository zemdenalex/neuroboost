package reminders

import (
	"testing"
	"time"
)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

func TestDueRemindersMatchesOffsetInsideWindow(t *testing.T) {
	occ := at(2026, 8, 1, 10, 0)
	// Window is the minute in which the 60-minute reminder comes due.
	got := DueReminders([]time.Time{occ}, []int{1440, 60}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 1))
	if len(got) != 1 {
		t.Fatalf("want 1 due reminder, got %d: %+v", len(got), got)
	}
	if got[0].MinutesBefore != 60 {
		t.Errorf("MinutesBefore = %d, want 60", got[0].MinutesBefore)
	}
	if !got[0].RemindAt.Equal(at(2026, 8, 1, 9, 0)) {
		t.Errorf("RemindAt = %v, want 09:00", got[0].RemindAt)
	}
	if !got[0].OccurrenceStart.Equal(occ) {
		t.Errorf("OccurrenceStart = %v, want %v", got[0].OccurrenceStart, occ)
	}
}

func TestDueRemindersIsHalfOpen(t *testing.T) {
	occ := at(2026, 8, 1, 10, 0)
	// remind_at is exactly 09:00. A window starting at 09:00 includes it;
	// a window ending at 09:00 does not. Without this, consecutive scan
	// windows would both claim the same reminder — harmless thanks to the
	// unique index, but it would hide real bugs behind ON CONFLICT.
	if n := len(DueReminders([]time.Time{occ}, []int{60}, at(2026, 8, 1, 9, 0), at(2026, 8, 1, 9, 1))); n != 1 {
		t.Errorf("window starting exactly at remind_at: got %d, want 1", n)
	}
	if n := len(DueReminders([]time.Time{occ}, []int{60}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 0))); n != 0 {
		t.Errorf("window ending exactly at remind_at: got %d, want 0", n)
	}
}

func TestDueRemindersEveryOccurrenceOfARecurringSeries(t *testing.T) {
	// Three daily occurrences; only the one whose reminder lands in the
	// window is due. This is the property that makes recurring events work:
	// the result carries which occurrence it belongs to.
	occs := []time.Time{
		at(2026, 8, 1, 10, 0),
		at(2026, 8, 2, 10, 0),
		at(2026, 8, 3, 10, 0),
	}
	got := DueReminders(occs, []int{60}, at(2026, 8, 2, 8, 59), at(2026, 8, 2, 9, 1))
	if len(got) != 1 {
		t.Fatalf("want 1, got %d", len(got))
	}
	if !got[0].OccurrenceStart.Equal(at(2026, 8, 2, 10, 0)) {
		t.Errorf("wrong occurrence: %v", got[0].OccurrenceStart)
	}
}

func TestDueRemindersMultipleOffsetsCanFireTogether(t *testing.T) {
	// One window, two different occurrences coming due via two different
	// offsets — both must survive as separate rows.
	//   Aug 1 10:00 − 60min   = Aug 1 09:00  (in window)
	//   Aug 2 09:00 − 1440min = Aug 1 09:00  (in window, same instant)
	occs := []time.Time{at(2026, 8, 1, 10, 0), at(2026, 8, 2, 9, 0)}
	got := DueReminders(occs, []int{60, 1440}, at(2026, 8, 1, 8, 59), at(2026, 8, 1, 9, 1))
	if len(got) != 2 {
		t.Fatalf("want 2 (60min before Aug1, 1440min before Aug2), got %d: %+v", len(got), got)
	}
	// They collide on remind_at but not on the dedupe key, because the key
	// includes occurrence_start — which is exactly why two reminders for
	// different occurrences can share a minute.
	if got[0].OccurrenceStart.Equal(got[1].OccurrenceStart) {
		t.Errorf("both rows claim the same occurrence: %+v", got)
	}
}

func TestDueRemindersEmptyInputs(t *testing.T) {
	if n := len(DueReminders(nil, []int{60}, at(2026, 8, 1, 0, 0), at(2026, 8, 1, 1, 0))); n != 0 {
		t.Errorf("no occurrences should yield nothing, got %d", n)
	}
	if n := len(DueReminders([]time.Time{at(2026, 8, 1, 10, 0)}, nil, at(2026, 8, 1, 0, 0), at(2026, 8, 2, 0, 0))); n != 0 {
		t.Errorf("no offsets should yield nothing, got %d", n)
	}
}

func TestDueRemindersIgnoresNegativeOffsets(t *testing.T) {
	// -1 is the snooze sentinel and -2 the digest sentinel in the reminder
	// table; neither may ever be read as "N minutes after the event".
	occ := at(2026, 8, 1, 10, 0)
	if n := len(DueReminders([]time.Time{occ}, []int{-1, -2}, at(2026, 8, 1, 9, 59), at(2026, 8, 1, 10, 3))); n != 0 {
		t.Errorf("negative offsets must be ignored, got %d", n)
	}
}
