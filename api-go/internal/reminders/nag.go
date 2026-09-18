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
		       r.occurrence_start,
		       -- 🔴 The owner's zone, because "the end of the day" is a question
		       -- only they can answer. occurrence_start comes back in UTC, and
		       -- 23:59 UTC is 02:59 the next morning in Moscow.
		       COALESCE(u.timezone, 'Europe/Moscow')
		FROM reminder r
		JOIN "user" u ON u.id = r.user_id
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
		timezone   string
	}
	var candidates []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.nagMinutes, &d.remindAt, &d.occurrence, &d.timezone); err != nil {
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

		if !WithinSameLocalDay(d.occurrence, next, d.timezone) {
			continue
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

// WithinSameLocalDay reports whether a re-armed reminder would still fall on the
// day it is about, where the OWNER lives.
//
// 🔴 Never past the end of that day. A 23:50 event nagging hourly would
// otherwise wake someone at 02:00 about something that finished yesterday, and
// that is how a person mutes the bot — after which nothing gets through.
//
// ⚠ The zone is load-bearing and was wrong in the first version. `occurrence_start`
// is a timestamptz and comes back in UTC, so computing 23:59:59 from it built
// the end of the UTC day: for a Moscow user that is 02:59 the following morning
// — precisely the 2am wake-up this guard exists to prevent — and for anyone west
// of UTC it cuts the evening short instead.
//
// A nil occurrence means the row is not about a particular day (a digest has
// none). Such a row has no day to overrun, so it is allowed through; the nag
// count still bounds it.
func WithinSameLocalDay(occurrence *time.Time, next time.Time, timezone string) bool {
	if occurrence == nil {
		return true
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// 🔴 UTC, not the server's local zone: the server's zone is an accident
		// of deployment, and falling back to it would make the same account
		// behave differently after a host move.
		loc = time.UTC
	}
	day := occurrence.In(loc)
	endOfDay := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, loc)
	return !next.After(endOfDay)
}
