package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// eventsHandler serves n upcoming events and each one by id.
func eventsHandler(t *testing.T, n int) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	start := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Hour)
	events := make([]map[string]any, 0, n)
	for i := 0; i < n; i++ {
		s := start.Add(time.Duration(i) * time.Hour)
		events = append(events, map[string]any{
			"id": fmt.Sprintf("e%02d", i), "title": fmt.Sprintf("событие %02d", i),
			"starts_at": s.Format(time.RFC3339), "ends_at": s.Add(time.Hour).Format(time.RFC3339),
		})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": events})
		case strings.HasPrefix(r.URL.Path, "/api/events/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/events/")
			for _, e := range events {
				if e["id"] == id {
					_ = json.NewEncoder(w).Encode(map[string]any{"data": e})
					return
				}
			}
			http.NotFound(w, r)
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"timezone": "Europe/Moscow", "settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9101)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	return h, fake, chat
}

func countButtons(markup, prefix string) int {
	return strings.Count(markup, `"callback_data":"`+prefix)
}

// Denis, 17.09: «Не могу открыть события позже первых 12, надо сделать первые
// 10, и … кнопки влево, номер страницы (вернуться к первой), вправо».
func TestEventPickerPagesByTen(t *testing.T) {
	h, fake, chat := eventsHandler(t, 23)
	press(h, chat, "event_pick")

	first := fake.last(t).Markup
	if n := countButtons(first, "ev_e"); n != 10 {
		t.Fatalf("page 1 has %d events, want 10: %s", n, first)
	}
	if !strings.Contains(first, `"1/3"`) || !strings.Contains(first, "evp_1") {
		t.Errorf("page 1 has no pager: %s", first)
	}

	press(h, chat, "evp_2")
	last := fake.last(t).Markup
	if n := countButtons(last, "ev_e"); n != 3 {
		t.Errorf("page 3 has %d events, want 3: %s", n, last)
	}
	if !strings.Contains(last, "ev_e22") {
		t.Errorf("the 23rd event is unreachable: %s", last)
	}
	// The page number goes back to the first page.
	if !strings.Contains(last, "evp_0") {
		t.Errorf("no way back to page 1: %s", last)
	}
}

func TestShortPickerHasNoPager(t *testing.T) {
	h, fake, chat := eventsHandler(t, 4)
	press(h, chat, "event_pick")
	if strings.Contains(fake.last(t).Markup, "evp_") {
		t.Errorf("four events do not need pages: %s", fake.last(t).Markup)
	}
}

// Denis, 17.09: «События — изменить — выбор события — изменить — изменить —
// выбор характеристики… зачем еще два раза это подтверждать». Picking an event
// opens its fields directly, with save and delete on the same screen.
func TestPickingAnEventOpensItsFields(t *testing.T) {
	h, fake, chat := eventsHandler(t, 3)
	press(h, chat, "event_pick")
	press(h, chat, "ev_e01")

	got := fake.last(t)
	for _, want := range []string{"dre_title", "dre_date", "dr_ok", "evd_e01"} {
		if !strings.Contains(got.Markup, want) {
			t.Errorf("the event screen lacks %s: %s", want, got.Markup)
		}
	}
	// And a field goes straight to its question, then back to the same screen.
	press(h, chat, "dre_title")
	say(h, chat, "новое название")
	back := fake.last(t)
	if !strings.Contains(back.Text, "новое название") || !strings.Contains(back.Markup, "dre_date") {
		t.Errorf("after an edit the fields are not one tap away: %q %s", back.Text, back.Markup)
	}
}
