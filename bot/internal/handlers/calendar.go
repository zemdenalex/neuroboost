package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// The month calendar, restored.
//
// v0.2.1 had it — a 6×7 grid with ⬅️/➡️ paging and a day view behind each cell
// (index.mjs:707, 750). The Go rewrite dropped it, and this file then said "a
// month view in a chat is still not the plan", which turned an accident into a
// decision nobody had made. Denis overturned that on 18.08: вернуть.
//
// Two things are new relative to v0.2.1, both because the data is now here to
// support them: a day carrying events is marked, so the grid answers "when am I
// busy" without 30 taps, and paging EDITS the message instead of sending a new
// one — v0.2.1 edited too, and the Go bot had no way to.

// dayCell is what one square knows about itself.
type dayCell struct {
	Date      time.Time
	InMonth   bool
	IsToday   bool
	HasEvents bool
}

// buildMonth lays out the six weeks a month view always shows.
//
// Always six rows, never five: a grid that changes height between months makes
// the navigation buttons move under the user's thumb between taps. ISO weeks,
// so the first column is Monday — the same rule the web grid follows.
func buildMonth(year int, month time.Month, today time.Time, busy map[string]bool, loc *time.Location) []dayCell {
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	// Go's Weekday is Sunday=0; ISO wants Monday=0.
	offset := (int(first.Weekday()) + 6) % 7
	start := first.AddDate(0, 0, -offset)

	todayKey := today.In(loc).Format("2006-01-02")
	cells := make([]dayCell, 0, 42)
	for i := 0; i < 42; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		cells = append(cells, dayCell{
			Date:      d,
			InMonth:   d.Month() == month && d.Year() == year,
			IsToday:   key == todayKey,
			HasEvents: busy[key],
		})
	}
	return cells
}

// cellLabel renders one square. Kept apart from the layout so the marks can be
// asserted directly: "today is marked" and "a busy day is marked" are the two
// things a user reads the grid for, and both are one character.
func cellLabel(c dayCell) string {
	n := strconv.Itoa(c.Date.Day())
	switch {
	case c.IsToday:
		return "🔸" + n
	case !c.InMonth:
		return "·" + n
	case c.HasEvents:
		return n + "•"
	default:
		return n
	}
}

// handleCalendar answers 🗓 Calendar and every paging tap.
//
// messageID > 0 means edit in place; 0 means send a new message. Paging that
// appends a new grid every tap buries the chat within a minute.
func (h *Handler) handleCalendar(chatID int64, messageID int, date time.Time) {
	loc := h.location()
	d := date.In(loc)
	h.showMonth(chatID, messageID, d.Year(), d.Month())
}

func (h *Handler) showMonth(chatID int64, messageID, year int, month time.Month) {
	loc := h.location()
	now := time.Now().In(loc)

	// The query spans the whole visible grid, not the calendar month: the first
	// and last rows show days from the neighbouring months, and leaving them
	// unmarked would say "nothing on the 31st" when something is.
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	offset := (int(first.Weekday()) + 6) % 7
	gridStart := first.AddDate(0, 0, -offset)
	gridEnd := gridStart.AddDate(0, 0, 42)

	busy := map[string]bool{}
	us := h.store.GetOrCreate(chatID)
	events, err := h.api.GetEvents(us.AuthToken,
		gridStart.UTC().Format(time.RFC3339), gridEnd.UTC().Format(time.RFC3339))
	if err == nil {
		for _, e := range events {
			if t, perr := time.Parse(time.RFC3339, e.StartsAt); perr == nil {
				busy[t.In(loc).Format("2006-01-02")] = true
			}
		}
	}
	// A failed load draws the grid without marks rather than an error screen:
	// the month is still navigable, and the marks are an aid, not the content.

	cells := buildMonth(year, month, now, busy, loc)
	labels := make([]string, len(cells))
	dates := make([]string, len(cells))
	for i, c := range cells {
		labels[i] = cellLabel(c)
		dates[i] = c.Date.Format("2006-01-02")
	}

	text := fmt.Sprintf("🗓 <b>%s %d</b>\n\nВыбери день. 🔸 сегодня · • есть события",
		monthNominative(month), year)
	kb := keyboards.MonthGrid(year, int(month), monthNominative(month), labels, dates,
		time.Now().In(loc).Format("2006-01-02"))

	h.editOrSend(chatID, messageID, text, kb)
}

