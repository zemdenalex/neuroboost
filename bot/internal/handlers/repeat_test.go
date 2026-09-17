package handlers

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func onboardedHandler(t *testing.T) (*Handler, *fakeTelegram, int64, *fakeAccount) {
	t.Helper()
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}
	h, fake, chat := onboardHandler(t, acc)
	return h, fake, chat, acc
}

// 🔴 End to end: the period and the count reach POST /api/events as one RRULE
// the API can parse — the card saying «раз в 3 дня» proves nothing on its own.
func TestCustomPeriodReachesTheAPI(t *testing.T) {
	h, fake, chat, acc := onboardedHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "таблетки раз в 3 дня 10 раз завтра 09:00")

	if card := fake.last(t).Text; !strings.Contains(card, "раз в 3 дня · 10 раз") {
		t.Fatalf("the card does not say the rule:\n%s", card)
	}
	press(h, chat, "dr_ok")

	for _, wr := range acc.writes {
		if wr.Method == "POST" && wr.Path == "/api/events" {
			if wr.Body["rrule"] != "FREQ=DAILY;INTERVAL=3;COUNT=10" {
				t.Errorf("rrule sent = %v", wr.Body["rrule"])
			}
			return
		}
	}
	t.Fatalf("no event was posted; writes = %+v", acc.writes)
}

// «✏️ Своя частота» takes the period as text.
func TestCustomFrequencyButtonTakesText(t *testing.T) {
	h, fake, chat, _ := onboardedHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "змей завтра 10:00 повтор")
	press(h, chat, "dre_repeat")
	press(h, chat, "dr_freq_CUSTOM")
	say(h, chat, "раз в 3 недели")

	if card := fake.last(t).Text; !strings.Contains(card, "раз в 3 недели") {
		t.Errorf("the typed period is not on the card:\n%s", card)
	}
}

// The series end is set under «Изменить», and «Никогда» clears it.
func TestRepeatEndIsEditable(t *testing.T) {
	h, fake, chat, _ := onboardedHandler(t)
	h.startNewEventFlow(chat)
	say(h, chat, "йога каждую неделю завтра 08:00")
	press(h, chat, "dre_rend")
	press(h, chat, "dr_rend_text")
	say(h, chat, "до 01.12")
	if card := fake.last(t).Text; !strings.Contains(card, "до 01.12") {
		t.Fatalf("the end is not on the card:\n%s", card)
	}
	press(h, chat, "dre_rend")
	press(h, chat, "dr_rend_never")
	if card := fake.last(t).Text; strings.Contains(card, "до 01.12") {
		t.Errorf("«Никогда» left the end:\n%s", card)
	}
}

// apiRRule is the grammar of api-go/internal/events/recurrence.go, parseRRule:
// FREQ of DAILY|WEEKLY|MONTHLY, and INTERVAL/COUNT (≥1) and UNTIL (a date).
// ⚠ A COPY of another module's contract — if the API learns YEARLY or BYDAY,
// this may widen; if it narrows, this must follow it.
var apiRRulePart = regexp.MustCompile(`^(FREQ=(DAILY|WEEKLY|MONTHLY)|INTERVAL=[1-9]\d*|COUNT=[1-9]\d*|UNTIL=\d{4}-\d{2}-\d{2})$`)

func parsesInAPI(rule string) bool {
	parts := strings.Split(rule, ";")
	if !strings.HasPrefix(parts[0], "FREQ=") {
		return false
	}
	for _, p := range parts {
		if !apiRRulePart.MatchString(p) {
			return false
		}
	}
	return true
}

// 🔴 An RRULE the API cannot parse is not rejected — the event is shown once
// and never repeats, silently. Every rule the bot can write must parse.
func TestEveryRuleTheBotWritesParsesInTheAPI(t *testing.T) {
	now := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	var rules []string
	for _, f := range []string{"DAILY", "WEEKLY", "MONTHLY", "YEARLY"} {
		rules = append(rules, parse.FreqRule(f))
	}
	for _, line := range []string{
		"a ежедневно", "a еженедельно", "a ежемесячно", "a ежегодно", "a daily", "a weekly", "a monthly", "a yearly", "a annually",
		"a каждый день", "a каждую неделю", "a каждый месяц", "a каждый год", "a every year", "a каждый вторник", "a every friday",
		"a раз в 3 дня", "a каждые 2 недели", "a every 5 years", "a через день", "a раз в год 3 раза", "a каждый день до 01.12",
	} {
		p := parse.ParseLine(line, now)
		if p.Draft.RRule() == "" {
			t.Errorf("%q produced no rule", line)
			continue
		}
		rules = append(rules, p.Draft.RRule())
	}
	for _, r := range rules {
		if !parsesInAPI(r) {
			t.Errorf("the bot writes %q, which the API cannot parse — the event would never repeat", r)
		}
	}
}
