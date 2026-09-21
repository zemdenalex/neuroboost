package events

import "time"

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
