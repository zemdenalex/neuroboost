// Package inbox stores what Denis sends the inbox bot — notes, ideas, links,
// feedback, reflections — to be gone through later in a session on the PC.
// Spec: docs/team/architecture/V003-20261010-arc-inbox-bot.md.
//
// Two clients and one secret: the bot on nl-2 writes, scripts/inbox.py on the
// PC reads and moves statuses. Both send X-Inbox-Token. There is no user
// account behind it — the bot is «just for me», and the owner is a Telegram id.
package inbox

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

const (
	// JSONBodyLimit: the bot joins posts forwarded together from one channel
	// into one note, and a full post is ~8 KB of Cyrillic — 64 KB answered 413
	// to a batch of eight, and resending hit the same wall. Still under the
	// host nginx's 1 MB.
	JSONBodyLimit int64 = 900 << 10
	// MediaBodyLimit: Telegram's getFile stops at 20 MB. ⚠ The host nginx in
	// front of the API has no client_max_body_size, so today it cuts anything
	// over 1 MB first; the bot sends larger files as metadata only.
	MediaBodyLimit int64 = 20 << 20
)

var (
	validTypes  = map[string]bool{"inbox": true, "idea": true, "link": true, "feedback": true, "reflection": true, "note": true}
	validStatus = map[string]bool{"new": true, "discussed": true, "done": true, "dropped": true}
)

type Handler struct{ db *database.DB }

func NewHandler(db *database.DB) *Handler { return &Handler{db: db} }

// TokenMiddleware admits a request carrying the inbox secret. Unset means
// closed: a deployment that forgets INBOX_API_TOKEN must not serve Denis's
// notes to anyone who sends an empty header.
func TokenMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				util.RespondError(w, http.StatusServiceUnavailable, "INBOX_DISABLED", "Inbox is not configured")
				return
			}
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Inbox-Token")), []byte(token)) != 1 {
				util.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid inbox token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Routes mounts the inbox under r. JSON and media get separate body limits:
// one number would either let a 20 MB JSON body in or keep every photo out.
func (h *Handler) Routes(r chi.Router, token string) {
	r.Route("/api/inbox", func(ir chi.Router) {
		ir.Use(TokenMiddleware(token))
		ir.Group(func(jr chi.Router) {
			jr.Use(middleware.BodyLimit(JSONBodyLimit))
			jr.Post("/", h.Create)
			jr.Get("/", h.List)
			jr.Get("/{id}", h.Get)
			jr.Patch("/{id}", h.Update)
			jr.Get("/media/{mid}", h.Media)
		})
		ir.With(middleware.BodyLimit(MediaBodyLimit)).Post("/{id}/media", h.AddMedia)
	})
}

type Source struct {
	Kind     string `json:"kind"` // user | hidden_user | chat | channel
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	URL      string `json:"url,omitempty"`
	Date     int64  `json:"date,omitempty"`
}

type MediaInfo struct {
	ID       int64  `json:"id"`
	Kind     string `json:"kind"`
	Mime     string `json:"mime"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
	Stored   bool   `json:"stored"`
}

type Note struct {
	ID        int64       `json:"id"`
	OwnerTgID int64       `json:"owner_tg_id"`
	Type      string      `json:"type"`
	Projects  []string    `json:"projects"`
	Text      string      `json:"text"`
	Source    *Source     `json:"source"`
	Via       string      `json:"via"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Media     []MediaInfo `json:"media"`
}

type createRequest struct {
	OwnerTgID int64    `json:"owner_tg_id"`
	Type      string   `json:"type"`
	Projects  []string `json:"projects"`
	Text      string   `json:"text"`
	Source    *Source  `json:"source"`
	Via       string   `json:"via"`
	ClientKey string   `json:"client_key"`
}

