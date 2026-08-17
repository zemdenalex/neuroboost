package reminders

import (
	"testing"
	"time"
)

func msk() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		panic(err)
	}
	return loc
}

func TestQuietHoursLeavesDaytimeAlone(t *testing.T) {
	// 14:00 Moscow, quiet 22:00–07:00 — nothing to do.
	in := time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC) // 14:00 MSK
	out, ok := ShiftForQuietHours(in, 60, "22:00", "07:00", msk())
	if !ok || !out.Equal(in) {
		t.Fatalf("daytime reminder changed: out=%v ok=%v", out, ok)
	}
}

func TestQuietHoursShiftsToEndOfWindow(t *testing.T) {
	// 02:00 MSK falls inside 22:00–07:00; a day-ahead reminder is still
	// useful at 07:00, so shift rather than drop.
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC) // 02:00 MSK Aug 1
	out, ok := ShiftForQuietHours(in, 1440, "22:00", "07:00", msk())
	if !ok {
		t.Fatal("day-ahead reminder was dropped, want shifted")
	}
	want := time.Date(2026, 8, 1, 7, 0, 0, 0, msk())
	if !out.Equal(want) {
		t.Errorf("shifted to %v, want %v", out.In(msk()), want)
	}
}

func TestQuietHoursShiftsFromPreMidnightHalf(t *testing.T) {
	// 23:00 MSK is in the pre-midnight half of a wrapping window: the end of
	// quiet hours is 07:00 TOMORROW, not 07:00 today (which is in the past).
	in := time.Date(2026, 8, 1, 20, 0, 0, 0, time.UTC) // 23:00 MSK Aug 1
	out, ok := ShiftForQuietHours(in, 1440, "22:00", "07:00", msk())
	if !ok {
		t.Fatal("dropped, want shifted")
	}
	want := time.Date(2026, 8, 2, 7, 0, 0, 0, msk())
	if !out.Equal(want) {
		t.Errorf("shifted to %v, want %v (next morning)", out.In(msk()), want)
	}
}

func TestQuietHoursDropsShortNoticeReminders(t *testing.T) {
	// A "15 minutes before" reminder delivered at 07:00 for a 02:15 event is
	// worse than useless — it is a lie. Drop it instead.
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC) // 02:00 MSK
	if _, ok := ShiftForQuietHours(in, 15, "22:00", "07:00", msk()); ok {
		t.Error("<=15min reminder in quiet hours should be dropped")
	}
	if _, ok := ShiftForQuietHours(in, 16, "22:00", "07:00", msk()); !ok {
		t.Error("16min reminder should still be shifted, not dropped")
	}
}

func TestQuietHoursNonWrappingWindow(t *testing.T) {
	// Not everyone sleeps across midnight. 01:00–07:00 must work too.
	in := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) // 03:00 MSK
	out, ok := ShiftForQuietHours(in, 1440, "01:00", "07:00", msk())
	if !ok {
		t.Fatal("dropped, want shifted")
	}
	if got := out.In(msk()).Hour(); got != 7 {
		t.Errorf("shifted to hour %d, want 7", got)
	}
}

func TestQuietHoursDisabledByEmptyStrings(t *testing.T) {
	in := time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC)
	out, ok := ShiftForQuietHours(in, 15, "", "", msk())
	if !ok || !out.Equal(in) {
		t.Errorf("empty quiet hours must be a no-op, got out=%v ok=%v", out, ok)
	}
}

func TestDigestFiresOnceInLocalMidnightTerms(t *testing.T) {
	// 08:00 MSK on Aug 1 == 05:00 UTC.
	start := time.Date(2026, 8, 1, 4, 59, 0, 0, time.UTC)
	end := time.Date(2026, 8, 1, 5, 1, 0, 0, time.UTC)
	day, fireAt, ok := DigestDue(start, end, "08:00", msk())
	if !ok {
		t.Fatal("digest should be due in this window")
	}
	// The returned day is the LOCAL midnight the digest belongs to — that is
	// what goes into occurrence_start, so the unique index dedupes per local
	// day rather than per UTC day.
	wantDay := time.Date(2026, 8, 1, 0, 0, 0, 0, msk())
	if !day.Equal(wantDay) {
		t.Errorf("digest day = %v, want %v", day.In(msk()), wantDay)
	}

	// 🔴 The fire time is the point: the row used to be written with
	// remind_at = NOW(), and because this window looks AHEAD (it starts at
	// 04:59 for an 05:00 digest) the notifier's `remind_at <= NOW()` gate was
	// already satisfied — 08:00 arrived at 07:59.
	wantAt := time.Date(2026, 8, 1, 8, 0, 0, 0, msk())
	if !fireAt.Equal(wantAt) {
		t.Errorf("digest fires at %v, want %v", fireAt.In(msk()), wantAt)
	}
	if !fireAt.After(start) {
		t.Error("fireAt must be later than the window start, or nothing is gained")
	}
}

func TestDigestNotDueOutsideWindow(t *testing.T) {
	start := time.Date(2026, 8, 1, 6, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 1, 6, 1, 0, 0, time.UTC)
	if _, _, ok := DigestDue(start, end, "08:00", msk()); ok {
		t.Error("digest fired an hour late")
	}
}

func TestDigestAcrossLocalMidnight(t *testing.T) {
	// A window straddling local midnight must still find a digest set for
	// 00:00 — checking only "today" would miss it.
	start := time.Date(2026, 7, 31, 20, 59, 0, 0, time.UTC) // 23:59 MSK
	end := time.Date(2026, 7, 31, 21, 1, 0, 0, time.UTC)    // 00:01 MSK Aug 1
	day, _, ok := DigestDue(start, end, "00:00", msk())
	if !ok {
		t.Fatal("midnight digest missed across the day boundary")
	}
	if want := time.Date(2026, 8, 1, 0, 0, 0, 0, msk()); !day.Equal(want) {
		t.Errorf("digest day = %v, want %v", day.In(msk()), want)
	}
}

func TestDigestBadTimeIsNotDue(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	if _, _, ok := DigestDue(start, end, "nonsense", msk()); ok {
		t.Error("unparseable digest_at must not fire")
	}
}
