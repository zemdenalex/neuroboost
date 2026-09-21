package events

import (
	"context"
	"errors"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/usersettings"
)

// Turning an event into a task — the other half of Настя's «превратить задачу
// в событие и обратно» (17–18.09). Denis 21.09: the card says what becomes
// what, and what is lost is named, not dropped.

// PlannedTask is what an event becomes. The bot draws its «что чем станет»
// card from this and from the lost codes — the mapping lives here and nowhere
// else, because the bot is a separate module and a copy there would drift.
type PlannedTask struct {
	Title            string   `json:"title"`
	Description      *string  `json:"description,omitempty"`
	CalendarID       string   `json:"calendar_id"`
	Tags             []string `json:"tags"`
	DueDate          string   `json:"due_date"` // YYYY-MM-DD, in the user's zone
	EstimatedMinutes *int     `json:"estimated_minutes,omitempty"`
	Rrule            *string  `json:"rrule,omitempty"`

	due time.Time // local midnight: what is stored, and the series anchor
}

// What an event has and a task cannot hold. Codes, not sentences: the bot
// words them in the user's language.
const (
	LostStartTime = "start_time"
	LostColor     = "color"
	LostLocation  = "location"
	LostReminders = "reminders"
)

// taskFromEvent maps one event — or one occurrence of it, given as start/end —
// onto a task.
//
// 🔴 The due day is the day in the USER's zone. 01:30 in Moscow is the
// previous day in UTC, and a task due «yesterday» is the whole shape of
// learning-the-right-time-in-the-wrong-zone.
func taskFromEvent(ev Event, start, end time.Time, tz string, series bool) (PlannedTask, []string) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	l := start.In(loc)
	due := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)

	p := PlannedTask{
		Title: ev.Title, Description: ev.Description, CalendarID: ev.CalendarID,
		Tags: ev.Tags, DueDate: due.Format("2006-01-02"), due: due,
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if series && ev.Rrule != nil && *ev.Rrule != "" {
		p.Rrule = ev.Rrule
	}

	var lost []string
	if !ev.AllDay {
		// An all-day event has no hour to lose and no length worth estimating.
		if mins := int(end.Sub(start).Minutes()); mins > 0 {
			p.EstimatedMinutes = &mins
		}
		lost = append(lost, LostStartTime)
	}
	if ev.Color != nil && *ev.Color != "" {
		lost = append(lost, LostColor)
	}
	if ev.Location != nil && *ev.Location != "" {
		lost = append(lost, LostLocation)
	}
	if len(ev.ReminderOffsets) > 0 {
		// Not carried over: the task gets the user's default task preset,
		// because «15 minutes before» means nothing without a start hour.
		lost = append(lost, LostReminders)
	}
	return p, lost
}

var (
	ErrToTaskMode           = errors.New("mode must be move or link")
	ErrToTaskRepeatRequired = errors.New("repeat must be series or once for a repeating event")
	ErrOccurrenceRequired   = errors.New("once needs the day of the occurrence")
)

