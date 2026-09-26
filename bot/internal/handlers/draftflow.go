package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
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
func creationGuide(lang i18n.Lang) string {
	return i18n.T(lang,
		`📅 <b>Новое событие</b>

Напиши одной строкой. Вот что я понимаю:

<code>Ужин завтра 19:00</code>
   → Ужин · завтра · 19:00–20:00

<code>среда 14:00-15:00 оркестр повтор</code>
   → оркестр · ср · 14:00–15:00 · повтор, спрошу частоту

<code>анализы весь день четверг</code>
   → анализы · чт · весь день

<code>созвон работа синий 16.09 10:00</code>
   → созвон · календарь «Работа» · синий · 16.09 10:00

<code>зарядка каждый день 07:00 напомнить за 10м</code>
   → зарядка · каждый день · 07:00 · напомню за 10 мин.

<b>Слова-триггеры</b>: пишешь в строке, я убираю их из названия:

<code>задача</code>: создам ещё и задачу, связанную с событием
<code>весь день</code>: без времени, на весь день
<code>с 14.10 по 29.10</code>: событие на несколько дней
<code>повтор</code>: спрошу, как часто
<code>каждый день</code> · <code>раз в 3 дня</code> · <code>через день</code>: период повтора
<code>10 раз</code> · <code>до 01.12</code>: когда повтор кончится
<code>синий</code> · <code>красный</code> · <code>зелёный</code> …: цвет события
<code>#тег</code>: тег
<code>напомнить за 15м</code> · <code>напомни за час</code>: напоминание
<b>название календаря</b>: положу событие в него

<b>Дни:</b> <code>завтра</code> · <code>среда</code> · <code>следующая среда</code> · <code>пн</code> · <code>wednesday</code> · <code>16.09</code>
<b>Время:</b> <code>14:00</code> · <code>14:00-15:30</code> · <code>1330</code> · <code>в 15</code> · <code>полдень</code>

<b>Своё слово</b>: ⚙️ Настройки → 🔤 Ключевые слова: любое слово можно назначить тегом, цветом, календарём, датой, временем или повтором.
<b>Списком</b>: несколько строк сразу, спрошу, одна это запись или список.

Покажу, что понял, и спрошу подтверждение. Создам только после него.`,
		`📅 <b>New event</b>

Write it in one line. Here is what I understand:

<code>Ужин завтра 19:00</code>
   → Ужин · tomorrow · 19:00–20:00

<code>среда 14:00-15:00 оркестр повтор</code>
   → оркестр · Wed · 14:00–15:00 · repeats, I will ask how often

<code>анализы весь день четверг</code>
   → анализы · Thu · all day

<code>созвон работа синий 16.09 10:00</code>
   → созвон · calendar «Работа» · blue · 16.09 10:00

<code>зарядка каждый день 07:00 напомнить за 10м</code>
   → зарядка · every day · 07:00 · reminder 10m before

<b>Trigger words</b>: write them in the line and I take them out of the title:

<code>задача</code>: also creates a task, linked to the event
<code>весь день</code>: no time, all day
<code>с 14.10 по 29.10</code>: an event over several days
<code>повтор</code>: I will ask how often
<code>каждый день</code> · <code>раз в 3 дня</code> · <code>через день</code>: how often it repeats
<code>10 раз</code> · <code>до 01.12</code>: when the series ends
<code>синий</code> · <code>красный</code> · <code>зелёный</code> …: the colour
<code>#тег</code>: a tag
<code>напомнить за 15м</code> · <code>напомни за час</code>: a reminder
<b>a calendar name</b>: puts the event in it

<b>Days:</b> <code>завтра</code> · <code>среда</code> · <code>следующая среда</code> · <code>пн</code> · <code>wednesday</code> · <code>16.09</code>
<b>Times:</b> <code>14:00</code> · <code>14:00-15:30</code> · <code>1330</code> · <code>в 15</code> · <code>полдень</code>

<b>Your own word</b>: ⚙️ Settings → 🔤 Keywords: any word can stand for a tag, a colour, a calendar, a date, a time or a repeat.
<b>As a list</b>: several lines at once, I will ask whether it is one entry or a list.

⚠ These words work in both languages and are not translated with the interface.

I will show what I understood and ask you to confirm. Nothing is created before that.`)
}

