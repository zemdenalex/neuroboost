package daytasks

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB follows the package-level-pool pattern the other packages use.
func InitDB(d *database.DB) { db = d }

var (
	ErrTooLate         = errors.New("today's tasks can only be removed before noon; past days not at all")
	ErrNotOpen         = errors.New("only an open task can be promised for a day")
	ErrTaskNotFound    = errors.New("task not found")
	ErrRangeTooLarge   = errors.New("range must be at most 62 days")
	ErrInvalidRange    = errors.New("from must be a date not after to")
	ErrNotAnOccurrence = errors.New("this series does not occur on that day")
)

type Item struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
}

type Day struct {
	Day       string `json:"day"`
	Target    int    `json:"target"`
	Confirmed bool   `json:"confirmed"`
	Items     []Item `json:"items"`
	Done      int    `json:"done"`
	Level     int    `json:"level"`
}

const dateFmt = "2006-01-02"

func userZoneAndTarget(ctx context.Context, userID string) (string, int, error) {
	var tz string
	var raw []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow'), COALESCE(settings, '{}') FROM "user" WHERE id = $1`,
		userID).Scan(&tz, &raw)
	return tz, Target(raw), err
}

// doneOnDay is the one definition of «closed on that day» (spec §2), shared by
// List and the «only open» check so the two cannot disagree.
//
//	one-off: completed_at falls on the day in the user's zone
//	series:  that day of the series is answered «done»
//
// Total: it is TRUE or FALSE, never NULL. A DONE row without completed_at
// (written by a door that did not stamp it) reads as not done on any day;
// before this it read NULL, and List's bool scan answered 500 (review C1).
const doneOnDay = `
	CASE WHEN COALESCE(t.rrule, '') <> ''
	     THEN EXISTS (SELECT 1 FROM task_occurrence o
	                   WHERE o.task_id = t.id AND o.occurrence = $DAY AND o.state = 'done')
	     ELSE t.status = 'DONE' AND t.completed_at IS NOT NULL
	          AND (t.completed_at AT TIME ZONE $TZ)::date = $DAY
	END`