// ToTaskRequest is POST /api/events/{id}/to-task.
type ToTaskRequest struct {
	Mode       string `json:"mode"`             // move | link
	Repeat     string `json:"repeat,omitempty"` // series | once, required for a repeating event
	Occurrence string `json:"occurrence,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

// ToTaskResult answers it. With dry_run nothing is written and Task.ID is
// empty: the bot shows the card first and only then asks for real.
type ToTaskResult struct {
	Task    PlannedTaskWithID `json:"task"`
	Lost    []string          `json:"lost"`
	EventID *string           `json:"event_id"`
	DryRun  bool              `json:"dry_run"`
}

// PlannedTaskWithID is the planned task plus the id it got, once it exists.
type PlannedTaskWithID struct {
	ID string `json:"id,omitempty"`
	PlannedTask
}

// EventToTask turns an event — a whole series, or one day of it — into a task,
// in one transaction.
//
//   - move + series/plain: the event is deleted.
//   - move + once: that one day is skipped; 🔴 never deleteEvent, which would
//     take the whole series with it.
//   - link + series/plain: the event stays and points at the new task.
//   - link + once: that day is detached into its own event pointing at the
//     task — the series itself is not linked to a one-off task.
func EventToTask(ctx context.Context, userID, rawID string, req ToTaskRequest) (*ToTaskResult, error) {
	if req.Mode != "move" && req.Mode != "link" {
		return nil, ErrToTaskMode
	}
	if req.Repeat != "" && req.Repeat != "series" && req.Repeat != "once" {
		return nil, ErrToTaskRepeatRequired
	}

	parentID, occDate, isInstance := parseInstanceID(rawID)
	parent, err := getEvent(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	repeating := parent.Rrule != nil && *parent.Rrule != ""
	once := false
	if repeating {
		if req.Repeat == "" {
			return nil, ErrToTaskRepeatRequired
		}
		once = req.Repeat == "once"
		if !once {
			// A rule this API cannot expand must not be copied onto a task
			// that would then claim to repeat and never do.
			if _, perr := parseRRule(*parent.Rrule); perr != nil {
				return nil, errors.Join(ErrInvalidRrule, perr)
			}
		}
	}

	start, end := parent.StartsAt, parent.EndsAt
	if once {
		if !isInstance {
			if req.Occurrence == "" {
				return nil, ErrOccurrenceRequired
			}
			d, perr := time.Parse(instanceIDDateLayout, req.Occurrence)
			if perr != nil {
				return nil, ErrOccurrenceRequired
			}
			occDate = d
		}
		s, e, ok := occurrenceWindow(*parent, occDate)
		if !ok {
			return nil, errNoSuchOccurrence
		}
		start, end = s, e
	}

	// The reader's day, not the event's author's: in a shared calendar the
	// person turning it into a task is the one whose «tomorrow» it is.
	tz := "Europe/Moscow"
	_ = db.Pool.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow') FROM "user" WHERE id = $1`, userID).Scan(&tz)

	planned, lost := taskFromEvent(*parent, start, end, tz, repeating && !once)
	if lost == nil {
		lost = []string{}
	}
	res := &ToTaskResult{Task: PlannedTaskWithID{PlannedTask: planned}, Lost: lost, DryRun: req.DryRun}
	if req.DryRun {
		return res, nil
	}

	// 🔴 Singular for the INSERT's destination, plural for the event's
	// UPDATE/DELETE — calendars.TestWritesScopeByWritableCalendars.
	calID, err := calendars.WritableIDFor(ctx, userID, parent.CalendarID)
	if err != nil {
		return nil, err
	}
	calIDs, err := calendars.WritableIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var anchor *time.Time
	if planned.Rrule != nil {
		a := planned.due
		anchor = &a
	}
	var taskID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO task (user_id, calendar_id, title, description, status, priority,
		                  estimated_minutes, due_date, tags, contexts, reminder_offsets,
		                  rrule, repeat_anchor)
		VALUES ($1, $2, $3, $4, 'TODO', 3, $5, $6, $7, '{}', $8, $9, $10)
		RETURNING id::text`,
		userID, calID, planned.Title, planned.Description, planned.EstimatedMinutes,
		planned.due, planned.Tags, usersettings.DefaultTaskOffsets(ctx, userID),
		planned.Rrule, anchor).Scan(&taskID); err != nil {
		return nil, err
	}
	res.Task.ID = taskID

	switch {
	case once && req.Mode == "link":
		occ := *parent
		occ.StartsAt, occ.EndsAt, occ.Rrule = start, end, nil
		occ.TaskID = &taskID
		detached, derr := detachOccurrenceTx(ctx, tx, userID, occ, parentID, occDate)
		if derr != nil {
			return nil, derr
		}
		res.EventID = &detached.ID
	case once:
		// move: skip this one day, never the series.
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_exception (event_id, user_id, occurrence, skipped)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (event_id, occurrence)
			DO UPDATE SET skipped = true, user_id = EXCLUDED.user_id`,
			parentID, userID, occDate); err != nil {
			return nil, err
		}
	case req.Mode == "link":
		if _, err := tx.Exec(ctx,
			`UPDATE event SET task_id = $2, updated_at = NOW() WHERE id = $1 AND calendar_id = ANY($3)`,
			parentID, taskID, calIDs); err != nil {
			return nil, err
		}
		res.EventID = &parentID
	default:
		if _, err := tx.Exec(ctx,
			`DELETE FROM event WHERE id = $1 AND calendar_id = ANY($2)`, parentID, calIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}
