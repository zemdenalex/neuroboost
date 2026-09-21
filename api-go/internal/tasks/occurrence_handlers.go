package tasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// occurrenceRequest is either "this day went like so" or "close the next N days".
//
// One endpoint for both because they are the same act from the user's side —
// answering for a day — and splitting them would make the bot hold two routes
// that differ by a field.
type occurrenceRequest struct {
	// Date is the calendar day, "2026-09-19". Empty means "today where I am",
	// which is what a button press means.
	Date  string `json:"date,omitempty"`
	State string `json:"state,omitempty"`

	// PostponeDays closes this many days from Date onward, leaving the rhythm
	// alone. Mutually exclusive with State.
	PostponeDays int `json:"postpone_days,omitempty"`
}

// MarkOccurrenceHandler handles POST /api/tasks/{id}/occurrences.
func MarkOccurrenceHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	var req occurrenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.State != "" && req.PostponeDays > 0 {
		// Refused rather than picked for them: the two mean opposite things, and
		// guessing which was meant is how a «done» becomes a «skipped».
		util.RespondError(w, http.StatusBadRequest, "AMBIGUOUS_REQUEST",
			"Send either state or postpone_days, not both")
		return
	}

	// The day is resolved where the USER is, not where the server is.
	rule, anchor, tz, err := repeatOf(r.Context(), userID, taskID)
	if err != nil {
		respondOccurrenceError(w, err)
		return
	}

	day := LocalDay(time.Now(), tz)
	namedADate := req.Date != ""
	if !namedADate && req.PostponeDays == 0 {
		// A press, not a date: resolve it to the day of the series the user is
		// actually looking at. See PressedDay for why today is not always it.
		if resolved, ok := PressedDay(rule, anchor, day); ok {
			day = resolved
		}
	}
	if req.Date != "" {
		parsed, perr := time.Parse("2006-01-02", req.Date)
		if perr != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_DATE", "Date must be YYYY-MM-DD")
			return
		}
		loc, lerr := time.LoadLocation(tz)
		if lerr != nil {
			loc = time.UTC
		}
		day = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, loc)
	}

	if req.PostponeDays > 0 {
		closed, perr := PostponeSeries(r.Context(), userID, taskID, day, req.PostponeDays)
		if perr != nil {
			respondOccurrenceError(w, perr)
			return
		}
		util.RespondJSON(w, http.StatusOK, map[string]any{
			"postponed_days": req.PostponeDays,
			"closed":         closed,
			"from":           day.Format("2006-01-02"),
		})
		return
	}

	state := req.State
	if state == "" {
		state = StateDone
	}
	if err := MarkOccurrence(r.Context(), userID, taskID, day, state); err != nil {
		respondOccurrenceError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]any{
		"occurrence": day.Format("2006-01-02"),
		"state":      state,
	})
}

// respondOccurrenceError maps each refusal to the status that describes it.
//
// 🔴 Not a flat 500. «This task does not repeat» and «that day is not in the
// series» are both the caller's mistake and both fixable by the caller; a 500
// reads as "the app is broken" and gets retried instead of corrected.
func respondOccurrenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// Not-found and not-yours share an answer on purpose: distinguishing
		// them tells a stranger the task exists.
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "No such task")
	case errors.Is(err, ErrNotRecurring):
		util.RespondError(w, http.StatusBadRequest, "NOT_RECURRING", "This task does not repeat")
	case errors.Is(err, ErrNotAnOccurrence):
		util.RespondError(w, http.StatusBadRequest, "NOT_AN_OCCURRENCE", "That day is not in the series")
	default:
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record the day")
	}
}
