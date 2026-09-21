package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/release"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
)

const (
	adminChat   = 5001
	okChat      = 700000001 // delivered
	blockedChat = 700000002 // Telegram answers 403: the bot is blocked
)

// bcWorld is a Telegram that refuses one chat and an API that knows two
// recipients, remembering every service call and every mark.
type bcWorld struct {
	mu       sync.Mutex
	sentTo   map[string]int // chat_id → sendMessage count
	texts    []string       // every text Telegram accepted
	svcCalls int
	marks    map[int64]int
	isAdmin  bool
	settings string
	patched  string
}

func newBCWorld(t *testing.T, admin bool) (*Handler, *bcWorld) {
	t.Helper()
	w := &bcWorld{sentTo: map[string]int{}, marks: map[int64]int{}, isAdmin: admin, settings: `{"bot":{"lang":"ru"}}`}

	tg := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		rw.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			_ = json.NewEncoder(rw).Encode(map[string]any{"ok": true, "result": map[string]any{"id": 1, "is_bot": true, "username": "t"}})
			return
		}
		chat := r.Form.Get("chat_id")
		if chat == "700000002" {
			rw.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(rw).Encode(map[string]any{"ok": false, "error_code": 403, "description": "Forbidden: bot was blocked by the user"})
			return
		}
		w.mu.Lock()
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			w.sentTo[chat]++
		}
		if txt := r.Form.Get("text"); txt != "" {
			w.texts = append(w.texts, txt)
		}
		w.mu.Unlock()
		_ = json.NewEncoder(rw).Encode(map[string]any{"ok": true, "result": map[string]any{"message_id": 1, "date": 0, "chat": map[string]any{"id": 1}}})
	}))
	t.Cleanup(tg.Close)

	apiSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		w.mu.Lock()
		defer w.mu.Unlock()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/svc/broadcast/recipients"):
			w.svcCalls++
			_, _ = rw.Write([]byte(`{"data":[{"tg_id":700000001,"lang":"ru"},{"tg_id":700000002,"lang":"en"}]}`))
		case r.URL.Path == "/api/svc/broadcast/mark":
			w.svcCalls++
			var m struct {
				TgID int64 `json:"tg_id"`
				Code int   `json:"code"`
			}
			_ = json.NewDecoder(r.Body).Decode(&m)
			w.marks[m.TgID] = m.Code
			_, _ = rw.Write([]byte(`{"data":{}}`))
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			w.patched = string(b)
			s := string(b)
			w.settings = s[strings.Index(s, ":")+1 : len(s)-1]
			_, _ = rw.Write([]byte(`{"data":{}}`))
		case r.URL.Path == "/api/auth/me":
			admin := "false"
			if w.isAdmin {
				admin = "true"
			}
			_, _ = rw.Write([]byte(`{"data":{"is_admin":` + admin + `,"settings":` + w.settings + `}}`))
		default:
			_, _ = rw.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(apiSrv.Close)

	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint("123:test", tg.URL+"/bot%s/%s")
	if err != nil {
		t.Fatal(err)
	}
	return New(bot, api.NewClient(apiSrv.URL), state.NewStore(), config.Config{ServiceToken: "svc"}), w
}

// Spec D2: a non-admin sends nothing — and asks the service nothing.
func TestOnlyAnAdminCanBroadcast(t *testing.T) {
	h, w := newBCWorld(t, false)
	h.handleBroadcastCommand(adminChat)
	h.handleBroadcastGo(adminChat, 0, release.Latest().Version)
	if w.svcCalls != 0 || w.sentTo["700000001"] != 0 {
		t.Errorf("a non-admin reached the service %d times, sent %d", w.svcCalls, w.sentTo["700000001"])
	}
}

// Spec D2: the dry run sends nothing to anybody but the admin.
func TestTheDryRunSendsNothing(t *testing.T) {
	h, w := newBCWorld(t, true)
	h.handleBroadcastCommand(adminChat)
	if w.sentTo["700000001"] != 0 || len(w.marks) != 0 {
		t.Errorf("dry run sent %d, marked %v", w.sentTo["700000001"], w.marks)
	}
	if last := w.texts[len(w.texts)-1]; !strings.Contains(last, "Получателей: 2 (ru 1 · en 1)") {
		t.Errorf("dry run text:\n%s", last)
	}
}

// Spec D1/D2: every recipient's result is recorded — 200 and 403 — and
// neither the admin's report nor the log carries a Telegram id.
func TestSendingRecordsEachRecipientAndLeaksNoIDs(t *testing.T) {
	h, w := newBCWorld(t, true)
	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(prev) })

	h.handleBroadcastGo(adminChat, 0, release.Latest().Version)

	if w.marks[okChat] != 200 || w.marks[blockedChat] != 403 {
		t.Errorf("marks = %v, want 200 and 403", w.marks)
	}
	report := w.texts[len(w.texts)-1]
	if !strings.Contains(report, "Дошло: 1 · заблокировали бота: 1 · не дошло: 0") {
		t.Errorf("report:\n%s", report)
	}
	for _, id := range []string{"700000001", "700000002"} {
		if strings.Contains(report, id) || strings.Contains(logs.String(), id) {
			t.Errorf("id %s leaked (report %q, log %q)", id, report, logs.String())
		}
	}
	if !strings.Contains(logs.String(), idHash(okChat)) {
		t.Errorf("the log does not name the recipient by hash: %q", logs.String())
	}
}

// A button from an older dry run must not send.
func TestAStaleBroadcastButtonSendsNothing(t *testing.T) {
	h, w := newBCWorld(t, true)
	h.handleBroadcastGo(adminChat, 0, "v0.0.1")
	if len(w.marks) != 0 || w.sentTo["700000001"] != 0 {
		t.Errorf("sent on a stale button: %v", w.marks)
	}
}

// Spec D1: «🔕 Не присылать обновления» under the broadcast, and back again.
func TestUnsubscribingKeepsTheRestOfTheSettings(t *testing.T) {
	h, w := newBCWorld(t, false)
	h.handleUpdatesSet(okChat, 0, "off")
	if !strings.Contains(w.patched, `"updates":"off"`) || !strings.Contains(w.patched, `"lang":"ru"`) {
		t.Errorf("PATCH = %s", w.patched)
	}
	h.handleUpdatesSet(okChat, 0, "on")
	if !strings.Contains(w.patched, `"updates":"on"`) {
		t.Errorf("PATCH = %s", w.patched)
	}
}
