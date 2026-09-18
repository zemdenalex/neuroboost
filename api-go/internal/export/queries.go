package export

import (
	"context"

	"neuroboost/api-go/internal/calendars"
)

func queryEvents(ctx context.Context, userID string) ([]EventRow, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// An empty list is a legitimate "nothing visible", not an error:
	// ANY('{}') returns zero rows.
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, description, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		       task_id, COALESCE(is_work_event, true), created_at, updated_at
		FROM event
		WHERE calendar_id = ANY($1)
		ORDER BY starts_at ASC
	`, calIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventRow
	for rows.Next() {
		var ev EventRow
		var tags []string
		err := rows.Scan(
			&ev.ID, &ev.UserID, &ev.Title, &ev.Description,
			&ev.StartsAt, &ev.EndsAt, &ev.AllDay, &ev.Rrule,
			&ev.Timezone, &ev.Location, &ev.Color, &tags,
			&ev.TaskID, &ev.IsWorkEvent, &ev.CreatedAt, &ev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		ev.Tags = tags
		events = append(events, ev)
	}

	if events == nil {
		events = []EventRow{}
	}
	return events, nil
}

func queryTasks(ctx context.Context, userID string) ([]TaskRow, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// recurrence-agnostic JOIN, deliberately: this query carries the repeat
	// RULE (rrule, repeat_anchor) so a restore does not turn every repeating
	// task into a one-off — but it must NOT join task_occurrence, because an
	// export is row-per-thing and a join would emit a task once per day it ran.
	// The days go out in their own list: queryTaskOccurrences, below.
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, description, status, category, priority,
		       estimated_minutes, actual_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		       energy, parent_id, completed_at, created_at, updated_at,
		       rrule, repeat_anchor, nag_minutes, event_id::text
		FROM task
		WHERE calendar_id = ANY($1)
		ORDER BY created_at ASC
	`, calIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskRow
	for rows.Next() {
		var tk TaskRow
		var tags, contexts []string
		err := rows.Scan(
			&tk.ID, &tk.UserID, &tk.Title, &tk.Description, &tk.Status, &tk.Category,
			&tk.Priority, &tk.EstimatedMinutes, &tk.ActualMinutes, &tk.DueDate, &tags, &contexts,
			&tk.Energy, &tk.ParentID, &tk.CompletedAt, &tk.CreatedAt, &tk.UpdatedAt,
			&tk.Rrule, &tk.RepeatAnchor, &tk.NagMinutes, &tk.EventID,
		)
		if err != nil {
			return nil, err
		}
		tk.Tags = tags
		tk.Contexts = contexts
		tasks = append(tasks, tk)
	}

	if tasks == nil {
		tasks = []TaskRow{}
	}
	return tasks, nil
}

// queryTaskOccurrences exports the per-day history of repeating tasks.
//
// 🔴 It is the only record that a given day was done or skipped. An export that
// leaves it out restores a series with its whole history erased — which looks
// like success, because the task is there.
func queryTaskOccurrences(ctx context.Context, userID string) ([]TaskOccurrenceRow, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT o.task_id::text, o.occurrence, o.state, o.acted_at
		FROM task_occurrence o
		JOIN task t ON t.id = o.task_id
		WHERE t.calendar_id = ANY($1)
		ORDER BY o.task_id, o.occurrence`, calIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TaskOccurrenceRow{}
	for rows.Next() {
		var r TaskOccurrenceRow
		if err := rows.Scan(&r.TaskID, &r.Occurrence, &r.State, &r.ActedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
