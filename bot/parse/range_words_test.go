package parse

import (
	"testing"
	"time"
)

// Denis, 23.09 (pass 3), from his chat: «завтра с 01:00 до 22:00 работать»
// became 01:00 with no end and the title «с до 22:00 работать». A dash range
// always worked; the words «с … до …» did not.
func TestARangeInWords(t *testing.T) {
	now := time.Date(2026, 9, 22, 15, 38, 0, 0, time.UTC)
	cases := []struct {
		line, title string
		start, end  string
	}{
		{"завтра с 01:00 до 22:00 работать", "работать", "01:00", "22:00"},
		{"завтра работать с 01:00 до 22:00", "работать", "01:00", "22:00"},
		{"работать с 14:00 до 15:30", "работать", "14:00", "15:30"},
		{"завтра с 10 до 12 работать", "работать", "10:00", "12:00"},
		{"meeting from 10:00 to 11:30", "meeting", "10:00", "11:30"},
		{"созвон в 15:00 до 16:00", "созвон", "15:00", "16:00"},
	}
	for _, c := range cases {
		p := ParseLine(c.line, now)
		got := [3]string{p.Title, p.Draft.StartsAt().Format("15:04"), p.Draft.EndsAt().Format("15:04")}
		if !p.Draft.HasEnd || got != [3]string{c.title, c.start, c.end} {
			t.Errorf("%q → title %q %s–%s (hasEnd %v), want %q %s–%s",
				c.line, got[0], got[1], got[2], p.Draft.HasEnd, c.title, c.start, c.end)
		}
	}
}

// What «с» must NOT become: company («с 2 друзьями») stays in the title, and a
// span of DATES stays a span.
func TestRangeWordsDoNotTakeWhatIsNotATime(t *testing.T) {
	now := time.Date(2026, 9, 22, 15, 38, 0, 0, time.UTC)
	if p := ParseLine("встреча с 2 друзьями", now); p.Draft.HasTime || p.Title != "встреча с 2 друзьями" {
		t.Errorf("«с 2 друзьями» read as a time: hasTime %v, title %q", p.Draft.HasTime, p.Title)
	}
	if p := ParseLine("отпуск с 14.10 до 16.10", now); p.Draft.EndDay.IsZero() || p.Draft.HasTime {
		t.Errorf("a date span became a clock range: endDay %v, hasTime %v", p.Draft.EndDay, p.Draft.HasTime)
	}
}
