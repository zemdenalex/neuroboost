package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func (h *Handler) handleTasks(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.editOrSend(chatID, messageID,
			h.t(chatID, "⚠️ Не дозвонился до сервера. Попробуй через минуту.", "⚠️ Could not reach the server. Try again in a minute."), keyboards.HomeInline(h.lang(chatID)))
		return
	}
	tasks = openTasks(tasks)

	if len(tasks) == 0 {
		h.editOrSend(chatID, messageID, h.t(chatID, "📋 <b>Задач нет</b>", "📋 <b>No tasks</b>"), keyboards.TaskListEmpty(h.lang(chatID)))
		return
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Priority < tasks[j].Priority })

	// 🔴 Denis, 18.09: «if completed for the day it should stop being in the
	// task list, but appear somewhere gray, and on the next day it pops up
	// again». A repeating task answered today is not outstanding — but deleting
	// it from the screen would hide the fact that it exists at all, and the
	// count is the point: «✅ сегодня: 3» is the reward for the morning.
	open := make([]api.Task, 0, len(tasks))
	answered := 0
	for _, t := range tasks {
		if t.AnsweredToday() {
			answered++
			continue
		}
		open = append(open, t)
	}
	tasks = open

	if len(tasks) == 0 && answered > 0 {
		h.editOrSend(chatID, messageID,
			fmt.Sprintf(h.t(chatID,
				"📋 <b>На сегодня всё</b>\n\n✅ сегодня: %d",
				"📋 <b>Nothing left today</b>\n\n✅ done today: %d"), answered),
			keyboards.TaskListEmpty(h.lang(chatID)))
		return
	}

	text := fmt.Sprintf(h.t(chatID, "📋 <b>Задачи (%d)</b>\n\n", "📋 <b>Tasks (%d)</b>\n\n"), len(tasks))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, t := range tasks {
		if i >= 10 {
			break
		}
		label := fmt.Sprintf("%s %s", format.PriorityEmoji(t.Priority), t.Title)
		if len(label) > 40 {
			label = label[:37] + "..."
		}
		text += fmt.Sprintf("%s %s\n", format.PriorityEmoji(t.Priority), format.Escape(t.Title))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "task_action_"+t.ID),
		))
	}
	// З2, Denis 16.09: «в пункте меню задачи нет кнопки создать, надо
	// добавить». The list was a dead end — every other screen offers the thing
	// it is a list of.
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(h.t(chatID, "➕ Задача", "➕ Task"), "new_task"),
		tgbotapi.NewInlineKeyboardButtonData(h.t(chatID, "« Меню", "« Menu"), "main_menu"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.editOrSend(chatID, messageID, text, kb)
}

func (h *Handler) handleTaskAction(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не удалось загрузить задачу.", "❌ Could not load the task."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}

	var found bool
	var title string
	var priority, estMin int
	var dueDate string
	var repeats bool
	for _, t := range tasks {
		if t.ID == taskID {
			found = true
			title = t.Title
			priority = t.Priority
			estMin = t.EstimatedMinutes
			dueDate = t.DueDate
			repeats = t.Repeats()
			break
		}
	}

	if !found {
		h.editOrSend(chatID, messageID, h.t(chatID, "Задача не найдена.", "Task not found."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}

	text := fmt.Sprintf("%s <b>%s</b>\n", format.PriorityEmoji(priority), format.Escape(title))
	if estMin > 0 {
		text += fmt.Sprintf("⏱ %s\n", format.Duration(estMin))
	}
	if dueDate != "" {
		// Both halves of this line were English regardless of the chat's
		// language: the word «Due» was never in i18n.T at all, and the date
		// came out of Go's own «Mon, Jan 2». Denis's card on 21.09 read
		// «📅 Due: Tue, Sep 22» with everything around it in Russian.
		text += fmt.Sprintf(h.t(chatID, "📅 Срок: %s\n", "📅 Due: %s\n"),
			dayLabelISO(h.lang(chatID), dueDate, h.timezone(chatID)))
	}

	if repeats {
		text += h.t(chatID, "🔁 Повторяется\n", "🔁 Repeats\n")
	}

	h.editOrSend(chatID, messageID, text, keyboards.TaskActions(h.lang(chatID), taskID, repeats))
}

// handleTaskDone ticks a task off — for today if it is a series, for good if it
// is not.
//
// 🔴 The branch is the whole point. For a repeating task task.status describes
// the SERIES, so status=DONE means «больше никогда» — and until 20.09 that is
// exactly what «✅ Готово» sent. The first tick of «пить таблетки» would have
// ended it permanently. The endpoint that closes a single day had existed since
// 18.09 with no caller at all.
func (h *Handler) handleTaskDone(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)

	var err error
	var msg string
	if h.taskRepeats(chatID, taskID) {
		var res api.OccurrenceResult
		res, err = h.api.MarkOccurrence(us.AuthToken, taskID, "done", 0)
		msg = h.closedDayText(chatID, res.Occurrence)
	} else {
		err = h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"status": "DONE"})
		msg = h.t(chatID, "✅ Задача выполнена.", "✅ Task done.")
	}
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получилось: ", "❌ That didn't work: ")+h.errorText(chatID, err))
		return
	}
	h.sendText(chatID, msg)
	h.handleTasks(chatID, messageID)
}

