package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/usersettings"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB sets the database connection for the events package
func InitDB(database *database.DB) {
	db = database
}

// ListHandler returns events for a date range
func ListHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	// Parse + validate the query window (defaults to the current week; rejects
	// unparseable or inverted/zero-length ranges instead of silently returning empty).
	startTime, endTime, errCode, errMsg := parseListRange(
		r.URL.Query().Get("start"), r.URL.Query().Get("end"), time.Now())
	if errCode != "" {
		util.RespondError(w, http.StatusBadRequest, errCode, errMsg)
		return
	}

	events, err := listEvents(r.Context(), userID, startTime, endTime)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch events")
		return
	}

	// Read the caller's calendars ONCE, above the loop. fetchExceptions used to
	// do this itself, on every recurring event, which made half of this
	// handler's queries redundant — and, worse, swallowed the failure as "no
	// exceptions", putting deleted occurrences back on the calendar.
	calIDs, err := calendars.CalendarIDsFor(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch events")
		return
	}

	// Expand recurring events
	var expanded []Event
	for _, ev := range events {
		if ev.Rrule != nil && *ev.Rrule != "" {
			_, err := parseRRule(*ev.Rrule)
			if err != nil {
				// Invalid rrule — include parent event as-is
				expanded = append(expanded, ev)
				continue
			}
			exceptions, err := fetchExceptions(r.Context(), calIDs, ev.ID)
			if err != nil {
				// Refusing is the honest answer. Continuing with no exceptions
				// would silently redraw occurrences the user has deleted.
				util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch events")
				return
			}
			instances := expandRecurrence(ev, startTime, endTime, exceptions)
			expanded = append(expanded, instances...)
		} else {
			expanded = append(expanded, ev)
		}
	}

	util.RespondJSON(w, http.StatusOK, expanded)
}

// CreateHandler creates a new event
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate required fields
	if req.Title == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_TITLE", "Title is required")
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_START", "Invalid start date format")
		return
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_END", "Invalid end date format")
		return
	}

	if err := validateTimeRange(startsAt, endsAt); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
		return
	}

	// Set defaults
	timezone := "Europe/Moscow"
	if req.Timezone != "" {
		timezone = req.Timezone
	}

	isWorkEvent := true
	if req.IsWorkEvent != nil {
		isWorkEvent = *req.IsWorkEvent
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	event, err := createEvent(r.Context(), userID, req, startsAt, endsAt, timezone, isWorkEvent, tags)
	if err != nil {
		// A refused calendar is the caller's mistake, not the server's. Left as
		// a flat 500 it would read as "the app is broken" for what is really
		// "you cannot write there" or "no such calendar".
		switch {
		case errors.Is(err, calendars.ErrCalendarNotFound):
			// 404 rather than 403 on purpose: a permission error would confirm
			// to a stranger that the calendar exists.
			util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Calendar not found")
		case errors.Is(err, calendars.ErrNotCalendarOwner):
			util.RespondError(w, http.StatusForbidden, "CALENDAR_READ_ONLY", "You can read this calendar but not add to it")
		default:
			util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create event")
		}
		return
	}

	util.RespondJSON(w, http.StatusCreated, event)
}

// GetHandler returns a single event
func GetHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	event, err := getEvent(r.Context(), userID, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch event")
		return
	}

	util.RespondJSON(w, http.StatusOK, event)
}

