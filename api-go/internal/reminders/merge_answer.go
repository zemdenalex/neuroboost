package reminders

import (
	"context"
	"log/slog"
	"net/http"

	"neuroboost/api-go/internal/accounts"
	"neuroboost/api-go/internal/util"
)

// handleMergeAnswer is the bot's side of «объединить аккаунты?».
//
// Two questions arrive on one notification, in order: which account stays, and
// then — only when both own a personal calendar — what happens to the two of
// them. The second question reuses the same reminder row rather than creating a
// new one, so the bot edits the message it already sent. A second row would be
// a second thing to deliver, to expire and to fail to deliver.
//
// 🔴 The person pressing is the Telegram side; that was established before this
// is called, by matching the reminder against their tg_id. Nothing here trusts
// the request row to say who is acting.
func handleMergeAnswer(w http.ResponseWriter, r *http.Request, requestID, userID, action string) {
	ctx := r.Context()

	switch action {
	case ActionKeepSite, ActionKeepTg:
		// Which side the answer names is read from the row, not from the
		// button: the button says «сайт» or «telegram», and only the row knows
		// which account id that is.
		column := "site_user_id"
		if action == ActionKeepTg {
			column = "tg_user_id"
		}
		tag, err := db.Pool.Exec(ctx,
			`UPDATE account_merge_request
			    SET keep_user_id = `+column+`
			  WHERE id = $1 AND status = 'pending' AND expires_at > NOW()`,
			requestID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record the answer")
			return
		}
		if tag.RowsAffected() == 0 {
			// Pressed twice, or answered on the other side, or simply too late.
			util.RespondJSON(w, http.StatusOK, map[string]any{
				"ok": true, "action": action, "already": true,
			})
			return
		}

		both, err := bothHavePersonalCalendars(ctx, requestID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read the accounts")
			return
		}
		if both {
			// Denis, 17.09: «about the calendar ask too». Asked only when there
			// is something to decide — a question with one possible answer is a
			// step, not a choice.
			util.RespondJSON(w, http.StatusOK, map[string]any{
				"ok": true, "action": action, "ask": "calendar",
			})
			return
		}
		finishMerge(w, ctx, requestID, action)

	case ActionCalMerge, ActionCalBoth:
		choice := string(accounts.CalendarsMerge)
		if action == ActionCalBoth {
			choice = string(accounts.CalendarsKeepBoth)
		}
		tag, err := db.Pool.Exec(ctx,
			`UPDATE account_merge_request SET calendar_choice = $2
			  WHERE id = $1 AND status = 'pending' AND expires_at > NOW()`,
			requestID, choice)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record the answer")
			return
		}
		if tag.RowsAffected() == 0 {
			util.RespondJSON(w, http.StatusOK, map[string]any{
				"ok": true, "action": action, "already": true,
			})
			return
		}
		finishMerge(w, ctx, requestID, action)

	default:
		util.RespondError(w, http.StatusBadRequest, "INVALID_ACTION", "Unknown merge answer")
	}
}

// finishMerge runs the irreversible part and reports plainly.
func finishMerge(w http.ResponseWriter, ctx context.Context, requestID, action string) {
	if err := accounts.Merge(ctx, db.Pool, requestID); err != nil {
		// 🔴 Reported as a failure, never as a success with a note. A merge that
		// half-happened is the one outcome nobody can act on, and Merge itself
		// guarantees it did not: it either completed or rolled back whole.
		if svcLog != nil {
			svcLog.Error("account merge failed",
				slog.String("merge_request_id", requestID), slog.String("error", err.Error()))
		}
		util.RespondError(w, http.StatusInternalServerError, "MERGE_FAILED",
			"Не удалось объединить аккаунты. Ничего не изменилось — попробуй ещё раз.")
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]any{"ok": true, "action": action, "merged": true})
}

// bothHavePersonalCalendars decides whether the second question has an answer
// worth asking for.
func bothHavePersonalCalendars(ctx context.Context, requestID string) (bool, error) {
	var both bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM calendar WHERE owner_id = m.site_user_id AND kind = 'personal')
		   AND EXISTS (SELECT 1 FROM calendar WHERE owner_id = m.tg_user_id   AND kind = 'personal')
		  FROM account_merge_request m WHERE m.id = $1`, requestID).Scan(&both)
	return both, err
}

// rejectMerge is «❌ Это не я».
//
// The request is closed rather than left to expire: an open request blocks the
// next one through the partial unique index, so somebody who pressed «не я» by
// mistake could not try again for ten minutes.
func rejectMerge(ctx context.Context, requestID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE account_merge_request SET status = 'rejected', resolved_at = NOW()
		  WHERE id = $1 AND status = 'pending'`, requestID)
	return err
}
