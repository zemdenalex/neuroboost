package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// fakeAccount is /api/auth/me for one user: a timezone column and a settings
// blob, with every PATCH body recorded.
type fakeAccount struct {
	mu       sync.Mutex
	timezone string
	settings map[string]any
	patches  []map[string]any
	down     bool
	// writes records every POST/PATCH outside /api/auth/me, body decoded.
	writes []recordedWrite
}

type recordedWrite struct {
	Method, Path string
	Body         map[string]any
}

func (a *fakeAccount) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.down {
		http.Error(w, "down", http.StatusBadGateway)
		return
	}
	if r.URL.Path != "/api/auth/me" {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]any
			_ = json.Unmarshal(body, &decoded)
			a.writes = append(a.writes, recordedWrite{r.Method, r.URL.Path, decoded})
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "e1", "title": decoded["title"]}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		return
	}
	if r.Method == http.MethodPatch {
		body, _ := io.ReadAll(r.Body)
		var patch map[string]any
		_ = json.Unmarshal(body, &patch)
		a.patches = append(a.patches, patch)
		if s, ok := patch["settings"].(map[string]any); ok {
			a.settings = s
		}
		if tz, ok := patch["timezone"].(string); ok {
			a.timezone = tz
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
		"timezone": a.timezone, "settings": a.settings,
	}})
}

func (a *fakeAccount) onboarded() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	bot, _ := a.settings["bot"].(map[string]any)
	done, _ := bot["onboarded"].(bool)
	return done
}

func onboardHandler(t *testing.T, acc *fakeAccount) (*Handler, *fakeTelegram, int64) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(acc.serve))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{Timezone: "Europe/Moscow"})
	const chat = int64(9001)
	h.store.SetAuth(chat, "jwt", time.Now().Add(time.Hour).Unix())
	return h, fake, chat
}

func sayAs(h *Handler, chat int64, text, languageCode string) {
	msg := &tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{ID: chat},
		From: &tgbotapi.User{ID: chat, FirstName: "Mufid", LanguageCode: languageCode},
	}
	if strings.HasPrefix(text, "/") {
		msg.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(text)}}
	}
	h.HandleMessage(msg)
}

// 🔴 Mufid, 16.09: «Почему нет поддержки языков?» — it was there, in settings,
// where a first-time user never looks. The first /start now asks, pre-picked
// from Telegram: his Arabic client lands on English, not on Russian.
func TestFirstStartAsksTheLanguageGuessedFromTelegram(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, fake, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ar")

	got := fake.last(t)
	if !strings.Contains(got.Text, "Choose language") {
		t.Fatalf("the first /start did not ask the language: %q", got.Text)
	}
	if !strings.Contains(got.Markup, "✅ English") {
		t.Errorf("an Arabic Telegram client should pre-pick English; markup = %s", got.Markup)
	}
}

// Negative control: someone who finished onboarding gets the menu.
func TestOnboardedUserGetsTheMenu(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"bot": map[string]any{"onboarded": true}}}
	h, fake, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ru")

	for _, s := range fake.sent() {
		if strings.Contains(s.Text, "Выбери язык") || strings.Contains(s.Text, "Choose language") {
			t.Fatalf("an onboarded user was onboarded again: %q", s.Text)
		}
	}
}

// A failed read must not put everyone into the questionnaire.
func TestOnboardingIsNotShownWhenTheFlagCannotBeRead(t *testing.T) {
	acc := &fakeAccount{down: true}
	h, fake, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ru")

	for _, s := range fake.sent() {
		if strings.Contains(s.Text, "Выбери язык") {
			t.Fatalf("onboarding shown on a failed read")
		}
	}
}

// 🔴 A menu button in the middle of onboarding is «skip»: the flag is written,
// or the very next press would start the questionnaire again.
func TestMenuPressDuringOnboardingSkipsIt(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{"work_start": "09:00"}}
	h, _, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ru")
	if h.store.GetOrCreate(chat).CurrentFlow != onboardFlow {
		t.Fatalf("onboarding did not start — the rest proves nothing")
	}
	sayAs(h, chat, "🗓 Календарь", "ru")

	if !acc.onboarded() {
		t.Error("a menu press left onboarding unfinished")
	}
	if h.store.GetOrCreate(chat).CurrentFlow != "" {
		t.Errorf("flow after the menu press = %q, want none", h.store.GetOrCreate(chat).CurrentFlow)
	}
	if acc.settings["work_start"] != "09:00" {
		t.Error("writing the flag erased another setting (gotcha 21)")
	}
}