// UpdateHandler updates an event
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Parse the requested bounds once; both the plain and the recurring paths
	// need them, and a malformed value is a 400 either way.
	var newStart, newEnd *time.Time
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_START", "Invalid start date format")
			return
		}
		newStart = &t
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_END", "Invalid end date format")
			return
		}
		newEnd = &t
	}

	targetID, occDate, mode := resolveMutation(eventID, r.URL.Query().Get("scope"))

	// A calendar change is a property of the SERIES, and refusing the
	// combination is better than picking a reading of it.
	//
	// "Move only this Tuesday to the shared calendar" would have to detach the
	// occurrence into its own row, so the thing that moved would no longer be
	// the occurrence the caller named — and the series it came from would stay
	// where it was, which is not what anyone asking for this wants. Answering
	// 400 says so; interpreting it silently would leave the caller looking at a
	// calendar that appears not to have changed.
	if req.CalendarID != nil && mode == mutateOccurrence {
		util.RespondError(w, http.StatusBadRequest, "CALENDAR_SCOPE_SERIES",
			"A calendar change applies to the whole series; retry without scope=occurrence")
		return
	}

	if mode != mutatePlain {
		parent, occStart, occEnd, err := loadOccurrence(r.Context(), userID, targetID, occDate)
		if err != nil {
			if err == pgx.ErrNoRows || err == errNoSuchOccurrence {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update event")
			return
		}

		if mode == mutateOccurrence {
			// Bounds the request left alone stay on the occurrence's own times —
			// validating against the parent would compare a September occurrence
			// with the series' July start.
			startsAt, endsAt := occStart, occEnd
			if newStart != nil {
				startsAt = *newStart
			}
			if newEnd != nil {
				endsAt = *newEnd
			}
			if err := validateTimeRange(startsAt, endsAt); err != nil {
				util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
				return
			}

			replacement, err := detachOccurrence(r.Context(), userID,
				mergeOccurrence(*parent, startsAt, endsAt, req), targetID, occStart)
			if err != nil {
				util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update occurrence")
				return
			}
			util.RespondJSON(w, http.StatusOK, replacement)
			return
		}

		// Series: a time change is a delta on every occurrence, not an absolute
		// rewrite that would re-anchor the series to the edited date.
		if newStart != nil || newEnd != nil {
			startsAt, endsAt := applySeriesDelta(parent.StartsAt, parent.EndsAt, occStart, occEnd, newStart, newEnd)
			if err := validateTimeRange(startsAt, endsAt); err != nil {
				util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
				return
			}
			s, e := startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339)
			req.StartsAt, req.EndsAt = &s, &e
		}
	} else if newStart != nil || newEnd != nil {
		// Validate the effective time window when either bound changes — the update
		// path must not be able to persist a zero-length or inverted event (the other
		// mutation paths all guard this).
		existing, err := getEvent(r.Context(), userID, targetID)
		if err != nil {
			if err == pgx.ErrNoRows {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update event")
			return
		}
		startsAt, endsAt := existing.StartsAt, existing.EndsAt
		if newStart != nil {
			startsAt = *newStart
		}
		if newEnd != nil {
			endsAt = *newEnd
		}
		if err := validateTimeRange(startsAt, endsAt); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
			return
		}
	}

	event, err := updateEvent(r.Context(), userID, targetID, req)
	if err != nil {
		// Same mapping as CreateHandler, and for the same reason: a refused
		// destination calendar is the caller's mistake, and a flat 500 would
		// read as "the app is broken".
		switch {
		case err == pgx.ErrNoRows:
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
		case errors.Is(err, calendars.ErrCalendarNotFound):
			util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Calendar not found")
		case errors.Is(err, calendars.ErrNotCalendarOwner):
			util.RespondError(w, http.StatusForbidden, "CALENDAR_READ_ONLY", "You can read this calendar but not move events into it")
		default:
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update event")
		}
		return
	}

	util.RespondJSON(w, http.StatusOK, event)
}

// DeleteHandler deletes an event
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	// A recurring instance arrives as "<parent uuid>:<date>", which is not a
	// uuid and would fail the cast in deleteEvent. Resolve it to the parent row
	// plus the scope the caller asked for.
	targetID, occurrence, mode := resolveMutation(eventID, r.URL.Query().Get("scope"))

	if mode == mutateOccurrence {
		// Skip this one occurrence; the series itself is untouched.
		if _, err := addException(r.Context(), userID, targetID, occurrence, true, nil); err != nil {
			if err == pgx.ErrNoRows {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete occurrence")
			return
		}
		util.RespondJSON(w, http.StatusOK, map[string]string{"message": "Occurrence deleted"})
		return
	}

	err := deleteEvent(r.Context(), userID, targetID)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete event")
		return
	}

	util.RespondJSON(w, http.StatusOK, map[string]string{"message": "Event deleted"})
}

