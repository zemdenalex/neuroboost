package handlers

import (
	"strings"
	"testing"
)

// ⚙️ → 🎯 Задач в день: 3…7, written through PatchSettings (read, merge,
// write — gotcha 21), so no other key is lost.
func TestTheTargetIsSavedWithoutLosingOtherSettings(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleDayTarget(960, 0, "4")
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
	h.handleDayTarget(961, 0, "9")
	if *patched != "" {
		t.Errorf("wrote %s", *patched)
	}
}
