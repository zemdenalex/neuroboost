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

// convertFake answers the way api-go does and remembers what the bot sent.
type convertFake struct {
	convertBody string
	toTaskBody  string
}

func convertAPI(t *testing.T, tasksJSON, eventJSON string) (*Handler, *fakeTelegram, *convertFake) {
	t.Helper()
	cf := &convertFake{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/convert"):
			cf.convertBody = string(b)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"ev1","title":"банк","starts_at":"2026-10-20T09:00:00Z","ends_at":"2026-10-20T10:00:00Z"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/to-task"):
			cf.toTaskBody = string(b)
			dry := strings.Contains(string(b), `"dry_run":true`)
			id, dr := `"id":"t9",`, "false"
			if dry {
				id, dr = "", "true"
			}
			_, _ = w.Write([]byte(`{"data":{"task":{` + id + `"title":"врач","due_date":"2026-10-20","estimated_minutes":70,"tags":[]},` +
				`"lost":["start_time","color"],"event_id":null,"dry_run":` + dr + `}}`))
		case strings.HasPrefix(r.URL.Path, "/api/events/"):
			_, _ = w.Write([]byte(`{"data":` + eventJSON + `}`))
		case r.URL.Path == "/api/events":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default: // /api/tasks
			_, _ = w.Write([]byte(`{"data":[` + tasksJSON + `]}`))
		}
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	return New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{}), fake, cf
}

const plainTask = `{"id":"t1","title":"банк","status":"TODO","priority":2,"estimated_minutes":60}`
const seriesTaskJSON = `{"id":"t1","title":"таблетки","status":"TODO","priority":3,"rrule":"FREQ=DAILY"}`

// The whole path, pressed as a user would: link → tomorrow → done.
func TestToEventLinkSendsWhatWasChosen(t *testing.T) {
	h, fake, cf := convertAPI(t, plainTask, `{}`)
	const chat = 900
	h.handleToEventStart(chat, 0, "t1")
	h.handleToEventStep(chat, 0, "t2m_l")
	h.handleToEventStep(chat, 0, "t2w_tmr")
	// The estimate is 60, so no duration step: the card is next.
	card := fake.last(t)
	if !strings.Contains(card.Text, "приоритет") || !strings.Contains(card.Markup, "t2ok") {
		t.Fatalf("expected the mapping card with a priority warning, got %q / %s", card.Text, card.Markup)
	}
	h.handleToEventStep(chat, 0, "t2ok")

	if !strings.Contains(cf.convertBody, `"mode":"link"`) {
		t.Errorf("convert body = %s", cf.convertBody)
	}
	if strings.Contains(cf.convertBody, `"repeat"`) {
		t.Errorf("a one-off task sent a repeat choice: %s", cf.convertBody)
	}
	var body struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	}
	_ = json.Unmarshal([]byte(cf.convertBody), &body)
	s, _ := time.Parse(time.RFC3339, body.StartsAt)
	e, _ := time.Parse(time.RFC3339, body.EndsAt)
	if e.Sub(s) != time.Hour {
		t.Errorf("length = %v, want the task's 60-minute estimate", e.Sub(s))
	}
}

// 🔴 A repeating task is ASKED «вся серия или этот раз» — Denis 21.09, «уточнять».
func TestToEventAsksSeriesOrOnceForARepeatingTask(t *testing.T) {
	h, fake, cf := convertAPI(t, seriesTaskJSON, `{}`)
	const chat = 901
	h.handleToEventStart(chat, 0, "t1")
	h.handleToEventStep(chat, 0, "t2m_m")
	if m := fake.last(t).Markup; !strings.Contains(m, "t2r_s") || !strings.Contains(m, "t2r_o") {
		t.Fatalf("no series/once question after the mode: %s", m)
	}
	h.handleToEventStep(chat, 0, "t2r_o")
	h.handleToEventStep(chat, 0, "t2w_tmr")
	h.handleToEventStep(chat, 0, "t2d_30") // no estimate → the length is asked
	h.handleToEventStep(chat, 0, "t2ok")
	if !strings.Contains(cf.convertBody, `"repeat":"once"`) || !strings.Contains(cf.convertBody, `"mode":"move"`) {
		t.Errorf("convert body = %s", cf.convertBody)
	}
}

