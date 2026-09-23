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

	// The user's own promises for this day and the day before — a personal
	// table, read by user_id. Tasks below are read through calendars only, as
	// in List.
	pinnedIDs, yesterdayIDs := map[string]bool{}, map[string]bool{}
	prom, err := db.Pool.Query(ctx, `
		SELECT task_id::text, day = $2::date FROM day_commitment
		 WHERE user_id = $1 AND day IN ($2::date, $3::date) AND removed_at IS NULL`,
		userID, d, y)
	if err != nil {
		return nil, err
	}
	for prom.Next() {
		var id string
		var onDay bool
		if err := prom.Scan(&id, &onDay); err != nil {
			prom.Close()
			return nil, err
		}
		if onDay {
			pinnedIDs[id] = true
		} else {
			yesterdayIDs[id] = true
		}
	}
	prom.Close()
	if err := prom.Err(); err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, `
		-- $1 calendars · $2 day · $3 zone · $4 yesterday
		SELECT t.id::text, t.title, COALESCE(t.priority, 0), t.created_at,
		       COALESCE(t.rrule, ''), t.repeat_anchor,
		       (t.due_date AT TIME ZONE $3)::date <= $2::date AS due,
		       `+replaceAll(doneOnDay, "$DAY", "$4::date", "$TZ", "$3")+` AS done_yesterday
		  FROM task t
		 WHERE t.calendar_id = ANY($1)
		   AND NOT (`+replaceAll(doneOnDay, "$DAY", "$2::date", "$TZ", "$3")+`)
		   AND NOT (COALESCE(t.rrule, '') = '' AND t.status = 'DONE')
		   AND t.status <> 'CANCELLED'`,
		calIDs, d, tz, y)
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
		var doneYesterday bool
		if err := rows.Scan(&c.TaskID, &c.Title, &c.priority, &c.created, &rrule, &anchor,
			&due, &doneYesterday); err != nil {
			return nil, err
		}
		pinned, yesterday := pinnedIDs[c.TaskID], yesterdayIDs[c.TaskID]
		series := rrule != ""
		// A series not occurring on the day is not the day's task, whatever
		// else holds: pinned or carried, it could never be done there (I5).
		if series && (anchor == nil || !occursOn(rrule, *anchor, day)) {
			continue
		}
		switch {
		case pinned:
			c.rank = 1
		// «Yesterday's UNDONE» (spec §4.2): a series done yesterday is not
		// carried; it comes back by its own rank (review I5).
		case yesterday && !doneYesterday:
			c.rank = 2
		case !series && due != nil && *due:
			c.rank = 3
		case series:
			c.rank = 4
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