// List returns every day of [from, to], empty ones included.
func List(ctx context.Context, userID string, from, to time.Time) ([]Day, error) {
	if ymd(to) < ymd(from) {
		return nil, ErrInvalidRange
	}
	if to.Sub(from) > 62*24*time.Hour {
		return nil, ErrRangeTooLarge
	}
	tz, target, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return nil, err
	}

	byDay := map[string]*Day{}
	var out []Day
	for d := from; ymd(d) <= ymd(to); d = d.AddDate(0, 0, 1) {
		out = append(out, Day{Day: d.Format(dateFmt), Target: target, Items: []Item{}})
	}
	for i := range out {
		byDay[out[i].Day] = &out[i]
	}

	confirmed, err := db.Pool.Query(ctx, `
		SELECT to_char(day, 'YYYY-MM-DD'), target FROM day_commitment_day
		 WHERE user_id = $1 AND day BETWEEN $2 AND $3`,
		userID, from.Format(dateFmt), to.Format(dateFmt))
	if err != nil {
		return nil, err
	}
	for confirmed.Next() {
		var d string
		var n int
		if err := confirmed.Scan(&d, &n); err != nil {
			confirmed.Close()
			return nil, err
		}
		// A taken day keeps the target it was taken with: changing the
		// setting later neither turns today green nor recolours the past.
		if day := byDay[d]; day != nil {
			day.Confirmed = true
			day.Target = n
		}
	}
	confirmed.Close()
	if err := confirmed.Err(); err != nil {
		return nil, err
	}

	// Two reads, on purpose. The promises are the user's own rows and are read
	// by user_id; the tasks behind them are read only through the calendars the
	// user can see now (calendars.CalendarIDsFor), like every other task read.
	// A task from a calendar the user has since left drops out of the day
	// instead of showing its title to someone who no longer has it.
	prom, err := db.Pool.Query(ctx, `
		SELECT to_char(day, 'YYYY-MM-DD'), task_id::text FROM day_commitment
		 WHERE user_id = $1 AND day BETWEEN $2 AND $3 AND removed_at IS NULL
		 ORDER BY day, added_at`,
		userID, from.Format(dateFmt), to.Format(dateFmt))
	if err != nil {
		return nil, err
	}
	var promDays, promTasks []string
	for prom.Next() {
		var d, id string
		if err := prom.Scan(&d, &id); err != nil {
			prom.Close()
			return nil, err
		}
		promDays = append(promDays, d)
		promTasks = append(promTasks, id)
	}
	prom.Close()
	if err := prom.Err(); err != nil {
		return nil, err
	}
	if len(promTasks) == 0 {
		return out, nil
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// $DAY is the row's own day, $TZ the user's zone: substituted by hand into
	// the shared fragment, parameters stay parameters.
	q := `
		SELECT x.day, t.id::text, t.title,
		       ` + replaceAll(doneOnDay, "$DAY", "x.day::date", "$TZ", "$4") + `
		  FROM unnest($1::text[], $2::text[]) WITH ORDINALITY AS x(day, task_id, n)
		  JOIN task t ON t.id = x.task_id::uuid
		 WHERE t.calendar_id = ANY($3)
		 ORDER BY x.n`
	rows, err := db.Pool.Query(ctx, q, promDays, promTasks, calIDs, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var it Item
		if err := rows.Scan(&d, &it.TaskID, &it.Title, &it.Done); err != nil {
			return nil, err
		}
		if day := byDay[d]; day != nil {
			day.Items = append(day.Items, it)
			if it.Done {
				day.Done++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Confirmed {
			out[i].Level = Level(out[i].Done, out[i].Target)
		}
	}
	return out, nil
}

// querier is what Add and Confirm write through: the pool, or Confirm's
// transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// isPast: a past day is closed to every edit, not only to removal — else an
// untaken ⬛ day could be taken the next morning (review I3).
func isPast(day time.Time, tz string) bool {
	return ymd(day) < ymd(Today(time.Now(), tz))
}

// checkOpen is the one «can this task be promised for that day» (spec §2):
// visible to the user, not closed (a one-off DONE, anything CANCELLED), not
// already done that day, and — for a series — a day the series occurs on.
func checkOpen(ctx context.Context, q querier, calIDs []string, day time.Time, tz, taskID string) error {
	var rrule string
	var anchor *time.Time
	var closed bool
	err := q.QueryRow(ctx, `
		SELECT COALESCE(t.rrule, ''), t.repeat_anchor,
		       `+replaceAll(doneOnDay, "$DAY", "$3::date", "$TZ", "$4")+`
		       OR (COALESCE(t.rrule, '') = '' AND t.status = 'DONE')
		       OR t.status = 'CANCELLED'
		  FROM task t WHERE t.id = $1 AND t.calendar_id = ANY($2)`,
		taskID, calIDs, day.Format(dateFmt), tz).Scan(&rrule, &anchor, &closed)
	// A task_id that is not a UUID fails the cast inside Postgres (22P02). To
	// the caller that is the same answer as a UUID nobody owns: no such task.
	if errors.Is(err, pgx.ErrNoRows) || pgErrCode(err, "22P02") {
		return ErrTaskNotFound
	}
	if err != nil {
		return err
	}
	if closed {
		return ErrNotOpen
	}
	if rrule != "" && (anchor == nil || !occursOn(rrule, *anchor, day)) {
		return ErrNotAnOccurrence
	}
	return nil
}

func promise(ctx context.Context, q querier, userID string, day time.Time, taskID string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO day_commitment (user_id, day, task_id) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day, task_id) DO UPDATE SET removed_at = NULL`,
		userID, day.Format(dateFmt), taskID)
	return err
}

// Add promises an OPEN task, visible to the caller, for today or a future
// day. Adding again what was removed brings it back.
func Add(ctx context.Context, userID string, day time.Time, taskID string) error {
	tz, _, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return err
	}
	if isPast(day, tz) {
		return ErrTooLate
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return err
	}
	if err := checkOpen(ctx, db.Pool, calIDs, day, tz, taskID); err != nil {
		return err
	}
	return promise(ctx, db.Pool, userID, day, taskID)
}

// Remove is spec §5: refused after noon today and on any past day.
func Remove(ctx context.Context, userID string, day time.Time, taskID string, now time.Time) error {
	tz, _, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return err
	}
	if !CanRemove(day, now, tz) {
		return ErrTooLate
	}
	_, err = db.Pool.Exec(ctx, `
		UPDATE day_commitment SET removed_at = NOW()
		 WHERE user_id = $1 AND day = $2 AND task_id = $3 AND removed_at IS NULL`,
		userID, day.Format(dateFmt), taskID)
	return err
}

// Confirm takes the day with these tasks. It only ADDS: removing goes through
// Remove and its noon rule, or «confirm» would be a way around it.
//
// One transaction: a refused task refuses the whole press, nothing is left
// half-written (review I2). A task already promised for the day is kept
// without the open check — «✅ Беру» on a proposal whose pinned task was done
// in the meantime takes the day instead of failing on the finished task.
// The day keeps the target it was taken with (review I1).
func Confirm(ctx context.Context, userID string, day time.Time, taskIDs []string) error {
	tz, target, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return err
	}
	if isPast(day, tz) {
		return ErrTooLate
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return err
	}

	kept := map[string]bool{}
	rows, err := db.Pool.Query(ctx, `
		SELECT task_id::text FROM day_commitment
		 WHERE user_id = $1 AND day = $2 AND removed_at IS NULL`,
		userID, day.Format(dateFmt))
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		kept[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, id := range taskIDs {
		if kept[id] {
			continue
		}
		if err := checkOpen(ctx, tx, calIDs, day, tz, id); err != nil {
			return err
		}
		if err := promise(ctx, tx, userID, day, id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO day_commitment_day (user_id, day, target) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, day) DO NOTHING`, userID, day.Format(dateFmt), target); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func pgErrCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func replaceAll(s string, pairs ...string) string {
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, pairs[i], pairs[i+1])
	}
	return s
}
