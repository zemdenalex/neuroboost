package menubutton

import (
	"encoding/json"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type fakeBot struct {
	endpoint string
	params   tgbotapi.Params
	err      error
}

func (f *fakeBot) MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error) {
	f.endpoint, f.params = endpoint, params
	if f.err != nil {
		return nil, f.err
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

// The chat menu button opens the web app inside Telegram (Mini App). The
// library predates setChatMenuButton, so the request is built by hand and its
// shape is what Telegram documents: menu_button = MenuButtonWebApp JSON.
func TestSetSendsAWebAppMenuButton(t *testing.T) {
	b := &fakeBot{}
	if err := Set(b, "https://dev.neuroboost.website/calendar", "Календарь"); err != nil {
		t.Fatal(err)
	}
	if b.endpoint != "setChatMenuButton" {
		t.Fatalf("endpoint = %q", b.endpoint)
	}
	var got struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		WebApp struct {
			URL string `json:"url"`
		} `json:"web_app"`
	}
	if err := json.Unmarshal([]byte(b.params["menu_button"]), &got); err != nil {
		t.Fatalf("menu_button is not JSON: %q", b.params["menu_button"])
	}
	if got.Type != "web_app" || got.Text != "Календарь" || got.WebApp.URL != "https://dev.neuroboost.website/calendar" {
		t.Fatalf("menu_button = %+v", got)
	}
	if _, ok := b.params["chat_id"]; ok {
		t.Fatal("chat_id set: the default button for every chat has none")
	}
}

// Telegram only opens https Mini Apps; an http or empty URL is refused here
// rather than silently installing a button that shows an error.
func TestSetRefusesANonHTTPSURL(t *testing.T) {
	for _, u := range []string{"", "http://dev.neuroboost.website", "dev.neuroboost.website"} {
		b := &fakeBot{}
		if err := Set(b, u, "x"); err == nil {
			t.Fatalf("%q accepted", u)
		}
		if b.endpoint != "" {
			t.Fatalf("%q: request sent anyway", u)
		}
	}
}

func TestSetReportsTelegramsRefusal(t *testing.T) {
	if err := Set(&fakeBot{err: errors.New("Bad Request")}, "https://x.example", "x"); err == nil {
		t.Fatal("error swallowed")
	}
}
