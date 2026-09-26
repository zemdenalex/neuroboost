package handlers

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// One language per person (Denis 26.09), review of 96c0d82: onboarding showed
// the language guessed from Telegram but saved nothing unless the person
// tapped a language, so the account kept `locale` ru and the Mini App opened
// in Russian for someone the bot spoke English to.
func TestOnboardingSavesTheLanguageItGuessed(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, _, chat := onboardHandler(t, acc)
	h.startOnboarding(chat, 0, "en-GB")

	for _, p := range acc.patches {
		bot, _ := p["settings"].(map[string]any)["bot"].(map[string]any)
		if p["locale"] == "en" && bot["lang"] == "en" {
			return
		}
	}
	t.Fatalf("the guessed language was not saved as locale + bot.lang: %v", acc.patches)
}

// The same review: the bot kept a chat's language for the whole process, so a
// language chosen in the web never reached the chat until a redeploy. After a
// while it asks again.
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
