package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 🔴 gotcha 21 again, and the stakes are higher here than for one keyword: the
// language is the FIRST thing a new user sets, so this write lands on a blob
// full of everything else they have ever configured.
func TestSetBotLangDoesNotWriteWhenTheReadFailed(t *testing.T) {
	wrote := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			wrote = true
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotLang("tok", "en"); err == nil {
		t.Error("a failed read reported success")
	}
	if wrote {
		t.Error("the bot wrote settings it had not read — this erases the rest of the blob")
	}
}

// Setting the language must not cost the user their keyword vocabulary, which
// lives in the same `bot` section.
func TestSetBotLangKeepsTheKeywords(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &sent)
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"settings": map[string]any{
				"work_start": "09:00",
				"bot": map[string]any{"keywords": map[string]any{
					"созвон": map[string]any{"field": "calendar", "value": "Работа"},
				}},
			}},
		})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotLang("tok", " EN "); err != nil {
		t.Fatalf("SetBotLang: %v", err)
	}

	settings, _ := sent["settings"].(map[string]any)
	if settings["work_start"] != "09:00" {
		t.Error("a setting outside the bot section was dropped")
	}
	bot, _ := settings["bot"].(map[string]any)
	if bot == nil {
		t.Fatalf("the bot section vanished: %+v", settings)
	}
	if bot["lang"] != "en" {
		t.Errorf("lang = %v, want «en» — trimmed and lowercased", bot["lang"])
	}
	if _, ok := bot["keywords"]; !ok {
		t.Error("setting the language erased the keyword vocabulary")
	}
}

func TestBotLangToleratesAnyShape(t *testing.T) {
	for _, settings := range []map[string]any{
		nil, {}, {"bot": "не карта"}, {"bot": map[string]any{"lang": 42}},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"settings": settings}})
		}))
		got, err := NewClient(srv.URL).BotLang("tok")
		srv.Close()
		if err != nil {
			t.Errorf("%+v gave an error: %v", settings, err)
		}
		if got != "" {
			t.Errorf("%+v gave %q, want an empty language", settings, got)
		}
	}
}

// One language per person (Denis 26.09): the bot's language is also the
// account's `locale`, so the Mini App opens in it. One PATCH carries both, so
// the two can never disagree after a half-done write.
func TestSetBotLangAlsoSetsTheAccountLocale(t *testing.T) {
	var patches []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var body map[string]any
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			patches = append(patches, body)
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"settings": map[string]any{}}})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotLang("tok", "en"); err != nil {
		t.Fatalf("SetBotLang: %v", err)
	}
	if len(patches) != 1 {
		t.Fatalf("%d PATCHes, want one carrying both", len(patches))
	}
	if patches[0]["locale"] != "en" {
		t.Errorf("locale = %v, want en", patches[0]["locale"])
	}
	bot, _ := patches[0]["settings"].(map[string]any)["bot"].(map[string]any)
	if bot["lang"] != "en" {
		t.Errorf("bot.lang = %v, want en", bot["lang"])
	}
}
