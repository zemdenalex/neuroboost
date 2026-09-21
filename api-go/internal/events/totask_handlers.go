package events

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// ToTaskHandler handles POST /api/events/{id}/to-task.
func ToTaskHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}
	var req ToTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	res, err := EventToTask(r.Context(), userID, id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrToTaskMode):
			util.RespondError(w, http.StatusBadRequest, "INVALID_MODE", "Mode must be move or link")
		case errors.Is(err, ErrToTaskRepeatRequired):
			// The bot asks «вся серия или только этот раз» on exactly this code.
			util.RespondError(w, http.StatusBadRequest, "REPEAT_CHOICE_REQUIRED", err.Error())
		case errors.Is(err, ErrOccurrenceRequired):
			util.RespondError(w, http.StatusBadRequest, "OCCURRENCE_REQUIRED", err.Error())
		case errors.Is(err, ErrInvalidRrule):
			util.RespondError(w, http.StatusBadRequest, "REPEAT_UNSUPPORTED", "The event's repeat rule cannot be copied")
		case errors.Is(err, pgx.ErrNoRows), errors.Is(err, errNoSuchOccurrence):
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "No such event")
		case errors.Is(err, calendars.ErrCalendarNotFound):
			util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Calendar not found")
		default:
			util.RespondError(w, http.StatusInternalServerError, "TO_TASK_ERROR", "Failed to turn the event into a task")
		}
		return
	}
	status := http.StatusCreated
	if res.DryRun {
		status = http.StatusOK
	}
	util.RespondJSON(w, status, res)
}
