package parse

import (
	"testing"
	"time"
)

// Thursday 13 August 2026, 01:40 local — the hour this was written, which is
// also the awkward case: "19:00" with no day must mean today, not yesterday.
func base() time.Time {
	return time.Date(2026, 8, 13, 1, 40, 0, 0, time.UTC)
}

func TestParseTitleAndTime(t *testing.T) {
	got := Parse("Ужин завтра 19:00", base())
	if got.NeedsTime {
		t.Fatal("a line with a day and a time must not need asking")
	}
	if got.Title != "Ужин" {
		t.Errorf("title = %q, want %q", got.Title, "Ужин")
	}
	want := time.Date(2026, 8, 14, 19, 0, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v", got.Start, want)
	}
	if got.End.Sub(got.Start) != time.Hour {
		t.Errorf("default duration = %v, want 1h", got.End.Sub(got.Start))
	}
}

// The whole point of the choice: no time means ask, never guess.
func TestParseWithoutTimeAsksRatherThanGuessing(t *testing.T) {
	got := Parse("Ужин", base())
	if !got.NeedsTime {
		t.Fatal("a line with no time must report NeedsTime")
	}
	if got.Title != "Ужин" {
		t.Errorf("title = %q, want %q", got.Title, "Ужин")
	}
	if !got.Start.IsZero() {
		t.Errorf("start must stay zero when there is no time, got %v", got.Start)
	}
}

// A bare time that has already passed today means tomorrow — nobody schedules
// into the past deliberately.
func TestParseBareTimeAlreadyPassedRollsToTomorrow(t *testing.T) {
	// base() is 01:40; 00:30 today is behind us.
	got := Parse("Зарядка 00:30", base())
	want := time.Date(2026, 8, 14, 0, 30, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v (tomorrow)", got.Start, want)
	}
}

// ...but a bare time still ahead today stays today.
func TestParseBareTimeStillAheadStaysToday(t *testing.T) {
	got := Parse("Созвон 09:00", base())
	want := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v (today)", got.Start, want)
	}
}

func TestParseExplicitRange(t *testing.T) {
	got := Parse("Созвон 19:00-20:30", base())
	if got.NeedsTime {
		t.Fatal("a range is a time")
	}
	if got.End.Sub(got.Start) != 90*time.Minute {
		t.Errorf("duration = %v, want 1h30m", got.End.Sub(got.Start))
	}
	if got.Title != "Созвон" {
		t.Errorf("title = %q", got.Title)
	}
}

// A range crossing midnight ends the next day, not before it began.
func TestParseRangeCrossingMidnight(t *testing.T) {
	got := Parse("Смена 23:00-01:00", base())
	if !got.End.After(got.Start) {
		t.Fatalf("end %v must be after start %v", got.End, got.Start)
	}
	if got.End.Sub(got.Start) != 2*time.Hour {
		t.Errorf("duration = %v, want 2h", got.End.Sub(got.Start))
	}
}

func TestParseExplicitDate(t *testing.T) {
	got := Parse("14.08 19:00 Ужин", base())
	want := time.Date(2026, 8, 14, 19, 0, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v", got.Start, want)
	}
	if got.Title != "Ужин" {
		t.Errorf("title = %q, want %q — the date must be cut out of it", got.Title, "Ужин")
	}
}

// A day.month already behind us means next year, not a date in the past.
func TestParseDateAlreadyPassedMeansNextYear(t *testing.T) {
	got := Parse("01.02 10:00 Годовщина", base())
	if got.Start.Year() != 2027 {
		t.Errorf("year = %d, want 2027", got.Start.Year())
	}
}

// "послезавтра" must not be read as "завтра" because one contains the other.
func TestParseLongestDayWordWins(t *testing.T) {
	got := Parse("Врач послезавтра 11:00", base())
	want := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	if !got.Start.Equal(want) {
		t.Errorf("start = %v, want %v", got.Start, want)
	}
	if got.Title != "Врач" {
		t.Errorf("title = %q — the day word must be cut out", got.Title)
	}
}

// Nonsense hours are not a time; asking beats inventing 25:00.
func TestParseRejectsImpossibleClock(t *testing.T) {
	got := Parse("Встреча 25:70", base())
	if !got.NeedsTime {
		t.Error("25:70 is not a time — must ask instead of accepting it")
	}
}

func TestParseEmptyLine(t *testing.T) {
	if got := Parse("   ", base()); !got.NeedsTime || got.Title != "" {
		t.Errorf("empty line: got %+v", got)
	}
}
