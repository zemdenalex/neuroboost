package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// creationGuide is what «➕ Создать → Событие» opens with.
//
// 🔴 Denis, 15.09: «краткий гайд по созданию надо увеличить и сделать чуть
// более понятным - то есть примеры и как бот их поймет». The second half is the
// part that was missing: the old guide showed three inputs and never said what
// came out of them, so the only way to learn the vocabulary was to guess.
//
// ⚠ Every example here is held against the parser by
// TestGuideExamplesParseAsAdvertised. A guide that promises what the bot does
// not do is worse than no guide.
const creationGuide = `📅 <b>Новое событие</b>

Напиши одной строкой. Вот что я понимаю:

<code>Ужин завтра 19:00</code>
   → Ужин · завтра · 19:00–20:00

<code>среда 14:00-15:00 оркестр повтор</code>
   → оркестр · ср 16.09 · 14:00–15:00 · повтор — спрошу частоту

<code>анализы весь день четверг</code>
   → анализы · чт · весь день

<code>созвон работа синий 16.09 10:00</code>
   → созвон · календарь «Работа» · синий · 16.09 10:00

<b>Слова:</b> задача · весь день · повтор · ежедневно · цвета · #тег
<b>Дни:</b> завтра · среда · следующая среда · пн · wednesday · 16.09

Покажу, что понял, и спрошу подтверждение — создам только после него.`

func (h *Handler) startNewEventFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	us.FlowData = map[string]any{}
	h.sendHTML(chatID, creationGuide)
}

// startNewEventForDay begins the same flow with the day already chosen, so
// «18:00 Ужин» is enough.
func (h *Handler) startNewEventForDay(chatID int64, date string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	us.FlowData = map[string]any{"date": date}
	h.sendHTML(chatID, "📅 <b>Событие на "+format.Escape(date)+"</b>\n\nНапиши время и название — например «18:00 Ужин».")
}

// draftOf returns the draft this chat is building, creating none if there is
// none: every caller has to be able to tell "no draft" from "empty draft".
func draftOf(h *Handler, chatID int64) (*draftState, bool) {
	us := h.store.GetOrCreate(chatID)
	st, ok := us.FlowData["draft"].(*draftState)
	return st, ok
}

func (h *Handler) handleNewEventFlow(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	switch us.FlowStep {
	case "line":
		st := h.parseIntoDraft(chatID, text)
		if date, ok := us.FlowData["date"].(string); ok && date != "" && !st.D.HasDay {
			if day, err := time.ParseInLocation("2006-01-02", date, h.location()); err == nil {
				st.D.Day, st.D.HasDay = day, true
			}
		}
		us.FlowData["draft"] = &st
		h.showCard(chatID, 0)

	case "edit:title":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		// 🔴 Raw. No recogniser runs here — that is the whole point of the
		// branch: «изменить название» means the text becomes the title even
		// when it is made of keywords.
		st.Title = parse.ParseLineRaw(text)
		h.showCard(chatID, 0)

	case "edit:tags":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		st.D.Tags = splitTags(text)
		h.showCard(chatID, 0)

	case "edit:date":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		p := parse.ParseLine(text, time.Now().In(h.location()))
		if !p.Draft.HasDay {
			h.sendText(chatID, "Не понял дату. Например: «завтра», «среда», «16.09».")
			return
		}
		st.D.Day, st.D.HasDay = p.Draft.Day, true
		h.showCard(chatID, 0)

	case "edit:time":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		p := parse.ParseLine(text, time.Now().In(h.location()))
		if !p.Draft.HasTime {
			h.sendText(chatID, "Не понял время. Например: «14:00» или «14:00-15:30».")
			return
		}
		st.D.Start, st.D.End = p.Draft.Start, p.Draft.End
		st.D.HasTime, st.D.HasEnd = true, p.Draft.HasEnd
		st.D.AllDay = false
		h.showCard(chatID, 0)

	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.HomeInline())
	}
}

// parseIntoDraft runs the pipeline and then resolves a calendar name, which
// needs data the parser cannot have.
//
// ⚠ The title is recomputed AFTER the calendar pass. Recomputing before it
// would leave the calendar's name in the title — the exact leftover-fragment
// failure this rewrite exists to end.
func (h *Handler) parseIntoDraft(chatID int64, text string) draftState {
	us := h.store.GetOrCreate(chatID)
	p := parse.ParseLine(text, time.Now().In(h.location()))
	st := draftState{D: p.Draft}

	if cals, err := h.api.Calendars(us.AuthToken); err == nil && len(cals) > 0 {
		names := make([]string, len(cals))
		for i, c := range cals {
			names[i] = c.Name
		}
		if idx, ok := parse.RecogniseCalendar(p.Tokens, names, &st.D); ok {
			st.CalendarID, st.CalendarName = cals[idx].ID, cals[idx].Name
		}
	}

	st.Title = parse.Title(p.Tokens)
	return st
}