// creationGuideShort is what opens by default since v0.4.11.2.
//
// 🔴 The full guide above is twenty lines, and the first outside users said
// «lots of steps» and «слишком много всего». Two examples say what the bot is
// for; the vocabulary is one tap away (📖), not deleted.
func creationGuideShort(lang i18n.Lang) string {
	return i18n.T(lang,
		`📅 <b>Новое событие</b>

Напиши одной строкой, например:
<code>Ужин завтра 19:00</code>
<code>зарядка каждый день 07:00</code>

Можно несколько строк сразу, списком.`,
		`📅 <b>New event</b>

Write it in one line, for example:
<code>Ужин завтра 19:00</code>
<code>зарядка каждый день 07:00</code>

Several lines at once work too, as a list.`)
}

func (h *Handler) startNewEventFlow(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	us.FlowData = map[string]any{}
	h.sendHTMLWithKeyboard(chatID, creationGuideShort(h.lang(chatID)), keyboards.GuideMore(h.lang(chatID), "event"))
}

// startNewEventForDay begins the same flow with the day already chosen, so
// «18:00 Ужин» is enough.
func (h *Handler) startNewEventForDay(chatID int64, date string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "new_event"
	us.FlowStep = "line"
	us.FlowData = map[string]any{"date": date}
	h.sendHTML(chatID, h.t(chatID, "📅 <b>Событие на ", "📅 <b>Event on ")+format.Escape(date)+
		h.t(chatID, "</b>\n\nНапиши время и название, например «18:00 Ужин».",
			"</b>\n\nWrite the time and the title, «18:00 Ужин» for instance."))
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
		// 🔴 Ask before assuming. A block of lines may be one entry with a long
		// title or several entries, and the bot is not the one who knows which.
		if parse.LooksLikeList(text, time.Now().In(h.location(chatID))) {
			h.askListOrSingle(chatID, text)
			return
		}
		st := h.parseIntoDraft(chatID, text)
		h.applyQuickKind(&st)
		if date, ok := us.FlowData["date"].(string); ok && date != "" && !st.D.HasDay {
			if day, err := time.ParseInLocation("2006-01-02", date, h.location(chatID)); err == nil {
				st.D.Day, st.D.HasDay = day, true
			}
		}
		us.FlowData["draft"] = &st
		if len(st.D.MoreDays) > 0 {
			// Several dates in one line mean a question, not a guess.
			h.askManyDates(chatID, 0)
			return
		}
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

	case "card":
		// 🔴 Review 17.09: text typed on the card used to wipe the draft. A new
		// draft retyped is a correction — read it as the line again. While a
		// list or an existing event is open, a retyped line cannot say which
		// entry it replaces, so the card stays and says so.
		st, ok := draftOf(h, chatID)
		_, inList := listOf(h, chatID)
		if ok && st.EventID == "" && !inList {
			us.FlowStep = "line"
			h.handleNewEventFlow(chatID, text)
			return
		}
		h.keepDraft(chatID)

	case "ask:freq", "edit:freq":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		read, valid := parse.RepeatText(text, time.Now().In(h.location(chatID)))
		if !valid {
			h.sendHTMLWithKeyboard(chatID, h.t(chatID,
				"Не понял частоту. Например «раз в 3 дня» или «каждые 2 недели».",
				"Could not read that. «every 3 days» or «every 2 weeks», say."), keyboards.DraftBack(h.lang(chatID)))
			return
		}
		st.D.Repeat, st.D.RepeatAsked = read.Repeat, false
		// An end typed with the period replaces the old one; none typed keeps it.
		if read.RepeatCount > 0 || !read.RepeatUntil.IsZero() {
			st.D.RepeatCount, st.D.RepeatUntil = read.RepeatCount, read.RepeatUntil
		}
		h.showCard(chatID, 0)

	case "edit:rend":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		if !parse.RepeatEndText(text, time.Now().In(h.location(chatID)), &st.D) {
			h.sendHTMLWithKeyboard(chatID, h.t(chatID,
				"Не понял. Например «10 раз» или «до 01.12».",
				"Could not read that. «10 times» or «until 01.12», say."), keyboards.DraftBack(h.lang(chatID)))
			return
		}
		h.showCard(chatID, 0)

	case "edit:remind", "pick:remind":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		n, read := parse.ReminderOffsetText(text)
		if !read {
			h.sendHTMLWithKeyboard(chatID, h.t(chatID,
				"Не понял время. Например «2ч», «45 минут», «за день».",
				"Could not read that. «2h», «45 min», «a day», say."), keyboards.DraftBack(h.lang(chatID)))
			return
		}
		toggleReminder(st, n, true)
		h.showReminderPicker(chatID, 0, st)

	case "edit:note":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		// Stored whole. Only the CARD shortens — truncating on the way in would
		// throw away what the person wrote, which no display decision is
		// allowed to do.
		st.Description = strings.TrimSpace(text)
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
		p := parse.ParseLine(text, time.Now().In(h.location(chatID)))
		if !p.Draft.HasDay {
			h.sendText(chatID, h.t(chatID, "Не понял дату. Например: «завтра», «среда», «16.09».", "Didn't get the date. Try «завтра», «среда», «16.09»."))
			return
		}
		st.D.Day, st.D.HasDay = p.Draft.Day, true
		// A span typed here replaces the old one; a single day ends it.
		st.D.EndDay = p.Draft.EndDay
		if !p.Draft.EndDay.IsZero() {
			st.D.AllDay, st.D.HasTime, st.D.HasEnd = p.Draft.AllDay, p.Draft.HasTime, p.Draft.HasEnd
			st.D.Start, st.D.End = p.Draft.Start, p.Draft.End
		}
		h.showCard(chatID, 0)

	case "edit:time":
		st, ok := draftOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return
		}
		p := parse.ParseLine(text, time.Now().In(h.location(chatID)))
		if !p.Draft.HasTime {
			h.sendText(chatID, h.t(chatID, "Не понял время. Например: «14:00» или «14:00-15:30».", "Didn't get the time. Try «14:00» or «14:00-15:30»."))
			return
		}
		st.D.Start, st.D.End = p.Draft.Start, p.Draft.End
		st.D.HasTime, st.D.HasEnd = true, p.Draft.HasEnd
		st.D.AllDay = false
		h.showCard(chatID, 0)

	default:
		// 🔴 Never ClearFlow here. This branch used to answer «Что-то пошло
		// не так» and throw the draft away for any text typed on a screen that
		// expects a button — the edit menu, a list question. The draft stays.
		h.keepDraft(chatID)
	}
}

