package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/recurrence"
)

// Writing a task's repeat — the half that did not exist until 20.09.
//
// Everything that READS a repeating task was built on 18.09: the occurrence
// table, «сделал на сегодня», postponing the series, the nagging. Nothing wrote
// one. The request types had no rrule field, so no client could make a task
// repeat, and the tests never noticed because they seeded the column directly.

// ErrInvalidRepeat is a rule or a nag interval the rest of the system could not
// act on. Refused at the door, because the alternative is a row the reminder
// scanner fails to parse once a minute for ever.
var ErrInvalidRepeat = errors.New("invalid repeat")

const (
	// A nag closer together than five minutes is an alarm, not a reminder; one
	// further apart than a day is a second reminder. The bot offers 10 and 60.
	minNagMinutes = 5
	maxNagMinutes = 24 * 60
)

// validateRepeat checks both fields the way the reader will later read them.
//
// The rule goes through recurrence.Parse — the SAME function repeatOf() uses —
// so «accepted here» and «understood there» cannot drift apart.
func validateRepeat(rrule *string, nagMinutes *int) error {
	if rrule != nil && *rrule != "" {
		if _, err := recurrence.Parse(*rrule); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidRepeat, err)
		}
	}
	if nagMinutes != nil && *nagMinutes != 0 {
		if *nagMinutes < minNagMinutes || *nagMinutes > maxNagMinutes {
			return fmt.Errorf("%w: nag_minutes must be 0 or %d–%d", ErrInvalidRepeat, minNagMinutes, maxNagMinutes)
		}
	}
	return nil
}

// anchorFor picks the day a series counts from.
//
// The due date when there is one: «каждый понедельник» typed on a Thursday with
// a Monday deadline must land on Mondays. Otherwise today — and today in the
// AUTHOR's zone is right here, unlike in the readers: the anchor is a fact about
// when the series was set up, not about who is looking at it.
func anchorFor(ctx context.Context, userID string, dueDate *time.Time) time.Time {
	tz := "Europe/Moscow"
	_ = db.Pool.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow') FROM "user" WHERE id = $1`, userID).Scan(&tz)
	at := time.Now()
	if dueDate != nil {
		at = *dueDate
	}
	return LocalDay(at, tz)
}

// applyRepeatOnCreate stores the rule on a task that was just inserted.
//
// A second statement rather than three more INSERT columns: createTask's INSERT
// and its RETURNING list are shared with the batch path, and widening both for
// fields most tasks never set is how the two would drift.
func applyRepeatOnCreate(ctx context.Context, userID string, t *Task, req CreateTaskRequest, dueDate *time.Time) error {
	hasRule := req.Rrule != nil && *req.Rrule != ""
	hasNag := req.NagMinutes != nil && *req.NagMinutes > 0
	if !hasRule && !hasNag {
		return nil
	}
	var anchor *time.Time
	if hasRule {
		a := anchorFor(ctx, userID, dueDate)
		anchor = &a
		t.Rrule = req.Rrule
	}
	var nag *int
	if hasNag {
		nag = req.NagMinutes
		t.NagMinutes = req.NagMinutes
	}
	var rule *string
	if hasRule {
		rule = req.Rrule
	}
	// Scoped by write access even though createTask authorised this very row a
	// statement ago. «It was just checked» is true today and is exactly the
	// sentence that stops being true when somebody calls this from elsewhere;
	// calendars.TestWritesScopeByWritableCalendars refuses to take it on trust.
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return err
	}
	_, err = db.Pool.Exec(ctx,
		`UPDATE task SET rrule = $2, repeat_anchor = $3, nag_minutes = $4
		  WHERE id = $1 AND calendar_id = ANY($5)`,
		t.ID, rule, anchor, nag, calIDs)
	return err
}

// repeatUpdates turns the repeat fields of a PATCH into SET clauses for
// updateTask's own statement.
//
// 🔴 Clauses, not statements. updateTask authorises the write with
// `WHERE calendar_id = ANY(writable)`, and a separate UPDATE here would have
// walked straight past it: a viewer of a shared calendar could not rename a
// task but could make it nag its owner every five minutes. Riding in the same
// statement means the repeat is exactly as writable as the title.
//
// nil means «не трогать», an empty string means «выключить» — the distinction
// every PATCH here already relies on. Turning repeat off clears the anchor too,
// but deliberately leaves task_occurrence alone: what was ticked on Tuesday was
// ticked on Tuesday, whatever the task is today.
func repeatUpdates(ctx context.Context, userID, taskID string, req UpdateTaskRequest, argNum int) ([]string, []interface{}, int) {
	var sets []string
	var args []interface{}

	if req.Rrule != nil {
		if *req.Rrule == "" {
			sets = append(sets, "rrule = NULL", "repeat_anchor = NULL")
		} else {
			// An existing anchor is kept when only the rule's text changes, so
			// editing «каждый день» into «через день» does not silently restart
			// the count.
			// The due date this same PATCH sets, when it sets one: the stored
			// value is read before the UPDATE and would be the OLD date (audit
			// 25.09: «каждый понедельник» with a Monday due date on a Thursday
			// task repeated on Thursdays).
			var due *time.Time
			if req.DueDate != nil && *req.DueDate != "" {
				if t, err := time.Parse(time.RFC3339, *req.DueDate); err == nil {
					due = &t
				}
			}
			if due == nil {
				_ = db.Pool.QueryRow(ctx, `
					-- recurrence-agnostic: the series' own deadline, to anchor it.
					SELECT due_date FROM task WHERE id = $1`, taskID).Scan(&due)
			}
			sets = append(sets,
				fmt.Sprintf("rrule = $%d", argNum),
				fmt.Sprintf("repeat_anchor = COALESCE(repeat_anchor, $%d)", argNum+1))
			args = append(args, *req.Rrule, anchorFor(ctx, userID, due))
			argNum += 2
		}
	}
	if req.NagMinutes != nil {
		if *req.NagMinutes > 0 {
			sets = append(sets, fmt.Sprintf("nag_minutes = $%d", argNum))
			args = append(args, *req.NagMinutes)
			argNum++
		} else {
			sets = append(sets, "nag_minutes = NULL")
		}
	}
	return sets, args, argNum
}