func (h *Handler) showCard(chatID int64, messageID int) {
	st, ok := draftOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return
	}
	us := h.store.GetOrCreate(chatID)
	us.FlowStep = "card"
	h.editOrSend(chatID, messageID, renderDraft(*st, time.Now().In(h.location())), keyboards.DraftCard())
}

func (h *Handler) lostDraft(chatID int64) {
	h.store.ClearFlow(chatID)
	h.sendHTMLWithKeyboard(chatID, "Черновик потерялся — начнём заново.", keyboards.HomeInline())
}

// handleDraftCallback answers every button of the confirmation card.
//
// Returns false when the callback is not one of ours, so HandleCallback can
// carry on to its own switch.
func (h *Handler) handleDraftCallback(chatID int64, messageID int, data string) bool {
	if !strings.HasPrefix(data, "dr_") && !strings.HasPrefix(data, "dre_") {
		return false
	}
	us := h.store.GetOrCreate(chatID)
	if us.CurrentFlow != "new_event" {
		// The card belongs to a flow that is over — most often because a menu
		// button interrupted it. Saying so beats editing a message the state no
		// longer backs.
		h.editOrSend(chatID, messageID, "Это создание уже закрыто.", keyboards.HomeInline())
		return true
	}
	st, ok := draftOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return true
	}

	switch {
	case data == "dr_ok":
		h.confirmDraft(chatID, messageID)

	case data == "dr_cancel":
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, "🗑 Отменено.", keyboards.HomeInline())

	case data == "dr_back":
		h.showCard(chatID, messageID)

	case data == "dr_edit":
		us.FlowStep = "edit:menu"
		h.editOrSend(chatID, messageID, "Что изменить?", keyboards.DraftEditMenu())

	case data == "dre_title":
		us.FlowStep = "edit:title"
		h.editOrSend(chatID, messageID,
			"Напиши название. Здесь оно берётся <b>как есть</b> — ключевые слова не действуют.", keyboards.DraftBack())

	case data == "dre_tags":
		us.FlowStep = "edit:tags"
		h.editOrSend(chatID, messageID,
			"Теги через запятую. Здесь любое слово — тег, даже «полдень».", keyboards.DraftBack())

	case data == "dre_date":
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, "Когда? Кнопкой или напиши: «среда», «16.09».", keyboards.DraftDay())

	case data == "dre_time":
		us.FlowStep = "edit:time"
		h.editOrSend(chatID, messageID, "Во сколько? Например «14:00» или «14:00-15:30».", keyboards.DraftBack())

	case data == "dre_repeat":
		us.FlowStep = "ask:freq"
		h.editOrSend(chatID, messageID, "Как часто повторять?", keyboards.FreqPicker())

	case data == "dre_colour":
		h.editOrSend(chatID, messageID, "Цвет события:", keyboards.ColourPicker())

	case data == "dre_cal":
		h.showCalendarPicker(chatID, messageID)

	case data == "dre_remind":
		h.editOrSend(chatID, messageID, "За сколько напомнить?", keyboards.ReminderPicker())

	case data == "dre_allday":
		st.D.AllDay = !st.D.AllDay
		if st.D.AllDay {
			st.D.HasTime, st.D.HasEnd = false, false
		}
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_freq_"):
		freq := strings.TrimPrefix(data, "dr_freq_")
		st.D.RepeatAsked = false
		if freq == "NONE" {
			st.D.Repeat = ""
		} else {
			st.D.Repeat = "FREQ=" + freq
		}
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_col_"):
		st.D.Colour = strings.TrimPrefix(data, "dr_col_")
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_cal_"):
		id := strings.TrimPrefix(data, "dr_cal_")
		st.CalendarID = id
		st.CalendarName = h.calendarName(chatID, id)
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_day_"):
		n, err := strconv.Atoi(strings.TrimPrefix(data, "dr_day_"))
		if err != nil {
			return true
		}
		now := time.Now().In(h.location())
		st.D.Day = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, n)
		st.D.HasDay = true
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_rem_"):
		mins := strings.TrimPrefix(data, "dr_rem_")
		h.setReminder(st, mins)
		h.showCard(chatID, messageID)

	default:
		return false
	}
	return true
}

// setReminder writes the reminder offsets, keeping "not stated" and "stated as
// none" apart.
//
// 🔴 They are different events. POST /api/events applies the user's default
// preset when reminder_offsets is ABSENT; an explicit empty array means stay
// silent forever. The nil pointer is "absent" and it is the default.
func (h *Handler) setReminder(st *draftState, mins string) {
	if mins == "none" {
		empty := []int{}
		st.ReminderOffsets = &empty
		return
	}
	if mins == "default" {
		st.ReminderOffsets = nil
		return
	}
	n, err := strconv.Atoi(mins)
	if err != nil {
		return
	}
	offsets := []int{n}
	st.ReminderOffsets = &offsets
}