// MoveHandler handles drag-and-drop move operations
func MoveHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	var req MoveEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_START", "Invalid start date format")
		return
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_END", "Invalid end date format")
		return
	}

	if err := validateTimeRange(startsAt, endsAt); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
		return
	}

	targetID, occDate, mode := resolveMutation(eventID, r.URL.Query().Get("scope"))

	if mode != mutatePlain {
		parent, occStart, occEnd, err := loadOccurrence(r.Context(), userID, targetID, occDate)
		if err != nil {
			if err == pgx.ErrNoRows || err == errNoSuchOccurrence {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "MOVE_ERROR", "Failed to move event")
			return
		}

		if mode == mutateOccurrence {
			replacement, err := detachOccurrence(r.Context(), userID,
				mergeOccurrence(*parent, startsAt, endsAt, UpdateEventRequest{}), targetID, occStart)
			if err != nil {
				util.RespondError(w, http.StatusInternalServerError, "MOVE_ERROR", "Failed to move occurrence")
				return
			}
			util.RespondJSON(w, http.StatusOK, replacement)
			return
		}

		startsAt, endsAt = applySeriesDelta(parent.StartsAt, parent.EndsAt, occStart, occEnd, &startsAt, &endsAt)
		if err := validateTimeRange(startsAt, endsAt); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
			return
		}
	}

	event, err := moveEvent(r.Context(), userID, targetID, startsAt, endsAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "MOVE_ERROR", "Failed to move event")
		return
	}

	util.RespondJSON(w, http.StatusOK, event)
}

// ResizeHandler handles resize operations
func ResizeHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	var req ResizeEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_END", "Invalid end date format")
		return
	}

	targetID, occDate, mode := resolveMutation(eventID, r.URL.Query().Get("scope"))

	if mode != mutatePlain {
		parent, occStart, occEnd, err := loadOccurrence(r.Context(), userID, targetID, occDate)
		if err != nil {
			if err == pgx.ErrNoRows || err == errNoSuchOccurrence {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "RESIZE_ERROR", "Failed to resize event")
			return
		}

		if mode == mutateOccurrence {
			// Validated against the occurrence's own start, not the parent's — the
			// parent's start is the *first* occurrence and says nothing about this one.
			if err := validateTimeRange(occStart, endsAt); err != nil {
				util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
				return
			}
			replacement, err := detachOccurrence(r.Context(), userID,
				mergeOccurrence(*parent, occStart, endsAt, UpdateEventRequest{}), targetID, occStart)
			if err != nil {
				util.RespondError(w, http.StatusInternalServerError, "RESIZE_ERROR", "Failed to resize occurrence")
				return
			}
			util.RespondJSON(w, http.StatusOK, replacement)
			return
		}

		_, seriesEnd := applySeriesDelta(parent.StartsAt, parent.EndsAt, occStart, occEnd, nil, &endsAt)
		if err := validateTimeRange(parent.StartsAt, seriesEnd); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
			return
		}
		endsAt = seriesEnd
	} else {
		// Query the event's current start time to validate the new end time
		calIDs, err := calendars.CalendarIDsFor(r.Context(), userID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to resolve calendars")
			return
		}

		var startsAt time.Time
		err = db.Pool.QueryRow(r.Context(),
			`SELECT starts_at FROM event WHERE id = $1 AND calendar_id = ANY($2)`,
			targetID, calIDs,
		).Scan(&startsAt)
		if err != nil {
			if err == pgx.ErrNoRows {
				util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
				return
			}
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch event")
			return
		}

		if err := validateTimeRange(startsAt, endsAt); err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_RANGE", "End time must be after start time")
			return
		}
	}

	event, err := resizeEvent(r.Context(), userID, targetID, endsAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "RESIZE_ERROR", "Failed to resize event")
		return
	}

	util.RespondJSON(w, http.StatusOK, event)
}

// AddExceptionHandler adds an exception to a recurring event
func AddExceptionHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	var req AddExceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	occurrence, err := time.Parse(time.RFC3339, req.Occurrence)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_OCCURRENCE", "Invalid occurrence date format")
		return
	}

	exception, err := addException(r.Context(), userID, eventID, occurrence, req.Skipped, req.ReplacementEventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Event not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "EXCEPTION_ERROR", "Failed to add exception")
		return
	}

	util.RespondJSON(w, http.StatusCreated, exception)
}

// Database operations