// keepDraft answers text the current screen cannot read: the draft survives,
// and the user is told what the screen wants.
func (h *Handler) keepDraft(chatID int64) {
	_, hasDraft := draftOf(h, chatID)
	_, inList := listOf(h, chatID)
	if !hasDraft && !inList {
		h.lostDraft(chatID)
		return
	}
	if !hasDraft {
		// A list with no entry picked: «back to the card» would find no card
		// and wipe the list, so the way back is to the list itself.
		h.sendHTMLWithKeyboard(chatID, h.t(chatID,
			"Здесь нужна кнопка, список на месте.",
			"This screen needs a button; the list is still here."),
			keyboards.BackToList(h.lang(chatID)))
		return
	}
	h.sendHTMLWithKeyboard(chatID, h.t(chatID,
		"Здесь нужна кнопка, черновик на месте. «🗑 Отменить», чтобы начать заново.",
		"This screen needs a button; the draft is still here. «🗑 Cancel» to start over."),
		keyboards.DraftBack(h.lang(chatID)))
}

// parseIntoDraft runs the pipeline and then resolves a calendar name, which
// needs data the parser cannot have.
//
// ⚠ The title is recomputed AFTER the calendar pass. Recomputing before it
// would leave the calendar's name in the title — the exact leftover-fragment
// failure this rewrite exists to end.
func (h *Handler) parseIntoDraft(chatID int64, text string) draftState {
	us := h.store.GetOrCreate(chatID)
	p := parse.ParseLine(text, time.Now().In(h.location(chatID)))
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

	// 🔴 The user's own words run LAST, after every built-in recogniser and
	// after the calendar. A custom word spelled like «синий» or «повтор» must
	// not take the built-in meaning away from the person who added it.
	if vocab, err := h.api.BotKeywords(us.AuthToken); err == nil && len(vocab) > 0 {
		rules := make(map[string]parse.Trigger, len(vocab))
		for word, kw := range vocab {
			field, ok := parse.FieldByName(kw.Field)
			if !ok {
				// A characteristic this build does not know — written by a
				// newer version, or renamed. Skipping it leaves the word in
				// the title, which is visible; guessing a field would not be.
				continue
			}
			rules[word] = parse.Trigger{Field: field, Value: kw.Value}
		}
		parse.RecogniseCustomTriggers(p.Tokens, rules, time.Now().In(h.location(chatID)), &st.D)
	}

	// A custom word may name a calendar. Resolving it needs the list again,
	// but only when the pipeline did not already resolve one from the line.
	if st.CalendarID == "" && st.D.Calendar != "" {
		if cals, err := h.api.Calendars(us.AuthToken); err == nil {
			for _, c := range cals {
				if strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(st.D.Calendar)) {
					st.CalendarID, st.CalendarName = c.ID, c.Name
					break
				}
			}
		}
	}

	if presets, err := h.api.ReminderPresets(us.AuthToken); err == nil && len(presets) > 0 {
		parse.RecogniseReminderPreset(p.Tokens, presets, &st.D)
	}
	st.ReminderOffsets = st.D.ReminderOffsets

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
	card := keyboards.DraftCard(h.lang(chatID))
	if st.EventID != "" {
		// 🔴 «Сохранить», and no «Создать» anywhere on the screen. The words are
		// the only thing telling the user whether they are about to add a second
		// event or change the one they opened.
		// Since 17.09 the fields themselves sit on this screen (EventEditor):
		// an opened event is one tap from any change.
		// The id the event was opened with, kept by handleEventCard: an
		// occurrence must stay one after a field is edited.
		rawID, _ := us.FlowData["rawID"].(string)
		if rawID == "" {
			rawID = st.EventID
		}
		card = keyboards.EventEditor(h.lang(chatID), st.EventID, rawID)
	} else if _, inList := listOf(h, chatID); inList {
		// ✅ is deliberately absent inside a list: creating is the list's own
		// button, and a per-entry ✅ would create one event and leave the rest
		// silently uncreated.
		card = keyboards.DraftCardInList(h.lang(chatID))
	}
	// The card names the calendar the event will actually land in. An unchosen
	// calendar is not «нет» — it is the personal one, and saying otherwise
	// teaches that the field is broken (Denis, 18.09).
	shown := *st
	shown.CalendarName = h.calendarNameFor(chatID, st.CalendarName)
	text := renderDraft(h.lang(chatID), shown, time.Now().In(h.location(chatID)))
	if series, _ := us.FlowData["series"].(bool); series && st.EventID != "" {
		// Repeated on every redraw, not only on opening: the warning matters
		// most right before «Сохранить».
		text += h.t(chatID,
			"\n\n⚠ Это повторяющееся событие. Изменения применятся ко всей серии.",
			"\n\n⚠ This event repeats. Changes apply to the whole series.")
	}
	h.editOrSend(chatID, messageID, text, card)
}

