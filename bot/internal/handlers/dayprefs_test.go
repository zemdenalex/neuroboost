package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Spec §11: no key means on and not painting; the target is 3…7, else 5.
func TestReadDayPrefsDefaults(t *testing.T) {
	p := readDayPrefs(map[string]any{})
	if !p.On || p.PaintBefore || p.Target != 5 {
		t.Errorf("empty settings: %+v, want on, not painting, 5", p)
	}
	p = readDayPrefs(map[string]any{"day_tasks_enabled": false, "day_tasks_paint_before": true, "day_tasks_target": float64(3)})
	if p.On || !p.PaintBefore || p.Target != 3 {
		t.Errorf("set settings: %+v", p)
	}
	if p := readDayPrefs(map[string]any{"day_tasks_target": float64(9)}); p.Target != 5 {
		t.Errorf("target 9 read as %d, want 5", p.Target)
	}
}

// Review Focus 1: a failed read hides nothing.
func TestAFailedReadKeepsDayTasksOn(t *testing.T) {
	a := &dayAPI{meDown: true}
	h, _, chat := dayHandler(t, a)
	if !h.dayTasksOn(chat) {
		t.Error("a failed settings read switched day tasks off")
	}
}

// Review Focus 5: switching keeps every other key (gotcha 21).
func TestSwitchingKeepsOtherSettings(t *testing.T) {
	h, _, patched := scaleAPI(t)
	if err := h.setDayPref(970, "day_tasks_enabled", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*patched, `"day_tasks_enabled":false`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if h.dayTasksOn(970) {
		t.Error("cache still says on")
	}
}

func TestTheHomeHidesDayTasksWhenOff(t *testing.T) {
	if m := markupJSON(keyboards.HomeInlineFor(i18n.RU, false)); strings.Contains(m, "dt_d_today") {
		t.Errorf("off, and the menu still has 📌: %s", m)
	}
	if m := markupJSON(keyboards.HomeInlineFor(i18n.RU, true)); !strings.Contains(m, "dt_d_today") {
		t.Errorf("on, and 📌 is missing: %s", m)
	}
}

func markupJSON(m any) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// Every handler draws the home keyboard through h.home, or a chat with day
// tasks off would still see 📌 under some screen.
func TestHandlersUseTheHomeMethod(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "dayprefs.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "keyboards.HomeInlineFor(") {
				t.Errorf("%s:%d draws the home keyboard directly; use h.home(chatID)", f, i+1)
			}
		}
	}
	if scanned < 20 {
		t.Fatalf("scanned %d files; the scan is not looking at the handlers", scanned)
	}
}
