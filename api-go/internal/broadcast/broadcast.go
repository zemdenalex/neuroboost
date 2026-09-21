// Package broadcast serves the bot's «что нового» broadcast (spec 21.09 §D).
//
// 🔴 Until 11.4 a broadcast was a one-off script on the bot host that read the
// token and reported only a tally — «200: 1 · 403: 1 · сеть: 1» — so nobody
// could tell whom to retry. Now the result is kept PER RECIPIENT, in that
// recipient's own settings (settings.bot.broadcasts.<version>), and the
// recipient list skips whoever already got it (200) or blocked the bot (403).
// Kept in settings rather than a table because 11.4 ships without migrations.
//
// Both routes sit behind the service token (/api/svc): the bot reads across
// users here, exactly like the notifier does.
package broadcast

import (
	"encoding/json"
	"net/http"
	"strconv"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB wires the pool.
func InitDB(d *database.DB) { db = d }

// Recipient is who gets a broadcast, and in which language.
type Recipient struct {
	TgID int64  `json:"tg_id"`
	Lang string `json:"lang"`
}

const maxVersionLen = 32

// RecipientsHandler handles GET /api/svc/broadcast/recipients?version=&active_days=.
//
// Denis 21.09: «только тем, кто пользовался ботом в последние 10 дней».
// tg_auth_date is refreshed on every Telegram login of the bot.
func RecipientsHandler(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	days, err := strconv.Atoi(r.URL.Query().Get("active_days"))
	if version == "" || len(version) > maxVersionLen || err != nil || days < 1 || days > 90 {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "version and active_days (1–90) are required")
		return
	}
	rows, err := db.Pool.Query(r.Context(), `
		SELECT tg_id, COALESCE(settings->'bot'->>'lang', '')
		  FROM "user"
		 WHERE tg_id IS NOT NULL
		   AND tg_auth_date > now() - make_interval(days => $2)
		   AND COALESCE(settings->'bot'->>'updates', 'on') <> 'off'
		   AND COALESCE(settings->'bot'->'broadcasts'->>$1, '') NOT IN ('200', '403')
		 ORDER BY tg_id`, version, days)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list recipients")
		return
	}
	defer rows.Close()
	out := []Recipient{}
	for rows.Next() {
		var rc Recipient
		if err := rows.Scan(&rc.TgID, &rc.Lang); err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list recipients")
			return
		}
		out = append(out, rc)
	}
	if rows.Err() != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list recipients")
		return
	}
	util.RespondJSON(w, http.StatusOK, out)
}

type markRequest struct {
	TgID    int64  `json:"tg_id"`
	Version string `json:"version"`
	Code    int    `json:"code"`
}

// MarkHandler handles POST /api/svc/broadcast/mark — one recipient's result.
//
// jsonb_set level by level: it creates only the LAST key of a path, so bot and
// broadcasts are ensured first. The rest of the blob is untouched — PATCH
// /api/auth/me replaces settings wholesale (gotcha 21); this must not.
func MarkHandler(w http.ResponseWriter, r *http.Request) {
	var req markRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgID == 0 ||
		req.Version == "" || len(req.Version) > maxVersionLen {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "tg_id, version and code are required")
		return
	}
	tag, err := db.Pool.Exec(r.Context(), `
		UPDATE "user" SET settings =
		  jsonb_set(
		    jsonb_set(
		      jsonb_set(COALESCE(settings, '{}'), '{bot}', COALESCE(settings->'bot', '{}'), true),
		      '{bot,broadcasts}', COALESCE(settings->'bot'->'broadcasts', '{}'), true),
		    ARRAY['bot', 'broadcasts', $2::text], to_jsonb($3::text), true)
		 WHERE tg_id = $1`, req.TgID, req.Version, strconv.Itoa(req.Code))
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record the result")
		return
	}
	if tag.RowsAffected() == 0 {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "No such recipient")
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]string{"message": "recorded"})
}
