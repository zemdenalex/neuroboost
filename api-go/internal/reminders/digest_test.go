package reminders

import (
	"strings"
	"testing"
	"time"
)

// msk() is the Europe/Moscow helper already defined in quiet_test.go.

// The bug this whole file exists for: insertDigest used to store an empty
// message, and Telegram rejects an empty send with "Bad Request: message text
// is empty". Every digest failed, every morning, and the only trace was a
// FAILED row nobody read. An empty digest must still be a sendable message.
func TestDigestTextIsNeverEmpty(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)

	got := DigestText(day, loc, nil, nil)

	if strings.TrimSpace(got) == "" {
		t.Fatal("digest text is empty; Telegram would reject the send")
	}
	if !strings.Contains(got, "Nothing scheduled") {
		t.Errorf("an empty day should say so plainly, got:\n%s", got)
	}
}

func TestDigestTextListsEventsInTimeOrder(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)
	events := []DigestEvent{
		{Title: "Второе", StartsAt: day.Add(14 * time.Hour), EndsAt: day.Add(15 * time.Hour)},
		{Title: "Первое", StartsAt: day.Add(9 * time.Hour), EndsAt: day.Add(10 * time.Hour)},
	}

	got := DigestText(day, loc, events, nil)

	first := strings.Index(got, "Первое")
	second := strings.Index(got, "Второе")
	if first == -1 || second == -1 {
		t.Fatalf("both events must appear, got:\n%s", got)
	}
	if first > second {
		t.Errorf("events must be ordered by start time, got:\n%s", got)
	}
	if !strings.Contains(got, "09:00") || !strings.Contains(got, "10:00") {
		t.Errorf("a timed event shows its start and end in local time, got:\n%s", got)
	}
}

// The times stored in the row are UTC instants; a digest that printed them
// raw would tell a Moscow user their 09:00 meeting is at 06:00.
func TestDigestTextRendersTimesInTheUsersZone(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)
	events := []DigestEvent{{
		Title:    "Планёрка",
		StartsAt: time.Date(2026, 8, 10, 6, 0, 0, 0, time.UTC), // 09:00 MSK
		EndsAt:   time.Date(2026, 8, 10, 7, 0, 0, 0, time.UTC),
	}}

	got := DigestText(day, loc, events, nil)

	if !strings.Contains(got, "09:00") {
		t.Errorf("expected the Moscow time 09:00, got:\n%s", got)
	}
	if strings.Contains(got, "06:00") {
		t.Errorf("UTC leaked into the digest, got:\n%s", got)
	}
}

func TestDigestTextMarksAllDayEventsWithoutATime(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)
	events := []DigestEvent{{Title: "Отпуск", StartsAt: day, EndsAt: day.Add(24 * time.Hour), AllDay: true}}

	got := DigestText(day, loc, events, nil)

	if !strings.Contains(got, "all day") {
		t.Errorf("an all-day event is labelled, not timed, got:\n%s", got)
	}
	if strings.Contains(got, "00:00") {
		t.Errorf("an all-day event must not print a clock time, got:\n%s", got)
	}
}

func TestDigestTextListsTasksDueToday(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)
	tasks := []DigestTask{{Title: "Дописать отчёт"}, {Title: "Позвонить в банк"}}

	got := DigestText(day, loc, nil, tasks)

	for _, want := range []string{"Дописать отчёт", "Позвонить в банк"} {
		if !strings.Contains(got, want) {
			t.Errorf("task %q missing from digest:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Nothing scheduled") {
		t.Errorf("a day with tasks is not empty, got:\n%s", got)
	}
}

// Telegram caps a message at 4096 characters and rejects anything longer, so
// a busy day must truncate rather than fail to send at all.
func TestDigestTextStaysWithinTelegramsLimit(t *testing.T) {
	loc := msk()
	day := time.Date(2026, 8, 10, 0, 0, 0, 0, loc)
	var events []DigestEvent
	for i := 0; i < 400; i++ {
		events = append(events, DigestEvent{
			Title:    strings.Repeat("длинное название события ", 4),
			StartsAt: day.Add(time.Duration(i) * time.Minute),
			EndsAt:   day.Add(time.Duration(i+30) * time.Minute),
		})
	}

	got := DigestText(day, loc, events, nil)

	if n := len([]rune(got)); n > 4096 {
		t.Errorf("digest is %d runes, Telegram rejects over 4096", n)
	}
	if !strings.Contains(got, "more") {
		t.Errorf("a truncated digest must say something was omitted, got tail:\n%s", got[len(got)-200:])
	}
}