func listEvents(ctx context.Context, userID string, start, end time.Time) ([]Event, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// An empty list is a legitimate "nothing visible", not an error:
	// ANY('{}') returns zero rows.
	rows, err := db.Pool.Query(ctx, `
		SELECT id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		       task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
		FROM event
		WHERE calendar_id = ANY($1)
		  AND (
		    (starts_at < $3 AND ends_at > $2)
		    OR (rrule IS NOT NULL AND rrule != '' AND starts_at < $3)
		  )
		ORDER BY starts_at ASC
	`, calIDs, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var tags []string
		err := rows.Scan(
			&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
			&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &tags,
			&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
		)
		if err != nil {
			return nil, err
		}
		e.Tags = tags
		events = append(events, e)
	}

	if events == nil {
		events = []Event{}
	}

	return events, nil
}

func createEvent(ctx context.Context, userID string, req CreateEventRequest, startsAt, endsAt time.Time, timezone string, isWorkEvent bool, tags []string) (*Event, error) {
	var e Event
	var resultTags []string

	// Absent means "apply my default preset"; an explicitly empty array means
	// "deliberately no reminders". The backend resolves the preset rather than
	// the frontend so an event created from the bot or an import gets
	// reminders too.
	reminderOffsets := []int{}
	if req.ReminderOffsets != nil {
		reminderOffsets = *req.ReminderOffsets
	} else {
		reminderOffsets = usersettings.DefaultEventOffsets(ctx, userID)
	}

	// The calendar the caller asked for, or their personal one when they asked
	// for none. user_id stays on the row either way — it means authorship now,
	// not access.
	requested := ""
	if req.CalendarID != nil {
		requested = *req.CalendarID
	}
	calID, err := calendars.WritableIDFor(ctx, userID, requested)
	if err != nil {
		return nil, err
	}

	err = db.Pool.QueryRow(ctx, `
		INSERT INTO event (user_id, calendar_id, title, description, starts_at, ends_at, all_day, rrule, timezone, location, color, tags, task_id, is_work_event, reminder_offsets)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, userID, calID, req.Title, req.Description, startsAt, endsAt, req.AllDay, req.Rrule, timezone, req.Location, req.Color, tags, req.TaskID, isWorkEvent, reminderOffsets).Scan(
		&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
		&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &resultTags,
		&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
	)

	if err != nil {
		return nil, err
	}

	e.Tags = resultTags
	return &e, nil
}

func getEvent(ctx context.Context, userID, eventID string) (*Event, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	var e Event
	var tags []string

	err = db.Pool.QueryRow(ctx, `
		SELECT id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		       task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
		FROM event
		WHERE id = $1 AND calendar_id = ANY($2)
	`, eventID, calIDs).Scan(
		&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
		&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &tags,
		&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
	)

	if err != nil {
		return nil, err
	}

	e.Tags = tags
	return &e, nil
}

func updateEvent(ctx context.Context, userID, eventID string, req UpdateEventRequest) (*Event, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *req.Title)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			return nil, err
		}
		updates = append(updates, fmt.Sprintf("starts_at = $%d", argNum))
		args = append(args, t)
		argNum++
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			return nil, err
		}
		updates = append(updates, fmt.Sprintf("ends_at = $%d", argNum))
		args = append(args, t)
		argNum++
	}
	if req.AllDay != nil {
		updates = append(updates, fmt.Sprintf("all_day = $%d", argNum))
		args = append(args, *req.AllDay)
		argNum++
	}
	if req.Rrule != nil {
		updates = append(updates, fmt.Sprintf("rrule = $%d", argNum))
		args = append(args, *req.Rrule)
		argNum++
	}
	if req.Timezone != nil {
		updates = append(updates, fmt.Sprintf("timezone = $%d", argNum))
		args = append(args, *req.Timezone)
		argNum++
	}
	if req.Location != nil {
		updates = append(updates, fmt.Sprintf("location = $%d", argNum))
		args = append(args, *req.Location)
		argNum++
	}
	if req.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argNum))
		args = append(args, *req.Color)
		argNum++
	}
	if req.Tags != nil {
		updates = append(updates, fmt.Sprintf("tags = $%d", argNum))
		args = append(args, req.Tags)
		argNum++
	}
	if req.ReminderOffsets != nil {
		updates = append(updates, fmt.Sprintf("reminder_offsets = $%d", argNum))
		args = append(args, *req.ReminderOffsets)
		argNum++
	}
	if req.IsWorkEvent != nil {
		updates = append(updates, fmt.Sprintf("is_work_event = $%d", argNum))
		args = append(args, *req.IsWorkEvent)
		argNum++
	}
	if req.CalendarID != nil {
		// Checked, never trusted: WritableIDFor refuses a calendar the user is
		// not an owner or editor of, and answers "not found" rather than
		// "forbidden" for a stranger so the reply cannot be used to discover
		// that a calendar exists.
		destination, err := calendars.WritableIDFor(ctx, userID, *req.CalendarID)
		if err != nil {
			return nil, err
		}
		updates = append(updates, fmt.Sprintf("calendar_id = $%d", argNum))
		args = append(args, destination)
		argNum++
	}

	if len(updates) == 0 {
		return getEvent(ctx, userID, eventID)
	}

	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, eventID, calIDs)

	query := fmt.Sprintf(`
		UPDATE event SET %s
		WHERE id = $%d AND calendar_id = ANY($%d)
		RETURNING id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'), 
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, strings.Join(updates, ", "), argNum, argNum+1)

	var e Event
	var tags []string

	err = db.Pool.QueryRow(ctx, query, args...).Scan(
		&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
		&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &tags,
		&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
	)

	if err != nil {
		return nil, err
	}

	e.Tags = tags
	return &e, nil
}

