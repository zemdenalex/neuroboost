package reminders

import (
	"context"
	"log/slog"
	"time"

	"neuroboost/api-go/internal/usersettings"
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

// 🔴 No cap and no midnight stop (Denis, 07.11: «надо сделать чтобы оно
// бесконечно напоминало а не 6 раз, там же есть функция отключить»). It was
// six nags and never past the end of the item's day; now it comes back until
// answered or switched off. What keeps it bearable instead:
//   - quiet hours: a nag that would land in them waits for their end;
//   - only the NEWEST delivered reminder of an item nags — with «every 5
//     minutes until the deadline» every older row nagging too would multiply;
//   - a task closed anywhere (web, another device) stops its nag.

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
	// 🔴 First retire what a newer reminder replaced. «Newer» is by created_at
	// (then id) — the moment the scanner wrote the row, which never moves.
	// remind_at cannot be the key: nagging moves it, so the row that was newest
	// stopped looking newest the moment it was re-armed, and the rows it held
	// back came back too (review 07.11: nags multiplied without end).
	//
	// A retired row is marked answered: the same «stop coming back» the buttons
	// write (action.go), and it is never delivered again.
	if _, err := db.Pool.Exec(ctx, `
		UPDATE reminder r
		   SET answered_at = NOW()
		 WHERE r.answered_at IS NULL
		   AND (r.status = 'SENT' OR (r.status = 'PENDING' AND r.nag_count > 0))
		   AND EXISTS (
		         SELECT 1 FROM reminder n
		          WHERE n.user_id = r.user_id
		            AND COALESCE(n.event_id, n.task_id) = COALESCE(r.event_id, r.task_id)
		            AND n.status = 'SENT'
		            AND (n.created_at, n.id) > (r.created_at, r.id)
		       )`); err != nil {
		return 0, err
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT r.id,
		       COALESCE(e.nag_minutes, t.nag_minutes) AS nag_minutes,
		       r.remind_at,
		       COALESCE(u.timezone, 'Europe/Moscow'),
		       COALESCE(u.settings, '{}')
		FROM reminder r
		JOIN "user" u ON u.id = r.user_id
		LEFT JOIN event e ON e.id = r.event_id
		LEFT JOIN task  t ON t.id = r.task_id
		WHERE r.status = 'SENT'
		  AND r.answered_at IS NULL
		  AND COALESCE(e.nag_minutes, t.nag_minutes) IS NOT NULL
		  AND r.remind_at <= $1
		  AND (t.id IS NULL OR t.status NOT IN ('DONE', 'CANCELLED'))`,
		now)
	if err != nil {
		return 0, err
	}

	type due struct {
		id         string
		nagMinutes int
		remindAt   time.Time
		timezone   string
		settings   []byte
	}
	var candidates []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.nagMinutes, &d.remindAt, &d.timezone, &d.settings); err != nil {
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
		// From now, not from the old remind_at: a row that waited (a deploy, a
		// long quiet night) must come back once, not replay every missed nag a
		// minute apart.
		base := d.remindAt
		if base.Before(now) {
			base = now
		}
		next := base.Add(time.Duration(d.nagMinutes) * time.Minute)
		if st := usersettings.ParseReminders(d.settings); st.QuietHoursRespected {
			loc, err := time.LoadLocation(d.timezone)
			if err != nil {
				loc = time.UTC
			}
			// A huge «minutes before» so the grace rule never drops a nag: it
			// only waits for the end of the quiet window.
			if shifted, ok := ShiftForQuietHours(next, 1<<30, st.QuietHoursStart, st.QuietHoursEnd, loc); ok {
				next = shifted
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
