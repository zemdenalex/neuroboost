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
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// Denis's pass 3 of 11.4, 23.09 — ref/feedback/bot-proverka-v04114-prohod3-otvet-denisa-2026-09-23.md,
// fix list docs/tasks-prohod3-2026-09-23.md.

const createdEventID = "eeeeeeee-1111-2222-3333-444444444444"

func eventAPIHandler(t *testing.T) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"timezone": "Europe/Moscow",
				"settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/events":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": createdEventID, "title": "ужин"}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9801)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat
}

// P1. «кнопка изменить после создания отправляет в список событий, надо чтобы
// после создания кнопка изменить отправляла в изменение созданного объекта».
func TestEditAfterCreatingAnEventOpensThatEvent(t *testing.T) {
	h, fake, chat := eventAPIHandler(t)
	say(h, chat, "ужин завтра 19:00")
	press(h, chat, "qa_event")
	press(h, chat, "dr_ok")

	got := fake.last(t)
	if !strings.Contains(got.Text, "Создано") {
		t.Fatalf("no creation answer: %q", got.Text)
	}
	if !strings.Contains(got.Markup, "eve_"+createdEventID) {
		t.Errorf("«✏️ Изменить» does not open the new event; markup = %s", got.Markup)
	}
	if strings.Contains(got.Markup, `"event_pick"`) {
		t.Errorf("«✏️ Изменить» still opens the event list; markup = %s", got.Markup)
	}
}
