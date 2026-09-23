package daytasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

func authed(w http.ResponseWriter, r *http.Request) (string, bool) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return "", false
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return "", false
	}
	return userID, true
}

func parseDay(w http.ResponseWriter, s string) (time.Time, bool) {
	d, err := time.Parse(dateFmt, s)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_DAY", "day must be YYYY-MM-DD")
		return time.Time{}, false
	}
	return d, true
}

func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTooLate):
		util.RespondError(w, http.StatusConflict, "TOO_LATE", err.Error())
	case errors.Is(err, ErrNotOpen):
		util.RespondError(w, http.StatusConflict, "NOT_OPEN", err.Error())
	case errors.Is(err, ErrNotAnOccurrence):
		util.RespondError(w, http.StatusConflict, "NOT_AN_OCCURRENCE", err.Error())
	case errors.Is(err, ErrTaskNotFound):
		util.RespondError(w, http.StatusNotFound, "TASK_NOT_FOUND", err.Error())
	case errors.Is(err, ErrRangeTooLarge):
		util.RespondError(w, http.StatusBadRequest, "RANGE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrInvalidRange):
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", err.Error())
	default:
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Day tasks failed")
	}
}

// respondDay answers a write with the day as it now is — the bot redraws from
// it without a second request.
func respondDay(w http.ResponseWriter, r *http.Request, userID string, day time.Time) {
	days, err := List(r.Context(), userID, day, day)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, days[0])
}

// ListHandler — GET /api/day-tasks?from=&to=
func ListHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	from, ferr := time.Parse(dateFmt, r.URL.Query().Get("from"))
	to, terr := time.Parse(dateFmt, r.URL.Query().Get("to"))
	if ferr != nil || terr != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "from and to must be YYYY-MM-DD")
		return
	}
	days, err := List(r.Context(), userID, from, to)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, days)
}

// ProposalHandler — GET /api/day-tasks/proposal?day=
func ProposalHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, r.URL.Query().Get("day"))
	if !ok {
		return
	}
	items, err := Propose(r.Context(), userID, day)
	if err != nil {
		respondErr(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, items)
}

type writeRequest struct {
	Day     string   `json:"day"`
	TaskID  string   `json:"task_id"`
	TaskIDs []string `json:"task_ids"`
}

func decode(w http.ResponseWriter, r *http.Request) (writeRequest, bool) {
	var req writeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return req, false
	}
	return req, true
}

// ConfirmHandler — POST /api/day-tasks/confirm {"day","task_ids"}
func ConfirmHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	req, ok := decode(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, req.Day)
	if !ok {
		return
	}
	if err := Confirm(r.Context(), userID, day, req.TaskIDs); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}

// AddHandler — POST /api/day-tasks {"day","task_id"}
func AddHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	req, ok := decode(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, req.Day)
	if !ok {
		return
	}
	if err := Add(r.Context(), userID, day, req.TaskID); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}

// RemoveHandler — DELETE /api/day-tasks/{day}/{task_id}
func RemoveHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := authed(w, r)
	if !ok {
		return
	}
	day, ok := parseDay(w, chi.URLParam(r, "day"))
	if !ok {
		return
	}
	if err := Remove(r.Context(), userID, day, chi.URLParam(r, "task_id"), time.Now()); err != nil {
		respondErr(w, err)
		return
	}
	respondDay(w, r, userID, day)
}
