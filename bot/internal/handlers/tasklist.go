package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// Tasks by the list, and the guide that says so.
//
// Denis, 15.09: «С задачами ты сделал тоже списки?» — no, the first pass put
// the list mode on events only, and a message of three lines became one task
// with a three-line title. It does now.
//
// ⚠ Tasks do NOT inherit a day from a header the way events do. An event block
// describes a schedule and its lines are positions in it; a task list is a list
// of things to do, and «среда» in the middle of one is far more likely to be
// part of what needs doing than a heading over the rest.

func taskGuide(lang i18n.Lang) string {
	return i18n.T(lang,
		`➕ <b>Новая задача</b>

Напиши название. Понимаю то же, что и в событиях:

<code>позвонить в банк завтра 30м !1 #дела</code>
   → позвонить в банк · завтра · 30 мин · приоритет 1 · тег «дела»

<b>Списком</b> — несколько строк сразу:

<code>1. Отжаться
2. Подтянуться
3. Присесть</code>
   → спрошу, одна это задача или три

Приоритет: <code>!1</code> срочно … <code>!5</code> если получится · Оценка: <code>30м</code>, <code>2ч</code>`,
		`➕ <b>New task</b>

Write the title. I understand the same words as for events:

<code>позвонить в банк завтра 30м !1 #дела</code>
   → позвонить в банк · tomorrow · 30 min · priority 1 · tag «дела»

<b>As a list</b> — several lines at once:

<code>1. Отжаться
2. Подтянуться
3. Присесть</code>
   → I will ask whether that is one task or three

Priority: <code>!1</code> urgent … <code>!5</code> if possible · Estimate: <code>30м</code>, <code>2ч</code>`)
}

func (h *Handler) startNewTaskFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_task"
	us.FlowStep = "title"
	us.FlowData = map[string]any{}
	h.sendHTML(chatID, taskGuide(h.lang(chatID)))
}

// handleTaskListCallback answers the one/many question for tasks.
//
// It exists next to handleListCallback rather than inside it because the two
// differ in what «many» means: events get a card with days and times to check,
// tasks get created. Folding them together would mean a flag that decides which
// half of the function runs, and that is two functions wearing one name.
func (h *Handler) handleTaskListCallback(chatID int64, messageID int, data string) bool {
	us := h.store.GetOrCreate(chatID)

	switch data {
	case "dr_one":
		raw, _ := us.FlowData["raw"].(string)
		h.showTaskCard(chatID, strings.ReplaceAll(raw, "\n", " "))
		return true

	case "dr_many":
		raw, _ := us.FlowData["raw"].(string)
		h.createTaskList(chatID, messageID, raw)
		return true
	}
	return false
}

// showTaskCard is the single-task path, unchanged in behaviour from before the
// list mode existed.
func (h *Handler) showTaskCard(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	r := parse.ParseTask(text, time.Now().In(h.location()))

	us.FlowData["title"] = r.Title
	if r.Priority != nil {
		us.FlowData["priority"] = *r.Priority
	}
	if r.EstimatedMinutes != nil {
		us.FlowData["minutes"] = *r.EstimatedMinutes
	}
	if r.DueDate != nil {
		us.FlowData["due"] = r.DueDate.Format(time.RFC3339)
	}
	if len(r.Tags) > 0 {
		us.FlowData["tags"] = r.Tags
	}
	us.FlowStep = "card"
	h.sendHTMLWithKeyboard(chatID, taskCardText(h.lang(chatID), r, h.cfg.Timezone), keyboards.TaskCard(h.lang(chatID)))
}

// createTaskList writes one task per entry and names what did not make it.
//
// 🔴 Named, not counted. «Создано 5 из 8» leaves the reader to work out which
// three need doing again, which is the only part of the answer they needed.
func (h *Handler) createTaskList(chatID int64, messageID int, raw string) {
	us := h.store.GetOrCreate(chatID)
	parsed := parse.ParseTaskList(raw, time.Now().In(h.location()))

	var made, failed []string
	for _, p := range parsed {
		if strings.TrimSpace(p.Title) == "" {
			continue
		}
		// Everything the task vocabulary understood, not just the title: a
		// priority or an estimate dropped here is a word the user typed and the
		// bot silently ignored.
		req := api.CreateTaskReq{Title: p.Title, Status: "TODO"}
		if p.Priority != nil {
			req.Priority = p.Priority
		}
		if p.EstimatedMinutes != nil {
			req.EstimatedMinutes = p.EstimatedMinutes
		}
		if p.DueDate != nil {
			due := p.DueDate.Format(time.RFC3339)
			req.DueDate = &due
		}
		if len(p.Tags) > 0 {
			req.Tags = p.Tags
		}
		if _, err := h.api.CreateTask(us.AuthToken, req); err != nil {
			failed = append(failed, format.Escape(p.Title)+" — "+format.Escape(err.Error()))
			continue
		}
		made = append(made, format.Escape(p.Title))
	}

	h.store.ClearFlow(chatID)

	var b strings.Builder
	if len(made) > 0 {
		fmt.Fprintf(&b, h.t(chatID, "✅ <b>Создано задач: %d</b>\n• %s\n", "✅ <b>Tasks created: %d</b>\n• %s\n"), len(made), strings.Join(made, "\n• "))
	}
	if len(failed) > 0 {
		b.WriteString(h.t(chatID, "\n❌ <b>Не создано</b>\n• ", "\n❌ <b>Not created</b>\n• ") + strings.Join(failed, "\n• "))
	}
	if b.Len() == 0 {
		b.WriteString(h.t(chatID, "Нечего создавать.", "Nothing to create."))
	}
	h.editOrSend(chatID, messageID, b.String(), keyboards.BackToTasks(h.lang(chatID)))
}
