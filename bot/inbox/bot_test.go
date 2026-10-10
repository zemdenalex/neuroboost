package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	owner     = int64(42)
	stranger  = int64(99)
	tokenTest = "123456:AAH-secret_shaped_token"
)

type call struct {
	method string
	form   map[string]string
}

// fakeTelegram records every Bot API call and answers each the way Telegram
// would, enough for the bot to carry on.
type fakeTelegram struct {
	mu    sync.Mutex
	calls []call
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	_ = r.ParseMultipartForm(1 << 20)
	form := map[string]string{}
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			form[k] = v[0]
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, call{method, form})
	f.mu.Unlock()
	result := `true`
	switch method {
	case "sendMessage":
		result = `{"message_id":1,"date":0,"chat":{"id":42,"type":"private"}}`
	case "getFile":
		result = `{"file_id":"f","file_unique_id":"u","file_path":"voice/file_1.oga"}`
	case "getMe":
		result = `{"id":1,"is_bot":true,"first_name":"inbox","username":"nb_inbox_bot"}`
	}
	_, _ = io.WriteString(w, `{"ok":true,"result":`+result+`}`)
}

func (f *fakeTelegram) byMethod(m string) []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []call
	for _, c := range f.calls {
		if c.method == m {
			out = append(out, c)
		}
	}
	return out
}

// fakeAPI is /api/inbox: it records what the bot posts.
type fakeAPI struct {
	mu      sync.Mutex
	creates []map[string]any
	media   []http.Header
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("X-Inbox-Token") != "api-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/inbox":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.creates = append(f.creates, body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"id":142,"type":"idea","projects":["avocar"],"status":"new"}}`)
	case strings.HasSuffix(r.URL.Path, "/media"):
		f.media = append(f.media, r.Header.Clone())
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"id":1}}`)
	default:
		_, _ = io.WriteString(w, `{"data":[]}`)
	}
}

// failingDownload stands in for a download that fails the way net/http does:
// the error text carries the URL, and the URL carries the token.
type failingDownload struct{}

func (failingDownload) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New(`Get "` + r.URL.String() + `": dial tcp: i/o timeout`)
}

func newService(t *testing.T) (*Service, *fakeTelegram, *fakeAPI, *bytes.Buffer) {
	t.Helper()
	tg, api := &fakeTelegram{}, &fakeAPI{}
	tgSrv, apiSrv := httptest.NewServer(tg), httptest.NewServer(api)
	t.Cleanup(tgSrv.Close)
	t.Cleanup(apiSrv.Close)
	b, err := bot.New(tokenTest, bot.WithServerURL(tgSrv.URL), bot.WithSkipGetMe())
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Owner: owner, Token: tokenTest, API: &API{Base: apiSrv.URL, Token: "api-token"},
		Projects: []string{"avocar", "graph"}, MaxBytes: 1 << 20,
		HTTP: &http.Client{Transport: failingDownload{}}}
	s.Bind(b)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(io.Discard) })
	return s, tg, api, &logs
}

func TestAStrangerGetsSilenceAndNothingIsSaved(t *testing.T) {
	s, tg, api, _ := newService(t)
	ctx := context.Background()
	s.Handle(ctx, nil, &models.Update{Message: &models.Message{ID: 1, Text: "/start",
		From: &models.User{ID: stranger}, Chat: models.Chat{ID: stranger}}})
	s.Handle(ctx, nil, &models.Update{ChosenInlineResult: &models.ChosenInlineResult{
		ResultID: "r1", From: models.User{ID: stranger}, Query: "#idea x"}})
	s.Handle(ctx, nil, &models.Update{InlineQuery: &models.InlineQuery{ID: "q1", From: &models.User{ID: stranger}, Query: "x"}})

	if len(api.creates) != 0 || len(tg.byMethod("sendMessage")) != 0 {
		t.Errorf("stranger got through: creates %v, sends %v", api.creates, tg.byMethod("sendMessage"))
	}
	inl := tg.byMethod("answerInlineQuery")
	if len(inl) != 1 || inl[0].form["results"] != "[]" {
		t.Errorf("stranger's inline query must get an empty list: %+v", inl)
	}
	// Positive control: the owner's /start is answered.
	s.Handle(ctx, nil, &models.Update{Message: &models.Message{ID: 2, Text: "/start",
		From: &models.User{ID: owner}, Chat: models.Chat{ID: owner}}})
	if len(tg.byMethod("sendMessage")) != 1 {
		t.Error("owner's /start was not answered")
	}
}

func TestInlineIsSavedOnTheChosenResultNotOnTheQuery(t *testing.T) {
	s, tg, api, _ := newService(t)
	ctx := context.Background()
	s.Handle(ctx, nil, &models.Update{InlineQuery: &models.InlineQuery{ID: "q1", From: &models.User{ID: owner}, Query: "#idea #avocar mini"}})
	if len(api.creates) != 0 {
		t.Fatal("an inline query (every keystroke) must not save")
	}
	if r := tg.byMethod("answerInlineQuery"); len(r) != 1 || !strings.Contains(r[0].form["results"], "#idea #avocar mini") {
		t.Errorf("inline answer: %+v", r)
	}
	s.Handle(ctx, nil, &models.Update{ChosenInlineResult: &models.ChosenInlineResult{
		ResultID: "r1", From: models.User{ID: owner}, Query: "#idea #avocar mini"}})
	if len(api.creates) != 1 {
		t.Fatalf("creates = %v", api.creates)
	}
	c := api.creates[0]
	if c["via"] != "inline" || c["client_key"] != "inline:r1" || c["type"] != "idea" {
		t.Errorf("create = %v", c)
	}
	if sends := tg.byMethod("sendMessage"); len(sends) != 1 || sends[0].form["chat_id"] != "42" || !strings.Contains(sends[0].form["text"], "#142") {
		t.Errorf("confirmation to the owner: %+v", sends)
	}
}

// gotcha 14: the download URL is …/file/bot<TOKEN>/…, so a failed download's
// error is the token. It must reach neither the log, nor the chat, nor the API.
func TestAFailedDownloadLeaksTheTokenNowhere(t *testing.T) {
	s, tg, api, logs := newService(t)
	s.save(context.Background(), owner, Draft{ClientKey: "chat:42:7", Text: "voice",
		Media: []Media{{Kind: "voice", FileID: "f", UniqueID: "u", Size: 10, Mime: "audio/ogg"}}})

	if len(api.media) != 1 || api.media[0].Get("X-Media-Omitted") != "1" {
		t.Fatalf("the file must be recorded without content: %v", api.media)
	}
	if !strings.Contains(logs.String(), "download") {
		t.Fatalf("control: the failure was not logged at all — the check below proves nothing:\n%s", logs)
	}
	secret := strings.SplitN(tokenTest, ":", 2)[1]
	everything := logs.String()
	for _, c := range tg.calls {
		everything += c.form["text"]
	}
	for _, h := range api.media {
		everything += strings.Join(h.Values("X-File-Name"), "")
	}
	if strings.Contains(everything, secret) {
		t.Errorf("token leaked:\n%s", everything)
	}
}