// Create answers 201 for a new note and 200 with the existing one when the
// client_key was seen before — a redelivered update must not become a twin.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondDecodeError(w, err)
		return
	}
	if req.OwnerTgID == 0 || strings.TrimSpace(req.ClientKey) == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID", "owner_tg_id and client_key are required")
		return
	}
	if req.Type == "" {
		req.Type = "inbox"
	}
	if !validTypes[req.Type] {
		util.RespondError(w, http.StatusBadRequest, "INVALID_TYPE", "unknown type")
		return
	}
	if req.Via == "" {
		req.Via = "chat"
	}
	projects := normProjects(req.Projects)
	var src []byte
	if req.Source != nil {
		src, _ = json.Marshal(req.Source)
	}

	var id int64
	var inserted bool
	err := h.db.Pool.QueryRow(r.Context(), `
		INSERT INTO inbox_note (owner_tg_id, type, projects, text, source, via, client_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (client_key) DO UPDATE SET client_key = EXCLUDED.client_key
		RETURNING id, (xmax = 0)`,
		req.OwnerTgID, req.Type, projects, req.Text, src, req.Via, req.ClientKey).Scan(&id, &inserted)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not save note")
		return
	}
	n, err := h.load(r, id)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read note")
		return
	}
	status := http.StatusOK
	if inserted {
		status = http.StatusCreated
	}
	util.RespondJSON(w, status, n)
}

type updateRequest struct {
	Type       *string   `json:"type"`
	Projects   *[]string `json:"projects"`
	Status     *string   `json:"status"`
	AppendText *string   `json:"append_text"`
}

// Update changes type, projects or status, and can append a line of text —
// a comment that arrived after its forward, not instead of it.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondDecodeError(w, err)
		return
	}
	if req.Type != nil && !validTypes[*req.Type] {
		util.RespondError(w, http.StatusBadRequest, "INVALID_TYPE", "unknown type")
		return
	}
	if req.Status != nil && !validStatus[*req.Status] {
		util.RespondError(w, http.StatusBadRequest, "INVALID_STATUS", "unknown status")
		return
	}
	var projects []string
	if req.Projects != nil {
		projects = normProjects(*req.Projects)
	}
	tag, err := h.db.Pool.Exec(r.Context(), `
		UPDATE inbox_note SET
			type     = COALESCE($2, type),
			projects = CASE WHEN $3::boolean THEN $4::text[] ELSE projects END,
			status   = COALESCE($5, status),
			text     = CASE WHEN $6::text IS NULL OR $6 = '' THEN text
			                WHEN text = '' THEN $6
			                ELSE text || E'\n\n' || $6 END,
			updated_at = NOW()
		WHERE id = $1`,
		id, req.Type, req.Projects != nil, projects, req.Status, req.AppendText)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not update note")
		return
	}
	if tag.RowsAffected() == 0 {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		return
	}
	n, err := h.load(r, id)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read note")
		return
	}
	util.RespondJSON(w, http.StatusOK, n)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	n, err := h.load(r, id)
	if errors.Is(err, pgx.ErrNoRows) {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		return
	}
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read note")
		return
	}
	util.RespondJSON(w, http.StatusOK, n)
}

// List filters by status, type, project (comma-separated each) and after_id;
// oldest first, so a session goes through them in the order they came.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	afterID, _ := strconv.ParseInt(q.Get("after_id"), 10, 64)
	rows, err := h.db.Pool.Query(r.Context(), `
		SELECT id, owner_tg_id, type, projects, text, source, via, status, created_at, updated_at
		FROM inbox_note
		WHERE ($1::text[] IS NULL OR status = ANY($1))
		  AND ($2::text[] IS NULL OR type = ANY($2))
		  AND ($3::text[] IS NULL OR projects && $3)
		  AND id > $4
		ORDER BY id`,
		csv(q.Get("status")), csv(q.Get("type")), csv(q.Get("project")), afterID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not list notes")
		return
	}
	notes := []Note{}
	byID := map[int64]int{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			rows.Close()
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read notes")
			return
		}
		byID[n.ID] = len(notes)
		notes = append(notes, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read notes")
		return
	}
	if len(notes) > 0 {
		ids := make([]int64, 0, len(notes))
		for _, n := range notes {
			ids = append(ids, n.ID)
		}
		media, err := h.mediaFor(r, ids)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read media")
			return
		}
		for noteID, ms := range media {
			notes[byID[noteID]].Media = ms
		}
	}
	util.RespondJSON(w, http.StatusOK, notes)
}

