package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 🔴 The rule gotcha 21 exists for: a failed read must not become a write.
//
// PATCH /api/auth/me replaces the whole settings blob. If the bot writes after
// a read it could not complete, it merges the new word onto an empty map and
// erases work hours, quiet hours and reminder presets — with a 200 in reply.
func TestSetBotKeywordDoesNotWriteWhenTheReadFailed(t *testing.T) {
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

	c := NewClient(srv.URL)
	if err := c.SetBotKeyword("tok", "спорт", "здоровье"); err == nil {
		t.Error("a failed read reported success")
	}
	if wrote {
		t.Error("the bot wrote settings it had not read — this erases the rest of the blob")
	}
}

// The word is added, and everything the bot knows nothing about survives.
func TestSetBotKeywordKeepsTheRestOfTheBlob(t *testing.T) {
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
				"work_start":        "09:00",
				"quiet_hours_start": "22:00",
				"bot":               map[string]any{"digest": true, "keywords": map[string]any{"дача": "отдых"}},
			}},
		})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotKeyword("tok", "Спорт", "Здоровье"); err != nil {
		t.Fatalf("SetBotKeyword: %v", err)
	}

	settings, _ := sent["settings"].(map[string]any)
	if settings == nil {
		t.Fatalf("no settings were sent: %+v", sent)
	}
	for _, k := range []string{"work_start", "quiet_hours_start"} {
		if _, ok := settings[k]; !ok {
			t.Errorf("%s was dropped", k)
		}
	}
	bot, _ := settings["bot"].(map[string]any)
	if bot == nil {
		t.Fatalf("the bot section vanished: %+v", settings)
	}
	if bot["digest"] != true {
		t.Error("a sibling key inside `bot` was dropped — the section must be rebuilt, not replaced")
	}
	words, _ := bot["keywords"].(map[string]any)
	if words["спорт"] != "здоровье" {
		t.Errorf("the new word is missing or not lowercased: %+v", words)
	}
	if words["дача"] != "отдых" {
		t.Error("an existing word was dropped")
	}
}

func TestSetBotKeywordWithAnEmptyTagRemovesIt(t *testing.T) {
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
				"bot": map[string]any{"keywords": map[string]any{"спорт": "здоровье", "дача": "отдых"}},
			}},
		})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotKeyword("tok", "спорт", ""); err != nil {
		t.Fatalf("SetBotKeyword: %v", err)
	}
	settings, _ := sent["settings"].(map[string]any)
	bot, _ := settings["bot"].(map[string]any)
	words, _ := bot["keywords"].(map[string]any)
	if _, still := words["спорт"]; still {
		t.Error("the word was not removed")
	}
	if words["дача"] != "отдых" {
		t.Error("removing one word removed another")
	}
}

// A blob shaped differently by another client is an empty vocabulary, not a
// crash: the bot has to keep working for whoever wrote it that way.
func TestKeywordsFromToleratesAnyShape(t *testing.T) {
	for _, settings := range []map[string]any{
		nil,
		{},
		{"bot": "не карта"},
		{"bot": map[string]any{"keywords": "не карта"}},
		{"bot": map[string]any{"keywords": map[string]any{"спорт": 42}}},
	} {
		if got := keywordsFrom(settings); len(got) != 0 {
			t.Errorf("%+v gave %+v, want an empty vocabulary", settings, got)
		}
	}
}