// closedDayText names the day that was actually closed.
//
// 🔴 «Сделано на сегодня» used to be a constant, and on 21.09 that constant
// turned into a lie. A task created «позвонить в банк ЗАВТРА … каждый день»
// has no occurrence today; the press now closes the first day the series does
// have, and calling that «сегодня» would hide the one fact the reader needs.
//
// ⚠ The date is numeric on purpose. Go's day and month names are English
// whatever the chat's language — the same defect Denis saw as «Mon, Sep 21» —
// and 22.09 needs no translation.
func (h *Handler) closedDayText(chatID int64, occurrence string) string {
	loc, err := time.LoadLocation(h.timezone(chatID))
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc).Format("2006-01-02")

	if occurrence == "" || occurrence == today {
		return h.t(chatID,
			"✅ Сделано на сегодня. Завтра напомню снова.",
			"✅ Done for today. It comes back tomorrow.")
	}

	day, perr := time.Parse("2006-01-02", occurrence)
	if perr != nil {
		return h.t(chatID, "✅ Сделано.", "✅ Done.")
	}
	return fmt.Sprintf(h.t(chatID,
		"✅ Закрыл %s — это ближайший день серии. Сегодня её в списке нет.",
		"✅ Closed %s — the nearest day of the series. It isn't due today."),
		day.Format("02.01"))
}

// taskRepeats answers whether a task is a series.
//
// ⚠ Asked of the server rather than remembered from the card: the card may be
// minutes old, and a stale «Готово» must not end a series that was made
// repeating in the web meanwhile. A failed lookup answers false — the button
// then does what it has always done rather than silently changing meaning.
func (h *Handler) taskRepeats(chatID int64, taskID string) bool {
	us := h.store.GetOrCreate(chatID)
	tasks, err := h.api.GetTasks(us.AuthToken, "")
	if err != nil {
		return false
	}
	for _, t := range tasks {
		if t.ID == taskID {
			return t.Repeats()
		}
	}
	return false
}

// handleTaskPostpone offers the intervals Denis listed on 18.09.
func (h *Handler) handleTaskPostpone(chatID int64, messageID int, taskID string) {
	h.editOrSend(chatID, messageID, h.t(chatID,
		"⏰ <b>Отложить</b>\n\nНа сколько? Ритм не сдвинется — просто пропущу эти дни.",
		"⏰ <b>Postpone</b>\n\nFor how long? The rhythm stays; these days are just skipped."),
		keyboards.TaskPostpone(h.lang(chatID), taskID))
}

// handleTaskPostponeDays sends the interval as DAYS.
//
// 🔴 Never as a new rule. Denis chose it on 18.09: «Пропустить закрытые дни,
// ритм не трогать» — rewriting the rrule is precisely the «ритм сдвинулся» he
// ruled out, and it is the easy mistake here because it looks equivalent.
func (h *Handler) handleTaskPostponeDays(chatID int64, messageID int, taskID string, days int) {
	us := h.store.GetOrCreate(chatID)
	res, err := h.api.MarkOccurrence(us.AuthToken, taskID, "", days)
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получилось: ", "❌ That didn't work: ")+h.errorText(chatID, err))
		return
	}

	// 🔴 Report what was actually skipped, not what was asked for.
	//
	// The server walks the days and closes only those the series has. «Отложить
	// на неделю» on a task that comes back monthly may close nothing at all,
	// and saying «отложил на 7 дн.» there is the same shape of defect as the
	// card that promised to ask and did not: a printed claim with nothing
	// behind it.
	if res.Closed == 0 {
		h.sendText(chatID, fmt.Sprintf(h.t(chatID,
			"⏰ В ближайшие %d дн. у этой серии дней нет — пропускать нечего. Ритм не тронут.",
			"⏰ The series has no days in the next %d — nothing to skip. The rhythm is untouched."), days))
		h.handleTasks(chatID, messageID)
		return
	}
	h.sendText(chatID, fmt.Sprintf(h.t(chatID,
		"⏰ Отложил на %d дн. Пропущено дней серии: %d. Ритм не тронут.",
		"⏰ Postponed by %d day(s); %d day(s) of the series skipped. The rhythm is untouched."),
		days, res.Closed))
	h.handleTasks(chatID, messageID)
}

func (h *Handler) handleTaskDelete(chatID int64, messageID int, taskID string) {
	us := h.store.GetOrCreate(chatID)
	err := h.api.DeleteTask(us.AuthToken, taskID)
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получилось: ", "❌ That didn't work: ")+h.errorText(chatID, err))
		return
	}
	h.sendText(chatID, h.t(chatID, "🗑 Задача удалена.", "🗑 Task deleted."))
	h.handleTasks(chatID, messageID)
}

