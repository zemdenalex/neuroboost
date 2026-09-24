package handlers

import (
	"strings"
	"testing"
)

// ⚙️ → 🎯 Задач в день: 3…7, written through PatchSettings (read, merge,
// write — gotcha 21), so no other key is lost.
func TestTheTargetIsSavedWithoutLosingOtherSettings(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleDaySettings(960, 0, "n:4")
	if !strings.Contains(*patched, `"day_tasks_target":4`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if got := fake.last(t); !strings.Contains(got.Markup, "✓ 4") {
		t.Errorf("the new target is not ticked: %s", got.Markup)
	}
}

// Callback data is user input: 9 is outside 3–7 and writes nothing.
func TestAnOutOfRangeTargetWritesNothing(t *testing.T) {
	h, _, patched := scaleAPI(t)
	h.handleDaySettings(961, 0, "n:9")
	if *patched != "" {
		t.Errorf("wrote %s", *patched)
	}
}

// Spec §11: ⚙️ → 📌 Задачи дня: the switch, the target, days before the start.
// Off leaves only the way back on; every write keeps the other keys.
func TestTheDaySettingsScreen(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleDaySettings(980, 0, "off")
	if !strings.Contains(*patched, `"day_tasks_enabled":false`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	m := fake.last(t).Markup
	if strings.Contains(m, "dtn_") || !strings.Contains(m, "dts_on") {
		t.Errorf("off: the target row must go, «включить» must stay: %s", m)
	}
	h.handleDaySettings(980, 0, "on")
	h.handleDaySettings(980, 0, "pb_on")
	if !strings.Contains(*patched, `"day_tasks_paint_before":true`) || !strings.Contains(*patched, `"day_tasks_enabled":true`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "dts_pb_off") || !strings.Contains(m, "dtn_") {
		t.Errorf("on + painting: %s", m)
	}
}
