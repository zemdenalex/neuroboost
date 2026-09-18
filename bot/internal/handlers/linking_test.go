package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/notifier"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// linkingHandler stands the bot up against an API that answers like the real
// one, and records which path was asked for.
func linkingHandler(t *testing.T) (*Handler, *fakeTelegram, int64, func() string) {
	t.Helper()
	var hit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = r.URL.Path
		switch r.URL.Path {
		case "/api/auth/login-link":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"token": "tok-abc123", "expires_in": 600},
			})
		case "/api/auth/link-code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"code": "048215", "expires_in": 600},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{APIBase: "https://dev.neuroboost.website"})
	const chat = int64(7400)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat, func() string { return hit }
}

// 🔴 The entrance must exist, in both languages. A button added in one language
// only is a feature that does not exist for half the users — and the settings
// menu is the only way in.
func TestSettingsOffersTheWebsiteAccount(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.RU, i18n.EN} {
		var found bool
		for _, row := range keyboards.SettingsMenu(lang).InlineKeyboard {
			for _, b := range row {
				if b.CallbackData != nil && *b.CallbackData == "lnk" {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s: settings has no website-account button", lang)
		}
	}
}

func TestBothDirectionsAreOffered(t *testing.T) {
	h, fake, chat, _ := linkingHandler(t)
	press(h, chat, "lnk")

	markup := fake.last(t).Markup
	for _, want := range []string{"lnk_web", "lnk_code"} {
		if !strings.Contains(markup, want) {
			t.Errorf("no %q button: %s", want, markup)
		}
	}
}

// The link the person receives must be a real address on the website, not the
// API's internal one, and it must carry the token.
func TestTheLoginLinkPointsAtTheWebsite(t *testing.T) {
	h, fake, chat, hit := linkingHandler(t)
	press(h, chat, "lnk_web")

	if hit() != "/api/auth/login-link" {
		t.Fatalf("the bot asked %q", hit())
	}
	text := fake.last(t).Text
	if !strings.Contains(text, "https://dev.neuroboost.website/login/link?t=tok-abc123") {
		t.Errorf("the message does not contain a usable link: %q", text)
	}
	// ⚠ Ten minutes must be SAID. A link that silently stops working is
	// indistinguishable from a broken one.
	if !strings.Contains(text, "10") {
		t.Errorf("the message does not say how long the link lives: %q", text)
	}
}

// 🔴 The leading zero is the point. A code shown as «48215» cannot be typed
// back, and the defect is invisible nine times out of ten.
func TestTheCodeKeepsItsLeadingZero(t *testing.T) {
	h, fake, chat, hit := linkingHandler(t)
	press(h, chat, "lnk_code")

	if hit() != "/api/auth/link-code" {
		t.Fatalf("the bot asked %q", hit())
	}
	if text := fake.last(t).Text; !strings.Contains(text, "048215") {
		t.Errorf("the code lost its leading zero or never arrived: %q", text)
	}
}

// The merge question must offer three answers, and «это не я» must not sit
// beside the two that mean yes.
func TestTheMergeQuestionHasThreeAnswersOnTwoRows(t *testing.T) {
	kb := notifier.Keyboard("LINK", "11111111-2222-3333-4444-555555555555", "ru")
	if kb == nil {
		t.Fatal("a merge request arrives with no buttons at all")
	}
	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("%d rows, expected the two answers above and the refusal below", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 2 || len(kb.InlineKeyboard[1]) != 1 {
		t.Errorf("rows are %d and %d, expected 2 and 1",
			len(kb.InlineKeyboard[0]), len(kb.InlineKeyboard[1]))
	}
	if !notifier.KeyboardFits(kb) {
		t.Error("a button is over Telegram's 64-byte callback_data cap — the whole message would be rejected")
	}

	// Every button must decode back into an action the API knows.
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			cb, ok := notifier.ParseCallback(*b.CallbackData)
			if !ok {
				t.Errorf("button %q does not decode", *b.CallbackData)
				continue
			}
			switch cb.Action {
			case notifier.ActionKeepSite, notifier.ActionKeepTg, notifier.ActionDecline:
			default:
				t.Errorf("button %q decodes to %q, which is not an answer to this question",
					*b.CallbackData, cb.Action)
			}
		}
	}
}