func (h *Handler) lostDraft(chatID int64) {
	h.store.ClearFlow(chatID)
	h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Черновик потерялся, начнём заново.", "Lost the draft; let's start over."), h.home(chatID))
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
	if us.CurrentFlow == "new_event" && h.handleListCallback(chatID, messageID, data) {
		return true
	}
	// Tasks share the one/many question and nothing else: for them «many»
	// means create, while for events it means show a card with days and times
	// to check first.
	if us.CurrentFlow == "new_task" && h.handleTaskListCallback(chatID, messageID, data) {
		return true
	}
	// 🔴 ONE cancel branch, and it is here: before the flow is checked and
	// before the draft is looked up.
	//
	// 🗑 is the one button that must work in every state — mid-list, on the
	// «одна запись или список?» question where no draft exists yet, and after a
	// menu press ended the flow. Handling it later would answer «черновик
	// потерялся» to someone who was trying to throw the draft away.
	//
	// ⚠ There were briefly three copies of this branch, two of them
	// unreachable, because each new state got its own. Unreachable code that
	// LOOKS like the handler is worse than none: the next reader fixes the copy
	// that never runs.
	// Several dates in one line have their own question (manydates.go).
	if strings.HasPrefix(data, "dr_d") && h.handleDatesCallback(chatID, messageID, data) {
		return true
	}

	if data == "dr_cancel" {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, h.t(chatID, "🗑 Отменено.", "🗑 Cancelled."), h.home(chatID))
		return true
	}

	if us.CurrentFlow != "new_event" {
		// The card belongs to a flow that is over — most often because a menu
		// button interrupted it. Saying so beats editing a message the state no
		// longer backs.
		h.editOrSend(chatID, messageID, h.t(chatID, "Это создание уже закрыто.", "That one is already closed."), h.home(chatID))
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

	case data == "dr_swap":
		st.D.Day, st.D.EndDay = st.D.EndDay, st.D.Day
		// The dates were read correctly; only their order was wrong, so the
		// ⚠ that flagged the order goes with it.
		st.D.Uncertain = withoutField(st.D.Uncertain, parse.FieldDay)
		h.showCard(chatID, messageID)

	case data == "dr_daytext":
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Напиши дату: «16.09», «среда», «завтра» или промежуток «с 14.10 по 29.10».",
			"Write the date: «16.09», «среда», «завтра» or a span «с 14.10 по 29.10»."),
			keyboards.DraftBack(h.lang(chatID)))

	case data == "dr_back":
		h.showCard(chatID, messageID)

	case data == "dr_edit":
		us.FlowStep = "edit:menu"
		h.editOrSend(chatID, messageID, h.t(chatID, "Что изменить?", "What should I change?"), keyboards.DraftEditMenu(h.lang(chatID)))

	case data == "dre_title":
		us.FlowStep = "edit:title"
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Напиши название. Здесь оно берётся <b>как есть</b>: ключевые слова не действуют.", "Write the title. Here it is taken <b>as is</b>: keywords do nothing."), keyboards.DraftBack(h.lang(chatID)))

	case data == "dre_note":
		us.FlowStep = "edit:note"
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Напиши описание. Сохранится целиком, на карточке видно начало.", "Write the description. It is saved in full; the card shows the beginning."), keyboards.DraftBack(h.lang(chatID)))

	case data == "dre_tags":
		us.FlowStep = "edit:tags"
		h.editOrSend(chatID, messageID,
			h.t(chatID, "Теги через запятую. Здесь любое слово считается тегом, даже «полдень».", "Tags, comma separated. Here any word is a tag, even «полдень»."), keyboards.DraftBack(h.lang(chatID)))

	case data == "dre_date":
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, h.t(chatID, "Когда? Кнопкой или напиши: «среда», «16.09».", "When? Use a button, or write «среда», «16.09»."), keyboards.DraftDay(h.lang(chatID)))

	case data == "dre_time":
		us.FlowStep = "edit:time"
		h.editOrSend(chatID, messageID, h.t(chatID, "Во сколько? Например «14:00» или «14:00-15:30».", "What time? «14:00» or «14:00-15:30»."), keyboards.DraftBack(h.lang(chatID)))

	case data == "dre_repeat":
		us.FlowStep = "ask:freq"
		h.editOrSend(chatID, messageID, h.t(chatID, "Как часто повторять?", "How often should it repeat?"), keyboards.FreqPicker(h.lang(chatID)))

	case data == "dre_rend":
		if st.D.Repeat == "" && !st.D.RepeatAsked {
			h.editOrSend(chatID, messageID, h.t(chatID, "Сначала выбери, как часто повторять.", "Choose how often it repeats first."), keyboards.FreqPicker(h.lang(chatID)))
			return true
		}
		// The end may be typed straight onto this screen (review 17.09).
		us.FlowStep = "edit:rend"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Когда закончить повтор? Можно сразу написать «10 раз» или «до 01.12».",
			"When should the series end? You can just write «10 times» or «until 01.12»."), keyboards.RepeatEndPicker(h.lang(chatID)))

	case data == "dr_rend_never":
		st.D.RepeatCount, st.D.RepeatUntil = 0, time.Time{}
		h.showCard(chatID, messageID)

	case data == "dr_rend_text":
		us.FlowStep = "edit:rend"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Напиши «10 раз» или «до 01.12».",
			"Write «10 times» or «until 01.12»."), keyboards.DraftBack(h.lang(chatID)))

	case data == "dre_colour":
		h.editOrSend(chatID, messageID, h.t(chatID, "Цвет события:", "Event colour:"), keyboards.ColourPicker(h.lang(chatID)))

	case data == "dre_cal":
		h.showCalendarPicker(chatID, messageID)

	case data == "dre_remind":
		h.showReminderPicker(chatID, messageID, st)

	case data == "dre_allday":
		st.D.AllDay = !st.D.AllDay
		if st.D.AllDay {
			st.D.HasTime, st.D.HasEnd = false, false
		}
		h.showCard(chatID, messageID)

	case strings.HasPrefix(data, "dr_freq_"):
		freq := strings.TrimPrefix(data, "dr_freq_")
		if freq == "CUSTOM" {
			us.FlowStep = "edit:freq"
			h.editOrSend(chatID, messageID, h.t(chatID,
				"Как часто? Например «раз в 3 дня», «каждые 2 недели», «раз в месяц 10 раз».",
				"How often? «every 3 days», «every 2 weeks», «every month 10 times», say."), keyboards.DraftBack(h.lang(chatID)))
			return true
		}
		st.D.RepeatAsked = false
		if freq == "NONE" {
			st.D.Repeat = ""
			st.D.RepeatCount, st.D.RepeatUntil = 0, time.Time{}
		} else {
			st.D.Repeat = parse.FreqRule(freq)
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
		now := time.Now().In(h.location(chatID))
		st.D.Day = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, n)
		st.D.HasDay = true
		// A day button picks ONE day: whatever span was there is gone.
		st.D.EndDay = time.Time{}
		h.showCard(chatID, messageID)

	case data == "dr_rem_done":
		h.showCard(chatID, messageID)

	case data == "dr_rem_custom":
		us.FlowStep = "edit:remind"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"За сколько напомнить? Например «2ч», «45 минут», «за день».",
			"How long before? «2h», «45 min», «a day», say."), keyboards.DraftBack(h.lang(chatID)))

	case strings.HasPrefix(data, "dr_rem_"):
		h.setReminder(st, strings.TrimPrefix(data, "dr_rem_"))
		h.showReminderPicker(chatID, messageID, st)

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
//
// A number TOGGLES: ticked joins the list, ticked again leaves it. Unticking the
// last one leaves an explicit empty list — «no reminder» — because that is what
// the picker then shows, and the card must not say otherwise.
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
	if err != nil || n <= 0 {
		return
	}
	toggleReminder(st, n, false)
}

