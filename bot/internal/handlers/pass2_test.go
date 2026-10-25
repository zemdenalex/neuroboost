package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
	"github.com/zemdenalex/neuroboost-bot/parse"
)

// Denis's second pass, 17.09 22:10 — ref/feedback/bot-povtor-otvet-denisa-2026-09-17.md.

// 🔴 «надо чтобы слово задача сделала на одно действие меньше, то есть бот
// воспринял как задачу, но уточнил, а не заставлял выбирать задачу еще раз».
// 23.09 took it one step further: no card, the task is saved at once
// (quicksave.go). What this test still guards: no «Что создать?», and
// changing your mind stays possible — now under the saved task.
func TestTaskWordGoesStraightToTheTaskCard(t *testing.T) {
	h, fake, _, chat := quickSaveHandler(t)
	say(h, chat, "задача на завтра умыться")

	got := fake.last(t)
	if strings.Contains(got.Text, "Что создать?") {
		t.Fatalf("«задача» still asks first: %q", got.Text)
	}
	if !strings.Contains(got.Text, "умыться") {
		t.Errorf("the saved task is not shown: %q", got.Text)
	}
	for _, want := range []string{"qs_event_", "qs_note_"} {
		if !strings.Contains(got.Markup, want) {
			t.Errorf("no way to make it %s under the saved task: %s", want, got.Markup)
		}
	}
}

// 🔴 Two cards were drawn for one choice (his log, 21:56: a ✅ card followed by
// a 📅 card for the same line).
func TestChoosingATypeDrawsOneCard(t *testing.T) {
	h, fake, chat := quickHandler(t)
	say(h, chat, "задача завтра в 15 позвонить в банк")
	before := len(fake.sent())
	press(h, chat, "qa_event")

	if n := len(fake.sent()) - before; n != 1 {
		t.Errorf("choosing «Событие» sent %d messages, want 1", n)
	}
}

// 🔴 «А список вообще не понял» — typing at the task list question answered
// «Что-то пошло не так» and threw everything away.
func TestTypingAtTheTaskListQuestionKeepsGoing(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewTaskFlow(chat)
	say(h, chat, "умыться, поесть, поспать")
	say(h, chat, "задачи на завтра умыться, поесть, поспать")

	if got := fake.last(t).Text; strings.Contains(got, "не так") {
		t.Fatalf("typing at the list question still breaks: %q", got)
	}
}

// 🔴 «день поставился только на первую задачу» — one line, one day, every task.
func TestOneLineTaskListSharesItsDay(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.startNewTaskFlow(chat)
	say(h, chat, "завтра помыться, поесть, поспать")
	press(h, chat, "dr_many")

	card := fake.last(t).Text
	if strings.Count(card, "📅 ") != 3 {
		t.Errorf("the day reached %d of 3 tasks:\n%s", strings.Count(card, "📅 "), card)
	}
}

// «❓ Как пользоваться» reads like a Q&A; it re-runs setup.
func TestSettingsSaysSetupNotHelp(t *testing.T) {
	h, fake, chat := quickHandler(t)
	h.handleSettings(chat, 0)
	got := fake.last(t)
	if !strings.Contains(got.Markup, "ob_start") {
		t.Fatalf("no onboarding entry in settings: %s", got.Markup)
	}
	if strings.Contains(got.Markup, "Как пользоваться") || strings.Contains(got.Markup, "How to use") {
		t.Errorf("the button still reads as a Q&A: %s", got.Markup)
	}
}

// The events picker must show all-day and multi-day events too: they start at
// midnight, which is already in the past by the time anyone looks — and the
// two-week horizon hid «отпуск с 14.10» entirely (Denis, 17.09).
func TestPickerAsksFromMidnightAndLooksFurther(t *testing.T) {
	var start, end string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			start, end = r.URL.Query().Get("start"), r.URL.Query().Get("end")
		}
		if r.URL.Path == "/api/auth/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"timezone": "Europe/Moscow", "settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()
	bot, _ := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9301)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())

	press(h, chat, "event_pick")

	from, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatalf("start = %q: %v", start, err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	if h, m, s := from.In(loc).Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("the picker asks from %s — an all-day event that began at midnight is missed", from.In(loc))
	}
	to, err := time.Parse(time.RFC3339, end)
	if err != nil {
		t.Fatalf("end = %q: %v", end, err)
	}
	if to.Sub(from) < 45*24*time.Hour {
		t.Errorf("the picker looks %v ahead — «отпуск с 14.10» was out of range", to.Sub(from))
	}
}

// The task guide's example must do what it says, like the event guide's.
func TestTaskGuideExampleParsesAsAdvertised(t *testing.T) {
	r := parseTaskForTest("позвонить в банк завтра 30м !1 #дела")
	if r.Title != "позвонить в банк" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.DueDate == nil || r.EstimatedMinutes == nil || *r.EstimatedMinutes != 30 || r.Priority == nil || *r.Priority != 1 {
		t.Errorf("due %v, estimate %v, priority %v", r.DueDate, r.EstimatedMinutes, r.Priority)
	}
	if len(r.Tags) != 1 || r.Tags[0] != "дела" {
		t.Errorf("tags %v", r.Tags)
	}
}

func parseTaskForTest(line string) parse.TaskResult {
	return parse.ParseTask(line, time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC))
}
