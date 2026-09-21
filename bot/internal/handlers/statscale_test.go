package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// scaleAPI keeps the settings blob and applies PATCHes, so a redraw after a
// write reads what was written — the way the real API behaves.
func scaleAPI(t *testing.T) (*Handler, *fakeTelegram, *string) {
	t.Helper()
	settings := `{"bot":{"lang":"ru"}}`
	var patched string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			patched = string(b)
			// {"settings": {...}} → keep the inner object.
			s := string(b)
			settings = s[strings.Index(s, ":")+1 : len(s)-1]
			_, _ = w.Write([]byte(`{"data":{}}`))
		case r.URL.Path == "/api/auth/me":
			_, _ = w.Write([]byte(`{"data":{"settings":` + settings + `}}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	return New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{}), fake, &patched
}

// Denis 22.09: the scale is chosen in ⚙️ Settings too.
func TestSettingsSavesTheScaleAndTicksIt(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	h.handleScalePick(950, 0, "work", false)
	if !strings.Contains(*patched, `"stats_scale":"work"`) || !strings.Contains(*patched, `"lang":"ru"`) {
		t.Fatalf("PATCH = %s", *patched)
	}
	got := fake.last(t)
	if !strings.Contains(got.Markup, "✓ рабочие часы") || !strings.Contains(got.Markup, `"settings_menu"`) {
		t.Errorf("markup: %s", got.Markup)
	}
}

// Callback data is user input: an unknown scale writes nothing.
func TestAnUnknownScaleWritesNothing(t *testing.T) {
	h, _, patched := scaleAPI(t)
	h.handleScalePick(951, 0, "yearly", false)
	if *patched != "" {
		t.Errorf("wrote %s", *patched)
	}
}

// Denis 22.09: «during onboarding it should ask too». The timezone screen
// leads to the scale, the scale leads to the end.
func TestOnboardingAsksForTheScale(t *testing.T) {
	h, fake, patched := scaleAPI(t)
	const chat = 952
	h.startOnboarding(chat, 0, "ru")
	h.handleOnboardCallback(chat, 0, "ob_tz", nil)
	// Zone → priority symbol (11.4 §B) → scale → end.
	if !strings.Contains(fake.last(t).Markup, `"ob_prio"`) {
		t.Fatalf("the timezone screen does not lead on: %s", fake.last(t).Markup)
	}
	h.handleOnboardCallback(chat, 0, "ob_prio", nil)
	if !strings.Contains(fake.last(t).Markup, `"ob_scale"`) {
		t.Fatalf("the priority step does not lead to the scale: %s", fake.last(t).Markup)
	}
	h.handleOnboardCallback(chat, 0, "ob_scale", nil)
	h.handleOnboardCallback(chat, 0, "ob_sc_peak", nil)
	if !strings.Contains(*patched, `"stats_scale":"peak"`) {
		t.Errorf("PATCH = %s", *patched)
	}
	if m := fake.last(t).Markup; !strings.Contains(m, `"ob_finish"`) || !strings.Contains(m, "✓ по максимуму") {
		t.Errorf("markup: %s", m)
	}
}

func TestScaleButtonsFitAndParse(t *testing.T) {
	for _, prefix := range []string{"scl_", "ob_sc_"} {
		eachButton(keyboards.StatsScale(i18n.RU, "day24", prefix, "settings_menu", "«"), func(data string) {
			if len(data) > 64 {
				t.Errorf("%q too long", data)
			}
			if kind := strings.TrimPrefix(data, prefix); data != "settings_menu" && !scaleKinds[kind] {
				t.Errorf("%q is not a known scale", data)
			}
		})
	}
}
