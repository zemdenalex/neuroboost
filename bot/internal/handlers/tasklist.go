package handlers

import (
	"fmt"
	"strconv"
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

Напиши, что сделать. Например:

<code>позвонить в банк завтра 30м !1 #дела</code>
   → позвонить в банк · срок завтра · 30 мин · приоритет 1 · тег «дела»

<b>Слова-триггеры в задаче</b> — убираю их из названия:

<code>!1</code> … <code>!5</code> — приоритет: <code>!1</code> срочно, <code>!5</code> если получится
<code>30м</code> · <code>2ч</code> — сколько займёт
<code>завтра</code> · <code>16.09</code> — срок
<code>#тег</code> — тег
<code>задача</code> — слово-подсказка, что это задача, в названии не останется

<b>Списком</b> — несколько строк или через запятую:

<code>завтра помыться, поесть, поспать</code>
   → спрошу, одна это задача или три; день из строки достанется всем

⚠ Время суток (<code>15:00</code>) делает из задачи ещё и событие — покажу карточку события.`,
		`➕ <b>New task</b>

Write what needs doing. For example:

<code>позвонить в банк завтра 30м !1 #дела</code>
   → позвонить в банк · due tomorrow · 30 min · priority 1 · tag «дела»

<b>Trigger words in a task</b> — I take them out of the title:

<code>!1</code> … <code>!5</code> — priority: <code>!1</code> urgent, <code>!5</code> if possible
<code>30м</code> · <code>2ч</code> — how long it takes
<code>завтра</code> · <code>16.09</code> — the due date
<code>#тег</code> — a tag
<code>задача</code> — the word that says it is a task; it does not stay in the title

<b>As a list</b> — several lines, or commas:

<code>завтра помыться, поесть, поспать</code>
   → I will ask whether that is one task or three; the day goes to all of them

⚠ A clock time (<code>15:00</code>) makes it an event as well — I will show the event card.`)
}

func (h *Handler) startNewTaskFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_task"
	us.FlowStep = "title"
	us.FlowData = map[string]any{}
	h.sendHTMLWithKeyboard(chatID, taskGuideShort(h.lang(chatID)), keyboards.GuideMore(h.lang(chatID), "task"))
}

// taskGuideShort is the default task guide — see creationGuideShort for why.
func taskGuideShort(lang i18n.Lang) string {
	return i18n.T(lang,
		`➕ <b>Новая задача</b>

Напиши, что сделать, например:
<code>позвонить в банк завтра 30м !1</code>

Можно несколько строк сразу — списком.`,
		`➕ <b>New task</b>

Write what needs doing, for example:
<code>позвонить в банк завтра 30м !1</code>

Several lines at once work too — as a list.`)
}

// handleGuideFull swaps a short guide for the full one, in place. The flow is
// not touched: the user is still in the middle of writing.
func (h *Handler) handleGuideFull(chatID int64, messageID int, kind string) {
	lang := h.lang(chatID)
	text := creationGuide(lang)
	if kind == "task" {
		text = taskGuide(lang)
	}
	h.editOrSend(chatID, messageID, text, keyboards.None())
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
		us.FlowData["tasks"] = parse.ParseTaskList(raw, time.Now().In(h.location(chatID)))
		h.showTaskList(chatID, messageID)
		return true
	}

	tasks, ok := us.FlowData["tasks"].([]parse.TaskResult)
	if !ok {
		return false
	}
	switch {
	case data == "dr_makeall":
		h.createTaskList(chatID, messageID, tasks)
	case data == "dr_list":
		us.FlowStep = "list"
		h.showTaskList(chatID, messageID)
	case data == "dr_pick":
		h.editOrSend(chatID, messageID, h.t(chatID, "Какую задачу изменить?", "Which task to change?"), keyboards.ListPick(h.lang(chatID), len(tasks)))
	case strings.HasPrefix(data, "dr_item_"):
		i, valid := listIndex(strings.TrimPrefix(data, "dr_item_"), len(tasks))
		if !valid {
			h.showTaskList(chatID, messageID)
			return true
		}
		h.editOrSend(chatID, messageID, taskCardText(h.lang(chatID), tasks[i], h.timezone(chatID)), keyboards.TaskListItem(h.lang(chatID), i))
	case strings.HasPrefix(data, "dr_tdel_"):
		i, valid := listIndex(strings.TrimPrefix(data, "dr_tdel_"), len(tasks))
		if valid {
			tasks = append(tasks[:i:i], tasks[i+1:]...)
			us.FlowData["tasks"] = tasks
		}
		if len(tasks) == 0 {
			h.store.ClearFlow(chatID)
			h.editOrSend(chatID, messageID, h.t(chatID, "Список пуст — ничего не создано.", "The list is empty — nothing was created."), keyboards.BackToTasks(h.lang(chatID)))
			return true
		}
		h.showTaskList(chatID, messageID)
	case strings.HasPrefix(data, "dr_trew_"):
		i, valid := listIndex(strings.TrimPrefix(data, "dr_trew_"), len(tasks))
		if !valid {
			h.showTaskList(chatID, messageID)
			return true
		}
		us.FlowStep = "list:rewrite:" + strconv.Itoa(i)
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Напиши эту задачу заново — приоритет, оценка и срок читаются как обычно.",
			"Write this task again — priority, estimate and due date are read as usual."), keyboards.None())
	default:
		return false
	}
	return true
}

// listIndex reads the entry number a button carries and checks it still
// exists: a list can shrink while an older message still shows its buttons.
func listIndex(raw string, n int) (int, bool) {
	i, err := strconv.Atoi(raw)
	return i, err == nil && i >= 0 && i < n
}

// rewriteTaskListEntry replaces one entry with a freshly typed line, read with
// the task vocabulary — the same words a new task understands.
func (h *Handler) rewriteTaskListEntry(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	tasks, ok := us.FlowData["tasks"].([]parse.TaskResult)
	i, valid := listIndex(strings.TrimPrefix(us.FlowStep, "list:rewrite:"), len(tasks))
	if !ok || !valid {
		h.lostDraft(chatID)
		return
	}
	tasks[i] = parse.ParseTask(text, time.Now().In(h.location(chatID)))
	h.showTaskList(chatID, 0)
}

// showTaskList is the task list's confirmation card.
//
// 🔴 Denis, 16.09: «он не подтверждает а сразу создает, надо чтобы подтверждал
// как с событиями». Same shape as the event list — one card for the whole list,
// numbered, because «Изменить» asks for a number.
func (h *Handler) showTaskList(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	tasks, _ := us.FlowData["tasks"].([]parse.TaskResult)
	lang := h.lang(chatID)

	var b strings.Builder
	fmt.Fprintf(&b, i18n.T(lang, "📋 <b>Список задач: %d</b>\n", "📋 <b>Task list: %d</b>\n"), len(tasks))
	for i, t := range tasks {
		title := t.Title
		if title == "" {
			title = i18n.T(lang, "(без названия)", "(untitled)")
		}
		fmt.Fprintf(&b, "\n%d. ", i+1)
		if t.Priority != nil {
			b.WriteString(format.PriorityEmoji(*t.Priority) + " ")
		}
		b.WriteString(format.Escape(title))
		var meta []string
		if t.DueDate != nil {
			meta = append(meta, "📅 "+t.DueDate.Format("02.01"))
		}
		if t.EstimatedMinutes != nil {
			meta = append(meta, "⏱ "+format.Duration(*t.EstimatedMinutes))
		}
		if len(t.Tags) > 0 {
			meta = append(meta, "🏷 "+format.Escape(strings.Join(t.Tags, ", ")))
		}
		if len(meta) > 0 {
			b.WriteString("\n    " + strings.Join(meta, " · "))
		}
	}
	us.FlowStep = "list"
	h.editOrSend(chatID, messageID, b.String(), keyboards.ListCard(lang, len(tasks)))
}

// showTaskCard is the single-task path, unchanged in behaviour from before the
// list mode existed.
func (h *Handler) showTaskCard(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	r := parse.ParseTask(text, time.Now().In(h.location(chatID)))

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
	// The kind can still be changed when the card came from a typed line.
	raw, fromLine := us.FlowData["raw"].(string)
	h.sendHTMLWithKeyboard(chatID, taskCardText(h.lang(chatID), r, h.timezone(chatID)),
		keyboards.TaskCard(h.lang(chatID), fromLine && raw != ""))
}

// createTaskList writes one task per entry and names what did not make it.
//
// 🔴 Named, not counted. «Создано 5 из 8» leaves the reader to work out which
// three need doing again, which is the only part of the answer they needed.
func (h *Handler) createTaskList(chatID int64, messageID int, parsed []parse.TaskResult) {
	us := h.store.GetOrCreate(chatID)

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
