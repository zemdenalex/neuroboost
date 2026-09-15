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
	if err := c.SetBotKeyword("tok", "спорт", "tag", "здоровье"); err == nil {
		t.Error("a failed read reported success")
	}
	if wrote {
		t.Error("the bot wrote settings it had not read — this erases the rest of the blob")
	}
}

// The word is added with its characteristic, and everything the bot knows
// nothing about survives.
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
				"bot": map[string]any{"digest": true, "keywords": map[string]any{
					"дача": map[string]any{"field": "tag", "value": "отдых"},
				}},
			}},
		})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotKeyword("tok", "Созвон", "Calendar", " Работа "); err != nil {
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
	added, _ := words["созвон"].(map[string]any)
	if added == nil {
		t.Fatalf("the new word is missing or not lowercased: %+v", words)
	}
	if added["field"] != "calendar" || added["value"] != "Работа" {
		t.Errorf("stored %+v, want field calendar and value «Работа» — the value keeps its case, the field does not", added)
	}
	if _, ok := words["дача"]; !ok {
		t.Error("an existing word was dropped")
	}
}

func TestSetBotKeywordWithAnEmptyFieldRemovesIt(t *testing.T) {
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
				"bot": map[string]any{"keywords": map[string]any{
					"спорт": map[string]any{"field": "tag", "value": "здоровье"},
					"дача":  map[string]any{"field": "tag", "value": "отдых"},
				}},
			}},
		})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetBotKeyword("tok", "спорт", "", ""); err != nil {
		t.Fatalf("SetBotKeyword: %v", err)
	}
	settings, _ := sent["settings"].(map[string]any)
	bot, _ := settings["bot"].(map[string]any)
	words, _ := bot["keywords"].(map[string]any)
	if _, still := words["спорт"]; still {
		t.Error("the word was not removed")
	}
	if _, ok := words["дача"]; !ok {
		t.Error("removing one word removed another")
	}
}

// 🔴 The first version of this feature stored every word as a bare string,
// because every word was a tag. Those blobs still exist. Dropping them would
// empty the vocabulary of whoever used that version — silently, which is the
// part that makes it unacceptable.
func TestLegacyStringKeywordsAreStillRead(t *testing.T) {
	got := keywordsFrom(map[string]any{
		"bot": map[string]any{"keywords": map[string]any{
			"спорт":  "Здоровье",
			"созвон": map[string]any{"field": "calendar", "value": "Работа"},
		}},
	})
	if got["спорт"] != (Keyword{Field: "tag", Value: "здоровье"}) {
		t.Errorf("legacy word read as %+v, want a tag «здоровье»", got["спорт"])
	}
	if got["созвон"] != (Keyword{Field: "calendar", Value: "Работа"}) {
		t.Errorf("new-shape word read as %+v", got["созвон"])
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
		{"bot": map[string]any{"keywords": map[string]any{"спорт": map[string]any{"value": "нет поля"}}}},
	} {
		if got := keywordsFrom(settings); len(got) != 0 {
			t.Errorf("%+v gave %+v, want an empty vocabulary", settings, got)
		}
	}
}
