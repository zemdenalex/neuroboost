package reminders

import (
	"context"
	"encoding/json"
	"time"

	"neuroboost/api-go/internal/remindrule"
	"neuroboost/api-go/internal/usersettings"
)

// scanRules writes the rows of «remind me regularly until the deadline»
// (remindrule): a task runs up to its due, an event up to its start.
//
// Each moment becomes an ordinary journal row with minutes_before = minutes
// left to the deadline, so the existing delivery, answer buttons and dedupe
// all apply unchanged. A moment that lands on a regular offset of the same
// deadline is one row, which is what a person would want.
//
// 🔴 Quiet hours DROP a rule's moment instead of shifting it. Shifting is right
// for a single «за день» reminder, but «every 5 minutes» shifted out of a
// 23:00–08:00 window would arrive as a hundred reminders at 08:00.
func scanRules(ctx context.Context, u scanUser, calIDs []string, st usersettings.Reminders, loc *time.Location, from, to time.Time) (int, error) {
	type item struct {
		kind     string
		id       string
		title    string
		deadline time.Time
		raw      []byte
	}
	var items []item

	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, due_date, remind_rule
		FROM task
		WHERE calendar_id = ANY($1)
		  AND remind_rule IS NOT NULL
		  AND due_date IS NOT NULL
		  AND rrule IS NULL
		  AND status NOT IN ('DONE', 'CANCELLED')
		  AND due_date > $2::timestamptz - interval '1 day'`,
		calIDs, from)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var it item
		var due time.Time
		if err := rows.Scan(&it.id, &it.title, &due, &it.raw); err != nil {
			rows.Close()
			return 0, err
		}
		it.kind, it.deadline = "TASK", remindrule.TaskDeadline(due, loc)
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	rows, err = db.Pool.Query(ctx, `
		SELECT id, title, starts_at, all_day, remind_rule
		FROM event
		WHERE calendar_id = ANY($1)
		  AND remind_rule IS NOT NULL
		  AND COALESCE(rrule, '') = ''
		  AND starts_at > $2::timestamptz - interval '1 day'`,
		calIDs, from)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		it := item{kind: "EVENT"}
		var allDay bool
		if err := rows.Scan(&it.id, &it.title, &it.deadline, &allDay, &it.raw); err != nil {
			rows.Close()
			return 0, err
		}
		// An all-day event's «до начала» is through its day, the same as a
		// date-only task: «экзамен 31.11 … в 12 и 16» reminds on the 31st too.
		if allDay {
			it.deadline = remindrule.AllDayDeadline(it.deadline, loc)
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, it := range items {
		var rule remindrule.Rule
		if json.Unmarshal(it.raw, &rule) != nil || rule.Validate() != nil {
			continue // a malformed rule reminds of nothing rather than of garbage
		}
		for _, at := range rule.Expand(it.deadline, loc, from, to) {
			if st.QuietHoursRespected {
				if shifted, ok := ShiftForQuietHours(at, 1<<30, st.QuietHoursStart, st.QuietHoursEnd, loc); !ok || !shifted.Equal(at) {
					continue
				}
			}
			minutes := int(it.deadline.Sub(at) / time.Minute)
			id := it.id
			var eventID, taskID *string
			if it.kind == "TASK" {
				taskID = &id
			} else {
				eventID = &id
			}
			n, err := insertReminder(ctx, u.id, it.kind, eventID, taskID,
				it.deadline, minutes, at, ReminderText(it.kind, it.title, it.deadline, minutes, loc))
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
	}
	return inserted, nil
}
