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

// taskRepeatHandler stands the bot against an API that records what a created
// task was made of.
func taskRepeatHandler(t *testing.T) (*Handler, *fakeTelegram, int64, func() map[string]any) {
	t.Helper()
	var created map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tasks" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &created)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "t1", "title": created["title"]}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	const chat = int64(7500)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.SetLang("ru")
	return h, fake, chat, func() map[string]any { return created }
}

// 🔴 The whole of section 2 of the v0.4.11.4 checklist, in one test. Denis,
// 20.09: «не могу создать повторяющуюся задачу» — the bot had no field for it,
// no word for it, and sent nothing about it. This presses the buttons he
// pressed and reads what reached the API.
func TestARepeatingTaskReachesTheAPI(t *testing.T) {
	h, fake, chat, created := taskRepeatHandler(t)

	press(h, chat, "new_task")
	say(h, chat, "пить таблетки каждый день")

	card := fake.last(t).Text
	if !strings.Contains(card, "🔁") {
		t.Fatalf("the task card has no repeat line at all:\n%s", card)
	}
	if !strings.Contains(card, "каждый день") {
		t.Errorf("the card does not say what it understood about the repeat:\n%s", card)
	}
	if strings.Contains(card, "пить таблетки каждый") {
		t.Errorf("the repeat words were left in the title:\n%s", card)
	}

	press(h, chat, "nt_save")
	got := created()
	if got == nil {
		t.Fatal("nothing was created")
	}
	if got["rrule"] != "FREQ=DAILY" {
		t.Errorf("rrule on the wire = %v — the card said «каждый день» and the API was told nothing", got["rrule"])
	}
	if got["title"] != "пить таблетки" {
		t.Errorf("title = %v", got["title"])
	}
}

// A task nobody asked to repeat must say «нет» on the card (so the field is
// learnable — Denis, 18.09) and must send NO rrule, not an empty one.
func TestAPlainTaskSaysNoRepeatAndSendsNone(t *testing.T) {
	h, fake, chat, created := taskRepeatHandler(t)

	press(h, chat, "new_task")
	say(h, chat, "купить молоко")
	if card := fake.last(t).Text; !strings.Contains(card, "🔁") || !strings.Contains(card, "нет") {
		t.Errorf("a plain task's card does not name the repeat field as «нет»:\n%s", card)
	}
	press(h, chat, "nt_save")
	if _, present := created()["rrule"]; present {
		t.Errorf("a plain task sent an rrule key: %v", created()["rrule"])
	}
}

// The wizard must offer the repeat, and the answer must be kept.
func TestTheWizardAsksAboutRepeat(t *testing.T) {
	h, fake, chat, created := taskRepeatHandler(t)

	press(h, chat, "new_task")
	say(h, chat, "полить цветы")
	press(h, chat, "nt_wizard") // priority
	press(h, chat, "nt_skip")   // → due
	press(h, chat, "nt_skip")   // → repeat

	step := fake.last(t)
	if !strings.Contains(step.Markup, "nt_r_") {
		t.Fatalf("the third wizard step is not the repeat question: %q / %s", step.Text, step.Markup)
	}
	press(h, chat, "nt_r_w")  // weekly → estimate
	press(h, chat, "nt_skip") // → done, saves

	if got := created(); got == nil || got["rrule"] != "FREQ=WEEKLY" {
		t.Errorf("the wizard's repeat answer did not reach the API: %v", got)
	}
}

// 🔴 Denis's chat log, 20.09: «Сколько времени займёт?» — «5м» — «Здесь нужна
// кнопка». His question: «почему здесь нет своего варианта?» The parser has read
// «5м» since v0.4.11.1; the wizard refused to listen.
func TestTheEstimateStepAcceptsTypedMinutes(t *testing.T) {
	h, fake, chat, created := taskRepeatHandler(t)

	press(h, chat, "new_task")
	say(h, chat, "пить таблетки")
	press(h, chat, "nt_wizard")
	press(h, chat, "nt_skip")
	press(h, chat, "nt_skip")
	press(h, chat, "nt_skip") // → estimate

	say(h, chat, "5м")
	if strings.Contains(fake.last(t).Text, "нужна кнопка") {
		t.Fatalf("typed «5м» on the estimate step and was told to press a button")
	}
	if got := created(); got == nil || got["estimated_minutes"] != float64(5) {
		t.Errorf("the typed estimate did not reach the API: %v", got)
	}

	// ⚠ And nonsense must not become a task: it is asked again, the draft kept.
	h2, fake2, chat2, created2 := taskRepeatHandler(t)
	press(h2, chat2, "new_task")
	say(h2, chat2, "пить таблетки")
	press(h2, chat2, "nt_wizard")
	press(h2, chat2, "nt_skip")
	press(h2, chat2, "nt_skip")
	press(h2, chat2, "nt_skip")
	say(h2, chat2, "быстро")
	if created2() != nil {
		t.Errorf("«быстро» was accepted as an estimate and the task saved: %v", created2())
	}
	if !strings.Contains(fake2.last(t).Text, "30м") {
		t.Errorf("after a bad estimate the bot did not say what it understands: %q", fake2.last(t).Text)
	}
}
