package reminders

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/util"
)

// Actions a notification's buttons can trigger.
const (
	ActionAck    = "ack"
	ActionSnooze = "snooze"
	ActionDone   = "done"
)

const (
	// DefaultSnoozeMinutes is what the "later" button means when the caller
	// sends no figure. The spec's button reads "Отложить 10 мин".
	DefaultSnoozeMinutes = 10
	// MaxSnoozeMinutes caps a snooze at a day. Beyond that the user wants a
	// different reminder, not a postponed one, and an unbounded value lets one
	// bad callback park a row in the table forever.
	MaxSnoozeMinutes = 24 * 60
)

// ErrInvalidAction is returned for a request no action can be derived from.
var ErrInvalidAction = errors.New("invalid action request")

// ActionRequest is what the notifier forwards when a user presses a button.
//
// The user is identified by tg_id rather than by a JWT: the caller is the bot,
// authenticated by the service token, and the person who pressed the button
// only ever exists to it as a Telegram id.
type ActionRequest struct {
	TgID       int64  `json:"tg_id"`
	ReminderID string `json:"reminder_id"`
	Action     string `json:"action"`
	Minutes    int    `json:"minutes,omitempty"`
}

// ValidateAction checks a request and fills in defaults.
//
// Kept separate from the handler so the rules are testable without a database
// — the same reason resizeCoords exists on the frontend. Out-of-range snoozes
// are clamped rather than rejected: a button that silently does nothing is
// worse than one that does something slightly different from what was asked.
func ValidateAction(req ActionRequest) (ActionRequest, error) {
	if req.TgID == 0 || req.ReminderID == "" {
		return req, ErrInvalidAction
	}
	switch req.Action {
	case ActionAck, ActionDone:
		req.Minutes = 0
	case ActionSnooze:
		if req.Minutes <= 0 {
			req.Minutes = DefaultSnoozeMinutes
		}
		if req.Minutes > MaxSnoozeMinutes {
			req.Minutes = MaxSnoozeMinutes
		}
	default:
		return req, ErrInvalidAction
	}
	return req, nil
}

// ActionHandler performs a button press from a notification.
func ActionHandler(w http.ResponseWriter, r *http.Request) {
	var raw ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	req, err := ValidateAction(raw)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_ACTION", "Unknown or incomplete action")
		return
	}

	ctx := r.Context()

	// Resolve the reminder AND the user in one statement. Matching on tg_id is
	// what stops one user's callback acting on another's reminder: the service
	// token authenticates the bot, not the person who pressed the button.
	var userID, sourceKind string
	var eventID, taskID *string
	var occurrenceStart *string
	var message string
	err = db.Pool.QueryRow(ctx, `
		SELECT r.user_id::text, r.source_kind,
		       r.event_id::text, r.task_id::text,
		       r.occurrence_start::text, COALESCE(r.message, '')
		FROM reminder r
		JOIN "user" u ON u.id = r.user_id
		WHERE r.id = $1 AND u.tg_id = $2`,
		req.ReminderID, req.TgID).
		Scan(&userID, &sourceKind, &eventID, &taskID, &occurrenceStart, &message)
	if err != nil {
		// Not found and not-yours are deliberately the same answer: telling the
		// caller which one it was would confirm that a reminder id exists.
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "No such reminder for this user")
		return
	}

	switch req.Action {
	case ActionAck:
		// Nothing to change — the row is already SENT. The button exists so the
		// message can be dismissed, and answering ok lets the bot edit it.
		util.RespondJSON(w, http.StatusOK, map[string]any{"ok": true, "action": ActionAck})

	case ActionSnooze:
		// minutes_before = -1 is the sentinel for "one-off, no offset"; the real
		// time lives in remind_at. It must NOT be NULL: the dedup index is
		// NULLS NOT DISTINCT, and a NULL in the key would make every snooze row
		// collide with the digest's.
		//
		// Re-snoozing pushes the time rather than failing, which is why this is
		// DO UPDATE and not DO NOTHING. The conflict target repeats the index's
		// COALESCE expression exactly — a plain column list does not match an
		// expression index.
		if _, err := db.Pool.Exec(ctx, `
			INSERT INTO reminder (user_id, source_kind, event_id, task_id, occurrence_start,
			                      minutes_before, remind_at, status, channel, message)
			VALUES ($1, $2, $3, $4, $5, -1, NOW() + make_interval(mins => $6), 'PENDING', 'TELEGRAM', $7)
			ON CONFLICT (user_id, source_kind, COALESCE(event_id, task_id), occurrence_start, minutes_before)
			DO UPDATE SET remind_at = EXCLUDED.remind_at, status = 'PENDING', sent_at = NULL`,
			userID, sourceKind, eventID, taskID, occurrenceStart, req.Minutes, message); err != nil {
			if svcLog != nil {
				svcLog.Error("snooze failed",
					slog.String("reminder_id", req.ReminderID), slog.String("error", err.Error()))
			}
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to snooze")
			return
		}
		util.RespondJSON(w, http.StatusOK, map[string]any{
			"ok": true, "action": ActionSnooze, "minutes": req.Minutes,
		})

	case ActionDone:
		// Only a task can be completed. An event reminder's "done" is an ack.
		if taskID == nil {
			util.RespondError(w, http.StatusBadRequest, "NOT_A_TASK", "Only a task can be completed")
			return
		}
		// A scoping clause in the WHERE, not just the id: the reminder row was
		// already matched against tg_id above, and this keeps that guarantee
		// inside the write. Access to a task comes from calendar membership,
		// so the clause is the membership list of that same verified user —
		// the check is not weakened, only re-expressed.
		calIDs, err := calendars.CalendarIDsFor(ctx, userID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to complete task")
			return
		}
		tag, err := db.Pool.Exec(ctx,
			`UPDATE task SET status = 'DONE', updated_at = NOW() WHERE id = $1 AND calendar_id = ANY($2)`,
			*taskID, calIDs)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to complete task")
			return
		}
		if tag.RowsAffected() == 0 {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task no longer exists")
			return
		}
		util.RespondJSON(w, http.StatusOK, map[string]any{"ok": true, "action": ActionDone})
	}
}
