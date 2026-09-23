package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

// Statistics — a screen you browse (Denis 20.09), filled by time (21.09),
// 24 hours by default (22.09). Spec: docs/superpowers/specs/2026-09-22-statistics-design.md.
//
// The arithmetic is statgrid's; this file only chooses what to measure and
// words it. The previous screen (one week, summed hours, «▪» per event) is
// gone: every number it printed is on this one, measured the way Denis asked.

// statsView is the whole state of the screen — it rides in the button.
type statsView struct {
	Period statgrid.Period
	Entity string // "a" all · "e" events · "t" tasks · "r" reflections
	Offset int
}

var statsPeriods = map[string]statgrid.Period{"w": statgrid.Week, "m": statgrid.Month, "y": statgrid.Year, "a": statgrid.All}

func periodCode(p statgrid.Period) string {
	for code, per := range statsPeriods {
		if per == p {
			return code
		}
	}
	return "w"
}

// maxStatsOffset bounds paging: ten years of weeks is not a screen anyone
// reaches by pressing ←, and a crafted button must not ask the API for 1900.
const maxStatsOffset = 120

// parseStatsView reads "w_a_-3". Callback data is user input: anything
// unknown is refused, not defaulted.
func parseStatsView(s string) (statsView, bool) {
	parts := strings.Split(s, "_")
	if len(parts) != 3 {
		return statsView{}, false
	}
	p, ok := statsPeriods[parts[0]]
	if !ok {
		return statsView{}, false
	}
	switch parts[1] {
	case "a", "e", "t", "r":
	default:
		return statsView{}, false
	}
	off, err := strconv.Atoi(parts[2])
	if err != nil {
		return statsView{}, false
	}
	if off > maxStatsOffset {
		off = maxStatsOffset
	}
	if off < -maxStatsOffset {
		off = -maxStatsOffset
	}
	if p == statgrid.All {
		off = 0
	}
	return statsView{Period: p, Entity: parts[1], Offset: off}, true
}

func (v statsView) code() string {
	return fmt.Sprintf("%s_%s_%d", periodCode(v.Period), v.Entity, v.Offset)
}

// Scale kinds (spec §3). The default is day24 — Denis 22.09: «the same for everyone».
const (
	scaleDay24 = "day24"
	scaleWork  = "work"
	scalePeak  = "peak"
)

func nextScale(kind string) string {
	switch kind {
	case scaleWork:
		return scalePeak
	case scalePeak:
		return scaleDay24
	default:
		return scaleWork
	}
}

func scaleLabel(lang i18n.Lang, kind string) string { return keyboards.StatsScaleName(lang, kind) }

// scaleKinds is the closed set a button may write; callback data is user input.
var scaleKinds = map[string]bool{scaleDay24: true, scaleWork: true, scalePeak: true}

// handleScalePick is the scale screen, from ⚙️ Settings or from onboarding
// (spec §3). kind == "" only shows it; a known kind is saved first.
func (h *Handler) handleScalePick(chatID int64, messageID int, kind string, onboarding bool) {
	origin := scaleFromSettings
	if onboarding {
		origin = scaleFromOnboarding
	}
	h.handleScalePickFrom(chatID, messageID, kind, origin)
}

// Where the scale picker was opened from: its buttons carry that place's
// prefix and its last button leads back there. One screen, three doors.
const (
	scaleFromSettings   = "settings"
	scaleFromOnboarding = "onboarding"
	scaleFromCalendar   = "calendar"
)

