package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/release"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

func feedbackHandler(t *testing.T) (*Handler, *fakeTelegram, int64, func() map[string]any) {
	t.Helper()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feedback" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})
	const chat = int64(7300)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.SetLang("ru")
	return h, fake, chat, func() map[string]any { return got }
}

// 🔴 The client call for this existed since v0.4.9 and was wired to nothing —
// api.SubmitFeedback had zero callers. The plumbing being present is not the
// feature; this test presses the button.
func TestBugReportReachesTheAPI(t *testing.T) {
	h, fake, chat, sent := feedbackHandler(t)

	press(h, chat, "fb_bug")
	say(h, chat, "Кнопка «Все слова» ничего не делает. Нажимаю — тишина.")

	got := sent()
	if got == nil {
		t.Fatal("ничего не ушло в API")
	}
	if got["type"] != "bug" {
		t.Errorf("type = %v, expected bug: %+v", got["type"], got)
	}
	if !strings.Contains(fmt.Sprint(got["description"]), "Нажимаю") {
		t.Errorf("текст обращения доехал не целиком: %+v", got)
	}
	// The title must be present and must be the FIRST sentence, not the whole
	// message: an admin list of sixty identical truncations is unreadable.
	title := fmt.Sprint(got["title"])
	if title == "" || strings.Contains(title, "Нажимаю") {
		t.Errorf("заголовок пуст или равен всему тексту: %q", title)
	}
	if !strings.Contains(fake.last(t).Text, "Спасибо") {
		t.Errorf("человеку не сказали, что приняли: %q", fake.last(t).Text)
	}
}

func TestIdeaIsSentAsFeature(t *testing.T) {
	h, _, chat, sent := feedbackHandler(t)

	press(h, chat, "fb_idea")
	say(h, chat, "Хочу повторяющиеся задачи")

	if got := sent(); got == nil || got["type"] != "feature" {
		t.Errorf("идея ушла не как feature: %+v", got)
	}
}

// An empty message must not create an empty ticket.
func TestEmptyFeedbackIsNotSent(t *testing.T) {
	h, _, chat, sent := feedbackHandler(t)

	press(h, chat, "fb_bug")
	say(h, chat, "   ")

	if got := sent(); got != nil {
		t.Errorf("пустое обращение всё же ушло: %+v", got)
	}
}

// 🔴 Both entrances must exist, in both languages. A button added in one
// language only is exactly the defect still open for the notification buttons.
func TestSettingsOffersFeedbackAndWhatsNew(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.RU, i18n.EN} {
		var data []string
		for _, row := range keyboards.SettingsMenu(lang).InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil {
					data = append(data, *b.CallbackData)
				}
			}
		}
		for _, want := range []string{"fb", "whatsnew", "cls"} {
			var found bool
			for _, d := range data {
				if d == want {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: settings has no %q button: %v", lang, want, data)
			}
		}
	}
}

// The release notes screen must show the version this release IS — otherwise
// the feature fails on its first use, quietly.
func TestWhatsNewShowsTheCurrentVersion(t *testing.T) {
	h, fake, chat, _ := feedbackHandler(t)
	press(h, chat, "whatsnew")

	got := fake.last(t)
	// ⚠ Asks the notes what the newest version IS rather than naming one.
	// Pinned to "v0.4.11.3" this went red the moment 11.4 was added — a test
	// that has to be edited on every release teaches people to edit tests.
	if !strings.Contains(got.Text, release.Latest().Version) {
		t.Errorf("«что нового» не показывает текущую версию: %q", got.Text)
	}
	if !strings.Contains(got.Markup, "whatsnew_all") {
		t.Errorf("нет кнопки прошлых версий: %s", got.Markup)
	}
}
