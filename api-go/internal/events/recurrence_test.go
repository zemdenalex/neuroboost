package events

import (
	"testing"
	"time"
)

func dt(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, time.UTC)
}

func mkEvent(id string, start time.Time, dur time.Duration, rrule string) Event {
	rr := rrule
	return Event{ID: id, StartsAt: start, EndsAt: start.Add(dur), Rrule: &rr}
}

var (
	wideStart = dt(2026, 1, 1, 0, 0)
	wideEnd   = dt(2027, 1, 1, 0, 0)
)

func dates(insts []Event) []string {
	out := make([]string, len(insts))
	for i, e := range insts {
		out[i] = e.StartsAt.Format("2006-01-02")
	}
	return out
}

func TestParseRRule(t *testing.T) {
	r, err := parseRRule("FREQ=WEEKLY;INTERVAL=2;COUNT=5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Freq != "WEEKLY" || r.Interval != 2 || r.Count == nil || *r.Count != 5 {
		t.Fatalf("parsed wrong: %+v", r)
	}

	// lowercase + default interval + UNTIL
	r2, err := parseRRule("freq=daily;until=2026-06-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r2.Freq != "DAILY" || r2.Interval != 1 || r2.Until == nil {
		t.Fatalf("parsed wrong: %+v", r2)
	}
}

func TestParseRRuleErrors(t *testing.T) {
	bad := []string{
		"",                      // empty → missing FREQ
		"INTERVAL=2",            // no FREQ
		"FREQ=YEARLY",           // unsupported FREQ
		"FREQ=DAILY;INTERVAL=0", // interval < 1
		"FREQ=DAILY;COUNT=x",    // non-numeric count
		"FREQ=DAILY;UNTIL=nope", // bad date
		"FREQ=DAILY;FOO=1",      // unknown key
		"FREQ=DAILY;malformed",  // no '='
	}
	for _, s := range bad {
		if _, err := parseRRule(s); err == nil {
			t.Errorf("expected error for %q, got nil", s)
		}
	}
}

func TestExpandDailyCount(t *testing.T) {
	e := mkEvent("e1", dt(2026, 6, 1, 10, 0), time.Hour, "FREQ=DAILY;COUNT=3")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-06-01", "2026-06-02", "2026-06-03"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("dates = %v, want %v", g, want)
	}
	for i, inst := range got {
		if inst.EndsAt.Sub(inst.StartsAt) != time.Hour {
			t.Errorf("instance %d duration not preserved", i)
		}
		if inst.ID != "e1:"+want[i] {
			t.Errorf("instance %d id = %q, want %q", i, inst.ID, "e1:"+want[i])
		}
		if inst.RecurringEventID == nil || *inst.RecurringEventID != "e1" {
			t.Errorf("instance %d RecurringEventID not set to parent", i)
		}
	}
}

func TestExpandWeeklyInterval(t *testing.T) {
	e := mkEvent("w", dt(2026, 6, 1, 9, 0), 30*time.Minute, "FREQ=WEEKLY;INTERVAL=2;COUNT=3")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-06-01", "2026-06-15", "2026-06-29"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("dates = %v, want %v", g, want)
	}
}

func TestExpandMonthly(t *testing.T) {
	e := mkEvent("m", dt(2026, 1, 15, 8, 0), time.Hour, "FREQ=MONTHLY;COUNT=3")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-01-15", "2026-02-15", "2026-03-15"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("dates = %v, want %v", g, want)
	}
}