func (h *Handler) handleScalePickFrom(chatID int64, messageID int, kind, origin string) {
	us := h.store.GetOrCreate(chatID)
	lang := h.lang(chatID)
	if kind != "" {
		if !scaleKinds[kind] {
			return
		}
		if err := h.api.SetBotSetting(us.AuthToken, "stats_scale", kind); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return
		}
	}
	current, _ := h.api.BotSetting(us.AuthToken, "stats_scale")
	if current == "" {
		current = scaleDay24
	}
	start, end := h.workHours(chatID)
	ws, we := hourOf(start, 8), hourOf(end, 20)
	six := 6 * time.Hour
	text := fmt.Sprintf(h.t(chatID,
		"📏 <b>Шкала статистики</b>\n\nЧто считать полным столбиком. Например, день, где занято 6 часов:\n"+
			"• 24 ч → %s, одинаково для всех\n• рабочие часы (%02d–%02d) → %s\n• по максимуму → самый занятый день экрана = █\n\n"+
			"Поменять можно в любой момент: и здесь, и кнопкой 📏 на экране статистики.",
		"📏 <b>Statistics scale</b>\n\nWhat counts as a full bar. For example, a day with 6 hours booked:\n"+
			"• 24 h → %s, the same for everyone\n• work hours (%02d–%02d) → %s\n• busiest = full → the fullest day on screen is █\n\n"+
			"Change it any time: here, or with 📏 on the statistics screen."),
		statgrid.Glyph(statgrid.Level(six, 24*time.Hour)), ws, we,
		statgrid.Glyph(statgrid.Level(six, time.Duration(we-ws)*time.Hour)))

	prefix, backData, backLabel := "scl_", "settings_menu", h.t(chatID, "« Настройки", "« Settings")
	switch origin {
	case scaleFromOnboarding:
		prefix, backData, backLabel = "ob_sc_", "ob_finish", h.t(chatID, "Дальше →", "Next →")
	case scaleFromCalendar:
		prefix, backData, backLabel = "cal_sc_", "cal_open", h.t(chatID, "« Календарь", "« Calendar")
	}
	h.editOrSend(chatID, messageID, text, keyboards.StatsScale(lang, current, prefix, backData, backLabel))
}

// statsData is what the screen is computed from.
type statsData struct {
	Events []api.Event
	Tasks  []api.Task
	Occ    []api.TaskOccurrence
	Refl   []api.Reflection
}

