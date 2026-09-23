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

// «Задачи дня» D2 — the handler, through the buttons (plan 2026-09-23 Task 4).

const (
	dtOne    = "11111111-1111-1111-1111-111111111111"
	dtSeries = "22222222-2222-2222-2222-222222222222"
)

type dayAPI struct {
	mu        sync.Mutex
	confirmed bool
	items     []map[string]any
	proposal  []map[string]any
	refuse    string // an error code the next write answers with
	calls     []string
	bodies    map[string]string
}

func (a *dayAPI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		b, _ := io.ReadAll(r.Body)
		call := r.Method + " " + r.URL.Path
		a.calls = append(a.calls, call)
		a.bodies[call] = string(b)
		w.Header().Set("Content-Type", "application/json")
		day := map[string]any{"day": r.URL.Query().Get("from"), "target": 5, "confirmed": a.confirmed,
			"items": a.items, "done": 0, "level": 0}
		if a.items == nil {
			day["items"] = []any{}
		}
		write := r.Method != http.MethodGet && r.URL.Path != "/api/auth/me"
		if write && a.refuse != "" {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": a.refuse, "message": "refused"}})
			return
		}
		switch {
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"timezone": "Europe/Moscow",
				"settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		case r.URL.Path == "/api/day-tasks" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{day}})
		case r.URL.Path == "/api/day-tasks/proposal":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": a.proposal})
		case strings.HasPrefix(r.URL.Path, "/api/day-tasks"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": day})
		case r.URL.Path == "/api/tasks" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
				map[string]any{"id": dtOne, "title": "отчёт", "status": "TODO", "priority": 2},
				map[string]any{"id": dtSeries, "title": "таблетки", "status": "TODO", "priority": 1, "rrule": "FREQ=DAILY"},
				map[string]any{"id": "done-one", "title": "сделано", "status": "DONE", "priority": 1},
			}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	}
}

func (a *dayAPI) called(call string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, c := range a.calls {
		if c == call {
			return true
		}
	}
	return false
}

func dayHandler(t *testing.T, a *dayAPI) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	a.bodies = map[string]string{}
	srv := httptest.NewServer(a.handler(t))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9901)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat
}

func moscowToday() string {
	loc, _ := time.LoadLocation("Europe/Moscow")
	return time.Now().In(loc).Format("2006-01-02")
}

func TestTheMenuOpensToday(t *testing.T) {
	a := &dayAPI{proposal: []map[string]any{{"task_id": dtOne, "title": "отчёт"}}}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_d_today")
	got := fake.last(t)
	if !strings.Contains(got.Text, "Задачи дня") || !strings.Contains(got.Markup, "dt_take_"+moscowToday()) {
		t.Errorf("today's screen: %q / %s", got.Text, got.Markup)
	}
}

// «✅ Беру» confirms what is offered at the moment of the press.
func TestTakeConfirmsTheOffer(t *testing.T) {
	a := &dayAPI{proposal: []map[string]any{{"task_id": dtOne, "title": "отчёт"}}}
	h, _, chat := dayHandler(t, a)
	press(h, chat, "dt_take_"+moscowToday())
	body := a.bodies["POST /api/day-tasks/confirm"]
	if !strings.Contains(body, dtOne) || !strings.Contains(body, moscowToday()) {
		t.Errorf("confirm body = %q", body)
	}
}

// Review focus 3: ✅ today on a series closes the series' day, on a one-off
// closes the task.
func TestTickingClosesTheRightWay(t *testing.T) {
	a := &dayAPI{confirmed: true}
	h, _, chat := dayHandler(t, a)
	press(h, chat, "dt_ok_"+dtSeries)
	if !a.called("POST /api/tasks/" + dtSeries + "/occurrences") {
		t.Errorf("a series was not closed through its day: %v", a.calls)
	}
	press(h, chat, "dt_ok_"+dtOne)
	if !a.called("PATCH /api/tasks/"+dtOne) || !strings.Contains(a.bodies["PATCH /api/tasks/"+dtOne], "DONE") {
		t.Errorf("a one-off was not closed: %v", a.calls)
	}
}

func TestRemovingAfterNoonSaysWhy(t *testing.T) {
	a := &dayAPI{confirmed: true, refuse: "TOO_LATE"}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_rm_"+moscowToday()+"_"+dtOne)
	if got := fake.last(t).Text; !strings.Contains(got, "до 12:00") {
		t.Errorf("TOO_LATE answered %q", got)
	}
}

// Review focus 4: the day-tasks NOT_AN_OCCURRENCE is «not on that day», not the
// generic «the series has ended».
func TestASeriesOnAWrongDaySaysSo(t *testing.T) {
	a := &dayAPI{refuse: "NOT_AN_OCCURRENCE"}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_pd_"+moscowToday()+"_"+dtSeries)
	got := fake.last(t).Text
	if !strings.Contains(got, "В этот день эта серия не повторяется") || strings.Contains(got, "закончилась") {
		t.Errorf("NOT_AN_OCCURRENCE answered %q", got)
	}
}

func TestAddingReturnsToTheEditScreen(t *testing.T) {
	a := &dayAPI{}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_put_"+moscowToday()+"_"+dtOne)
	if !a.called("POST /api/day-tasks") {
		t.Fatalf("nothing was added: %v", a.calls)
	}
	if got := fake.last(t); !strings.Contains(got.Markup, "dt_add_"+moscowToday()) {
		t.Errorf("after ➕ the screen is not «Поменять»: %s", got.Markup)
	}
}

// The ➕ list: open tasks only, not the done one.
func TestTheAddListOffersOpenTasks(t *testing.T) {
	a := &dayAPI{}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_add_"+moscowToday())
	got := fake.last(t).Markup
	if !strings.Contains(got, "dt_put_"+moscowToday()+"_"+dtOne) || strings.Contains(got, "done-one") {
		t.Errorf("add list: %s", got)
	}
}

// Denis 23.09: «✏️ Дата» takes a typed date. A past date asks again and writes
// nothing; a good one adds the task.
func TestATypedDateAddsTheTask(t *testing.T) {
	a := &dayAPI{}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_pdt_"+dtOne)
	say(h, chat, "вчера")
	if a.called("POST /api/day-tasks") {
		t.Fatalf("a past date was written")
	}
	if got := fake.last(t).Text; !strings.Contains(got, "прошл") {
		t.Errorf("a past date answered %q", got)
	}
	say(h, chat, "послезавтра")
	if !a.called("POST /api/day-tasks") {
		t.Errorf("a good date added nothing: %v", a.calls)
	}
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow != "" {
		t.Errorf("the date flow is still running: %q", flow)
	}
}

// Review focus 5: a long title is shortened in the button.
func TestLongTitlesAreShortened(t *testing.T) {
	long := strings.Repeat("очень длинное название задачи ", 5)
	a := &dayAPI{confirmed: true, items: []map[string]any{{"task_id": dtOne, "title": long, "done": false}}}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_d_today")
	if got := fake.last(t).Markup; strings.Contains(got, long) || !strings.Contains(got, "…") {
		t.Errorf("the title went into the button whole: %s", got)
	}
}
