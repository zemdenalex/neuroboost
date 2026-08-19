package handlers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

func TestTaskCardShowsOnlyWhatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if !strings.Contains(got, "позвонить в банк") {
		t.Errorf("the title is missing:\n%s", got)
	}
	if strings.Contains(got, "📅") || strings.Contains(got, "⏱") {
		t.Errorf("the card invented a due date or an estimate:\n%s", got)
	}
}

func TestTaskCardShowsEverythingThatWasParsed(t *testing.T) {
	r := parse.ParseTask("позвонить в банк завтра 30м !1", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	for _, want := range []string{"📅", "⏱", "30"} {
		if !strings.Contains(got, want) {
			t.Errorf("card is missing %q:\n%s", want, got)
		}
	}
}

func TestTaskCardEscapesTheTitle(t *testing.T) {
	r := parse.ParseTask("R&D <срочно>", time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC))
	got := taskCardText(r, "UTC")
	if strings.Contains(got, "<срочно>") {
		t.Errorf("an unescaped title reached an HTML message:\n%s", got)
	}
}

func TestUnsetPriorityIsAbsentFromTheWire(t *testing.T) {
	// 🔴 The whole point of *int. 0 is Buffer — a real choice. If an unset
	// priority serialised as 0, "skip" would silently mean "lowest urgency".
	body, err := json.Marshal(api.CreateTaskReq{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "priority") {
		t.Errorf("an unset priority was sent anyway: %s", body)
	}
}

func TestBufferPriorityIsSentAsZero(t *testing.T) {
	zero := 0
	body, err := json.Marshal(api.CreateTaskReq{Title: "x", Priority: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"priority":0`) {
		t.Errorf("Buffer (0) did not reach the wire: %s", body)
	}
}

// Spec, part 3: a step whose value is already known shows it as the current
// value. wizardStepText carries it even when the value does not land on one
// of the keyboard's quick choices — an arbitrary parsed date, here.
func TestWizardStepTextShowsTheKnownValue(t *testing.T) {
	loc := time.UTC

	flowData := map[string]any{"priority": 1}
	got := wizardStepText("priority", flowData, loc)
	if !strings.Contains(got, "Emergency") {
		t.Errorf("priority step text does not show the known value:\n%s", got)
	}

	due := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	flowData = map[string]any{"due": due}
	got = wizardStepText("due", flowData, loc)
	if !strings.Contains(got, "25.08") {
		t.Errorf("due step text does not show the known date:\n%s", got)
	}

	flowData = map[string]any{"minutes": 45}
	got = wizardStepText("estimate", flowData, loc)
	if !strings.Contains(got, "45") {
		t.Errorf("estimate step text does not show the known minutes:\n%s", got)
	}

	// Nothing known — the old, unconditional prompt.
	got = wizardStepText("priority", map[string]any{}, loc)
	if strings.Contains(got, "Сейчас") {
		t.Errorf("priority step text claims a current value with nothing known:\n%s", got)
	}
}

func TestWizardKeyboardForMarksTheKnownValue(t *testing.T) {
	loc := time.UTC
	kb := wizardKeyboardFor("priority", map[string]any{"priority": 3}, loc)
	found := false
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != nil && *b.CallbackData == "nt_p_3" && strings.HasPrefix(b.Text, "✓") {
				found = true
			}
		}
	}
	if !found {
		t.Error("wizardKeyboardFor(priority) with priority=3 known did not mark nt_p_3")
	}
}

// B1's blocker: nt_save must work from a wizard step, not just from the card.
// This tests the guard directly rather than button presence — the button
// already existed on every wizard screen (TestEveryWizardKeyboardOffersBothEscapes
// proves that), and it was still dead.
func TestSaveFromAWizardStepActuallySaves(t *testing.T) {
	h, fake := newTestHandler(t)
	chatID := int64(777)

	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_task"
	us.FlowStep = "wizard:due"
	us.FlowData["title"] = "позвонить в банк"
	us.FlowData["priority"] = 1

	// newTestHandler's bot construction itself calls getMe against the fake —
	// snapshot after setup so the assertion below is about what
	// handleTaskCardSave did, not about bot start-up noise.
	before := len(fake.calls())

	h.handleTaskCardSave(chatID, 0)

	if len(fake.calls()) == before {
		t.Fatal("nt_save from wizard:due produced no Telegram call at all — the guard silently ate it")
	}
	if h.store.GetOrCreate(chatID).CurrentFlow != "" {
		t.Error("the flow was not cleared — handleTaskCardSave returned before reaching its work")
	}
}