// renderStatsScreen writes the screen. Pure: the handler fetches, this draws.
func renderStatsScreen(lang i18n.Lang, v statsView, g statgrid.Grid, d statsData,
	sc statgrid.Scale, loc *time.Location, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📊 <b>%s</b>\n", statsTitle(lang, g))

	spans, allDay := statgrid.EventSpans(d.Events)
	points := statgrid.TaskPoints(d.Tasks, d.Occ, loc)
	reflDays := statgrid.ReflectionDays(d.Refl, loc)
	whole := statgrid.Cell{From: g.From, To: g.To}

	busy := statgrid.Busy(spans, g.From, g.To)
	planned := statgrid.Planned(spans, g.From, g.To)
	taskTime := statgrid.SumPoints(points, whole)
	reflN := daysWith(reflDays, g.From, g.To)

	if busy == 0 && allDay == 0 && taskTime == 0 && reflN == 0 {
		b.WriteString("\n" + i18n.T(lang,
			"Пока пусто. Заведи событие или задачу, и здесь появятся числа.",
			"Nothing yet. Add an event or a task and the numbers will appear."))
		return b.String()
	}

	b.WriteString("\n")
	if v.Entity == "r" && g.Period == statgrid.Week {
		// A reflection has no length (spec §2): a tick per day, not an hour grid.
		var ticks []string
		for i, row := range g.Rows {
			mark := "·"
			if daysWith(reflDays, row.From, row.To) > 0 {
				mark = "✓"
			}
			ticks = append(ticks, weekdayShort(lang, i)+" "+mark)
		}
		b.WriteString(strings.Join(ticks, "  ") + "\n")
	} else {
		b.WriteString("<pre>" + statsGrid(lang, v, g, sc, spans, points, reflDays) + "</pre>")
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	upTo := g.To
	if tomorrow := today.AddDate(0, 0, 1); tomorrow.Before(upTo) {
		upTo = tomorrow
	}

	b.WriteString("\n")
	if v.Entity == "a" || v.Entity == "e" {
		fmt.Fprintf(&b, i18n.T(lang, "⏳ Занято: %s · В планах: %s\n", "⏳ Booked: %s · Planned: %s\n"),
			fmtHours(lang, busy), fmtHours(lang, planned))
		if allDay > 0 {
			fmt.Fprintf(&b, i18n.T(lang, "🗓 Весь день: %d\n", "🗓 All-day: %d\n"), allDay)
		}
		if n := repeatingSeries(d.Events); n > 0 {
			fmt.Fprintf(&b, i18n.T(lang, "🔁 Повторяющихся: %d\n", "🔁 Repeating: %d\n"), n)
		}
	}
	if v.Entity == "a" || v.Entity == "t" {
		b.WriteString(taskTotals(lang, d, g, today, upTo, loc))
	}
	if (v.Entity == "a" || v.Entity == "r") && upTo.After(g.From) {
		fmt.Fprintf(&b, i18n.T(lang, "📝 Рефлексии: %d из %d дней\n", "📝 Reflections: %d of %d days\n"),
			daysWith(reflDays, g.From, upTo), daysBetween(g.From, upTo))
	}
	return b.String()
}

// repeatingSeries counts the series behind the events: occurrences arrive as
// «uuid:date», and a daily series is one repeating thing, not seven.
func repeatingSeries(events []api.Event) int {
	seen := map[string]bool{}
	for _, e := range events {
		if e.Rrule == nil || *e.Rrule == "" {
			continue
		}
		parent, _, _ := splitInstanceID(e.ID)
		seen[parent] = true
	}
	return len(seen)
}

// statsGrid is the <pre> block: a header of columns and a line per row.
func statsGrid(lang i18n.Lang, v statsView, g statgrid.Grid, sc statgrid.Scale,
	spans []statgrid.Span, points []statgrid.Point, reflDays map[string]bool) string {

	short := g.Period == statgrid.Week // cells are hours
	var value func(statgrid.Cell) time.Duration
	switch v.Entity {
	case "t":
		value = func(c statgrid.Cell) time.Duration {
			if short {
				return statgrid.SumPointsTimed(points, c)
			}
			return statgrid.SumPoints(points, c)
		}
	case "r":
		value = nil
	default:
		value = func(c statgrid.Cell) time.Duration { return statgrid.Busy(spans, c.From, c.To) }
	}

	var levels [][]int
	if value != nil {
		levels = statgrid.Levels(g, sc, value)
	} else {
		// Reflections: the share of the cell's days that have one.
		levels = make([][]int, len(g.Rows))
		for r, row := range g.Rows {
			levels[r] = make([]int, len(row.Cells))
			for i, c := range row.Cells {
				if !c.Out {
					levels[r][i] = statgrid.Level(time.Duration(daysWith(reflDays, c.From, c.To)),
						time.Duration(daysBetween(c.From, c.To)))
				}
			}
		}
	}

	rowTotal := func(row statgrid.Row) string {
		switch v.Entity {
		case "t":
			return fmtHours(lang, statgrid.SumPoints(points, statgrid.Cell{From: row.From, To: row.To}))
		case "r":
			if n := daysWith(reflDays, row.From, row.To); n > 0 {
				return fmt.Sprintf(i18n.T(lang, "%d дн", "%d d"), n)
			}
			return "—"
		default:
			return fmtHours(lang, statgrid.Busy(spans, row.From, row.To))
		}
	}

	var b strings.Builder
	switch g.Period {
	case statgrid.Week:
		hdr := []rune(strings.Repeat(" ", len(g.ColHours)))
		for i, h := range g.ColHours {
			if i%6 == 0 && i+1 < len(hdr) {
				s := fmt.Sprintf("%02d", h)
				hdr[i], hdr[i+1] = rune(s[0]), rune(s[1])
			}
		}
		b.WriteString("   " + strings.TrimRight(string(hdr), " ") + "\n")
		for r, row := range g.Rows {
			fmt.Fprintf(&b, "%s %s %s\n", weekdayShort(lang, r), glyphRow(levels[r], row.Cells, ""), rowTotal(row))
		}
	case statgrid.Month:
		var hdr []string
		for i := 0; i < 7; i++ {
			hdr = append(hdr, weekdayShort(lang, i))
		}
		b.WriteString("   " + strings.Join(hdr, " ") + "\n")
		for r, row := range g.Rows {
			fmt.Fprintf(&b, "%2d %s %s\n", row.Label, glyphRow(levels[r], row.Cells, "  "), rowTotal(row))
		}
	case statgrid.Year:
		b.WriteString("    1  2  3  4  5  6\n")
		for r, row := range g.Rows {
			line := glyphRow(levels[r], row.Cells, "  ")
			line += strings.Repeat("   ", 6-len(row.Cells))
			fmt.Fprintf(&b, "%s %s %s\n", monthShort(lang, time.Month(row.Label)), line, rowTotal(row))
		}
	case statgrid.All:
		var hdr []string
		for m := 1; m <= 12; m++ {
			r, _ := utf8.DecodeRuneInString(monthShort(lang, time.Month(m)))
			hdr = append(hdr, string(r))
		}
		b.WriteString("     " + strings.Join(hdr, " ") + "\n")
		for r, row := range g.Rows {
			fmt.Fprintf(&b, "%d %s %s\n", row.Label, glyphRow(levels[r], row.Cells, " "), rowTotal(row))
		}
	}
	return b.String()
}

// glyphRow draws one row's cells; a cell outside the period is blank.
func glyphRow(levels []int, cells []statgrid.Cell, sep string) string {
	var b strings.Builder
	for i, c := range cells {
		if c.Out {
			b.WriteString(" ")
		} else {
			b.WriteString(statgrid.Glyph(levels[i]))
		}
		if i < len(cells)-1 {
			b.WriteString(sep)
		}
	}
	return b.String()
}

// taskTotals is the tasks' line (spec §2).
func taskTotals(lang i18n.Lang, d statsData, g statgrid.Grid, today, upTo time.Time, loc *time.Location) string {
	var closed, open, overdue, seriesDue, seriesDone int
	series := map[string]bool{}
	for _, t := range d.Tasks {
		switch {
		case t.Status == "DONE":
			if t.Rrule != "" || t.CompletedAt == "" {
				continue
			}
			if at, err := time.Parse(time.RFC3339, t.CompletedAt); err == nil && !at.Before(g.From) && at.Before(g.To) {
				closed++
			}
		case t.Status == "CANCELLED":
		default:
			if t.Rrule != "" {
				series[t.ID] = true
				if upTo.After(g.From) {
					seriesDue += statgrid.SeriesDays(t.Rrule, t.RepeatAnchor, g.From, upTo, loc)
				}
				continue
			}
			open++
			if due, err := time.Parse(time.RFC3339, t.DueDate); err == nil && due.In(loc).Before(today) {
				overdue++
			}
		}
	}
	for _, o := range d.Occ {
		if o.State != "done" || !series[o.TaskID] {
			continue
		}
		if day, err := time.ParseInLocation("2006-01-02", o.Occurrence, loc); err == nil && !day.Before(g.From) && day.Before(upTo) {
			seriesDone++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, i18n.T(lang, "✅ Закрыто: %d · 📋 Открыто: %d", "✅ Done: %d · 📋 Open: %d"), closed, open)
	if overdue > 0 {
		fmt.Fprintf(&b, i18n.T(lang, " · 🔴 Просрочено: %d", " · 🔴 Overdue: %d"), overdue)
	}
	b.WriteString("\n")
	if seriesDue > 0 {
		fmt.Fprintf(&b, i18n.T(lang, "🔁 Серии: сделано %d из %d\n", "🔁 Series: done %d of %d\n"), seriesDone, seriesDue)
	}
	return b.String()
}

func daysWith(days map[string]bool, from, to time.Time) int {
	n := 0
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		if days[d.Format("2006-01-02")] {
			n++
		}
	}
	return n
}

func daysBetween(from, to time.Time) int {
	n := 0
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		n++
	}
	return n
}

