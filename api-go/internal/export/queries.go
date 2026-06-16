package export

import (
	"context"
)

func queryEvents(ctx context.Context, userID string) ([]EventRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, description, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), location, color, COALESCE(tags, '{}'),
		       task_id, COALESCE(is_work_event, true), created_at, updated_at
		FROM event
		WHERE user_id = $1
		ORDER BY starts_at ASC
	`, userID)
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
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, description, status, category, priority,
		       estimated_minutes, actual_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		       energy, parent_id, completed_at, created_at, updated_at
		FROM task
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, userID)
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
