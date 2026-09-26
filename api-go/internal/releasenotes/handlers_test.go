package releasenotes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zemdenalex/neuroboost-bot/release"
)

// The web shows what the bot's «Что нового» says: the first note is the bot's
// latest, with its text in both languages.
func TestTheWebGetsTheBotsNotesNewestFirst(t *testing.T) {
	r := chi.NewRouter()
	Register(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/release-notes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Data []Note `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	latest := release.Latest()
	if len(body.Data) == 0 || body.Data[0].Version != latest.Version || body.Data[0].RU != latest.RU || body.Data[0].EN != latest.EN {
		t.Fatalf("first note %+v, want the bot's latest %s", body.Data, latest.Version)
	}
}
