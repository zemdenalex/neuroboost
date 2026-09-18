package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Статистика — по-настоящему.
//
// 🔴 До 18.09 эта кнопка отвечала «Скоро — здесь будут тренды по твоей неделе»
// (today.go:71). Она существовала с самого начала и не показала ни одного числа
// ни разу. Денис: «статистика не работает» — и он прав, она никогда и не
// работала.
//
// Считается в БОТЕ из событий и задач, которые он и так умеет читать: своего
// эндпоинта у статистики нет, а заводить его значило бы ждать релиза API ради
// экрана, который можно собрать из готовых данных.

// StatsWeek is everything the screen shows, as plain numbers.
type StatsWeek struct {
	Events      int
	Hours       float64
	AllDay      int
	Repeating   int
	TasksDone   int
	TasksOpen   int
	TasksOverdue int
	// PerDay holds seven counts, Monday first — the ISO week this product uses
	// everywhere else.
	PerDay [7]int
	// Busiest is the weekday index of the fullest day, or -1 when the week is
	// empty.
	Busiest int
}

// BuildStats turns a week of events and the task list into the numbers shown.
//
// A pure function of its inputs and the clock, so the arithmetic can be tested
// without a Telegram or an API — the same reason renderDraft and dayBounds are
// pure. The bug this screen replaces survived because there was nothing to test.
func BuildStats(events []api.Event, tasks []api.Task, now time.Time, loc *time.Location) StatsWeek {
	if loc == nil {
		loc = time.UTC
	}
	var s StatsWeek
	s.Busiest = -1

	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	// ISO week: Monday is day 0.
	weekStart := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	weekEnd := weekStart.AddDate(0, 0, 7)

	for _, e := range events {
		start, err := time.Parse(time.RFC3339, e.StartsAt)
		if err != nil {
			continue
		}
		startLocal := start.In(loc)
		if startLocal.Before(weekStart) || !startLocal.Before(weekEnd) {
			continue
		}

		s.Events++
		if e.Rrule != nil && *e.Rrule != "" {
			s.Repeating++
		}
		if e.AllDay {
			s.AllDay++
		} else if end, perr := time.Parse(time.RFC3339, e.EndsAt); perr == nil && end.After(start) {
			s.Hours += end.Sub(start).Hours()
		}

		day := int(startLocal.Sub(weekStart).Hours() / 24)
		if day >= 0 && day < 7 {
			s.PerDay[day]++
		}
	}

	for i, n := range s.PerDay {
		if s.Busiest < 0 || n > s.PerDay[s.Busiest] {
			if n > 0 {
				s.Busiest = i
			}
		}
	}

	for _, t := range tasks {
		switch {
		case t.Status == "DONE":
			// Only this week's closures: "закрыто за неделю" is the number that
			// answers "did the week go anywhere", and an all-time total does not.
			if t.CompletedAt == "" {
				continue
			}
			done, err := time.Parse(time.RFC3339, t.CompletedAt)
			if err != nil {
				continue
			}
			if d := done.In(loc); !d.Before(weekStart) && d.Before(weekEnd) {
				s.TasksDone++
			}
		case t.Status == "CANCELLED":
			// Neither open nor an achievement.
		default:
			s.TasksOpen++
			if t.DueDate == "" {
				continue
			}
			due, err := time.Parse(time.RFC3339, t.DueDate)
			if err != nil {
				continue
			}
			// 🔴 Overdue is measured against the START of today in the user's
			// zone, not against now: a task due today is not late at 09:00.
			if due.In(loc).Before(today) {
				s.TasksOverdue++
			}
		}
	}

	return s
}

func (h *Handler) handleStats(chatID int64, messageID int) {
	lang := h.lang(chatID)
	loc := h.location(chatID)
	us := h.store.GetOrCreate(chatID)

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	weekStart := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	weekEnd := weekStart.AddDate(0, 0, 7)

	events, err := h.api.GetEvents(us.AuthToken, weekStart.UTC().Format(time.RFC3339), weekEnd.UTC().Format(time.RFC3339))
	if err != nil {
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"⚠️ Не дозвонился до сервера. Попробуй через минуту.",
			"⚠️ Could not reach the server. Try again in a minute."), keyboards.BackToMenu(lang))
		return
	}
	// Every task, not just the open ones: the closed ones are the point.
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		tasks = nil
	}

	s := BuildStats(events, tasks, now, loc)
	h.editOrSend(chatID, messageID, renderStats(lang, s, weekStart), keyboards.BackToMenu(lang))
}

// renderStats writes the screen.
func renderStats(lang i18n.Lang, s StatsWeek, weekStart time.Time) string {
	var b strings.Builder

	fmt.Fprintf(&b, i18n.T(lang, "📊 <b>Неделя с %s</b>\n\n", "📊 <b>Week of %s</b>\n\n"),
		weekStart.Format("02.01"))

	if s.Events == 0 && s.TasksOpen == 0 && s.TasksDone == 0 {
		b.WriteString(i18n.T(lang,
			"Пока пусто. Заведи событие или задачу — здесь появятся числа.",
			"Nothing yet. Add an event or a task and the numbers will appear."))
		return b.String()
	}

	b.WriteString(fieldLine("📅", i18n.T(lang, "Событий:", "Events:"), fmt.Sprintf("%d", s.Events)))
	if s.Hours > 0 {
		b.WriteString(fieldLine("⏳", i18n.T(lang, "Занято:", "Booked:"),
			fmt.Sprintf(i18n.T(lang, "%.1f ч", "%.1f h"), s.Hours)))
	}
	if s.AllDay > 0 {
		b.WriteString(fieldLine("🗓", i18n.T(lang, "Весь день:", "All-day:"), fmt.Sprintf("%d", s.AllDay)))
	}
	if s.Repeating > 0 {
		b.WriteString(fieldLine("🔁", i18n.T(lang, "Повторяющихся:", "Repeating:"), fmt.Sprintf("%d", s.Repeating)))
	}

	b.WriteString("\n")
	b.WriteString(fieldLine("✅", i18n.T(lang, "Закрыто задач:", "Tasks done:"), fmt.Sprintf("%d", s.TasksDone)))
	b.WriteString(fieldLine("📋", i18n.T(lang, "Открыто:", "Open:"), fmt.Sprintf("%d", s.TasksOpen)))
	if s.TasksOverdue > 0 {
		b.WriteString(fieldLine("🔴", i18n.T(lang, "Просрочено:", "Overdue:"), fmt.Sprintf("%d", s.TasksOverdue)))
	}

	// A bar per day, so the shape of the week is visible without reading.
	b.WriteString("\n")
	names := [7]string{"пн", "вт", "ср", "чт", "пт", "сб", "вс"}
	namesEN := [7]string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	for i := 0; i < 7; i++ {
		name := i18n.T(lang, names[i], namesEN[i])
		bar := strings.Repeat("▪", min(s.PerDay[i], 10))
		if s.PerDay[i] == 0 {
			bar = "·"
		}
		fmt.Fprintf(&b, "%s %s %d\n", name, format.Escape(bar), s.PerDay[i])
	}

	if s.Busiest >= 0 {
		b.WriteString("\n")
		fmt.Fprintf(&b, i18n.T(lang, "Плотнее всего — %s.", "Busiest day — %s."),
			i18n.T(lang, names[s.Busiest], namesEN[s.Busiest]))
	}

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
