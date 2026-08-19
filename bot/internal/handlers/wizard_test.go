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

// Renamed from TestWizardSkipsStepsTheLineAlreadyAnswered — that test
// ratified the deviation from the spec instead of catching it. Spec, part 3,
// verbatim: "Мастер стартует из карточки, то есть уже с тем, что разобрано
// из строки: шаг, значение которого уже известно, показывает его текущим и
// позволяет заменить." A step already answered by the line is SHOWN, with
// its known value, not skipped — skipping it made "📝 Подробнее" on a fully
// parsed line create the task with zero screens, indistinguishable from
// pressing "✅ Создать" outright, and left no way to fix a wrongly parsed
// field. `has` is deliberately still all-true here: the point is that it no
// longer changes the answer.
func TestWizardVisitsStepsTheLineAlreadyAnswered(t *testing.T) {
	has := map[string]bool{"priority": true, "due": true, "estimate": true}
	if got := nextWizardStep("start", has); got != "priority" {
		t.Errorf("nextWizardStep(start) with everything already known = %q, want priority — "+
			"the wizard must show the known value, not skip straight to done", got)
	}
}

func TestWizardStopsAtDone(t *testing.T) {
	// A wizard that walks past its last step loops forever on the user's screen.
	if got := nextWizardStep("done", map[string]bool{}); got != "done" {
		t.Errorf("nextWizardStep(done) = %q, want done", got)
	}
}
