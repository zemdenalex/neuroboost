package handlers

import (
	"fmt"
	"strings"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// taskCardText renders what was understood — and nothing else.
//
// A card that shows "Срок: —" for a task with no due date trains the reader to
// skip the line. Absent fields are absent.
func taskCardText(r parse.TaskResult, tz string) string {
	title := r.Title
	if title == "" {
		title = "(без названия)"
	}
	text := "➕ <b>Новая задача</b>\n"
	if r.Priority != nil {
		text += format.PriorityEmoji(*r.Priority) + " "
	}
	text += fmt.Sprintf("<b>%s</b>\n", format.Escape(title))

	var meta []string
	if r.DueDate != nil {
		meta = append(meta, "📅 "+r.DueDate.Format("02.01"))
	}
	if r.EstimatedMinutes != nil {
		meta = append(meta, "⏱ "+format.Duration(*r.EstimatedMinutes))
	}
	if len(r.Tags) > 0 {
		meta = append(meta, "🏷 "+format.Escape(strings.Join(r.Tags, ", ")))
	}
	if len(meta) > 0 {
		text += strings.Join(meta, " · ") + "\n"
	}
	return text
}
