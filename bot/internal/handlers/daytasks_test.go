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
	// proposalDown makes /proposal answer 500; noDay makes /day-tasks answer
	// an empty list; meDownAfterWrite fails every settings read after a PATCH.
	proposalDown, noDay, meDownAfterWrite, meWritten bool
	// settings adds top-level keys to /api/auth/me; meDown fails every read of it.
	settings map[string]any
	// days, when set, is what GET /api/day-tasks answers, whatever the range.
	days   []map[string]any
	meDown bool
	// eventsDown fails GET /api/events: the API is unreachable.
	eventsDown bool
	calls      []string
	bodies     map[string]string
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
		down := (r.URL.Path == "/api/day-tasks/proposal" && a.proposalDown) ||
			(r.URL.Path == "/api/events" && a.eventsDown) ||
			(r.URL.Path == "/api/auth/me" && r.Method == http.MethodGet && (a.meDown || (a.meDownAfterWrite && a.meWritten)))
		if r.URL.Path == "/api/auth/me" && r.Method != http.MethodGet {
			a.meWritten = true
		}
		if down {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "INTERNAL", "message": "down"}})
			return
		}
		switch {
		case r.URL.Path == "/api/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		case r.URL.Path == "/api/day-tasks" && r.Method == http.MethodGet && a.days != nil:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": a.days})
		case r.URL.Path == "/api/day-tasks" && r.Method == http.MethodGet && a.noDay:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		case r.URL.Path == "/api/auth/me":
			settings := map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}
			for k, v := range a.settings {
				settings[k] = v
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"timezone": "Europe/Moscow",
				"settings": settings}})
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
	a := &dayAPI{confirmed: true, items: []map[string]any{
		{"task_id": dtSeries, "title": "таблетки", "done": false},
		{"task_id": dtOne, "title": "отчёт", "done": false}}}
	h, _, chat := dayHandler(t, a)
	press(h, chat, "dt_ok_"+moscowToday()+"_"+dtSeries)
	if !a.called("POST /api/tasks/" + dtSeries + "/occurrences") {
		t.Errorf("a series was not closed through its day: %v", a.calls)
	}
	// Review I1: the day is named, so an old press cannot close a later day.
	if body := a.bodies["POST /api/tasks/"+dtSeries+"/occurrences"]; !strings.Contains(body, `"date":"`+moscowToday()+`"`) {
		t.Errorf("the series day was not named: %s", body)
	}
	press(h, chat, "dt_ok_"+moscowToday()+"_"+dtOne)
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

// Review I1: ✅ on a screen that is no longer today's writes nothing.
func TestAnOldTickWritesNothing(t *testing.T) {
	a := &dayAPI{confirmed: true, items: []map[string]any{{"task_id": dtOne, "title": "отчёт", "done": false}}}
	h, fake, chat := dayHandler(t, a)
	loc, _ := time.LoadLocation("Europe/Moscow")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	press(h, chat, "dt_ok_"+yesterday+"_"+dtOne)
	if a.called("PATCH /api/tasks/" + dtOne) {
		t.Errorf("an old ✅ closed the task: %v", a.calls)
	}
	if got := fake.sent(); len(got) == 0 || !strings.Contains(got[0].Text, "уже не сегодняшний") {
		t.Errorf("an old ✅ did not say why: %v", got)
	}
}

// Review I1: a task already done is not closed again (that would move its
// completed_at to now, and its day's colour with it).
func TestTickingADoneTaskWritesNothing(t *testing.T) {
	a := &dayAPI{confirmed: true, items: []map[string]any{{"task_id": dtOne, "title": "отчёт", "done": true}}}
	h, _, chat := dayHandler(t, a)
	press(h, chat, "dt_ok_"+moscowToday()+"_"+dtOne)
	if a.called("PATCH /api/tasks/" + dtOne) {
		t.Errorf("a done task was closed again: %v", a.calls)
	}
}