// MONTHLY must anchor to the start day-of-month and SKIP months that lack it,
// per RFC 5545 §3.3.10 — not drift into the next month (Jan-31 + 1mo ≠ Mar-3).
func TestExpandMonthlyDay31SkipsShortMonths(t *testing.T) {
	e := mkEvent("m31", dt(2026, 1, 31, 9, 0), time.Hour, "FREQ=MONTHLY;COUNT=4")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	// Feb, Apr, Jun have no 31st → skipped.
	want := []string{"2026-01-31", "2026-03-31", "2026-05-31", "2026-07-31"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("Jan-31 monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandMonthlyDay30SkipsFebruary(t *testing.T) {
	e := mkEvent("m30", dt(2026, 1, 30, 9, 0), time.Hour, "FREQ=MONTHLY;COUNT=3")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-01-30", "2026-03-30", "2026-04-30"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("Jan-30 monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandMonthlyDay29NonLeapSkipsFebruary(t *testing.T) {
	// 2026 is not a leap year → February has 28 days, so the 29th is skipped.
	e := mkEvent("m29", dt(2026, 1, 29, 8, 0), time.Hour, "FREQ=MONTHLY;COUNT=3")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-01-29", "2026-03-29", "2026-04-29"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("Jan-29 monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandMonthlyIntervalPreservesDay(t *testing.T) {
	// Every 2 months from Jan-31; all target months (Mar, May, Jul) have a 31st.
	e := mkEvent("m31i2", dt(2026, 1, 31, 9, 0), time.Hour, "FREQ=MONTHLY;INTERVAL=2;COUNT=4")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-01-31", "2026-03-31", "2026-05-31", "2026-07-31"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("Jan-31 bi-monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandMonthlyCrossesYearBoundary(t *testing.T) {
	// Exercises the total/12 year rollover: Nov → Dec → Jan(next year).
	e := mkEvent("mny", dt(2026, 11, 30, 9, 0), time.Hour, "FREQ=MONTHLY;COUNT=3")
	got := expandRecurrence(e, wideStart, dt(2027, 6, 1, 0, 0), nil)
	want := []string{"2026-11-30", "2026-12-30", "2027-01-30"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("year-crossing monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandMonthlyLeapDayAnchor(t *testing.T) {
	// Feb-29 in a leap year is a valid anchor; following months keep day 29.
	e := mkEvent("m29l", dt(2028, 2, 29, 8, 0), time.Hour, "FREQ=MONTHLY;COUNT=3")
	got := expandRecurrence(e, dt(2028, 1, 1, 0, 0), dt(2028, 12, 1, 0, 0), nil)
	want := []string{"2028-02-29", "2028-03-29", "2028-04-29"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("leap-day anchor monthly; dates = %v, want %v", g, want)
	}
}

func TestExpandRangeFiltering(t *testing.T) {
	e := mkEvent("r", dt(2026, 6, 1, 10, 0), time.Hour, "FREQ=DAILY;COUNT=10")
	got := expandRecurrence(e, dt(2026, 6, 3, 0, 0), dt(2026, 6, 5, 0, 0), nil)
	want := []string{"2026-06-03", "2026-06-04"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("dates = %v, want %v", g, want)
	}
}

// A long-running recurring event must still appear in a query window far in the
// future. Regression test for a cap that counted occurrences from the event's
// StartsAt: a daily event would hit the 366 safety cap ~1 year after its start —
// before the loop ever reached the requested window — and silently vanish from
// the calendar. This is the ONE test here that fails on the pre-fix code; the
// other two below are forward guards (bounded payload, COUNT exhaustion) that
// pass on both versions.
func TestExpandDailyLongRunningStillAppearsInWindow(t *testing.T) {
	e := mkEvent("lr", dt(2026, 1, 1, 10, 0), time.Hour, "FREQ=DAILY")
	// Window ~424 days after the start — well past the old 366 cap.
	got := expandRecurrence(e, dt(2027, 3, 1, 0, 0), dt(2027, 3, 2, 0, 0), nil)
	want := []string{"2027-03-01"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("long-running daily; dates = %v, want %v", g, want)
	}
}

// The number of instances returned for a single event is bounded even when the
// query window is enormous (defends against unbounded expansion / huge payloads).
func TestExpandDailyResponseBounded(t *testing.T) {
	e := mkEvent("hb", dt(2026, 1, 1, 10, 0), time.Hour, "FREQ=DAILY")
	got := expandRecurrence(e, dt(2026, 1, 1, 0, 0), dt(2031, 1, 1, 0, 0), nil)
	if len(got) > 366 {
		t.Fatalf("expected at most 366 instances, got %d", len(got))
	}
	if len(got) == 0 {
		t.Fatal("expected a bounded but non-empty set of instances")
	}
}

// COUNT counts from the event's start, so once all occurrences are exhausted the
// event must not reappear in a later window.
func TestExpandCountRespectedAfterAllOccurrences(t *testing.T) {
	e := mkEvent("ce", dt(2026, 1, 1, 10, 0), time.Hour, "FREQ=DAILY;COUNT=3")
	got := expandRecurrence(e, dt(2027, 1, 1, 0, 0), dt(2027, 2, 1, 0, 0), nil)
	if len(got) != 0 {
		t.Fatalf("COUNT=3 exhausted by 2026-01-03; window in 2027 should be empty, got %v", dates(got))
	}
}

func TestExpandExceptionsSkip(t *testing.T) {
	e := mkEvent("x", dt(2026, 6, 1, 10, 0), time.Hour, "FREQ=DAILY;COUNT=3")
	exc := []time.Time{dt(2026, 6, 2, 0, 0)}
	got := expandRecurrence(e, wideStart, wideEnd, exc)
	want := []string{"2026-06-01", "2026-06-03"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("dates = %v, want %v", g, want)
	}
}

// UNTIL with a DATE value is inclusive of the whole final day. An occurrence on
// the UNTIL date at a non-midnight time must NOT be dropped.
func TestExpandUntilIncludesLastDay(t *testing.T) {
	e := mkEvent("u", dt(2026, 6, 1, 10, 0), time.Hour, "FREQ=DAILY;UNTIL=2026-06-03")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-06-01", "2026-06-02", "2026-06-03"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("UNTIL=2026-06-03 should include Jun 1–3; dates = %v, want %v", g, want)
	}
}

// Closest case to the old fencepost: a midnight-start event on the UNTIL day
// (occurrence == UNTIL-day 00:00) must be included.
func TestExpandUntilMidnightStartOnUntilDay(t *testing.T) {
	e := mkEvent("um", dt(2026, 6, 1, 0, 0), time.Hour, "FREQ=DAILY;UNTIL=2026-06-02")
	got := expandRecurrence(e, wideStart, wideEnd, nil)
	want := []string{"2026-06-01", "2026-06-02"}
	if g := dates(got); !equal(g, want) {
		t.Fatalf("midnight-start UNTIL day; dates = %v, want %v", g, want)
	}
}

func TestIsException(t *testing.T) {
	exc := []time.Time{dt(2026, 6, 2, 0, 0)}
	if !isException(dt(2026, 6, 2, 10, 0), exc) {
		t.Error("same calendar date should be treated as an exception")
	}
	if isException(dt(2026, 6, 3, 10, 0), exc) {
		t.Error("a different date should not be an exception")
	}
}

func TestBuildRRuleRoundTrip(t *testing.T) {
	c := 5
	u := dt(2026, 6, 1, 0, 0)
	s := buildRRule("weekly", 2, &c, &u)
	r, err := parseRRule(s)
	if err != nil {
		t.Fatalf("round-trip parse failed for %q: %v", s, err)
	}
	if r.Freq != "WEEKLY" || r.Interval != 2 || r.Count == nil || *r.Count != 5 || r.Until == nil {
		t.Fatalf("round-trip mismatch: %q → %+v", s, r)
	}
	// INTERVAL=1 is omitted.
	if got := buildRRule("daily", 1, nil, nil); got != "FREQ=DAILY" {
		t.Errorf("buildRRule(daily,1,nil,nil) = %q, want FREQ=DAILY", got)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
