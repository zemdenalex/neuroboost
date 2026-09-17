package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The onboarding flag lands in the same `bot` section as the language and the
// keywords, on a blob PATCH replaces wholesale (gotcha 21).
func TestSetOnboardedKeepsEverythingElse(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &sent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"settings": map[string]any{
			"work_start": "09:00",
			"bot":        map[string]any{"lang": "en", "keywords": map[string]any{"x": "y"}},
		}}})
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetOnboarded("tok"); err != nil {
		t.Fatalf("SetOnboarded: %v", err)
	}
	settings, _ := sent["settings"].(map[string]any)
	bot, _ := settings["bot"].(map[string]any)
	if settings["work_start"] != "09:00" || bot["lang"] != "en" || bot["keywords"] == nil {
		t.Errorf("onboarding erased something: %+v", settings)
	}
	if bot["onboarded"] != true {
		t.Errorf("onboarded = %v, want true", bot["onboarded"])
	}
}

func TestSetOnboardedDoesNotWriteWhenTheReadFailed(t *testing.T) {
	wrote := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			wrote = true
			return
		}
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetOnboarded("tok"); err == nil {
		t.Error("a failed read reported success")
	}
	if wrote {
		t.Error("wrote settings it had not read")
	}
}

// 🔴 The timezone is a column. Sending `settings` alongside it — even an empty
// one — would REPLACE the blob, so the request must carry the timezone and
// nothing else.
func TestSetTimezoneSendsOnlyTheTimezone(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetTimezone("tok", "Asia/Yekaterinburg"); err != nil {
		t.Fatalf("SetTimezone: %v", err)
	}
	if len(sent) != 1 || sent["timezone"] != "Asia/Yekaterinburg" {
		t.Errorf("body = %+v, want only the timezone", sent)
	}
}