func deleteEvent(ctx context.Context, userID, eventID string) error {
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return err
	}

	result, err := db.Pool.Exec(ctx, `
		DELETE FROM event WHERE id = $1 AND calendar_id = ANY($2)
	`, eventID, calIDs)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func moveEvent(ctx context.Context, userID, eventID string, startsAt, endsAt time.Time) (*Event, error) {
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	var e Event
	var tags []string

	err = db.Pool.QueryRow(ctx, `
		UPDATE event SET starts_at = $3, ends_at = $4, updated_at = NOW()
		WHERE id = $1 AND calendar_id = ANY($2)
		RETURNING id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, eventID, calIDs, startsAt, endsAt).Scan(
		&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
		&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &tags,
		&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
	)

	if err != nil {
		return nil, err
	}

	e.Tags = tags
	return &e, nil
}

func resizeEvent(ctx context.Context, userID, eventID string, endsAt time.Time) (*Event, error) {
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	var e Event
	var tags []string

	err = db.Pool.QueryRow(ctx, `
		UPDATE event SET ends_at = $3, updated_at = NOW()
		WHERE id = $1 AND calendar_id = ANY($2)
		RETURNING id, calendar_id::text, user_id, title, description, starts_at, ends_at, all_day, rrule,
		          COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		          task_id, COALESCE(is_work_event, true), created_at, updated_at,
		          COALESCE(reminder_offsets, '{}')
	`, eventID, calIDs, endsAt).Scan(
		&e.ID, &e.CalendarID, &e.UserID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt,
		&e.AllDay, &e.Rrule, &e.Timezone, &e.Location, &e.Color, &tags,
		&e.TaskID, &e.IsWorkEvent, &e.CreatedAt, &e.UpdatedAt, &e.ReminderOffsets,
	)

	if err != nil {
		return nil, err
	}

	e.Tags = tags
	return &e, nil
}

func addException(ctx context.Context, userID, eventID string, occurrence time.Time, skipped bool, replacementID *string) (*EventException, error) {
	// Writing an exception needs the same access as reading the event it
	// targets — getEvent already resolves that against calendar membership
	// and returns pgx.ErrNoRows for "doesn't exist" and "not yours to see"
	// alike, which callers already treat as 404.
	if _, err := getEvent(ctx, userID, eventID); err != nil {
		return nil, err
	}

	var ex EventException

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO event_exception (event_id, user_id, occurrence, skipped, replacement_event_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, event_id, user_id, occurrence, skipped, replacement_event_id, created_at
	`, eventID, userID, occurrence, skipped, replacementID).Scan(
		&ex.ID, &ex.EventID, &ex.UserID, &ex.Occurrence, &ex.Skipped, &ex.ReplacementEventID, &ex.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &ex, nil
}
