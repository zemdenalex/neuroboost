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

	got := DigestText(day, loc, nil, nil, "en")

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

	got := DigestText(day, loc, events, nil, "en")

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

	got := DigestText(day, loc, events, nil, "en")

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

	got := DigestText(day, loc, events, nil, "en")

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

	got := DigestText(day, loc, nil, tasks, "en")

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

	got := DigestText(day, loc, events, nil, "en")

	if n := len([]rune(got)); n > 4096 {
		t.Errorf("digest is %d runes, Telegram rejects over 4096", n)
	}
	if !strings.Contains(got, "more") {
		t.Errorf("a truncated digest must say something was omitted, got tail:\n%s", got[len(got)-200:])
	}
}

// 🔴 Настя через Дениса, 17.09: the morning digest arrived in English for
// everyone, because every string in it was a literal.
//
// This is the same defect class as the notification buttons, one process over:
// a message written by the SERVER, where nothing forces a language choice the
// way i18n.T does in the bot.
func TestDigestSpeaksTheReadersLanguage(t *testing.T) {
	day := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)
	events := []DigestEvent{{
		Title:    "стоматолог",
		StartsAt: time.Date(2026, time.September, 18, 15, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, time.September, 18, 16, 0, 0, 0, time.UTC),
	}}
	tasks := []DigestTask{{Title: "доделать сайт"}}

	ru := DigestText(day, time.UTC, events, tasks, "ru")
	for _, want := range []string{"Сегодня", "18 сентября", "События (1)", "Задачи на сегодня (1)"} {
		if !strings.Contains(ru, want) {
			t.Errorf("русский дайджест не содержит %q:\n%s", want, ru)
		}
	}
	for _, unwanted := range []string{"Today", "Events (", "Tasks due today"} {
		if strings.Contains(ru, unwanted) {
			t.Errorf("русский дайджест содержит английское %q:\n%s", unwanted, ru)
		}
	}

	en := DigestText(day, time.UTC, events, tasks, "en")
	for _, want := range []string{"Today", "Events (1)", "Tasks due today (1)"} {
		if !strings.Contains(en, want) {
			t.Errorf("the English digest is missing %q:\n%s", want, en)
		}
	}
	// The control: the two must actually differ, or a bug that ignores the
	// language argument would satisfy both halves above.
	if ru == en {
		t.Error("both languages produced the same text; the lang argument is ignored")
	}
}

func TestEmptyDigestSpeaksTheLanguageToo(t *testing.T) {
	day := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)
	ru := DigestText(day, time.UTC, nil, nil, "ru")
	if !strings.Contains(ru, "Ничего не запланировано") {
		t.Errorf("пустой русский дайджест по-английски:\n%s", ru)
	}
	if strings.Contains(ru, "Nothing scheduled") {
		t.Errorf("пустой русский дайджест содержит английский текст:\n%s", ru)
	}
}

// The all-day marker is inside a line, which is exactly where a translation is
// forgotten.
func TestAllDayLineIsTranslated(t *testing.T) {
	day := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)
	events := []DigestEvent{{Title: "отпуск", AllDay: true, StartsAt: day}}

	ru := DigestText(day, time.UTC, events, nil, "ru")
	if !strings.Contains(ru, "весь день") || strings.Contains(ru, "all day") {
		t.Errorf("строка всё-дневного события не переведена:\n%s", ru)
	}
}

// 🔴 The bot's language beats the account locale: the digest is read in
// Telegram. On production these already disagree for one account —
// locale = ru, settings.bot.lang = en.
func TestDigestLangPrefersTheBotSetting(t *testing.T) {
	for _, c := range []struct {
		name     string
		settings string
		locale   string
		want     string
	}{
		{"bot wins over locale", `{"bot":{"lang":"en"}}`, "ru", "en"},
		{"locale when the bot has none", `{"bot":{}}`, "en", "en"},
		{"locale when settings are empty", `{}`, "en", "en"},
		{"ru when nothing is known", `{}`, "", "ru"},
		{"unreadable settings fall back rather than fail", `not json`, "en", "en"},
	} {
		if got := DigestLang([]byte(c.settings), c.locale); got != c.want {
			t.Errorf("%s: DigestLang(%s, %q) = %q, want %q", c.name, c.settings, c.locale, got, c.want)
		}
	}
}
