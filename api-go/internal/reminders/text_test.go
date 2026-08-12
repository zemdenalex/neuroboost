package reminders

import (
	"testing"
	"time"
)

func moscow(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skip("tzdata unavailable on this host")
	}
	return loc
}

// The exact shape Denis chose: context line, then the thing itself.
func TestReminderTextEventTwoLines(t *testing.T) {
	loc := moscow(t)
	occ := time.Date(2026, 8, 13, 1, 15, 0, 0, loc).UTC()

	got := ReminderText("EVENT", "YIIIIPIIIIEEEEEE", occ, 15, loc)
	want := "⏰ Через 15 минут — 01:15\nYIIIIPIIIIEEEEEE"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// A task says "срок" and carries its own marker, so a glance separates it from
// an event without reading the words.
func TestReminderTextTaskSaysDeadline(t *testing.T) {
	loc := moscow(t)
	occ := time.Date(2026, 8, 13, 18, 0, 0, 0, loc).UTC()

	got := ReminderText("TASK", "Купить молоко", occ, 60, loc)
	want := "⏰ Срок через 1 час — 18:00\n📋 Купить молоко"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// "Через 1440 минут" is not actionable; a day-away reminder names the day.
func TestReminderTextNamesTheDayWhenNotToday(t *testing.T) {
	loc := moscow(t)
	occ := time.Date(2026, 8, 14, 9, 0, 0, 0, loc).UTC()

	if got, want := ReminderText("EVENT", "Встреча с клиентом", occ, 1440, loc),
		"⏰ Завтра в 09:00\nВстреча с клиентом"; got != want {
		t.Errorf("one day out:\ngot  %q\nwant %q", got, want)
	}

	// A week out names the date rather than saying "завтра" of the wrong day.
	occFar := time.Date(2026, 8, 20, 9, 0, 0, 0, loc).UTC()
	if got, want := ReminderText("EVENT", "Отпуск", occFar, 10080, loc),
		"⏰ 20 августа в 09:00\nОтпуск"; got != want {
		t.Errorf("a week out:\ngot  %q\nwant %q", got, want)
	}
}

// Russian numerals are not English ones: 1 минуту / 2 минуты / 5 минут, and the
// 11-14 band takes the "many" form despite ending in 1-4.
func TestPluralRu(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "минуту"}, {2, "минуты"}, {4, "минуты"}, {5, "минут"},
		{11, "минут"}, {12, "минут"}, {14, "минут"},
		{21, "минуту"}, {22, "минуты"}, {25, "минут"},
		{101, "минуту"}, {111, "минут"},
	}
	for _, c := range cases {
		if got := pluralRu(c.n, "минуту", "минуты", "минут"); got != c.want {
			t.Errorf("pluralRu(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestReminderTextHoursAndMinutes(t *testing.T) {
	loc := moscow(t)
	occ := time.Date(2026, 8, 13, 12, 0, 0, 0, loc).UTC()

	if got, want := ReminderText("EVENT", "Созвон", occ, 90, loc),
		"⏰ Через 1 час 30 минут — 12:00\nСозвон"; got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// Snoozing must not keep repeating the interval it just invalidated.
func TestSnoozedTextReplacesOnlyTheContextLine(t *testing.T) {
	original := "⏰ Через 15 минут — 01:15\nYIIIIPIIIIEEEEEE"

	got := SnoozedText(original, 10)
	want := "😴 Отложено на 10 минут\nYIIIIPIIIIEEEEEE"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if want, bad := "10 минут", "Через 15"; !contains(got, want) || contains(got, bad) {
		t.Errorf("snoozed text must state the new delay and drop the old one: %q", got)
	}
}

// A message with no newline (anything written before this format existed) must
// still snooze into something sensible rather than losing its title.
func TestSnoozedTextHandlesLegacySingleLine(t *testing.T) {
	got := SnoozedText("YIIIIPIIIIEEEEEE", 10)
	want := "😴 Отложено на 10 минут\nYIIIIPIIIIEEEEEE"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
