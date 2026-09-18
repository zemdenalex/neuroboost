package reminders

import (
	"context"
	"log/slog"
	"time"
)

// Repeating an unanswered reminder — «долбёжка».
//
// Denis, 18.09: «можно настроить чтобы оно напоминало о себе каждые 10 минут
// или каждый час», and the same property on events as on tasks.
//
// 🔴 A nag MOVES the existing row. It does not insert.
//
// The dedupe index is
//
//	(user_id, source_kind, COALESCE(event_id, task_id), calendar_id,
//	 occurrence_start, minutes_before)      -- 000015:34
//
// and `remind_at` is NOT in it. Two rows for 09:00 and 09:10 of the same
// occurrence differ only in a column outside the key, so the second INSERT
// collides and the second nag silently does not exist. Moving the row forward
// touches no key column at all.
//
// That is the same class of defect as the snooze that answered 500 to everyone
// from 000015 until 18.09 — a migration changed an index and a query that never
// changed became wrong.

const (
	// maxNags bounds how many times one reminder may come back.
	//
	// 🔴 A bot that nags forever gets muted, and a muted bot delivers NOTHING —
	// including the reminder that mattered. Six is enough to be hard to ignore
	// and few enough to stay inside one morning.
	maxNags = 6
)

// NagUnanswered moves every delivered-but-unanswered reminder forward by the
// nag interval of the thing it is about.
//
// Returns how many were moved. An error stops the round rather than skipping
// rows: a partial round that reports success would hide a broken query for as
// long as nobody counted.
func NagUnanswered(ctx context.Context, now time.Time, log *slog.Logger) (int, error) {
	// The interval lives on the SOURCE — event.nag_minutes or task.nag_minutes —
	// because it is a property of the thing being reminded about, not of one
	// delivery. Joined here rather than copied onto the reminder row, so
	// changing it changes the next nag instead of only future reminders.
	rows, err := db.Pool.Query(ctx, `
		SELECT r.id,
		       COALESCE(e.nag_minutes, t.nag_minutes) AS nag_minutes,
		       r.remind_at,
		       r.occurrence_start
		FROM reminder r
		LEFT JOIN event e ON e.id = r.event_id
		LEFT JOIN task  t ON t.id = r.task_id
		WHERE r.status = 'SENT'
		  AND r.answered_at IS NULL
		  AND r.nag_count < $1
		  AND COALESCE(e.nag_minutes, t.nag_minutes) IS NOT NULL
		  AND r.remind_at <= $2`,
		maxNags, now)
	if err != nil {
		return 0, err
	}

	type due struct {
		id         string
		nagMinutes int
		remindAt   time.Time
		occurrence *time.Time
	}
	var candidates []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.nagMinutes, &d.remindAt, &d.occurrence); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	moved := 0
	for _, d := range candidates {
		next := d.remindAt.Add(time.Duration(d.nagMinutes) * time.Minute)

		// 🔴 Never past the end of the day the reminder is about. A 23:50 event
		// nagging hourly would otherwise wake someone at 02:00 about something
		// that finished yesterday — and that is how a person turns the bot off
		// altogether.
		if d.occurrence != nil {
			endOfDay := time.Date(d.occurrence.Year(), d.occurrence.Month(), d.occurrence.Day(),
				23, 59, 59, 0, d.occurrence.Location())
			if next.After(endOfDay) {
				continue
			}
		}

		tag, err := db.Pool.Exec(ctx, `
			UPDATE reminder
			   SET remind_at = $2, status = 'PENDING', sent_at = NULL, nag_count = nag_count + 1
			 WHERE id = $1 AND answered_at IS NULL AND status = 'SENT'`,
			d.id, next)
		if err != nil {
			return moved, err
		}
		// The WHERE repeats the conditions on purpose: between the SELECT and
		// this UPDATE the user may have answered, and re-arming a reminder they
		// just dismissed is the most annoying thing this code could do.
		moved += int(tag.RowsAffected())
	}

	if moved > 0 && log != nil {
		log.Info("reminders re-armed", slog.Int("count", moved))
	}
	return moved, nil
}