// Срок / Оценка / Теги — the three fields the card could show but never let
// you set. Due and estimate are button-driven, and reuse keyboards.TaskDue /
// TaskEstimate (which themselves reuse the wizard's own dueRow/estimateRow —
// see keyboards.go). Tags are free text: there is no closed set of values to
// offer, so "🏷 Теги" opens a normal FlowData text prompt, same shape as
// startNoteFlow / startNewTaskFlow.

var dueOffsets = map[string]bool{"0": true, "1": true, "7": true}
var estimateOptions = map[string]bool{"15": true, "30": true, "60": true, "120": true}

func (h *Handler) handleTaskDueMenu(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "📅 <b>%s</b>\n\nКогда срок?", "📅 <b>%s</b>\n\nWhen is it due?"), format.Escape(title)),
		keyboards.TaskDue(h.lang(chatID), taskID))
}

// handleTaskDueSet answers task_due_set_<uuid>_<offset>, cut from the right —
// same convention as parsePlanCallback in schedule.go, for the same reason: a
// task id is opaque to this bot and the offset is ours and known-shaped.
func (h *Handler) handleTaskDueSet(chatID int64, messageID int, data string) {
	idx := strings.LastIndex(data, "_")
	if idx < 0 {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял кнопку.", "Didn't understand that button."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	taskID, offsetStr := data[:idx], data[idx+1:]
	if taskID == "" || !dueOffsets[offsetStr] {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял кнопку.", "Didn't understand that button."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	offset, _ := strconv.Atoi(offsetStr)

	due := time.Now().In(h.location(chatID)).AddDate(0, 0, offset)
	us := h.store.GetOrCreate(chatID)
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{
		"due_date": due.Format(time.RFC3339),
	}); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

func (h *Handler) handleTaskEstimateMenu(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "⏱ <b>%s</b>\n\nСколько времени займёт?", "⏱ <b>%s</b>\n\nHow long will it take?"), format.Escape(title)),
		keyboards.TaskEstimate(h.lang(chatID), taskID))
}

// handleTaskEstimateSet answers task_est_set_<uuid>_<minutes>.
func (h *Handler) handleTaskEstimateSet(chatID int64, messageID int, data string) {
	idx := strings.LastIndex(data, "_")
	if idx < 0 {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял кнопку.", "Didn't understand that button."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	taskID, minStr := data[:idx], data[idx+1:]
	if taskID == "" || !estimateOptions[minStr] {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не понял кнопку.", "Didn't understand that button."), keyboards.BackToTasks(h.lang(chatID)))
		return
	}
	minutes, _ := strconv.Atoi(minStr)

	us := h.store.GetOrCreate(chatID)
	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{
		"estimated_minutes": minutes,
	}); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	h.handleTaskAction(chatID, messageID, taskID)
}

// handleTaskTagsPrompt answers task_tag_<uuid> — "🏷 Теги" on the card. It
// opens a text flow rather than a keyboard: tags are not a closed set.
func (h *Handler) handleTaskTagsPrompt(chatID int64, messageID int, taskID string) {
	title, ok := h.taskTitle(chatID, taskID)
	if !ok {
		return
	}
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "edit_task_tags"
	us.FlowStep = "text"
	us.FlowData["taskID"] = taskID
	h.editOrSend(chatID, messageID,
		fmt.Sprintf(h.t(chatID, "🏷 <b>%s</b>\n\nТеги через запятую (или «cancel»):", "🏷 <b>%s</b>\n\nTags, comma separated (or «cancel»):"), format.Escape(title)),
		keyboards.None())
}

// handleEditTaskTags is the text-flow answer to handleTaskTagsPrompt, routed
// from handleFlowInput (flows.go) by CurrentFlow == "edit_task_tags".
//
// The project rule is empty slices, never nil: typing "-" or a blank line
// clears the tags rather than leaving the field untouched, and it does so by
// sending an explicit [] — omitting the key here would mean "don't change
// this field", which is not what a user asking to clear their tags wants.
func (h *Handler) handleEditTaskTags(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	taskID, _ := us.FlowData["taskID"].(string)
	h.store.ClearFlow(chatID)
	if taskID == "" {
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Не помню, к какой задаче это относится.", "I've lost track of which task this is."), keyboards.HomeInline(h.lang(chatID)))
		return
	}

	tags := []string{}
	for _, tag := range strings.Split(text, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" && tag != "-" {
			tags = append(tags, tag)
		}
	}

	if err := h.api.UpdateTask(us.AuthToken, taskID, map[string]any{"tags": tags}); err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+h.errorText(chatID, err))
		return
	}
	h.sendText(chatID, h.t(chatID, "🏷 Теги обновлены", "🏷 Tags updated"))
	// 0: the tags arrived as a text message, so there is no screen of ours
	// under the user thumb to edit — the card is posted fresh.
	h.handleTaskAction(chatID, 0, taskID)
}
