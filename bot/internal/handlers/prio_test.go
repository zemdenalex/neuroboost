package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// 🔴 Spec 21.09 §B1: every screen draws the priority through the chat's style.
// One direct format.PriorityEmoji left in a handler is a screen that stays in
// circles after the user picked dashes — and nobody notices.
func TestNoHandlerDrawsPriorityOutsideTheStyle(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no source files: %v", err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "PriorityEmoji(") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				t.Errorf("%s:%d draws a priority past the style: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned < 20 {
		t.Fatalf("scanned %d files — the glob is not reading the package", scanned)
	}
}

// prioAPI serves settings (applying PATCHes) and one task.
func prioAPI(t *testing.T, bot map[string]string) (*Handler, *fakeTelegram, *string) {
	t.Helper()
	settings := `{"bot":{"lang":"ru"`
	for k, v := range bot {
		settings += `,"` + k + `":"` + v + `"`
	}
	settings += `,"onboarded":true}}`
	var patched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			patched = string(b)
			s := string(b)
			settings = s[strings.Index(s, ":")+1 : len(s)-1]
			_, _ = w.Write([]byte(`{"data":{}}`))
		case r.URL.Path == "/api/auth/me":
			_, _ = w.Write([]byte(`{"data":{"settings":` + settings + `}}`))
		case r.URL.Path == "/api/tasks":
			_, _ = w.Write([]byte(`{"data":[{"id":"t1","title":"позвонить в банк","status":"TODO","priority":1}]}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	botAPI, fake := newFakeTelegram(t)
	return New(botAPI, api.NewClient(srv.URL), state.NewStore(), config.Config{}), fake, &patched
}

func TestTheChosenStyleDrawsTheTaskList(t *testing.T) {
	h, fake, _ := prioAPI(t, map[string]string{"priority_style": "dot"})
	h.handleTasks(960, 0)
	if text := fake.last(t).Text; !strings.Contains(text, "●1 позвонить в банк") {
		t.Errorf("list:\n%s", text)
	}
}

func TestSettingsSavesThePriorityStyle(t *testing.T) {
	h, fake, patched := prioAPI(t, nil)
	h.handlePriorityPick(961, 0, "dash", "prs_")
	if !strings.Contains(*patched, `"priority_style":"dash"`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	if m := fake.last(t).Markup; !strings.Contains(m, "✓ — — —") || !strings.Contains(m, `"settings_menu"`) {
		t.Errorf("markup: %s", m)
	}
	h2, _, patched2 := prioAPI(t, nil)
	h2.handlePriorityPick(962, 0, "rainbow", "prs_")
	if *patched2 != "" {
		t.Errorf("an unknown style wrote %s", *patched2)
	}
}

// Spec §B2: people onboarded before the choice existed are asked once — and
// only once, even if they walk away without choosing.
func TestExistingUsersAreAskedOnce(t *testing.T) {
	h, fake, patched := prioAPI(t, nil)
	const chat = 963
	h.store.GetOrCreate(chat).Onboarded = true

	h.handleMenu(chat, 0)
	if !strings.Contains(fake.last(t).Text, "Символ приоритета") || !strings.Contains(*patched, `"priority_style_asked":"1"`) {
		t.Fatalf("first menu: %q, PATCH %s", fake.last(t).Text, *patched)
	}
	h.handleMenu(chat, 0)
	if strings.Contains(fake.last(t).Text, "Символ приоритета") {
		t.Error("asked a second time")
	}
}

// Spec §B1: with dashes the card names the priority in words, or it would be
// visible nowhere.
func TestADashCardNamesThePriority(t *testing.T) {
	h, fake, _ := prioAPI(t, map[string]string{"priority_style": "dash"})
	h.handleTaskAction(964, 0, "t1")
	if text := fake.last(t).Text; !strings.Contains(text, "Приоритет: срочно") {
		t.Errorf("card:\n%s", text)
	}
}
