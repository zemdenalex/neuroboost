package statgrid

import (
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
)

// EventSpans turns timed events into spans. All-day events are not «busy
// hours» — a birthday is not twenty-four booked hours — so they are counted
// apart (spec §1).
func EventSpans(events []api.Event) (spans []Span, allDay int) {
	for _, e := range events {
		if e.AllDay {
			allDay++
			continue
		}
		s, err1 := time.Parse(time.RFC3339, e.StartsAt)
		en, err2 := time.Parse(time.RFC3339, e.EndsAt)
		if err1 != nil || err2 != nil || !en.After(s) {
			continue
		}
		spans = append(spans, Span{Start: s, End: en})
	}
	return spans, allDay
}

// defaultTaskMinutes is what a closed task without an estimate is worth.
const defaultTaskMinutes = 30

// TaskMinutes is a task's time: logged, else estimated, else 30 minutes.
func TaskMinutes(t api.Task) int {
	switch {
	case t.ActualMinutes > 0:
		return t.ActualMinutes
	case t.EstimatedMinutes > 0:
		return t.EstimatedMinutes
	default:
		return defaultTaskMinutes
	}
}

// Point is time that belongs to one moment.
type Point struct {
	At  time.Time
	Dur time.Duration
}

// TaskPoints places done work in time: a closed one-off task at the moment it
// was closed, each «done» day of a series at that day's local midnight.
func TaskPoints(tasks []api.Task, occ []api.TaskOccurrence, loc *time.Location) []Point {
	byID := map[string]api.Task{}
	var pts []Point
	for _, t := range tasks {
		byID[t.ID] = t
		if t.Rrule != "" || t.Status != "DONE" || t.CompletedAt == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, t.CompletedAt)
		if err != nil {
			continue
		}
		pts = append(pts, Point{At: at, Dur: time.Duration(TaskMinutes(t)) * time.Minute})
	}
	for _, o := range occ {
		if o.State != "done" {
			continue
		}
		day, err := time.ParseInLocation("2006-01-02", o.Occurrence, loc)
		if err != nil {
			continue
		}
		pts = append(pts, Point{At: day, Dur: time.Duration(TaskMinutes(byID[o.TaskID])) * time.Minute})
	}
	return pts
}

// ReflectionDays is the set of the user's days with a reflection.
func ReflectionDays(refl []api.Reflection, loc *time.Location) map[string]bool {
	days := map[string]bool{}
	for _, r := range refl {
		at, err := time.Parse(time.RFC3339, r.CreatedAt)
		if err != nil {
			continue
		}
		days[at.In(loc).Format("2006-01-02")] = true
	}
	return days
}

// SumPoints is the time of the points inside the cell.
func SumPoints(points []Point, c Cell) time.Duration {
	var total time.Duration
	for _, p := range points {
		if !p.At.Before(c.From) && p.At.Before(c.To) {
			total += p.Dur
		}
	}
	return total
}
