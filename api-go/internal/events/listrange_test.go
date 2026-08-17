package events

import (
	"testing"
	"time"
)

func TestParseListRangeExplicit(t *testing.T) {
	now := dt(2026, 6, 17, 12, 0)
	s, e, code, _ := parseListRange("2026-06-01T09:00:00Z", "2026-06-02T09:00:00Z", now)
	if code != "" {
		t.Fatalf("valid range errored: %s", code)
	}
	if !s.Equal(dt(2026, 6, 1, 9, 0)) || !e.Equal(dt(2026, 6, 2, 9, 0)) {
		t.Fatalf("parsed wrong: %v .. %v", s, e)
	}
}

func TestParseListRangeDefaultsToWeek(t *testing.T) {
	now := dt(2026, 6, 17, 12, 0)
	s, e, code, _ := parseListRange("", "", now)
	if code != "" {
		t.Fatalf("default errored: %s", code)
	}
	if s.Weekday() != time.Sunday {
		t.Errorf("week start weekday = %v, want Sunday", s.Weekday())
	}
	if e.Sub(s) != 7*24*time.Hour {
		t.Errorf("week span = %v, want 168h", e.Sub(s))
	}
	if s.After(now) || e.Before(now) {
		t.Errorf("now %v not within default week [%v, %v]", now, s, e)
	}
	// A single empty param also falls back to the full week (matches prior behaviour).
	if _, _, c, _ := parseListRange("2026-06-01T09:00:00Z", "", now); c != "" {
		t.Errorf("one empty param should default to week, got code %q", c)
	}
}

func TestParseListRangeBadFormats(t *testing.T) {
	now := dt(2026, 6, 17, 12, 0)
	if _, _, code, _ := parseListRange("nope", "2026-06-02T09:00:00Z", now); code != "INVALID_START" {
		t.Errorf("bad start code = %q, want INVALID_START", code)
	}
	if _, _, code, _ := parseListRange("2026-06-01T09:00:00Z", "nope", now); code != "INVALID_END" {
		t.Errorf("bad end code = %q, want INVALID_END", code)
	}
}

func TestParseListRangeRejectsInvertedAndEqual(t *testing.T) {
	now := dt(2026, 6, 17, 12, 0)
	if _, _, code, _ := parseListRange("2026-06-02T09:00:00Z", "2026-06-01T09:00:00Z", now); code != "INVALID_RANGE" {
		t.Errorf("inverted range code = %q, want INVALID_RANGE", code)
	}
	if _, _, code, _ := parseListRange("2026-06-01T09:00:00Z", "2026-06-01T09:00:00Z", now); code != "INVALID_RANGE" {
		t.Errorf("equal range code = %q, want INVALID_RANGE", code)
	}
}
