package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// Denis, 23.09: a line typed to the bot without a command becomes a task at
// once — «seamless task creation» — with ↩️ Отменить and ✏️ Изменить under it,
// and a way to say «I meant an event / a note».

type taskAPI struct {
	mu      sync.Mutex
	created []map[string]any
	deleted []string
	rules   map[string]map[string]any // PUT …/remind-rule, by path
}

func (a *taskAPI) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		switch {
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"timezone": "Europe/Moscow",
				"settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			a.created = append(a.created, body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id": "11111111-2222-3333-4444-555555555555", "title": body["title"], "status": "TODO"}})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/remind-rule"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if a.rules == nil {
				a.rules = map[string]map[string]any{}
			}
			a.rules[r.URL.Path] = body
			_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/tasks/"):
			a.deleted = append(a.deleted, strings.TrimPrefix(r.URL.Path, "/api/tasks/"))
			w.WriteHeader(http.StatusNoContent)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
	}
}

const quickTaskID = "11111111-2222-3333-4444-555555555555"

func quickSaveHandler(t *testing.T) (*Handler, *fakeTelegram, *taskAPI, int64) {
	t.Helper()
	a := &taskAPI{}
	srv := httptest.NewServer(a.handler())
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9701)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.SetLang("ru")
	return h, fake, a, chat
}

func TestAPlainLineIsSavedAtOnce(t *testing.T) {
	h, fake, a, chat := quickSaveHandler(t)
	say(h, chat, "купить молоко завтра")

	if len(a.created) != 1 {
		t.Fatalf("created %d tasks, want 1 without a single press", len(a.created))
	}
	if a.created[0]["title"] != "купить молоко" || a.created[0]["due_date"] == nil {
		t.Errorf("created %v, want title «купить молоко» with a due date", a.created[0])
	}
	got := fake.last(t)
	if !strings.Contains(got.Text, "Задача создана") || !strings.Contains(got.Text, "купить молоко") {
		t.Errorf("answer = %q", got.Text)
	}
	for _, button := range []string{"qs_undo_" + quickTaskID, "task_action_" + quickTaskID,
		"qs_event_" + quickTaskID, "qs_note_" + quickTaskID} {
		if !strings.Contains(got.Markup, button) {
			t.Errorf("no %s under the saved task; markup = %s", button, got.Markup)
		}
	}
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "" {
		t.Errorf("a saved task leaves flow %q running; the next line must be a fresh one", flow)
	}
}

// «задача …» is the same request, said explicitly.
func TestATaskWordLineIsSavedAtOnceToo(t *testing.T) {
	h, _, a, chat := quickSaveHandler(t)
	say(h, chat, "задача позвонить маме")
	if len(a.created) != 1 || a.created[0]["title"] != "позвонить маме" {
		t.Errorf("created %v, want one task «позвонить маме»", a.created)
	}
}

func TestUndoDeletesTheQuickTask(t *testing.T) {
	h, fake, a, chat := quickSaveHandler(t)
	say(h, chat, "купить молоко")
	press(h, chat, "qs_undo_"+quickTaskID)
	if len(a.deleted) != 1 || a.deleted[0] != quickTaskID {
		t.Fatalf("deleted %v, want the quick task", a.deleted)
	}
	if got := fake.last(t).Text; !strings.Contains(got, "Отменено") {
		t.Errorf("answer after undo = %q", got)
	}
}

// «I meant an event»: the task goes, the same line opens the event flow.
func TestQuickTaskBecomesAnEvent(t *testing.T) {
	h, _, a, chat := quickSaveHandler(t)
	say(h, chat, "купить молоко завтра")
	press(h, chat, "qs_event_"+quickTaskID)
	if len(a.deleted) != 1 {
		t.Fatalf("deleted %v, want the task removed before the event is made", a.deleted)
	}
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "new_event" {
		t.Errorf("flow = %q, want the event flow on the same line", flow)
	}
}

// A note is saved in one step by the note flow (a priority-5 task), so «→
// Заметка» is: the task goes, the note is written from the same line.
func TestQuickTaskBecomesANote(t *testing.T) {
	h, fake, a, chat := quickSaveHandler(t)
	say(h, chat, "купить молоко")
	press(h, chat, "qs_note_"+quickTaskID)
	if len(a.deleted) != 1 {
		t.Fatalf("deleted %v, want the task removed", a.deleted)
	}
	if len(a.created) != 2 || a.created[1]["title"] != "купить молоко" || a.created[1]["priority"] != float64(5) {
		t.Errorf("created %v, want a second, priority-5 entry «купить молоко»", a.created)
	}
	if got := fake.last(t).Text; !strings.Contains(got, "Заметка") {
		t.Errorf("answer = %q, want the note confirmation", got)
	}
}

// What still asks, as before: a clock time (task or event is a real choice
// there), a list (one or many), «повтор» without a frequency (the card asks).
func TestLinesThatStillAsk(t *testing.T) {
	for _, line := range []string{
		"завтра в 15 стоматолог",
		"хлеб\nмолоко\nяйца",
		"задача повтор зарядка",
	} {
		h, _, a, chat := quickSaveHandler(t)
		say(h, chat, line)
		if len(a.created) != 0 {
			t.Errorf("%q was saved at once; it needs a question first", line)
		}
	}
}

// «Напоминать до срока» typed in the line (Denis, 07.11) reaches the API: the
// task is created without the clause in its title, and the rule is PUT to its
// own door. A line with the clause but no due date stores no rule and says so.
func TestARuleTypedInTheLineIsSentAfterTheTask(t *testing.T) {
	h, fake, a, chat := quickSaveHandler(t)
	say(h, chat, "сдать отчёт 20.11 напоминать каждый день в 9 и 18")

	if len(a.created) != 1 || a.created[0]["title"] != "сдать отчёт" || a.created[0]["due_date"] == nil {
		t.Fatalf("created %v, want «сдать отчёт» with a due date", a.created)
	}
	rule := a.rules["/api/tasks/"+quickTaskID+"/remind-rule"]
	if rule == nil || rule["every_days"] != float64(1) {
		t.Fatalf("rule sent = %v, want every_days 1", a.rules)
	}
	if times, _ := rule["times"].([]any); len(times) != 2 || times[0] != "09:00" || times[1] != "18:00" {
		t.Errorf("times = %v, want 09:00 and 18:00", rule["times"])
	}
	if got := fake.last(t).Text; !strings.Contains(got, "Буду напоминать") {
		t.Errorf("the receipt does not say the rule: %q", got)
	}

	h2, fake2, a2, chat2 := quickSaveHandler(t)
	say(h2, chat2, "позвонить маме напоминать каждый час")
	if len(a2.rules) != 0 {
		t.Errorf("a rule was stored for a task with no due date: %v", a2.rules)
	}
	if got := fake2.last(t).Text; !strings.Contains(got, "нужен срок") {
		t.Errorf("no word that the rule needs a due date: %q", got)
	}
}
