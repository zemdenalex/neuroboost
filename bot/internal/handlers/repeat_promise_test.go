package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// «⚠ повтор — частота не указана, спрошу» must be followed by asking.
//
// 🔴 Denis, 21.09, writing directly under the checkbox he had just ticked:
// «Ну пишет спрошу, но создать даёт без спроса». The card had printed that
// promise since 18.09 and no code anywhere acted on it.
//
// The test drives handleTaskCardSave with a draft in exactly that state. The
// handler's API client points at a dead port, so a creation attempt cannot be
// mistaken for success: if the guard is gone, what reaches the screen is the
// failure to create, which is what this asserts against.
func taskDraft(h *Handler, chatID int64, data map[string]any) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_task"
	us.FlowStep = "card"
	us.FlowData = data
}

func TestAPromisedQuestionIsAsked(t *testing.T) {
	h, fake := newTestHandler(t)
	taskDraft(h, 500, map[string]any{"title": "пить таблетки", "repeat_asked": true})

	h.handleTaskCardSave(500, 0)

	last := fake.last(t).Text
	if strings.Contains(last, "Не удалось создать") || strings.Contains(last, "Задача создана") {
		t.Fatalf("the card promised to ask and went straight to creating:\n\t%s", last)
	}
	if !strings.Contains(last, "Повторять") && !strings.Contains(last, "Repeat") {
		t.Errorf("the repeat question never reached the screen:\n\t%s", last)
	}
	if step := h.store.GetOrCreate(500).FlowStep; step != "wizard:repeat" {
		t.Errorf("flow step = %q, want wizard:repeat — the answer has nowhere to land", step)
	}
}

// 🔴 And the answer comes straight back to creating.
//
// «повтор» is the third of four wizard steps, so advancing from it lands on
// «Сколько времени займёт?» — a second screen for someone who pressed ✅
// Создать. Denis's rule for this bot, from the 17.09 passes: «каждое
// замечание — в сторону меньшего числа шагов».
func TestAnsweringFromTheCardGoesBackToCreating(t *testing.T) {
	h, fake := newTestHandler(t)
	taskDraft(h, 503, map[string]any{"title": "пить таблетки", "repeat_asked": true})

	h.handleTaskCardSave(503, 0)      // asks
	h.handleWizardRepeat(503, 0, "d") // «Каждый день»

	last := fake.last(t).Text
	if strings.Contains(last, "займёт") || strings.Contains(last, "take") {
		t.Errorf("answering the repeat question walked on into the estimate step:\n\t%s", last)
	}
	// The API client points at a dead port, so «could not create» IS the
	// evidence that creation was attempted — the screen after this one.
	if !strings.Contains(last, "Не удалось создать") && !strings.Contains(last, "Задача создана") {
		t.Errorf("the answer did not lead back to creating:\n\t%s", last)
	}
}

// And the answered rule reaches the wire.
//
// ⚠ Not assertable from FlowData: handleTaskCardSave clears the flow once it
// has sent the request, so looking afterwards finds an empty draft whether the
// rule was carried or dropped. The first draft of this test asserted exactly
// that and failed for the wrong reason. The only place the answer is visible
// is the request body.
func TestTheAnsweredRuleReachesTheRequest(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only the create: the home screen reads settings afterwards (GET),
		// and keeping the last body would lose the one this test is about.
		if r.Method == http.MethodPost {
			body, _ = io.ReadAll(r.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"t1","title":"пить таблетки"}}`))
	}))
	defer srv.Close()

	bot, _ := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	taskDraft(h, 504, map[string]any{"title": "пить таблетки", "repeat_asked": true})

	h.handleTaskCardSave(504, 0)
	h.handleWizardRepeat(504, 0, "d")

	if !strings.Contains(string(body), `"rrule":"FREQ=DAILY"`) {
		t.Errorf("the answered rule never reached the API:\n\t%s", string(body))
	}
}

// 🔴 Asked ONCE. Skipping the step, or pressing ✅ again, must move on: a
// reading of «спрошу» as «refuse until answered» builds a screen the user
// cannot leave, and «пропустить» is a button on that very step.
func TestTheQuestionIsAskedOnceNotUntilAnswered(t *testing.T) {
	h, fake := newTestHandler(t)
	taskDraft(h, 501, map[string]any{"title": "пить таблетки", "repeat_asked": true})

	h.handleTaskCardSave(501, 0) // asks
	if asked, _ := h.store.GetOrCreate(501).FlowData["repeat_asked"].(bool); asked {
		t.Fatal("the flag survived the question; the second press would ask again")
	}

	h.store.GetOrCreate(501).FlowStep = "card"
	h.handleTaskCardSave(501, 0) // second press: must go through

	last := fake.last(t).Text
	if strings.Contains(last, "Повторять") || strings.Contains(last, "Repeat") {
		t.Errorf("asked a second time — this is the loop:\n\t%s", last)
	}
}

// And a draft that already carries a rule is not interrogated about it.
func TestAKnownRuleIsNotAskedAbout(t *testing.T) {
	h, fake := newTestHandler(t)
	taskDraft(h, 502, map[string]any{
		"title":        "пить таблетки",
		"repeat_asked": true,
		"rrule":        "FREQ=DAILY",
	})

	h.handleTaskCardSave(502, 0)

	if last := fake.last(t).Text; strings.Contains(last, "Повторять") || strings.Contains(last, "Repeat") {
		t.Errorf("asked about a frequency the draft already has:\n\t%s", last)
	}
}
