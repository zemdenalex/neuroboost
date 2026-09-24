package handlers

import (
	"strings"
	"testing"
)

// Spec §11: onboarding has a day-tasks step after the scale. «Не сейчас»
// switches them off and ends onboarding; «Включить» asks N, then finishes.
func TestOnboardingAsksAboutDayTasks(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"lang": "ru"}}}
	h, fake, chat := onboardHandler(t, acc)
	h.startOnboarding(chat, 0, "ru")
	h.handleOnboardCallback(chat, 0, "ob_scale", nil)
	if m := fake.last(t).Markup; !strings.Contains(m, `"ob_dt"`) {
		t.Fatalf("the scale step does not lead to day tasks: %s", m)
	}
	h.handleOnboardCallback(chat, 0, "ob_dt", nil)
	if m := fake.last(t).Markup; !strings.Contains(m, "ob_dt_on") || !strings.Contains(m, "ob_dt_off") {
		t.Fatalf("the day-tasks step: %s", m)
	}
	h.handleOnboardCallback(chat, 0, "ob_dt_off", nil)
	if v, ok := acc.settings["day_tasks_enabled"].(bool); !ok || v {
		t.Errorf("«Не сейчас» wrote %v, want false", acc.settings["day_tasks_enabled"])
	}
	if !acc.onboarded() {
		t.Error("«Не сейчас» did not finish onboarding")
	}

	acc = &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"lang": "ru"}}}
	h, fake, chat = onboardHandler(t, acc)
	h.startOnboarding(chat, 0, "ru")
	h.handleOnboardCallback(chat, 0, "ob_dt_on", nil)
	if v, _ := acc.settings["day_tasks_enabled"].(bool); !v {
		t.Errorf("«Включить» wrote %v", acc.settings["day_tasks_enabled"])
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "ob_dtn_") || !strings.Contains(m, `"ob_finish"`) {
		t.Fatalf("after «Включить» the target row: %s", m)
	}
	h.handleOnboardCallback(chat, 0, "ob_dtn_4", nil)
	if v, _ := acc.settings["day_tasks_target"].(float64); v != 4 {
		t.Errorf("target = %v, want 4", acc.settings["day_tasks_target"])
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "✓ 4") {
		t.Errorf("4 is not ticked: %s", m)
	}
}

// Review Focus 3: people onboarded before D3 are asked once; someone who has
// already answered (the key is there) is never asked.
func TestOldUsersAreAskedAboutDayTasksOnce(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{
		"bot": map[string]any{"lang": "ru", "onboarded": true}, "day_tasks_enabled": true}}
	h, _, chat := onboardHandler(t, acc)
	h.store.GetOrCreate(chat).Onboarded = true
	if h.askDayTasksOnce(chat, 0) {
		t.Error("asked someone who already chose")
	}

	acc = &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{
		"bot": map[string]any{"lang": "ru", "onboarded": true}}}
	h, fake, chat2 := onboardHandler(t, acc)
	chat = chat2
	h.store.GetOrCreate(chat).Onboarded = true
	if !h.askDayTasksOnce(chat, 0) {
		t.Fatal("did not ask an old user")
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "dtq_on") || !strings.Contains(m, "dtq_off") {
		t.Errorf("the question: %s", m)
	}
	if h.askDayTasksOnce(chat, 0) {
		t.Error("asked twice")
	}
	press(h, chat, "dtq_off")
	if v, ok := acc.settings["day_tasks_enabled"].(bool); !ok || v {
		t.Errorf("«Не сейчас» wrote %v, want false", acc.settings["day_tasks_enabled"])
	}
}
