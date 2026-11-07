package remindrule

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// The rule has its own door — GET/PUT/DELETE /api/{tasks|events}/{id}/remind-rule
// — instead of riding in the task and event bodies. Two reasons:
//
//   - the web editor saves whole bodies and has never heard of the field; a
//     field it does not send cannot be wiped by it (gotcha 21 is the same
//     class: one writer replacing what another wrote);
//   - a dozen SELECTs return a task or an event, and none of them needs to
//     change for a field that only the bot and the scanner read.

var db *database.DB

func InitDB(d *database.DB) { db = d }

// kind maps the URL to its table. Never interpolated from user input: the two
// literals are the only values that reach the SQL.
type kind struct{ table, noun string }

var (
	TaskKind  = kind{"task", "Task"}
	EventKind = kind{"event", "Event"}
)

var (
	errNotFound   = errors.New("not found")
	errNoDeadline = errors.New("no deadline")
	errRepeating  = errors.New("repeating")
)

// Get answers the rule, or null when there is none.
func Get(k kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := ids(w, r)
		if !ok {
			return
		}
		calIDs, err := calendars.CalendarIDsFor(r.Context(), userID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "READ_ERROR", "Failed to read the rule")
			return
		}
		var raw []byte
		err = db.Pool.QueryRow(r.Context(),
			`SELECT remind_rule FROM `+k.table+` WHERE id = $1 AND calendar_id = ANY($2)`, id, calIDs).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", k.noun+" not found")
			return
		}
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "READ_ERROR", "Failed to read the rule")
			return
		}
		if raw == nil {
			util.RespondJSON(w, http.StatusOK, nil)
			return
		}
		var rule Rule
		if json.Unmarshal(raw, &rule) != nil {
			util.RespondJSON(w, http.StatusOK, nil)
			return
		}
		util.RespondJSON(w, http.StatusOK, rule)
	}
}

// Put sets the rule. Only a one-off with a deadline may carry one: a task
// without a due date has nothing to run up to, and a repeating item has no
// single deadline (v1, docs/tasks-napominaniya-do-sroka.md «Не входит»).
func Put(k kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := ids(w, r)
		if !ok {
			return
		}
		var rule Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if err := rule.Validate(); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_RULE", err.Error())
			return
		}
		switch err := write(r.Context(), k, userID, id, &rule); {
		case err == nil:
			util.RespondJSON(w, http.StatusOK, rule)
		case errors.Is(err, errNotFound):
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", k.noun+" not found")
		case errors.Is(err, errNoDeadline):
			util.RespondError(w, http.StatusBadRequest, "NO_DEADLINE", "Set a due date first: the rule runs up to it")
		case errors.Is(err, errRepeating):
			util.RespondError(w, http.StatusBadRequest, "REPEATING", "A repeating item cannot have a rule yet")
		default:
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to save the rule")
		}
	}
}

// Delete removes the rule. Removing one that is not there is not an error.
func Delete(k kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := ids(w, r)
		if !ok {
			return
		}
		switch err := write(r.Context(), k, userID, id, nil); {
		case err == nil:
			util.RespondJSON(w, http.StatusOK, nil)
		case errors.Is(err, errNotFound):
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", k.noun+" not found")
		default:
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to remove the rule")
		}
	}
}

func write(ctx context.Context, k kind, userID, id string, rule *Rule) error {
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return err
	}
	// The deadline and repeat checks read the row the write is about, inside
	// the same scope, so a viewer cannot learn anything a 404 would hide.
	var hasDeadline, repeats bool
	q := `SELECT due_date IS NOT NULL, COALESCE(rrule, '') <> '' FROM task WHERE id = $1 AND calendar_id = ANY($2)`
	if k == EventKind {
		q = `SELECT true, COALESCE(rrule, '') <> '' FROM event WHERE id = $1 AND calendar_id = ANY($2)`
	}
	err = db.Pool.QueryRow(ctx, q, id, calIDs).Scan(&hasDeadline, &repeats)
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	var value any // nil → SQL NULL
	if rule != nil {
		if !hasDeadline {
			return errNoDeadline
		}
		if repeats {
			return errRepeating
		}
		raw, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		value = raw
	}
	_, err = db.Pool.Exec(ctx,
		`UPDATE `+k.table+` SET remind_rule = $3, updated_at = NOW() WHERE id = $1 AND calendar_id = ANY($2)`,
		id, calIDs, value)
	return err
}

func ids(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return "", "", false
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return "", "", false
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "ID is required")
		return "", "", false
	}
	return userID, id, true
}
