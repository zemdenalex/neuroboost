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

// calendarHandler wires a handler to a fake API serving three calendars: the
// caller's personal one, a shared one they own, and someone else's.
//
// Field names are the API's own (kind / role / status, api-go
// internal/calendars/types.go:21) — a fake that answers with invented names
// would let a misspelt tag pass, and a misspelt tag in Go decodes to the zero
// value with no error at all.
func calendarHandler(t *testing.T) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	cals := []any{
		map[string]any{"id": "cal-personal", "name": "Личный", "color": nil, "kind": "personal", "role": "owner", "status": "active"},
		map[string]any{"id": "cal-work", "name": "Работа", "color": "#7c3aed", "kind": "shared", "role": "owner", "status": "active"},
		map[string]any{"id": "cal-family", "name": "Семья", "color": "#22c55e", "kind": "shared", "role": "viewer", "status": "active"},
	}
	members := []any{
		map[string]any{"user_id": "u1", "email": "d@example.com", "display_name": "Denis", "role": "owner", "status": "active"},
		map[string]any{"user_id": "u2", "email": nil, "display_name": nil, "role": "editor", "status": "active"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/calendars" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": cals})
		case strings.HasSuffix(r.URL.Path, "/members"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": members})
		case strings.HasSuffix(r.URL.Path, "/invite-links"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"token": "Zm9vYmFyYmF6"}})
		case r.URL.Path == "/api/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id":       "u1",
				"timezone": "Europe/Moscow",
				"settings": map[string]any{"bot": map[string]any{"onboarded": true, "lang": "ru"}}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		}
	}))
	t.Cleanup(srv.Close)

	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{BotUsername: "NeuroBoost_dev_bot"})
	const chat = int64(7100)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	us := h.store.GetOrCreate(chat)
	us.Lang, us.LangKnown = "ru", true
	return h, fake, chat
}

func TestCalendarListShowsAllThree(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cls")

	got := fake.last(t)
	for _, name := range []string{"Личный", "Работа", "Семья"} {
		if !strings.Contains(got.Text+got.Markup, name) {
			t.Errorf("список не показал %q: %q / %s", name, got.Text, got.Markup)
		}
	}
	if !strings.Contains(got.Markup, "cl_new") {
		t.Errorf("нет кнопки «создать»: %s", got.Markup)
	}
}

// 🔴 Кнопка — заявка, обработчик — свидетельство. Offering «Удалить» on a
// calendar the API will refuse to delete teaches the user the bot lies —
// «есть кнопка, но она не работает» was three separate defects on 17.09.
func TestForeignCalendarOffersLeaveAndNotDelete(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cl_cal-family")

	got := fake.last(t)
	if strings.Contains(got.Markup, "cl_del_") {
		t.Errorf("чужой календарь предлагает удаление: %s", got.Markup)
	}
	if !strings.Contains(got.Markup, "cl_leave_") {
		t.Errorf("чужой календарь не предлагает выйти: %s", got.Markup)
	}
	for _, forbidden := range []string{"cl_name_", "cl_col_", "cl_inv_"} {
		if strings.Contains(got.Markup, forbidden) {
			t.Errorf("не владелец видит %s: %s", forbidden, got.Markup)
		}
	}
}

// The personal calendar can be neither deleted (ErrCalendarIsPersonal guards
// the delete path) nor left (an owner may not leave — members.go:304).
func TestPersonalCalendarOffersNeitherLeaveNorDelete(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cl_cal-personal")

	got := fake.last(t)
	for _, forbidden := range []string{"cl_del_", "cl_leave_"} {
		if strings.Contains(got.Markup, forbidden) {
			t.Errorf("личный календарь предлагает %s: %s", forbidden, got.Markup)
		}
	}
	if !strings.Contains(got.Markup, "cl_name_") {
		t.Errorf("личный календарь нельзя переименовать: %s", got.Markup)
	}
}

// The owned shared calendar is the one with everything.
func TestOwnedSharedCalendarOffersEverything(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cl_cal-work")

	got := fake.last(t)
	for _, want := range []string{"cl_name_", "cl_col_", "cl_inv_", "cl_mem_", "cl_del_"} {
		if !strings.Contains(got.Markup, want) {
			t.Errorf("владелец не видит %s: %s", want, got.Markup)
		}
	}
	if strings.Contains(got.Markup, "cl_leave_") {
		t.Errorf("владельцу предложили выйти — API это отвергнет: %s", got.Markup)
	}
}

// The card follows the same rule as Task 1: every characteristic is named,
// empty ones say «нет».
func TestCalendarCardNamesEveryField(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cl_cal-personal")

	got := fake.last(t)
	for _, label := range []string{"Цвет:", "Участник", "Роль:"} {
		if !strings.Contains(got.Text, label) {
			t.Errorf("карточка календаря не называет %q:\n%s", label, got.Text)
		}
	}
	// The personal calendar is created colourless by design, so this is the
	// real «нет» case rather than a contrived one.
	if !strings.Contains(got.Text, "нет") {
		t.Errorf("пустой цвет не сказал «нет»:\n%s", got.Text)
	}
}

// 🔴 Every callback this screen emits must fit Telegram's 64-byte
// callback_data cap. A UUID is 36 bytes, so the prefix budget is 27 — and an
// oversized button makes Telegram refuse the WHOLE message, not just that
// button.
func TestCalendarCallbacksFitTheBudget(t *testing.T) {
	h, fake, chat := calendarHandler(t)
	press(h, chat, "cl_11111111-2222-3333-4444-555555555555")
	press(h, chat, "cls")

	for _, m := range fake.sent() {
		pieces := strings.Split(m.Markup, `"callback_data":"`)
		for _, piece := range pieces[1:] {
			end := strings.Index(piece, `"`)
			if end < 0 {
				continue
			}
			if data := piece[:end]; len(data) > 64 {
				t.Errorf("callback_data %d bytes, cap is 64: %q", len(data), data)
			}
		}
	}
}

// 🔴 The control that was missing when this screen was written, and the reason
// it is here: the calendars screen was first given the prefix «cal_», which the
// month grid has owned since 19.08 (cal_prev_, cal_next_, cal_day_ —
// keyboards/keyboards.go:237,252). The new handler is asked BEFORE the switch
// that routes those, so every day press in the month calendar would have been
// swallowed and answered with «calendar not found».
//
// Every test in this file still passed. Nothing in the suite pressed a day
// button, so the whole month view could break in silence — which is precisely
// the shape of «есть кнопка, но она не работает».
func TestCalendarScreenDoesNotSwallowTheMonthGrid(t *testing.T) {
	h, _, chat := calendarHandler(t)

	for _, data := range []string{"cal_day_2026-09-19", "cal_prev_2026_9", "cal_next_2026_9"} {
		if h.handleCalendarsCallback(chat, 55, data) {
			t.Errorf("the calendars screen claimed %q, which belongs to the month grid", data)
		}
	}

	// The positive control: it must still claim its own.
	for _, data := range []string{"cls", "cl_new", "cl_cal-work"} {
		if !h.handleCalendarsCallback(chat, 55, data) {
			t.Errorf("the calendars screen did not claim its own callback %q", data)
		}
	}
}