// A time typed without an hour is asked again, never guessed — a task has a
// due DAY, and inventing 09:00 puts it in the calendar at an hour nobody chose.
func TestToEventCustomTimeWithoutAnHourIsAskedAgain(t *testing.T) {
	h, fake, cf := convertAPI(t, plainTask, `{}`)
	const chat = 902
	h.handleToEventStart(chat, 0, "t1")
	h.handleToEventStep(chat, 0, "t2m_l")
	h.handleToEventStep(chat, 0, "t2c")
	h.handleToEventText(chat, "завтра")
	if cf.convertBody != "" {
		t.Fatalf("converted with no hour: %s", cf.convertBody)
	}
	if !strings.Contains(fake.last(t).Text, "время") {
		t.Errorf("did not ask for the hour again: %q", fake.last(t).Text)
	}
	h.handleToEventText(chat, "завтра 15:00")
	if !strings.Contains(fake.last(t).Markup, "t2ok") {
		t.Errorf("a full time did not reach the card: %s", fake.last(t).Markup)
	}
}

// A button from an abandoned flow does nothing to the calendar.
//
// ⚠ Two cases, because the easy one cannot fail: with no flow at all there is
// no task id, and the lookup refuses on its own — a first version of this test
// stayed green with the flow check removed. The second case is the one only
// that check stops: the path reached its card, then the user went elsewhere,
// and the ✅ from the old message is pressed while the old choices still sit
// in the store under a different flow.
func TestAStaleToEventButtonWritesNothing(t *testing.T) {
	h, _, cf := convertAPI(t, plainTask, `{}`)
	h.handleToEventStep(903, 0, "t2ok")
	if cf.convertBody != "" {
		t.Errorf("a stale ✅ with no flow converted: %s", cf.convertBody)
	}

	const chat = 904
	h.handleToEventStart(chat, 0, "t1")
	h.handleToEventStep(chat, 0, "t2m_l")
	h.handleToEventStep(chat, 0, "t2w_tmr")
	h.store.GetOrCreate(chat).CurrentFlow = "note" // the user moved on
	h.handleToEventStep(chat, 0, "t2ok")
	if cf.convertBody != "" {
		t.Errorf("a ✅ from an abandoned card converted: %s", cf.convertBody)
	}
}

// Every code the A1 endpoints answer with has a sentence, not the fallback.
func TestA1ErrorCodesHaveWords(t *testing.T) {
	h, _, _ := convertAPI(t, "", `{}`)
	fallback := h.errorText(920, &api.Error{Status: 400, Code: "SOMETHING_UNKNOWN"})
	for _, code := range []string{"REPEAT_CHOICE_REQUIRED", "OCCURRENCE_REQUIRED", "REPEAT_UNSUPPORTED", "NEEDS_TIME"} {
		if got := h.errorText(920, &api.Error{Status: 400, Code: code}); got == fallback {
			t.Errorf("%s falls back to the generic text", code)
		}
	}
}

// A slot on a day the series skips answers NOT_AN_OCCURRENCE. Here that does
// not mean «the series ended» — the ✅ button's wording — but «pick a day it
// has», and saying the first would send the user looking for a problem that
// is not there.
func TestADayOutsideTheSeriesIsExplainedAsSuch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/convert") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_AN_OCCURRENCE","message":"x"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[` + seriesTaskJSON + `]}`))
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	const chat = 921
	h.handleToEventStart(chat, 0, "t1")
	h.handleToEventStep(chat, 0, "t2m_l")
	h.handleToEventStep(chat, 0, "t2r_o")
	h.handleToEventStep(chat, 0, "t2w_tmr")
	h.handleToEventStep(chat, 0, "t2d_30")
	h.handleToEventStep(chat, 0, "t2ok")
	text := fake.last(t).Text
	if !strings.Contains(text, "не входит в серию") || strings.Contains(text, "закончилась") {
		t.Errorf("reply: %q", text)
	}
}
