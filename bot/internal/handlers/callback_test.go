package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

// A fake Telegram, recording which methods the handler called.
//
// Built against a real *tgbotapi.BotAPI rather than an interface: the panic
// under test happened inside the handler's own field access, so replacing the
// client with a mock type would have moved the code being tested.
type fakeTelegram struct {
	srv    *httptest.Server
	mu     sync.Mutex
	called []string
	// texts holds the text and the reply markup of every sendMessage and
	// editMessageText, in order — what the user would have read.
	texts []sentText
}

type sentText struct {
	Method string
	Text   string
	Markup string
	// Entities is the formatting sent with the text as entities (not HTML) —
	// how a restored screen keeps its bold.
	Entities string
}

func newFakeTelegram(t *testing.T) (*tgbotapi.BotAPI, *fakeTelegram) {
	t.Helper()
	f := &fakeTelegram{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path is /bot<token>/<method>
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		method := parts[len(parts)-1]

		_ = r.ParseForm()
		f.mu.Lock()
		f.called = append(f.called, method)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		// 🔴 Real Telegram rejects a keyboard serialised as null — «Bad
		// Request: field "inline_keyboard" must be of type Array». This fake
		// used to accept anything, so three dead buttons (Другое…, 📖,
		// Переписать) passed their tests and failed on Denis's phone, 17.09.
		// A fake that cannot say no is not a control.
		if strings.Contains(r.Form.Get("reply_markup"), `"inline_keyboard":null`) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": false, "error_code": 400,
				"description": `Bad Request: field "inline_keyboard" must be of type Array`,
			})
			return
		}
		// Recorded only once accepted: a message Telegram refused was never read.
		if text := r.Form.Get("text"); text != "" {
			f.mu.Lock()
			f.texts = append(f.texts, sentText{Method: method, Text: text, Markup: r.Form.Get("reply_markup"),
				Entities: r.Form.Get("entities")})
			f.mu.Unlock()
		}
		switch method {
		case "getMe":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":     true,
				"result": map[string]any{"id": 1, "is_bot": true, "username": "test_bot"},
			})
		case "sendMessage", "editMessageText":
			// 🔴 23.09: these answered «result: true» like every other method,
			// but real Telegram returns the Message. The library failed to
			// decode «true» into a Message, so every edit looked failed and
			// editOrSend fell back to a new message — an in-place edit could
			// not be observed as one in any test. A fake that cannot say what
			// Telegram says is not a control.
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{
				"message_id": 1, "date": 0, "chat": map[string]any{"id": 1, "type": "private"}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
		}
	}))
	t.Cleanup(f.srv.Close)

	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint("123:test", f.srv.URL+"/bot%s/%s")
	if err != nil {
		t.Fatalf("could not build a bot against the fake: %v", err)
	}
	return bot, f
}

func (f *fakeTelegram) sent() []sentText {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentText(nil), f.texts...)
}

func (f *fakeTelegram) last(t *testing.T) sentText {
	t.Helper()
	all := f.sent()
	if len(all) == 0 {
		t.Fatalf("the bot sent nothing; calls = %v", f.calls())
	}
	return all[len(all)-1]
}

func (f *fakeTelegram) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.called...)
}

func (f *fakeTelegram) called_(method string) bool {
	for _, c := range f.calls() {
		if c == method {
			return true
		}
	}
	return false
}

func newTestHandler(t *testing.T) (*Handler, *fakeTelegram) {
	t.Helper()
	bot, fake := newFakeTelegram(t)
	return New(bot, api.NewClient("http://127.0.0.1:1"), state.NewStore(), config.Config{}), fake
}

// 🔴 The defect: CallbackQuery.Message is optional in the Bot API — Telegram
// omits it for messages that are too old, and for inline messages. The handler
// dereferenced it unconditionally, and this loop is the whole process, so one
// user pressing a button under an old reminder killed the bot for everybody.
func TestCallbackWithoutMessageDoesNotPanic(t *testing.T) {
	h, fake := newTestHandler(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a callback with no Message panicked — this kills the process: %v", r)
		}
	}()

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:   "cb-1",
		Data: "today_focus",
		From: &tgbotapi.User{ID: 42, FirstName: "Denis"},
		// Message deliberately nil — this is what Telegram sends.
	})

	// Not panicking is not enough. Telegram spins the button forever until the
	// callback is answered, which the user reads as a hung bot rather than as a
	// stale message.
	if !fake.called_("answerCallbackQuery") {
		t.Errorf("the callback was not answered; calls = %v", fake.calls())
	}
}

// The negative control. Without it, a HandleCallback that returned immediately
// for EVERY callback would satisfy the test above while breaking every button.
func TestCallbackWithAMessageStillReachesItsHandler(t *testing.T) {
	h, fake := newTestHandler(t)

	h.HandleCallback(&tgbotapi.CallbackQuery{
		ID:      "cb-2",
		Data:    "today_focus",
		From:    &tgbotapi.User{ID: 42, FirstName: "Denis"},
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 4242}},
	})

	// With no reachable API the auth step fails and the handler tells the user
	// so — which proves execution got past the nil guard and into the body.
	if !fake.called_("sendMessage") {
		t.Errorf("a normal callback produced no message; calls = %v", fake.calls())
	}
}
