package handlers

import (
	"encoding/json"
	"io"
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

// Real-length ids: callback_data is 64 bytes at most, and a uuid is 36.
const (
	stParent = "11111111-1111-4111-8111-111111111111"
	stTodo   = "22222222-2222-4222-8222-222222222222"
	stDone   = "33333333-3333-4333-8333-333333333333"
	stSeries = "44444444-4444-4444-8444-444444444444"
	stHome   = "cal-home-shared"
)

type subAPI struct {
	mu     sync.Mutex
	calls  []string
	bodies map[string]string
}

func (a *subAPI) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		b, _ := io.ReadAll(r.Body)
		call := r.Method + " " + r.URL.Path
		a.calls = append(a.calls, call)
		a.bodies[call] = string(b)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/tasks" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"id": stParent, "title": "ремонт", "status": "TODO", "priority": 2, "calendar_id": stHome},
				map[string]any{"id": stTodo, "title": "купить краску", "status": "TODO", "priority": 3, "calendar_id": stHome, "parent_id": stParent},
				map[string]any{"id": stDone, "title": "снять обои", "status": "DONE", "priority": 3, "calendar_id": stHome, "parent_id": stParent},
				map[string]any{"id": stSeries, "title": "проветрить", "status": "TODO", "priority": 3, "calendar_id": stHome, "parent_id": stParent, "rrule": "FREQ=DAILY"},
			}})
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"timezone": "Europe/Moscow",
				"settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		case r.URL.Path == "/api/tasks" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "55555555-5555-4555-8555-555555555555", "title": "x", "status": "TODO"}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
	}
}

func (a *subAPI) called(call string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, c := range a.calls {
		if c == call {
			return true
		}
	}
	return false
}

func subHandler(t *testing.T) (*Handler, *fakeTelegram, int64, *subAPI) {
	t.Helper()
	a := &subAPI{bodies: map[string]string{}}
	srv := httptest.NewServer(a.handler())
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9911)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	us.Onboarded = true
	return h, fake, chat, a
}

// Plan 23.09, day 2: the card lists its subtasks with ✅ and offers
// «➕ Подзадача». A task with subtasks is a project: it says how far along.
func TestTheCardListsSubtasks(t *testing.T) {
	h, fake, chat, _ := subHandler(t)
	h.handleTaskAction(chat, 0, stParent)
	got := fake.last(t)
	if !strings.Contains(got.Text, "1 из 3") {
		t.Errorf("no progress line: %q", got.Text)
	}
	for _, want := range []string{"sb_d_" + stTodo, "task_action_" + stDone, "sb_add_" + stParent} {
		if !strings.Contains(got.Markup, want) {
			t.Errorf("no %s: %s", want, got.Markup)
		}
	}
}

// A subtask's card leads back to its task, and a card without subtasks has
// no progress line.
func TestASubtaskCardLeadsBackToItsTask(t *testing.T) {
	h, fake, chat, _ := subHandler(t)
	h.handleTaskAction(chat, 0, stTodo)
	got := fake.last(t)
	if !strings.Contains(got.Markup, "task_action_"+stParent) {
		t.Errorf("no way back to the task: %s", got.Markup)
	}
	if strings.Contains(got.Text, " из ") {
		t.Errorf("a task without subtasks shows progress: %q", got.Text)
	}
}

// 🔴 The API files a task without calendar_id into the author's personal
// calendar. A subtask of a task in a shared calendar must land beside it, or
// the other member never sees it.
func TestANewSubtaskLandsInTheTasksCalendar(t *testing.T) {
	h, fake, chat, a := subHandler(t)
	press(h, chat, "sb_add_"+stParent)
	say(h, chat, "купить валик завтра")
	body := a.bodies["POST /api/tasks"]
	for _, want := range []string{`"parent_id":"` + stParent + `"`, `"calendar_id":"` + stHome + `"`, `"title":"купить валик"`, `"due_date"`} {
		if !strings.Contains(body, want) {
			t.Errorf("POST body has no %s: %s", want, body)
		}
	}
	if got := fake.last(t); !strings.Contains(got.Text, "ремонт") {
		t.Errorf("after adding, the task's card is not shown: %q", got.Text)
	}
}

// ✅ goes through the one done path: a series closes for the day, never for
// good (the 20.09 defect); a one-off task is closed. Then the task's card.
func TestTickingASubtaskUsesTheDonePath(t *testing.T) {
	h, fake, chat, a := subHandler(t)
	press(h, chat, "sb_d_"+stSeries)
	if !a.called("POST /api/tasks/" + stSeries + "/occurrences") {
		t.Errorf("a series subtask was not closed for the day: %v", a.calls)
	}
	if a.called("PATCH /api/tasks/" + stSeries) {
		t.Errorf("a series subtask was closed for good: %s", a.bodies["PATCH /api/tasks/"+stSeries])
	}
	press(h, chat, "sb_d_"+stTodo)
	if !strings.Contains(a.bodies["PATCH /api/tasks/"+stTodo], `"status":"DONE"`) {
		t.Errorf("a one-off subtask was not closed: %v", a.calls)
	}
	if got := fake.last(t); !strings.Contains(got.Text, "ремонт") {
		t.Errorf("after ✅ the task's card is not shown: %q", got.Text)
	}
}

// «❌ Отмена» ends the question: the next line is an ordinary line again,
// not a subtask of the task left behind.
func TestCancellingASubtaskEndsTheQuestion(t *testing.T) {
	h, _, chat, a := subHandler(t)
	press(h, chat, "sb_add_"+stParent)
	press(h, chat, "sb_x_"+stParent)
	say(h, chat, "позвонить маме")
	if strings.Contains(a.bodies["POST /api/tasks"], "parent_id") {
		t.Errorf("a line after cancel became a subtask: %s", a.bodies["POST /api/tasks"])
	}
}
