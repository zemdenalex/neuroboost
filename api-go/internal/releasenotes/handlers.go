// Package releasenotes serves the bot's «Что нового» to the web (gap list
// row 19). One text for both: the notes live in bot/release, next to the code
// they describe, so the web cannot say something the bot does not.
package releasenotes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zemdenalex/neuroboost-bot/release"

	"neuroboost/api-go/internal/util"
)

// Note is one release in both languages; the web picks its own.
type Note struct {
	Version  string `json:"version"`
	Released string `json:"released"`
	RU       string `json:"ru"`
	EN       string `json:"en"`
}

// Register adds GET /api/release-notes.
func Register(r chi.Router) {
	r.Get("/api/release-notes", Handler)
}

// Handler answers the notes, newest first. Always an array (gotcha 6).
func Handler(w http.ResponseWriter, _ *http.Request) {
	all := release.All()
	out := make([]Note, 0, len(all))
	for _, n := range all {
		out = append(out, Note{Version: n.Version, Released: n.Released, RU: n.RU, EN: n.EN})
	}
	util.RespondJSON(w, http.StatusOK, out)
}
