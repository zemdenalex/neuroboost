package reminders

import (
	"context"
	"log/slog"
	"time"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/events"
	"neuroboost/api-go/internal/usersettings"
)

var db *database.DB

// InitDB follows the package-level-pool pattern the events and tasks packages
// already use (see cmd/api/main.go).
func InitDB(d *database.DB) { db = d }

// maxOffsetMinutes bounds how far ahead the scan must look for occurrences:
// the largest offset any preset uses is a month.
const maxOffsetMinutes = 43200

type scanUser struct {
	id       string
	timezone string
	settings []byte
}

// Scan finds everything due in [from, to) and writes PENDING journal rows.
// It returns how many rows it actually inserted — normally a small number,
// and the one the worker logs.
func Scan(ctx context.Context, from, to time.Time, log *slog.Logger) (int, error) {
	// Users without tg_id are skipped here rather than downstream, so we never
	// accumulate rows nobody can deliver.
	rows, err := db.Pool.Query(ctx, `
		SELECT id, COALESCE(timezone, 'Europe/Moscow'), COALESCE(settings, '{}')
		FROM "user"
		WHERE tg_id IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	var users []scanUser
	for rows.Next() {
		var u scanUser
		if err := rows.Scan(&u.id, &u.timezone, &u.settings); err != nil {
			rows.Close()
			return 0, err
		}
		users = append(users, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, u := range users {
		loc, err := time.LoadLocation(u.timezone)
		if err != nil {
			loc = time.UTC
		}
		st := usersettings.ParseReminders(u.settings)

		n, err := scanEvents(ctx, u, st, loc, from, to)
		if err != nil {
			// One user's bad data must not stop everyone else's reminders.
			log.Error("reminder event scan failed",
				slog.String("user_id", u.id), slog.String("error", err.Error()))
		}
		inserted += n

		n, err = scanTasks(ctx, u, st, loc, from, to)
		if err != nil {
			log.Error("reminder task scan failed",
				slog.String("user_id", u.id), slog.String("error", err.Error()))
		}
		inserted += n

		if st.DigestEnabled {
			if day, fireAt, ok := DigestDue(from, to, st.DigestAt, loc); ok {
				n, err := insertDigest(ctx, u.id, day, fireAt, loc)
				if err != nil {
					log.Error("digest insert failed",
						slog.String("user_id", u.id), slog.String("error", err.Error()))
				}
				inserted += n
			}
		}
	}
	return inserted, nil
}

func scanEvents(ctx context.Context, u scanUser, st usersettings.Reminders, loc *time.Location, from, to time.Time) (int, error) {
	// Look ahead by the largest offset in use: an occurrence up to a month
	// away can have a reminder due right now.
	horizon := to.Add(maxOffsetMinutes * time.Minute)

	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow'), COALESCE(reminder_offsets, '{}')
		FROM event
		WHERE user_id = $1
		  AND cardinality(reminder_offsets) > 0
		  AND (
		    (rrule IS NOT NULL AND rrule != '')
		    OR starts_at BETWEEN $2 AND $3
		  )`,
		u.id, from, horizon)
	if err != nil {
		return 0, err
	}

	type candidate struct {
		ev      events.Event
		offsets []int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ev.ID, &c.ev.UserID, &c.ev.Title, &c.ev.StartsAt,
			&c.ev.EndsAt, &c.ev.AllDay, &c.ev.Rrule, &c.ev.Timezone, &c.offsets); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, c := range candidates {
		exceptions := fetchSkippedOccurrences(ctx, u.id, c.ev.ID)
		occurrences := events.OccurrencesInRange(c.ev, from, horizon, exceptions)
		for _, d := range DueReminders(occurrences, c.offsets, from, to) {
			remindAt := d.RemindAt
			if st.QuietHoursRespected {
				shifted, ok := ShiftForQuietHours(remindAt, d.MinutesBefore, st.QuietHoursStart, st.QuietHoursEnd, loc)
				if !ok {
					continue
				}
				remindAt = shifted
			}
			eventID := c.ev.ID
			n, err := insertReminder(ctx, u.id, "EVENT", &eventID, nil,
				d.OccurrenceStart, d.MinutesBefore, remindAt, c.ev.Title)
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
	}
	return inserted, nil
}

// fetchSkippedOccurrences mirrors the events package's own exception lookup:
// only rows with skipped = true remove an occurrence.
func fetchSkippedOccurrences(ctx context.Context, userID, eventID string) []time.Time {
	rows, err := db.Pool.Query(ctx,
		`SELECT occurrence FROM event_exception
		 WHERE user_id = $1 AND event_id = $2 AND skipped = true`,
		userID, eventID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []time.Time
	for rows.Next() {
		var t time.Time
		if rows.Scan(&t) == nil {
			out = append(out, t)
		}
	}
	return out
}

func scanTasks(ctx context.Context, u scanUser, st usersettings.Reminders, loc *time.Location, from, to time.Time) (int, error) {
	horizon := to.Add(maxOffsetMinutes * time.Minute)

	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, due_date, COALESCE(reminder_offsets, '{}')
		FROM task
		WHERE user_id = $1
		  AND due_date IS NOT NULL
		  AND cardinality(reminder_offsets) > 0
		  AND status NOT IN ('DONE', 'CANCELLED')
		  AND due_date BETWEEN $2 AND $3`,
		u.id, from, horizon)
	if err != nil {
		return 0, err
	}

	type candidate struct {
		id      string
		title   string
		due     time.Time
		offsets []int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.title, &c.due, &c.offsets); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	inserted := 0
	for _, c := range candidates {
		// A task has exactly one "occurrence": its due date.
		for _, d := range DueReminders([]time.Time{c.due}, c.offsets, from, to) {
			remindAt := d.RemindAt
			if st.QuietHoursRespected {
				shifted, ok := ShiftForQuietHours(remindAt, d.MinutesBefore, st.QuietHoursStart, st.QuietHoursEnd, loc)
				if !ok {
					continue
				}
				remindAt = shifted
			}
			taskID := c.id
			n, err := insertReminder(ctx, u.id, "TASK", nil, &taskID,
				d.OccurrenceStart, d.MinutesBefore, remindAt, c.title)
			if err != nil {
				return inserted, err
			}
			inserted += n
		}
	}
	return inserted, nil
}

// insertReminder writes one journal row. A unique-index conflict means this
// exact (occurrence, offset) pair was already scheduled — not an error.
func insertReminder(ctx context.Context, userID, kind string, eventID, taskID *string,
	occurrence time.Time, minutesBefore int, remindAt time.Time, message string) (int, error) {
	tag, err := db.Pool.Exec(ctx, `
		INSERT INTO reminder (user_id, source_kind, event_id, task_id, occurrence_start,
		                      minutes_before, remind_at, status, channel, message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', 'TELEGRAM', $8)
		ON CONFLICT DO NOTHING`,
		userID, kind, eventID, taskID, occurrence, minutesBefore, remindAt, message)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// digestMinutesBefore is the sentinel stored in reminder.minutes_before for a
// daily digest, which has no offset. Explicit rather than NULL: NULLS NOT
// DISTINCT protects the index, but a real value keeps the dedupe key readable
// and does not depend on index semantics alone. (-1 is reserved for snooze.)
const digestMinutesBefore = -2

// insertDigest uses the user's local midnight as occurrence_start, so the
// unique index means "one digest per user per local day".
//
// 🔴 The message is composed here, and it must never be empty: this row used to
// be written with '' and Telegram answers an empty send with
// "Bad Request: message text is empty" — so every digest failed, every morning,
// leaving only a FAILED row nobody read. Composing at insert time is also the
// right moment: the scan writes this row when the digest is due, so the day's
// contents are current rather than hours stale.
// remind_at carries the digest's real time rather than NOW(). The scan window
// looks ahead, so the row is written slightly BEFORE the digest is due; with
// NOW() the notifier's `remind_at <= NOW()` gate was already satisfied and the
// digest went out up to a minute early.
func insertDigest(ctx context.Context, userID string, localDay, fireAt time.Time, loc *time.Location) (int, error) {
	message := DigestText(localDay, loc, digestEvents(ctx, userID, localDay), digestTasks(ctx, userID, localDay))

	tag, err := db.Pool.Exec(ctx, `
		INSERT INTO reminder (user_id, source_kind, occurrence_start, minutes_before,
		                      remind_at, status, channel, message)
		VALUES ($1, 'DIGEST', $2, $3, $4, 'PENDING', 'TELEGRAM', $5)
		ON CONFLICT DO NOTHING`,
		userID, localDay, digestMinutesBefore, fireAt, message)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// digestEvents collects what the user actually has on that local day,
// recurring occurrences included.
//
// A query error yields an empty list rather than an error: a digest listing
// only tasks is worth sending, whereas propagating the failure would abort the
// whole scan and cost the user their per-event reminders too.
func digestEvents(ctx context.Context, userID string, localDay time.Time) []DigestEvent {
	dayStart := localDay
	dayEnd := localDay.AddDate(0, 0, 1)

	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, title, starts_at, ends_at, all_day, rrule,
		       COALESCE(timezone, 'Europe/Moscow')
		FROM event
		WHERE user_id = $1
		  AND ((rrule IS NOT NULL AND rrule != '') OR (starts_at < $3 AND ends_at > $2))`,
		userID, dayStart, dayEnd)
	if err != nil {
		return nil
	}
	var candidates []events.Event
	for rows.Next() {
		var ev events.Event
		if err := rows.Scan(&ev.ID, &ev.UserID, &ev.Title, &ev.StartsAt, &ev.EndsAt,
			&ev.AllDay, &ev.Rrule, &ev.Timezone); err != nil {
			rows.Close()
			return nil
		}
		candidates = append(candidates, ev)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil
	}

	var out []DigestEvent
	for _, ev := range candidates {
		duration := ev.EndsAt.Sub(ev.StartsAt)
		exceptions := fetchSkippedOccurrences(ctx, userID, ev.ID)
		for _, start := range events.OccurrencesInRange(ev, dayStart, dayEnd, exceptions) {
			out = append(out, DigestEvent{
				Title:    ev.Title,
				StartsAt: start,
				EndsAt:   start.Add(duration),
				AllDay:   ev.AllDay,
			})
		}
	}
	return out
}

// digestTasks lists what is due that local day and still open.
func digestTasks(ctx context.Context, userID string, localDay time.Time) []DigestTask {
	rows, err := db.Pool.Query(ctx, `
		SELECT title FROM task
		WHERE user_id = $1
		  AND due_date IS NOT NULL
		  AND due_date >= $2 AND due_date < $3
		  AND status NOT IN ('DONE', 'CANCELLED')
		ORDER BY priority, due_date`,
		userID, localDay, localDay.AddDate(0, 0, 1))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []DigestTask
	for rows.Next() {
		var t DigestTask
		if rows.Scan(&t.Title) == nil {
			out = append(out, t)
		}
	}
	return out
}