// fmtHours prints «5,5 ч» / «5.5 h»; nothing is «—».
func fmtHours(lang i18n.Lang, d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	h := math.Round(d.Hours()*10) / 10
	s := strconv.FormatFloat(h, 'f', -1, 64)
	if lang == i18n.RU {
		s = strings.ReplaceAll(s, ".", ",")
	}
	return s + i18n.T(lang, " ч", " h")
}

func statsTitle(lang i18n.Lang, g statgrid.Grid) string {
	switch g.Period {
	case statgrid.Month:
		return fmt.Sprintf("%s %d", monthNominative(lang, g.From.Month()), g.From.Year())
	case statgrid.Year:
		return strconv.Itoa(g.From.Year())
	case statgrid.All:
		return i18n.T(lang, "Всё время", "All time")
	default:
		last := g.To.AddDate(0, 0, -1)
		if last.Month() == g.From.Month() {
			return fmt.Sprintf(i18n.T(lang, "Неделя %d–%d %s", "Week %d–%d %s"),
				g.From.Day(), last.Day(), monthGenitive(lang, last.Month()))
		}
		return fmt.Sprintf(i18n.T(lang, "Неделя %d %s – %d %s", "Week %d %s – %d %s"),
			g.From.Day(), monthGenitive(lang, g.From.Month()), last.Day(), monthGenitive(lang, last.Month()))
	}
}

