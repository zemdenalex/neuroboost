package tasks

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

// ConvertHandler handles POST /api/tasks/{id}/convert.
func ConvertHandler(w http.ResponseWriter, r *http.Request) {
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

	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	event, err := Convert(r.Context(), userID, taskID, req)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "No such task")
		case errors.Is(err, ErrUnknownMode):
			util.RespondError(w, http.StatusBadRequest, "INVALID_MODE", "Mode must be move or link")
		case errors.Is(err, ErrNeedsTime):
			// 🔴 Its own code, because this is the one the BOT has to act on:
			// a task with no time cannot become an event until the user is
			// asked for one. A generic 400 would leave the bot guessing.
			util.RespondError(w, http.StatusBadRequest, "NEEDS_TIME", err.Error())
		case errors.Is(err, ErrRepeatChoiceRequired):
			// The bot asks «вся серия или только этот раз» on exactly this code.
			util.RespondError(w, http.StatusBadRequest, "REPEAT_CHOICE_REQUIRED", err.Error())
		case errors.Is(err, ErrNotAnOccurrence):
			util.RespondError(w, http.StatusBadRequest, "NOT_AN_OCCURRENCE", "That day is not in the series")
		case errors.Is(err, ErrInvalidRepeat):
			util.RespondError(w, http.StatusBadRequest, "REPEAT_UNSUPPORTED", "The task's repeat rule cannot be copied")
		case errors.Is(err, calendars.ErrCalendarNotFound):
			util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Calendar not found")
		default:
			util.RespondError(w, http.StatusInternalServerError, "CONVERT_ERROR", "Failed to convert the task")
		}
		return
	}

	util.RespondJSON(w, http.StatusCreated, event)
}