func (h *Handler) showCalendarPicker(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	cals, err := h.api.Calendars(us.AuthToken)
	if err != nil || len(cals) == 0 {
		h.editOrSend(chatID, messageID, "Не удалось прочитать календари.", keyboards.DraftCard())
		return
	}
	names := make([]string, len(cals))
	ids := make([]string, len(cals))
	for i, c := range cals {
		names[i], ids[i] = c.Name, c.ID
	}
	h.editOrSend(chatID, messageID, "В какой календарь?", keyboards.CalendarPicker(names, ids))
}

func (h *Handler) calendarName(chatID int64, id string) string {
	us := h.store.GetOrCreate(chatID)
	cals, err := h.api.Calendars(us.AuthToken)
	if err != nil {
		return ""
	}
	for _, c := range cals {
		if c.ID == id {
			return c.Name
		}
	}
	return ""
}

// confirmDraft is the ✅ button: it asks for what is still missing, and creates
// only when nothing is.
func (h *Handler) confirmDraft(chatID int64, messageID int) {
	st, ok := draftOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return
	}
	us := h.store.GetOrCreate(chatID)

	switch nextQuestion(*st) {
	case askTitle:
		us.FlowStep = "edit:title"
		h.editOrSend(chatID, messageID, "Как назвать? Текст пойдёт в название как есть.", keyboards.DraftBack())
	case askFreq:
		us.FlowStep = "ask:freq"
		h.editOrSend(chatID, messageID, "Как часто повторять?", keyboards.FreqPicker())
	case askDate:
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, "На какой день?", keyboards.DraftDay())
	case askTime:
		us.FlowStep = "edit:time"
		h.editOrSend(chatID, messageID, "Во сколько? Например «14:00» или «14:00-15:30».", keyboards.DraftBack())
	default:
		h.createFromDraft(chatID, messageID, *st)
	}
}

// createFromDraft writes the draft to the API.
//
// 🔴 A draft marked «задача» becomes TWO objects: a task, and an event bound to
// it by task_id. If the second call fails the first one has already happened,
// and that is said out loud rather than compensated for — a silent rollback
// that can itself fail is worse than an honest sentence.
func (h *Handler) createFromDraft(chatID int64, messageID int, st draftState) {
	us := h.store.GetOrCreate(chatID)
	loc := h.location()

	var taskID *string
	if st.D.IsTask {
		task, err := h.api.CreateTask(us.AuthToken, api.CreateTaskReq{Title: st.Title, Status: "TODO", Tags: st.D.Tags})
		if err != nil {
			h.store.ClearFlow(chatID)
			h.editOrSend(chatID, messageID, "❌ Не удалось создать задачу: "+err.Error(), keyboards.HomeInline())
			return
		}
		taskID = &task.ID
	}

	start := st.D.StartsAt()
	end := start.Add(time.Hour)
	switch {
	case st.D.AllDay:
		start = st.D.Day
		end = st.D.Day.AddDate(0, 0, 1)
	case st.D.HasEnd:
		end = st.D.EndsAt()
	}

	req := api.CreateEventReq{
		Title:           st.Title,
		StartsAt:        start.UTC().Format(time.RFC3339),
		EndsAt:          end.UTC().Format(time.RFC3339),
		AllDay:          st.D.AllDay,
		Tags:            st.D.Tags,
		TaskID:          taskID,
		ReminderOffsets: st.ReminderOffsets,
	}
	if st.D.Repeat != "" {
		rrule := st.D.Repeat
		req.Rrule = &rrule
	}
	if st.D.Colour != "" {
		colour := st.D.Colour
		req.Colour = &colour
	}
	if st.CalendarID != "" {
		id := st.CalendarID
		req.CalendarID = &id
	}

	ev, err := h.api.CreateEvent(us.AuthToken, req)
	h.store.ClearFlow(chatID)
	if err != nil {
		msg := "❌ Не удалось создать событие: " + err.Error()
		if taskID != nil {
			msg += "\n\n⚠ Задача при этом создана — она в списке задач."
		}
		h.editOrSend(chatID, messageID, msg, keyboards.HomeInline())
		return
	}

	h.editOrSend(chatID, messageID, fmt.Sprintf("✅ <b>Создано</b>\n%s\n🕐 %s",
		format.Escape(ev.Title), humanRange(start.In(loc), end.In(loc), time.Now().In(loc))),
		keyboards.AgendaActions())
}