// toggleReminder flips one offset; onlyAdd is the typed-time path, where
// writing a time that is already ticked must not untick it.
func toggleReminder(st *draftState, n int, onlyAdd bool) {
	offsets := []int{}
	if st.ReminderOffsets != nil {
		offsets = append(offsets, (*st.ReminderOffsets)...)
	}
	for i, m := range offsets {
		if m == n {
			if !onlyAdd {
				offsets = append(offsets[:i], offsets[i+1:]...)
			}
			st.ReminderOffsets = &offsets
			return
		}
	}
	offsets = append(offsets, n)
	sort.Ints(offsets)
	st.ReminderOffsets = &offsets
}

func (h *Handler) showReminderPicker(chatID int64, messageID int, st *draftState) {
	// Its own step, so a time typed onto the picker is read as one.
	h.store.GetOrCreate(chatID).FlowStep = "pick:remind"
	h.editOrSend(chatID, messageID,
		h.t(chatID, "🔔 Когда напомнить? Можно отметить несколько.", "🔔 When should I remind you? Tick as many as you like."),
		keyboards.ReminderPicker(h.lang(chatID), st.ReminderOffsets))
}

func (h *Handler) showCalendarPicker(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	cals, err := h.api.Calendars(us.AuthToken)
	if err != nil || len(cals) == 0 {
		h.editOrSend(chatID, messageID, h.t(chatID, "Не удалось прочитать календари.", "Could not read your calendars."), keyboards.DraftCard(h.lang(chatID)))
		return
	}
	names := make([]string, len(cals))
	ids := make([]string, len(cals))
	for i, c := range cals {
		names[i], ids[i] = c.Name, c.ID
	}
	h.editOrSend(chatID, messageID, h.t(chatID, "В какой календарь?", "Which calendar?"), keyboards.CalendarPicker(h.lang(chatID), names, ids))
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
		h.editOrSend(chatID, messageID, h.t(chatID, "Как назвать? Текст пойдёт в название как есть.", "What should it be called? The text goes in as is."), keyboards.DraftBack(h.lang(chatID)))
	case askFreq:
		us.FlowStep = "ask:freq"
		h.editOrSend(chatID, messageID, h.t(chatID, "Как часто повторять?", "How often should it repeat?"), keyboards.FreqPicker(h.lang(chatID)))
	case askDate:
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, h.t(chatID,
			"На какой день? Кнопкой или напиши: «16.09», «среда», «с 14.10 по 29.10».",
			"Which day? Use a button, or write «16.09», «среда», «с 14.10 по 29.10»."),
			keyboards.DraftDay(h.lang(chatID)))
	case askSpan:
		us.FlowStep = "edit:date"
		h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
			"Конец раньше начала: %s – %s. Поменять местами или написать даты заново?",
			"It ends before it starts: %s – %s. Swap them, or write the dates again?"),
			st.D.Day.Format("02.01"), st.D.EndDay.Format("02.01")),
			keyboards.SpanFix(h.lang(chatID), st.D.Day.Format("02.01"), st.D.EndDay.Format("02.01")))
	case askTime:
		us.FlowStep = "edit:time"
		h.editOrSend(chatID, messageID, h.t(chatID, "Во сколько? Например «14:00» или «14:00-15:30».", "What time? «14:00» or «14:00-15:30»."), keyboards.DraftBack(h.lang(chatID)))
	default:
		if st.EventID != "" {
			h.updateFromDraft(chatID, messageID, *st)
			return
		}
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
	loc := h.location(chatID)
	start, end := draftBounds(st)

	eventID, err := h.createOne(chatID, st)
	if err != nil {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, "❌ "+h.errorText(chatID, err), h.home(chatID))
		return
	}

	h.store.ClearFlow(chatID)
	// 🔴 Denis, 23.09 (pass 3): «кнопка изменить после создания отправляет в
	// список событий». The answer carried the События screen's keyboard, whose
	// ✏️ opens the picker. Under «✅ Создано», «изменить» means the event just
	// made — the task → calendar path already answered with its card.
	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID, "✅ <b>Создано</b>\n%s\n🕐 %s", "✅ <b>Created</b>\n%s\n🕐 %s"),
		format.Escape(st.Title), humanRange(h.lang(chatID), start.In(loc), end.In(loc), time.Now().In(loc))),
		keyboards.EventCard(h.lang(chatID), eventID))
}

