package daytasks

import (
	"context"
	"sort"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/recurrence"
)

type candidate struct {
	Item
	rank     int // 1..5, spec §4
	priority int
	created  time.Time
}

// Propose is spec §4 — what the morning offers, in order, up to N. It writes
// nothing: the day is taken only by Confirm.
func Propose(ctx context.Context, userID string, day time.Time) ([]Item, error) {
	tz, target, err := userZoneAndTarget(ctx, userID)
	if err != nil {
		return nil, err
	}
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	d := day.Format(dateFmt)
	y := day.AddDate(0, 0, -1).Format(dateFmt)

	rows, err := db.Pool.Query(ctx, `
		-- $1 calendars · $2 day · $3 zone · $4 user · $5 yesterday
		SELECT t.id::text, t.title, COALESCE(t.priority, 0), t.created_at,
		       COALESCE(t.rrule, ''), t.repeat_anchor,
		       (t.due_date AT TIME ZONE $3)::date <= $2::date AS due,
		       EXISTS (SELECT 1 FROM day_commitment c WHERE c.user_id = $4 AND c.task_id = t.id
		                 AND c.day = $2::date AND c.removed_at IS NULL) AS pinned,
		       EXISTS (SELECT 1 FROM day_commitment c WHERE c.user_id = $4 AND c.task_id = t.id
		                 AND c.day = $5::date AND c.removed_at IS NULL) AS yesterday
		  FROM task t
		 WHERE t.calendar_id = ANY($1)
		   AND NOT (`+replaceAll(doneOnDay, "$DAY", "$2::date", "$TZ", "$3")+`)
		   AND NOT (COALESCE(t.rrule, '') = '' AND t.status = 'DONE')`,
		calIDs, d, tz, userID, y)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cands []candidate
	for rows.Next() {
		var c candidate
		var rrule string
		var anchor *time.Time
		var due *bool
		var pinned, yesterday bool
		if err := rows.Scan(&c.TaskID, &c.Title, &c.priority, &c.created, &rrule, &anchor,
			&due, &pinned, &yesterday); err != nil {
			return nil, err
		}
		series := rrule != ""
		switch {
		case pinned:
			c.rank = 1
		case yesterday:
			c.rank = 2
		case !series && due != nil && *due:
			c.rank = 3
		case series && anchor != nil && occursOn(rrule, *anchor, day):
			c.rank = 4
		case series:
			continue // a series not due today is not today's task
		default:
			c.rank = 5
		}
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Priority: 1 is the most urgent … 5; 0 is the buffer and goes last (gotcha 4).
	pri := func(p int) int {
		if p == 0 {
			return 99
		}
		return p
	}
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if pri(a.priority) != pri(b.priority) {
			return pri(a.priority) < pri(b.priority)
		}
		return a.created.Before(b.created)
	})

	out := []Item{}
	for _, c := range cands {
		if len(out) == target {
			break
		}
		out = append(out, c.Item)
	}
	return out, nil
}

func occursOn(rrule string, anchor, day time.Time) bool {
	rule, err := recurrence.Parse(rrule)
	if err != nil {
		return false
	}
	return recurrence.Occurs(rule, anchor, day)
}
