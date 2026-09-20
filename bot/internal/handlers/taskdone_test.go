package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// call is one request the bot made.
type call struct {
	method, path string
	body         map[string]any
}

// doneHandler serves one task — repeating or not — and records every write.
func doneHandler(t *testing.T, task map[string]any) (*Handler, *fakeTelegram, int64, func() []call) {
	t.Helper()
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			calls = append(calls, call{r.Method, r.URL.Path, body})
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/tasks") {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{task}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	const chat = int64(7600)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat, func() []call { return calls }
}

// 🔴 «Готово» on a repeating task means «сделал СЕГОДНЯ», never «серия
// закончена». Denis, 18.09: «если выполнено на день, оно уходит из списка …
// а на следующий день всплывает снова».
//
// Until 20.09 the button sent status=DONE, which ends the series for good: the
// pills task would have vanished permanently the first time it was ticked. The
// endpoint that marks one day has existed since 18.09 and had no caller.
func TestTickingARepeatingTaskClosesTheDayNotTheSeries(t *testing.T) {
	h, _, chat, calls := doneHandler(t, map[string]any{
		"id": "t-rep", "title": "пить таблетки", "status": "TODO", "rrule": "FREQ=DAILY",
	})

	press(h, chat, "task_done_t-rep")

	var wrote []string
	for _, c := range calls() {
		wrote = append(wrote, c.method+" "+c.path)
		if c.path == "/api/tasks/t-rep" && c.body["status"] == "DONE" {
			t.Error("the whole series was marked DONE — tomorrow it never comes back")
		}
	}
	var marked bool
	for _, c := range calls() {
		if c.path == "/api/tasks/t-rep/occurrences" && c.body["state"] == "done" {
			marked = true
		}
	}
	if !marked {
		t.Errorf("today's occurrence was never marked; the bot sent: %v", wrote)
	}
}

// A one-off task still ends outright — that is what «Готово» means for it.
func TestTickingAPlainTaskStillCompletesIt(t *testing.T) {
	h, _, chat, calls := doneHandler(t, map[string]any{
		"id": "t-one", "title": "купить молоко", "status": "TODO",
	})

	press(h, chat, "task_done_t-one")

	var completed bool
	for _, c := range calls() {
		if c.path == "/api/tasks/t-one" && c.body["status"] == "DONE" {
			completed = true
		}
		if strings.HasSuffix(c.path, "/occurrences") {
			t.Error("a one-off task was given an occurrence, which it does not have")
		}
	}
	if !completed {
		t.Error("a plain task was not completed")
	}
}

// «Отложить» must exist on a repeating task and must not shift the rhythm —
// which is what postpone_days means to the API. Denis chose it on 18.09:
// «Пропустить закрытые дни, ритм не трогать».
func TestPostponingASeriesSendsDaysNotANewRule(t *testing.T) {
	h, fake, chat, calls := doneHandler(t, map[string]any{
		"id": "t-rep", "title": "пить таблетки", "status": "TODO", "rrule": "FREQ=DAILY",
	})

	press(h, chat, "task_action_t-rep")
	if card := fake.last(t).Markup; !strings.Contains(card, "task_pp_") {
		t.Fatalf("a repeating task's card offers no «Отложить»: %s", card)
	}

	press(h, chat, "task_pp_t-rep")
	if !strings.Contains(fake.last(t).Markup, "task_ppd_t-rep_7") {
		t.Errorf("the postpone screen does not offer a week: %s", fake.last(t).Markup)
	}

	press(h, chat, "task_ppd_t-rep_7")
	var sent bool
	for _, c := range calls() {
		if c.path == "/api/tasks/t-rep/occurrences" {
			sent = true
			if c.body["postpone_days"] != float64(7) {
				t.Errorf("postpone_days = %v, want 7", c.body["postpone_days"])
			}
			// 🔴 The rule itself must be untouched: rewriting it is exactly the
			// «ритм сдвинулся» Denis ruled out.
			if _, moved := c.body["rrule"]; moved {
				t.Error("postponing rewrote the repeat rule")
			}
		}
	}
	if !sent {
		t.Error("postponing sent nothing")
	}
}

// A one-off task has no series to postpone, and must not be offered one.
func TestAPlainTaskIsNotOfferedPostpone(t *testing.T) {
	h, fake, chat, _ := doneHandler(t, map[string]any{
		"id": "t-one", "title": "купить молоко", "status": "TODO",
	})

	press(h, chat, "task_action_t-one")
	if strings.Contains(fake.last(t).Markup, "task_pp_") {
		t.Errorf("a one-off task offers «Отложить серию»: %s", fake.last(t).Markup)
	}
}
