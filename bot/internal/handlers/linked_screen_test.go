package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// A linked task's time shows where the user looks — the list line and the
// card — through the real handlers, against an API that answers the way
// api-go does: the task in /api/tasks, its event (with task_id) in
// /api/events. The helpers' own tests prove the arithmetic; this proves
// somebody calls them.
func linkedScreenAPI(t *testing.T) (*Handler, *fakeTelegram) {
	t.Helper()
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	start := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/events") {
			_, _ = w.Write([]byte(`{"data":[{"id":"ev1","title":"банк","starts_at":"` + start +
				`","ends_at":"` + start + `","task_id":"t1"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"t1","title":"позвонить в банк","status":"SCHEDULED","priority":1},` +
			`{"id":"t2","title":"просто задача","status":"TODO","priority":2}]}`))
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	return New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{}), fake
}

func TestTheListMarksALinkedTask(t *testing.T) {
	h, fake := linkedScreenAPI(t)
	h.handleTasks(930, 0)
	text := fake.last(t).Text
	if !strings.Contains(text, "позвонить в банк   📅 завтра") {
		t.Errorf("the scheduled task is missing or unmarked:\n%s", text)
	}
	if strings.Contains(text, "просто задача   📅") {
		t.Errorf("an unlinked task got a mark:\n%s", text)
	}
}

func TestTheCardSaysWhenAndOpensTheEvent(t *testing.T) {
	h, fake := linkedScreenAPI(t)
	h.handleTaskAction(931, 0, "t1")
	got := fake.last(t)
	if !strings.Contains(got.Text, "Запланирована на завтра") {
		t.Errorf("card text:\n%s", got.Text)
	}
	if !strings.Contains(got.Markup, `"ev_ev1"`) {
		t.Errorf("no way to the event: %s", got.Markup)
	}
}
