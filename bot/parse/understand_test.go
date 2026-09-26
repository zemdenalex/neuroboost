package parse

import (
	"testing"
	"time"
)

var understandNow = time.Date(2026, time.September, 26, 10, 0, 0, 0, time.UTC)

func TestUnderstandAppliesTheUsersOwnWordsAfterTheBuiltIns(t *testing.T) {
	v := Vocabulary{
		Calendars: []Calendar{{ID: "c-work", Name: "Работа"}},
		Keywords:  map[string]Keyword{"созвон": {Field: "calendar", Value: "работа"}},
	}
	u := Understand("созвон с Петей завтра 15:00", understandNow, v)
	if u.Title != "с Петей" {
		t.Errorf("title = %q, want the custom word out of it", u.Title)
	}
	if u.CalendarID != "c-work" || u.CalendarName != "Работа" {
		t.Errorf("calendar = %q/%q, want the one the word names", u.CalendarID, u.CalendarName)
	}
	if !u.Draft.HasTime || u.Draft.Start != 15*time.Hour {
		t.Errorf("time = %v (has %v), want 15:00", u.Draft.Start, u.Draft.HasTime)
	}
}

// Negative control for the test above: without the word in the vocabulary the
// same line keeps it in the title and lands in no calendar.
func TestUnderstandWithoutTheWordKeepsItInTheTitle(t *testing.T) {
	u := Understand("созвон с Петей завтра 15:00", understandNow, Vocabulary{
		Calendars: []Calendar{{ID: "c-work", Name: "Работа"}},
	})
	if u.Title != "созвон с Петей" || u.CalendarID != "" {
		t.Errorf("got title %q calendar %q, want the word kept and no calendar", u.Title, u.CalendarID)
	}
}

func TestUnderstandReadsACalendarNamedInTheLine(t *testing.T) {
	u := Understand("отчёт завтра 10:00 календарь работа", understandNow, Vocabulary{
		Calendars: []Calendar{{ID: "c-home", Name: "Дом"}, {ID: "c-work", Name: "Работа"}},
	})
	if u.CalendarID != "c-work" || u.Title != "отчёт" {
		t.Errorf("got calendar %q title %q", u.CalendarID, u.Title)
	}
}

func TestUnderstandUsesAReminderPreset(t *testing.T) {
	u := Understand("врач завтра 9:00 важное", understandNow, Vocabulary{
		Presets: map[string][]int{"важное": {60, 1440}},
	})
	if u.Draft.ReminderOffsets == nil || len(*u.Draft.ReminderOffsets) != 2 {
		t.Fatalf("reminders = %v, want the preset", u.Draft.ReminderOffsets)
	}
	if u.Title != "врач" {
		t.Errorf("title = %q", u.Title)
	}
}

func TestPlainTaskIsALineWithoutAClockTime(t *testing.T) {
	r, ok := PlainTask("купить молоко завтра !1 30м #дом", understandNow)
	if !ok {
		t.Fatal("a line with no time must be a plain task")
	}
	if r.Title != "купить молоко" || r.Priority == nil || *r.Priority != 1 ||
		r.EstimatedMinutes == nil || *r.EstimatedMinutes != 30 || r.DueDate == nil {
		t.Errorf("got %+v", r)
	}
	for _, line := range []string{
		"стоматолог завтра 15:00", // a clock time: task or event is a real choice
		"молоко\nхлеб\nяйца",      // a list: one or many is a real choice
		"зарядка повтор",          // «повтор» with no frequency: the card asks
	} {
		if _, ok := PlainTask(line, understandNow); ok {
			t.Errorf("%q must not be a plain task", line)
		}
	}
}

func TestBoundsDefaultsToAnHourAndSpansAllDay(t *testing.T) {
	d := ParseLine("встреча завтра 15:00", understandNow).Draft
	start, end := Bounds(d)
	if end.Sub(start) != time.Hour {
		t.Errorf("no end given: %v, want one hour", end.Sub(start))
	}
	d = ParseLine("отпуск завтра весь день", understandNow).Draft
	start, end = Bounds(d)
	if !d.AllDay || end.Sub(start) != 24*time.Hour {
		t.Errorf("all day: %v–%v", start, end)
	}
}

func TestSettingsReadersReadBothKeywordShapes(t *testing.T) {
	s := map[string]any{
		"bot": map[string]any{"keywords": map[string]any{
			"Дом":    "home",
			"созвон": map[string]any{"field": "calendar", "value": "Работа"},
		}},
		"reminders": map[string]any{"presets": map[string]any{
			"без": []any{}, "важное": []any{float64(60)}, "кривое": "x",
		}},
	}
	kw := KeywordsFromSettings(s)
	if kw["дом"] != (Keyword{Field: "tag", Value: "home"}) || kw["созвон"].Field != "calendar" {
		t.Errorf("keywords = %+v", kw)
	}
	p := PresetsFromSettings(s)
	if len(p) != 2 || p["без"] == nil || len(p["без"]) != 0 || p["важное"][0] != 60 {
		t.Errorf("presets = %+v", p)
	}
}

func TestMissingNamesWhatTheCardStillHasToAsk(t *testing.T) {
	cases := map[string]string{
		"стоматолог завтра 15:00":    "",
		"стоматолог 15:00":           "date",
		"завтра 15:00":               "title",
		"зарядка завтра 8:00 повтор": "freq",
	}
	for line, want := range cases {
		u := Understand(line, understandNow, Vocabulary{})
		if got := Missing(u.Title, u.Draft); got != want {
			t.Errorf("%q: missing %q, want %q", line, got, want)
		}
	}
}