// draftBounds turns the draft's day and offsets into two instants. An all-day
// event spans local midnight to local midnight; anything else runs an hour
// unless an end was given.
func draftBounds(st draftState) (time.Time, time.Time) {
	if st.D.AllDay {
		// An all-day event ends at the midnight AFTER its last day — one day
		// or a span, the same convention.
		last := st.D.Day
		if !st.D.EndDay.IsZero() && st.D.EndDay.After(last) {
			last = st.D.EndDay
		}
		return st.D.Day, last.AddDate(0, 0, 1)
	}
	start := st.D.StartsAt()
	if st.D.HasEnd {
		return start, st.D.EndsAt()
	}
	return start, start.Add(time.Hour)
}

// createOne writes one draft and reports what went wrong, if anything.
//
// 🔴 A draft marked «задача» becomes TWO objects: a task, and an event bound to
// it by task_id. If the second call fails the first has already happened, and
// that is said out loud rather than compensated for — a silent rollback that
// can itself fail is worse than an honest sentence.
// It returns the id of the event it made.
func (h *Handler) createOne(chatID int64, st draftState) (string, error) {
	us := h.store.GetOrCreate(chatID)

	var taskID *string
	if st.D.IsTask {
		task, err := h.api.CreateTask(us.AuthToken, api.CreateTaskReq{
			Title: st.Title, Status: "TODO", Tags: st.D.Tags,
		})
		if err != nil {
			return "", fmt.Errorf(h.t(chatID, "задача не создана: %w", "task not created: %w"), err)
		}
		taskID = &task.ID
	}

	start, end := draftBounds(st)

	req := api.CreateEventReq{
		Title:           st.Title,
		StartsAt:        start.UTC().Format(time.RFC3339),
		EndsAt:          end.UTC().Format(time.RFC3339),
		AllDay:          st.D.AllDay,
		Tags:            st.D.Tags,
		TaskID:          taskID,
		Description:     optional(st.Description),
		ReminderOffsets: st.ReminderOffsets,
	}
	if rrule := st.D.RRule(); rrule != "" {
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
	if err != nil {
		if taskID != nil {
			return "", fmt.Errorf(h.t(chatID,
				"событие не создано (%w), но задача создана: она в списке задач",
				"event not created (%w), but the task was: it is in your task list"), err)
		}
		return "", fmt.Errorf(h.t(chatID, "событие не создано: %w", "event not created: %w"), err)
	}
	return ev.ID, nil
}

// withoutField drops one field from the "read loosely" list.
func withoutField(fields []parse.Field, drop parse.Field) []parse.Field {
	out := fields[:0]
	for _, f := range fields {
		if f != drop {
			out = append(out, f)
		}
	}
	return out
}

// optional turns an empty string into an absent JSON field.
//
// 🔴 Not a plain string: the API's Description is *string, and «» sent as a
// present-but-empty value would CLEAR a note on every edit that did not touch
// it — the same shape as the settings blob that erases what it does not know
// (gotcha 21).
func optional(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