// monthShort is the first three letters, lower case — the year grid's rows.
func monthShort(lang i18n.Lang, m time.Month) string {
	r := []rune(strings.ToLower(monthNominative(lang, m)))
	if len(r) > 3 {
		r = r[:3]
	}
	return string(r)
}

// handleStatsView fetches what the view needs and draws it.
func (h *Handler) handleStatsView(chatID int64, messageID int, v statsView) {
	lang := h.lang(chatID)
	loc := h.location(chatID)
	us := h.store.GetOrCreate(chatID)
	now := time.Now().In(loc)

	sc := h.statsScale(chatID)
	kind := sc.Kind

	fetch := func(from, to time.Time) ([]api.Event, error) {
		return h.api.GetEvents(us.AuthToken, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	}

	var events []api.Event
	firstYear := now.Year()
	if v.Period == statgrid.All {
		// From the earliest year that has anything, walking back while years
		// have events — at most five (spec §6).
		for y := now.Year(); y >= now.Year()-5; y-- {
			start := time.Date(y, 1, 1, 0, 0, 0, 0, loc)
			ev, err := fetch(start, start.AddDate(1, 0, 0))
			if err != nil {
				if y == now.Year() {
					h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не получилось: ", "❌ Did not work: ")+h.errorText(chatID, err),
						keyboards.BackToMenu(lang))
					return
				}
				break
			}
			if len(ev) == 0 && y < now.Year() {
				break
			}
			events = append(events, ev...)
			firstYear = y
		}
	}
	g := statgrid.Layout(v.Period, now, v.Offset, loc, sc, firstYear)
	if v.Period != statgrid.All {
		ev, err := fetch(g.From, g.To)
		if err != nil {
			h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не получилось: ", "❌ Did not work: ")+h.errorText(chatID, err),
				keyboards.BackToMenu(lang))
			return
		}
		events = ev
	}

	// The rest is best effort: totals without them beat no screen at all.
	tasks, _ := h.api.GetTasks(us.AuthToken, "")
	var occ []api.TaskOccurrence
	for from := g.From; from.Before(g.To); from = from.AddDate(1, 0, 0) {
		to := from.AddDate(1, 0, 0)
		if g.To.Before(to) {
			to = g.To
		}
		// The API range is inclusive and at most 366 days; ask year by year.
		part, err := h.api.TaskOccurrences(us.AuthToken, from.Format("2006-01-02"), to.AddDate(0, 0, -1).Format("2006-01-02"))
		if err != nil {
			break
		}
		occ = append(occ, part...)
	}
	refl, _ := h.api.Reflections(us.AuthToken)

	text := renderStatsScreen(lang, v, g, statsData{Events: events, Tasks: tasks, Occ: occ, Refl: refl}, sc, loc, now)
	h.editOrSend(chatID, messageID, text,
		keyboards.StatsNav(lang, periodCode(v.Period), v.Entity, v.Offset, scaleLabel(lang, kind)))
}

// statsScale is the user's scale — the one statistics and the month calendar
// share, so one day never looks different on the two screens (spec §4).
func (h *Handler) statsScale(chatID int64) statgrid.Scale {
	us := h.store.GetOrCreate(chatID)
	kind, _ := h.api.BotSetting(us.AuthToken, "stats_scale")
	if !scaleKinds[kind] {
		kind = scaleDay24
	}
	sc := statgrid.Scale{Kind: kind}
	if kind == scaleWork {
		start, end := h.workHours(chatID)
		sc.WorkStart, sc.WorkEnd = hourOf(start, 8), hourOf(end, 20)
	}
	return sc
}

// handleStatsScale moves to the next scale, remembers it, and redraws.
func (h *Handler) handleStatsScale(chatID int64, messageID int, v statsView) {
	us := h.store.GetOrCreate(chatID)
	kind, _ := h.api.BotSetting(us.AuthToken, "stats_scale")
	if err := h.api.SetBotSetting(us.AuthToken, "stats_scale", nextScale(kind)); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
	}
	h.handleStatsView(chatID, messageID, v)
}

// hourOf reads the hour of «08:00».
func hourOf(hhmm string, fallback int) int {
	if len(hhmm) >= 2 {
		if n, err := strconv.Atoi(hhmm[:2]); err == nil && n >= 0 && n <= 24 {
			return n
		}
	}
	return fallback
}