// handleCalendarNav steps a month at a time. The current position travels in
// the callback data, so a grid left open for a week still pages from where it
// is showing rather than from wherever the bot last thought it was.
func (h *Handler) handleCalendarNav(chatID int64, messageID int, data string) {
	dir, rest, ok := strings.Cut(data, "_")
	if !ok || (dir != "prev" && dir != "next") {
		return
	}
	yearStr, monthStr, ok := strings.Cut(rest, "_")
	if !ok {
		return
	}
	year, err1 := strconv.Atoi(yearStr)
	month, err2 := strconv.Atoi(monthStr)
	if err1 != nil || err2 != nil || month < 1 || month > 12 || year < 1970 || year > 9999 {
		return
	}

	step := 1
	if dir == "prev" {
		step = -1
	}
	// AddDate on the first of the month, rather than arithmetic on the month
	// number: it carries the year for free and cannot produce month 0 or 13.
	shifted := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, h.location()).AddDate(0, step, 0)
	h.showMonth(chatID, messageID, shifted.Year(), shifted.Month())
}

// shiftDay moves an ISO date by whole days.
//
// Deliberately date-only arithmetic with time.UTC: the value comes from
// callback_data as "2026-08-18" and goes straight back into it. Doing this in a
// zone with DST would give 23- and 25-hour days, and "yesterday" could land on
// today again on the switchover — a bug that appears twice a year and is
// invisible the rest of the time.
func shiftDay(iso string, days int) (string, bool) {
	d, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return "", false
	}
	return d.AddDate(0, 0, days).Format("2006-01-02"), true
}

// handleCalendarDay lists one day. messageID > 0 means edit in place; paging
// day to day must not bury the chat under a new message per tap, same reason
// as the month grid.
func (h *Handler) handleCalendarDay(chatID int64, messageID int, date string) {
	loc := h.location()
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		h.editOrSend(chatID, messageID, "Не понял дату.", keyboards.HomeInline())
		return
	}

	prev, okPrev := shiftDay(date, -1)
	next, okNext := shiftDay(date, 1)
	if !okPrev || !okNext {
		// Unparseable date from callback_data: say so rather than render a day
		// that is not the one asked for.
		h.editOrSend(chatID, messageID, "Не понял дату.", keyboards.HomeInline())
		return
	}

	// Built in the user's zone and converted, not stamped as UTC. Midnight local
	// is 21:00 the previous day in UTC for Moscow, and a query that ignores that
	// returns the wrong 24 hours — three of yesterday's evening, missing three
	// of tonight's.
	from := day.UTC().Format(time.RFC3339)
	to := day.AddDate(0, 0, 1).UTC().Format(time.RFC3339)

	us := h.store.GetOrCreate(chatID)
	events, err := h.api.GetEvents(us.AuthToken, from, to)
	if err != nil {
		h.editOrSend(chatID, messageID, "❌ Не удалось загрузить день: "+err.Error(), keyboards.HomeInline())
		return
	}

	text := fmt.Sprintf("📅 <b>%d %s %d</b>\n\n", day.Day(), monthGenitive(day.Month()), day.Year())
	if len(events) == 0 {
		text += "Пусто."
	} else {
		sort.Slice(events, func(i, j int) bool { return events[i].StartsAt < events[j].StartsAt })
		for _, e := range events {
			text += fmt.Sprintf("🕐 %s — %s\n",
				format.FormatTime(e.StartsAt, h.cfg.Timezone), format.Escape(e.Title))
		}
	}

	h.editOrSend(chatID, messageID, text,
		keyboards.DayActions(date, prev, next, day.Year(), int(day.Month())))
}

// handleCalendarBack returns from a day to the month it was opened from.
//
// ⚠ It used to force a new message, and a comment in handler.go explained why:
// the day view was "its own message", and editing it would destroy what the
// user was reading. That stopped being true once the day view itself began
// editing in place — grid and day now share one message, and going back to the
// month is the same navigation step as paging it. Denis, 23.08: «нажатие „к
// месяцу“ пишет новое».
func (h *Handler) handleCalendarBack(chatID int64, messageID int, pos string) {
	yearStr, monthStr, ok := strings.Cut(pos, "_")
	if !ok {
		return
	}
	year, err1 := strconv.Atoi(yearStr)
	month, err2 := strconv.Atoi(monthStr)
	if err1 != nil || err2 != nil || month < 1 || month > 12 {
		return
	}
	h.showMonth(chatID, messageID, year, time.Month(month))
}

func monthNominative(m time.Month) string {
	names := [...]string{
		"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
		"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
	}
	return names[int(m)-1]
}
