package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// recordingAPI answers GET /api/tasks with one repeating task and remembers
// the body of any PATCH.
func recordingAPI(t *testing.T, task string) (*Handler, *fakeTelegram, *string) {
	t.Helper()
	var patched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch {
			b, _ := io.ReadAll(r.Body)
			patched = string(b)
			_, _ = w.Write([]byte(`{"data":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[` + task + `]}`))
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	return New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{}), fake, &patched
}

const repeatingTask = `{"id":"t1","title":"пить таблетки","status":"TODO","priority":3,` +
	`"rrule":"FREQ=DAILY","nag_minutes":10}`

// 🔴 «Не повторять» must send an EMPTY rule, not leave the key out.
//
// api-go reads `"rrule": ""` as «clear it», and an absent key as «don't
// touch». A button that omits the key answers with success and changes
// nothing — the failure mode this project keeps meeting: a control that
// cannot fail, on the user's side of the screen.
func TestTurningARepeatOffClearsTheRule(t *testing.T) {
	h, _, patched := recordingAPI(t, repeatingTask)

	h.handleTaskRepeatSet(700, 0, "t1", "n")

	if !strings.Contains(*patched, `"rrule":""`) {
		t.Errorf("PATCH body %q does not clear the rule", *patched)
	}
}

func TestChangingARepeatSendsTheNewRule(t *testing.T) {
	h, _, patched := recordingAPI(t, repeatingTask)

	h.handleTaskRepeatSet(701, 0, "t1", "w")

	if !strings.Contains(*patched, `"rrule":"FREQ=WEEKLY"`) {
		t.Errorf("PATCH body %q does not carry the new rule", *patched)
	}
}

// An unknown code writes nothing at all. Callback data comes from the user's
// client, and a stale or crafted button must not reach the database.
func TestAnUnknownRepeatCodeWritesNothing(t *testing.T) {
	h, _, patched := recordingAPI(t, repeatingTask)

	h.handleTaskRepeatSet(702, 0, "t1", "yearly")

	if *patched != "" {
		t.Errorf("an unknown code produced a write: %q", *patched)
	}
}

func TestNaggingCanBeSetAndSwitchedOff(t *testing.T) {
	h, _, patched := recordingAPI(t, repeatingTask)
	h.handleTaskNagSet(703, 0, "t1", "30")
	if !strings.Contains(*patched, `"nag_minutes":30`) {
		t.Errorf("PATCH body %q does not set the interval", *patched)
	}

	h2, _, patched2 := recordingAPI(t, repeatingTask)
	h2.handleTaskNagSet(704, 0, "t1", "off")
	if !strings.Contains(*patched2, `"nag_minutes":0`) {
		t.Errorf("PATCH body %q does not switch nagging off", *patched2)
	}
}

// 🔴 A number that could not be read keeps the question open.
//
// Throwing the flow away on a typo is the defect Denis named twice — 17.09,
// «текст на вопросе о списке задач снова стирал всё», and 20.09 for the
// wizard steps. A screen that asks for a number and forgets everything when
// it gets a word has punished the user for answering.
func TestAnUnreadableNumberKeepsThePostponeQuestionOpen(t *testing.T) {
	h, fake := newTestHandler(t)
	us := h.store.GetOrCreate(705)
	us.CurrentFlow = "postpone_custom"
	us.FlowStep = "text"
	us.FlowData["taskID"] = "t1"

	h.handlePostponeCustomText(705, "быстро")

	if got := h.store.GetOrCreate(705).CurrentFlow; got != "postpone_custom" {
		t.Errorf("flow = %q, want it still open", got)
	}
	if last := fake.last(t).Text; !strings.Contains(last, "число") {
		t.Errorf("the screen did not say what it wants:\n\t%s", last)
	}
}

func TestPostponeRefusesDaysOutsideItsRange(t *testing.T) {
	for _, text := range []string{"0", "-3", "4000", "10 дней и ещё"} {
		h, _ := newTestHandler(t)
		us := h.store.GetOrCreate(706)
		us.CurrentFlow = "postpone_custom"
		us.FlowStep = "text"
		us.FlowData["taskID"] = "t1"

		h.handlePostponeCustomText(706, text)

		if got := h.store.GetOrCreate(706).CurrentFlow; got != "postpone_custom" {
			t.Errorf("%q was accepted — flow is now %q", text, got)
		}
	}
}

// 🔴 A hint only where typing actually works.
//
// Denis asked for «нажми кнопку или напиши своё» after finding out «5м» was
// accepted and unadvertised. Printing it on a step that ignores text would be
// the same defect as the card that promised to ask and did not.
func TestTheHintAppearsOnlyWhereTypingIsHeard(t *testing.T) {
	for _, step := range []string{"estimate", "repeat"} {
		if wizardHint(i18n.RU, step) == "" {
			t.Errorf("step %q takes a typed answer and says nothing about it", step)
		}
		if !strings.Contains(wizardStepText(i18n.RU, step, map[string]any{}, nil, format.StyleCircles), "напиши") {
			t.Errorf("step %q: the hint never reaches the screen", step)
		}
	}
	for _, step := range []string{"priority", "due"} {
		if wizardHint(i18n.RU, step) != "" {
			t.Errorf("step %q promises typing it does not accept", step)
		}
	}
}