// AddMedia takes the raw file as the body; kind, mime, name and the Telegram
// file_unique_id ride in headers. X-Media-Omitted: 1 with an empty body records
// a file that was too large to carry.
func (h *Handler) AddMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	kind := r.Header.Get("X-Media-Kind")
	uniq := r.Header.Get("X-File-Unique-Id")
	if kind == "" || uniq == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID", "X-Media-Kind and X-File-Unique-Id are required")
		return
	}
	size, _ := strconv.ParseInt(r.Header.Get("X-File-Size"), 10, 64)
	var data []byte
	if r.Header.Get("X-Media-Omitted") != "1" {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			util.RespondDecodeError(w, err)
			return
		}
		data = b
		size = int64(len(b))
	}
	var mid int64
	err := h.db.Pool.QueryRow(r.Context(), `
		INSERT INTO inbox_media (note_id, kind, mime, file_name, size, tg_file_unique_id, data)
		SELECT $1, $2, $3, $4, $5, $6, $7 WHERE EXISTS (SELECT 1 FROM inbox_note WHERE id = $1)
		ON CONFLICT (note_id, tg_file_unique_id) DO UPDATE SET data = COALESCE(EXCLUDED.data, inbox_media.data)
		RETURNING id`,
		id, kind, r.Header.Get("Content-Type"), r.Header.Get("X-File-Name"), size, uniq, data).Scan(&mid)
	if errors.Is(err, pgx.ErrNoRows) {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		return
	}
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not save media")
		return
	}
	util.RespondJSON(w, http.StatusCreated, map[string]int64{"id": mid})
}

// Media returns the stored bytes; 404 also when only metadata was kept.
func (h *Handler) Media(w http.ResponseWriter, r *http.Request) {
	mid, ok := idParam(w, r, "mid")
	if !ok {
		return
	}
	var mime string
	var data []byte
	err := h.db.Pool.QueryRow(r.Context(),
		`SELECT mime, data FROM inbox_media WHERE id = $1 AND data IS NOT NULL`, mid).Scan(&mime, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "media not found")
		return
	}
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "could not read media")
		return
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) load(r *http.Request, id int64) (Note, error) {
	row := h.db.Pool.QueryRow(r.Context(), `
		SELECT id, owner_tg_id, type, projects, text, source, via, status, created_at, updated_at
		FROM inbox_note WHERE id = $1`, id)
	n, err := scanNote(row)
	if err != nil {
		return n, err
	}
	media, err := h.mediaFor(r, []int64{id})
	if err != nil {
		return n, err
	}
	if ms, ok := media[id]; ok {
		n.Media = ms
	}
	return n, nil
}

func (h *Handler) mediaFor(r *http.Request, ids []int64) (map[int64][]MediaInfo, error) {
	rows, err := h.db.Pool.Query(r.Context(), `
		SELECT note_id, id, kind, mime, file_name, size, data IS NOT NULL
		FROM inbox_media WHERE note_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]MediaInfo{}
	for rows.Next() {
		var noteID int64
		var m MediaInfo
		if err := rows.Scan(&noteID, &m.ID, &m.Kind, &m.Mime, &m.FileName, &m.Size, &m.Stored); err != nil {
			return nil, err
		}
		out[noteID] = append(out[noteID], m)
	}
	return out, rows.Err()
}

func scanNote(row pgx.Row) (Note, error) {
	var n Note
	var src []byte
	if err := row.Scan(&n.ID, &n.OwnerTgID, &n.Type, &n.Projects, &n.Text, &src, &n.Via, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return n, err
	}
	if len(src) > 0 {
		var s Source
		if json.Unmarshal(src, &s) == nil {
			n.Source = &s
		}
	}
	if n.Projects == nil {
		n.Projects = []string{}
	}
	n.Media = []MediaInfo{}
	return n, nil
}

// normProjects lowercases, drops '#' and blanks, and keeps first-seen order
// without repeats: «#AvoCar avocar» is one project.
func normProjects(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "#")))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func csv(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func idParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		util.RespondError(w, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return 0, false
	}
	return id, true
}
