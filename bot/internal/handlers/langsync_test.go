package handlers

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Onboarding shows the language guessed from Telegram and saves it, even when
// the person keeps the pre-selected one. It is the BOT's language only: the
// account's locale belongs to the web and the Mini App (Denis 26.09, the bot
// «should stay in [its] own language»).
func TestOnboardingSavesTheLanguageItGuessedForTheBotOnly(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, _, chat := onboardHandler(t, acc)
	h.startOnboarding(chat, 0, "en-GB")

	saved := false
	for _, p := range acc.patches {
		bot, _ := p["settings"].(map[string]any)["bot"].(map[string]any)
		saved = saved || bot["lang"] == "en"
	}
	if !saved {
		t.Fatalf("the guessed language was not saved as bot.lang: %v", acc.patches)
	}
}

// The bot kept a chat's language for the whole process, so a change made by
// another bot process (dev and prod redeploys, a second chat) stuck until a
// redeploy. After a while it asks again.
func TestTheBotPicksUpALanguageChangedElsewhere(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"lang": "ru"}}}
	h, _, chat := onboardHandler(t, acc)
	if got := h.lang(chat); got != i18n.RU {
		t.Fatalf("first read: %v", got)
	}
	acc.mu.Lock()
	acc.settings = map[string]any{"bot": map[string]any{"lang": "en"}}
	acc.mu.Unlock()

	if got := h.lang(chat); got != i18n.RU {
		t.Errorf("re-read at once: %v — the cache should still answer", got)
	}
	h.store.GetOrCreate(chat).LangAt = time.Now().Add(-langTTL - time.Second)
	if got := h.lang(chat); got != i18n.EN {
		t.Errorf("after the cache aged: %v, want en (changed in the web)", got)
	}
}

// Gap list row 16 (Denis 26.09): the priority style is set in the web too, so
// the chat's cached copy ages out like the language does.
func TestTheBotPicksUpAPriorityStyleChangedElsewhere(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"lang": "ru", "priority_style": "circles"}}}
	h, _, chat := onboardHandler(t, acc)
	if got := h.priorityStyle(chat); got != "circles" {
		t.Fatalf("first read: %q", got)
	}
	acc.mu.Lock()
	acc.settings = map[string]any{"bot": map[string]any{"lang": "ru", "priority_style": "dot"}}
	acc.mu.Unlock()
	h.store.GetOrCreate(chat).PriorityStyleAt = time.Now().Add(-langTTL - time.Second)
	if got := h.priorityStyle(chat); got != "dot" {
		t.Errorf("after the cache aged: %q, want dot (changed in the web)", got)
	}
}
