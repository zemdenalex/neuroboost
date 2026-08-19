package handlers

import "testing"

func TestWizardVisitsEveryStepInOrder(t *testing.T) {
	has := map[string]bool{}
	order := []string{"priority", "due", "estimate", "done"}
	cur := "start"
	for _, want := range order {
		cur = nextWizardStep(cur, has)
		if cur != want {
			t.Fatalf("after %q the wizard went to %q, want %q", cur, cur, want)
		}
	}
}

func TestWizardSkipsStepsTheLineAlreadyAnswered(t *testing.T) {
	// 🔴 The line "позвонить завтра 30м !1" answered all three. Asking anyway
	// makes the fast path slower than the wizard, which defeats having one.
	has := map[string]bool{"priority": true, "due": true, "estimate": true}
	if got := nextWizardStep("start", has); got != "done" {
		t.Errorf("wizard asked %q about a line that stated everything", got)
	}
}

func TestWizardStopsAtDone(t *testing.T) {
	// A wizard that walks past its last step loops forever on the user's screen.
	if got := nextWizardStep("done", map[string]bool{}); got != "done" {
		t.Errorf("nextWizardStep(done) = %q, want done", got)
	}
}