// Review I2: a past day's write is refused by the bot with a sentence that fits,
// before the server is asked.
func TestPastDayWritesAreRefusedWithAFittingSentence(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	for _, data := range []string{"dt_take_" + yesterday, "dt_takeset_" + yesterday,
		"dt_put_" + yesterday + "_" + dtOne, "dt_pd_" + yesterday + "_" + dtOne} {
		a := &dayAPI{proposal: []map[string]any{{"task_id": dtOne, "title": "отчёт"}}}
		h, fake, chat := dayHandler(t, a)
		press(h, chat, data)
		if a.called("POST /api/day-tasks/confirm") || a.called("POST /api/day-tasks") {
			t.Errorf("%s wrote to a past day: %v", data, a.calls)
		}
		all := ""
		for _, m := range fake.sent() {
			all += m.Text + " | "
		}
		if !strings.Contains(all, "уже прошёл") || strings.Contains(all, "Убрать") {
			t.Errorf("%s answered: %s", data, all)
		}
	}
}

// Review I2: TOO_LATE on an ADD is not a sentence about removing.
func TestTooLateOnAddIsNotAboutRemoving(t *testing.T) {
	a := &dayAPI{refuse: "TOO_LATE"}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_put_"+moscowToday()+"_"+dtOne)
	if got := fake.last(t).Text; strings.Contains(got, "Убрать") {
		t.Errorf("TOO_LATE on add answered %q", got)
	}
}

// Review I3: the date step has a way out, and a new line is a new thing, not
// a date for the old task (Denis 18.09: the latest line is what the user wants).
func TestTheDateStepHasAWayOut(t *testing.T) {
	a := &dayAPI{}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_pdt_"+dtOne)
	if got := fake.last(t).Markup; !strings.Contains(got, "dt_pin_"+dtOne) {
		t.Errorf("the date step has no ❌ Отмена: %s", got)
	}
	say(h, chat, "купить хлеб завтра")
	if a.called("POST /api/day-tasks") {
		t.Errorf("a new line pinned the old task: %v", a.calls)
	}
	if !a.called("POST /api/tasks") {
		t.Errorf("the new line did not become a task: %v", a.calls)
	}
	if flow := h.store.GetOrCreate(chat).CurrentFlow; flow == dayTaskDateFlow {
		t.Errorf("still in the date step")
	}
}

// Review M1: an offer that could not be read is not «nothing to take».
func TestAFailedOfferIsNotAnEmptyOne(t *testing.T) {
	a := &dayAPI{proposalDown: true}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_d_today")
	got := fake.last(t).Text
	if strings.Contains(got, "Взять нечего") || !strings.Contains(got, "❌") {
		t.Errorf("a failed offer read as %q", got)
	}
}

// Review M2: a read that brought back no day still says something.
func TestAnEmptyReadSaysSomething(t *testing.T) {
	a := &dayAPI{noDay: true}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_d_today")
	if got := strings.TrimSpace(strings.TrimPrefix(fake.last(t).Text, "❌")); got == "" {
		t.Errorf("an empty read printed a bare ❌")
	}
}

// Review M3: «Поменять» on a past day opens the day itself, which is read-only,
// rather than ❌/➕ buttons that every press would refuse.
func TestEditingAPastDayShowsTheDay(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	a := &dayAPI{confirmed: true, items: []map[string]any{{"task_id": dtOne, "title": "отчёт", "done": false}}}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_edit_"+yesterday)
	if got := fake.last(t).Markup; strings.Contains(got, "dt_rm_") || strings.Contains(got, "dt_add_") {
		t.Errorf("a past day opened for editing: %s", got)
	}
}

// Review M5: the tick marks what was saved, even when the read after the write
// fails.
func TestTheSavedTargetIsTickedWhenTheReReadFails(t *testing.T) {
	a := &dayAPI{meDownAfterWrite: true}
	h, fake, chat := dayHandler(t, a)
	h.handleDaySettings(chat, 0, "n:4")
	if got := fake.last(t).Markup; !strings.Contains(got, "✓ 4") {
		t.Errorf("saved 4, ticked: %s", got)
	}
}

// Review Focus 2: an old 📌 button, day tasks off → a sentence and the way back
// on; no screen, no write.
func TestAnOldDayTasksButtonWhenOff(t *testing.T) {
	a := &dayAPI{settings: map[string]any{"day_tasks_enabled": false}}
	h, fake, chat := dayHandler(t, a)
	press(h, chat, "dt_put_"+moscowToday()+"_"+dtOne)
	if a.called("POST /api/day-tasks") {
		t.Errorf("wrote while off: %v", a.calls)
	}
	if got := fake.last(t); !strings.Contains(got.Text, "выключены") || !strings.Contains(got.Markup, "dts_on") {
		t.Errorf("off answered %q / %s", got.Text, got.Markup)
	}
}