// Free text is a quick add even for someone never onboarded: the line they
// typed is what they want, and a questionnaire would lose it.
func TestFreeTextIsNotSwallowedByOnboarding(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, fake, chat := onboardHandler(t, acc)

	sayAs(h, chat, "завтра в 15 стоматолог", "ru")

	if !strings.Contains(fake.last(t).Markup, "qa_event") {
		t.Errorf("free text from a new user did not reach quick add: %+v", fake.last(t))
	}
}

// Pressing a clock writes the matching zone — and the sign is the trap.
func TestTappingAClockWritesTheZone(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, _, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ru")
	press(h, chat, "ob_tz")
	press(h, chat, "ob_tzset_5")

	if acc.timezone != "Etc/GMT-5" {
		t.Fatalf("timezone = %q, want Etc/GMT-5", acc.timezone)
	}
	loc, err := time.LoadLocation(acc.timezone)
	if err != nil {
		t.Fatalf("the written zone does not load: %v", err)
	}
	if _, off := time.Now().In(loc).Zone(); off != 5*3600 {
		t.Errorf("the zone for UTC+5 is %+d hours — the POSIX sign is inverted", off/3600)
	}
	// The bot itself now reads the user's zone, not the container's.
	if h.timezone(chat) != "Etc/GMT-5" {
		t.Errorf("the bot still uses %q", h.timezone(chat))
	}
}

// Pressing the time the account already has keeps the NAMED zone.
func TestTappingTheCurrentClockKeepsTheNamedZone(t *testing.T) {
	acc := &fakeAccount{timezone: "Europe/Moscow", settings: map[string]any{}}
	h, _, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "ru")
	press(h, chat, "ob_tz")
	press(h, chat, "ob_tzset_3")

	if acc.timezone != "Europe/Moscow" {
		t.Errorf("timezone = %q — Moscow was replaced by a bare offset", acc.timezone)
	}
}

func TestZoneFromText(t *testing.T) {
	cases := map[string]string{
		"UTC+5": "Etc/GMT-5", "+5": "Etc/GMT-5", "gmt-3": "Etc/GMT+3",
		"Asia/Yekaterinburg": "Asia/Yekaterinburg", "Екатеринбург": "", "UTC+99": "",
	}
	for in, want := range cases {
		if got := zoneFromText(in); got != want {
			t.Errorf("zoneFromText(%q) = %q, want %q", in, got, want)
		}
	}
}

// The closing message teaches one example per language. Both must do what the
// message says, or the first thing a new user copies fails.
func TestOnboardingExamplesParseAsAdvertised(t *testing.T) {
	now := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	for _, line := range []string{"завтра в 15 стоматолог напомни за час", "tomorrow at 15 dentist remind in 1h"} {
		p := parse.ParseLine(line, now)
		if !p.Draft.HasTime || p.Draft.Start != 15*time.Hour || p.Draft.Day.Day() != 17 {
			t.Errorf("%q: day %s time %v", line, p.Draft.Day.Format("2 Jan"), p.Draft.Start)
		}
		if p.Draft.ReminderOffsets == nil || (*p.Draft.ReminderOffsets)[0] != 60 {
			t.Errorf("%q: reminder %v, want 60", line, p.Draft.ReminderOffsets)
		}
		if p.Title != "стоматолог" && p.Title != "dentist" {
			t.Errorf("%q: title %q", line, p.Title)
		}
	}
}

// Review finding 3: the clock buttons are offsets from the ACCOUNT's zone, so a
// user in New York sees their own time marked, not a fixed UTC+2…+12 row.
func TestClockButtonsCentreOnTheAccountZone(t *testing.T) {
	acc := &fakeAccount{timezone: "America/New_York", settings: map[string]any{}}
	h, fake, chat := onboardHandler(t, acc)

	sayAs(h, chat, "/start", "en")
	press(h, chat, "ob_tz")

	markup := fake.last(t).Markup
	// A ticked TIME — «✅ Correct, next» carries a tick of its own.
	if !regexp.MustCompile(`✅ \d{2}:\d{2}`).MatchString(markup) {
		t.Errorf("no clock is marked for an account in New York: %s", markup)
	}
}
