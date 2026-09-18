package handlers

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Calendars in the bot.
//
// 🔴 Denis, 18.09: «сейчас нет способа добавить календарь или человека в
// календарь в боте, только через веб». Every endpoint this screen needs already
// existed — /api/calendars has full CRUD plus invites, invite links and members
// — so the gap was never the API, it was that nothing in the bot called it.
//
// Two entrances, both to the same callback: the 🗓 Calendar screen and
// ⚙️ Settings. Not the reply keyboard — that is a fixed [3][2] grid
// (keyboards/menu.go:56) and a seventh button breaks both MainMenu and the
// guard that tells a button press from a typed line.

// calendarFlowPrefix marks the text steps this screen owns.
const calendarFlowPrefix = "cal:"

// handleCalendarsCallback owns every cal_/cals callback. It returns false for
// anything that is not its own, so HandleCallback can go on asking.
func (h *Handler) handleCalendarsCallback(chatID int64, messageID int, data string) bool {
	switch {
	case data == "cls":
		h.showCalendars(chatID, messageID)
	case data == "cl_new":
		h.startCalendarName(chatID, "", i18n.T(h.lang(chatID),
			"Как назвать новый календарь?", "What should the new calendar be called?"))
	case strings.HasPrefix(data, "cl_name_"):
		h.startCalendarName(chatID, strings.TrimPrefix(data, "cl_name_"), i18n.T(h.lang(chatID),
			"Новое имя календаря?", "New calendar name?"))
	case strings.HasPrefix(data, "cl_col_"):
		id := strings.TrimPrefix(data, "cl_col_")
		h.editOrSend(chatID, messageID, h.t(chatID, "Какой цвет?", "Which colour?"),
			keyboards.CalendarColours(h.lang(chatID), id))
	case strings.HasPrefix(data, "cl_acc_"):
		h.acceptInviteLink(chatID, messageID, strings.TrimPrefix(data, "cl_acc_"))
	case data == "cl_dec":
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Хорошо, не добавляю.", "All right, not adding you."), keyboards.None())
	case strings.HasPrefix(data, "cl_setcol_"):
		h.setCalendarColour(chatID, messageID, strings.TrimPrefix(data, "cl_setcol_"))
	case strings.HasPrefix(data, "cl_mem_"):
		h.showCalendarMembers(chatID, messageID, strings.TrimPrefix(data, "cl_mem_"))
	case strings.HasPrefix(data, "cl_inv_"):
		id := strings.TrimPrefix(data, "cl_inv_")
		h.editOrSend(chatID, messageID, h.t(chatID, "Как пригласить?", "How to invite?"),
			keyboards.CalendarInvite(h.lang(chatID), id))
	case strings.HasPrefix(data, "cl_link_"):
		h.sendInviteLink(chatID, strings.TrimPrefix(data, "cl_link_"))
	case strings.HasPrefix(data, "cl_mail_"):
		h.startCalendarEmail(chatID, strings.TrimPrefix(data, "cl_mail_"))
	case strings.HasPrefix(data, "cl_leaveok_"):
		h.leaveCalendar(chatID, messageID, strings.TrimPrefix(data, "cl_leaveok_"))
	case strings.HasPrefix(data, "cl_leave_"):
		h.confirmCalendar(chatID, messageID, strings.TrimPrefix(data, "cl_leave_"), "leave")
	case strings.HasPrefix(data, "cl_delok_"):
		h.deleteCalendar(chatID, messageID, strings.TrimPrefix(data, "cl_delok_"))
	case strings.HasPrefix(data, "cl_del_"):
		h.confirmCalendar(chatID, messageID, strings.TrimPrefix(data, "cl_del_"), "delete")
	case strings.HasPrefix(data, "cl_"):
		// 🔴 Last, because «cal_» is a prefix of every case above it. Putting
		// the card first would swallow «cl_new» and open a calendar whose id
		// is the word «new».
		h.showCalendarCard(chatID, messageID, strings.TrimPrefix(data, "cl_"))
	default:
		return false
	}
	return true
}

// showCalendars lists every calendar the user can see.
func (h *Handler) showCalendars(chatID int64, messageID int) {
	lang := h.lang(chatID)
	cals, err := h.calendarList(chatID)
	if err != nil {
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"Не смог получить календари. Попробуй ещё раз.",
			"Could not load the calendars. Try again."), keyboards.HomeInline(lang))
		return
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, "📁 <b>Календари</b>\n\n", "📁 <b>Calendars</b>\n\n"))
	names := make([]string, 0, len(cals))
	ids := make([]string, 0, len(cals))
	for _, c := range cals {
		line := "📁 " + format.Escape(c.Name)
		switch {
		case c.IsPersonal():
			line += i18n.T(lang, " · личный", " · personal")
		case !c.IsOwner():
			line += i18n.T(lang, " · чужой", " · shared with me")
		}
		b.WriteString(line + "\n")
		names = append(names, c.Name)
		ids = append(ids, c.ID)
	}
	if len(cals) == 0 {
		b.WriteString(i18n.T(lang, "Пока ни одного.\n", "None yet.\n"))
	}
	h.editOrSend(chatID, messageID, b.String(), keyboards.CalendarList(lang, names, ids))
}

// showCalendarCard prints one calendar with every characteristic named.
//
// The same rule as the event card: an empty field says «нет» rather than being
// skipped, so a person learns the field exists.
func (h *Handler) showCalendarCard(chatID int64, messageID int, id string) {
	lang := h.lang(chatID)
	c, ok := h.findCalendar(chatID, id)
	if !ok {
		h.showCalendars(chatID, messageID)
		return
	}

	members := "—"
	if ms, err := h.api.CalendarMembers(h.store.GetOrCreate(chatID).AuthToken, id); err == nil {
		members = fmt.Sprintf("%d", len(ms))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "📁 <b>%s</b>\n", format.Escape(c.Name))
	b.WriteString(fieldLine("🎨", i18n.T(lang, "Цвет:", "Colour:"), orNone(lang, format.Escape(c.ColourOrEmpty()))))
	b.WriteString(fieldLine("👥", i18n.T(lang, "Участников:", "Members:"), members))
	b.WriteString(fieldLine("🔑", i18n.T(lang, "Роль:", "Role:"), roleName(lang, c.Role)))
	if c.IsPersonal() {
		b.WriteString(i18n.T(lang,
			"\nЛичный календарь — его нельзя удалить или покинуть.",
			"\nThe personal calendar — it cannot be deleted or left."))
	}

	h.editOrSend(chatID, messageID, b.String(),
		keyboards.CalendarCard(lang, id, c.Role, c.IsPersonal()))
}

func (h *Handler) showCalendarMembers(chatID int64, messageID int, id string) {
	lang := h.lang(chatID)
	ms, err := h.api.CalendarMembers(h.store.GetOrCreate(chatID).AuthToken, id)
	if err != nil {
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"Не смог получить участников.", "Could not load the members."),
			keyboards.CalendarCard(lang, id, "viewer", false))
		return
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, "👥 <b>Участники</b>\n\n", "👥 <b>Members</b>\n\n"))
	for _, m := range ms {
		line := "👤 " + format.Escape(m.Label()) + " · " + roleName(lang, m.Role)
		if m.Status != "active" {
			line += i18n.T(lang, " · приглашён", " · invited")
		}
		b.WriteString(line + "\n")
	}
	c, _ := h.findCalendar(chatID, id)
	h.editOrSend(chatID, messageID, b.String(),
		keyboards.CalendarCard(lang, id, c.Role, c.IsPersonal()))
}

// sendInviteLink hands over ONE message, ready to forward.
//
// 🔴 Not «here is a token, now write your own invitation». The person
// forwarding it is doing us a favour; making them compose the message is a step
// that buys nothing. And the link is the only path that reaches someone with no
// email at all — which is every Telegram-only account, including the tester who
// asked for this.
func (h *Handler) sendInviteLink(chatID int64, id string) {
	lang := h.lang(chatID)
	c, ok := h.findCalendar(chatID, id)
	if !ok {
		return
	}
	token, err := h.api.CalendarInviteLink(h.store.GetOrCreate(chatID).AuthToken, id, "editor")
	if err != nil {
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"Не смог создать ссылку. Попробуй ещё раз.",
			"Could not create the link. Try again."), keyboards.CalendarCard(lang, id, c.Role, c.IsPersonal()))
		return
	}

	h.sendHTMLWithKeyboard(chatID, fmt.Sprintf(i18n.T(lang,
		"Перешли это сообщение тому, кого зовёшь:\n\nЗову тебя в календарь «%s» в NeuroBoost.\nОткрой: https://t.me/%s?start=inv_%s",
		"Forward this message to whoever you are inviting:\n\nJoin my calendar «%s» in NeuroBoost.\nOpen: https://t.me/%s?start=inv_%s"),
		format.Escape(c.Name), h.botUsername(), token),
		keyboards.CalendarCard(lang, id, c.Role, c.IsPersonal()))
}

func (h *Handler) confirmCalendar(chatID int64, messageID int, id, action string) {
	lang := h.lang(chatID)
	c, ok := h.findCalendar(chatID, id)
	if !ok {
		h.showCalendars(chatID, messageID)
		return
	}
	text := i18n.T(lang, "Выйти из «"+c.Name+"»?", "Leave «"+c.Name+"»?")
	if action == "delete" {
		text = i18n.T(lang,
			"Удалить «"+c.Name+"»? Это навсегда.",
			"Delete «"+c.Name+"»? This is permanent.")
	}
	h.editOrSend(chatID, messageID, text, keyboards.CalendarConfirm(lang, id, c.Name, action))
}

func (h *Handler) leaveCalendar(chatID int64, messageID int, id string) {
	lang := h.lang(chatID)
	us := h.store.GetOrCreate(chatID)
	h.invalidateCalendars(chatID)
	if err := h.api.LeaveCalendar(us.AuthToken, id, h.myUserID(chatID)); err != nil {
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"Не смог выйти из календаря.", "Could not leave the calendar."),
			keyboards.CalendarList(lang, nil, nil))
		return
	}
	h.showCalendars(chatID, messageID)
}

// deleteCalendar reports a refusal in words.
//
// 🔴 The API refuses a non-empty calendar with 409 and the counts inside, and
// refuses the personal one outright. Both arrive as a plain error here, so
// saying «удалил» on a failure would be a lie the user only discovers when the
// calendar is still in the list.
func (h *Handler) deleteCalendar(chatID int64, messageID int, id string) {
	lang := h.lang(chatID)
	h.invalidateCalendars(chatID)
	if err := h.api.DeleteCalendar(h.store.GetOrCreate(chatID).AuthToken, id); err != nil {
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"Не смог удалить: в календаре ещё есть события или задачи. Перенеси их и попробуй снова.",
			"Could not delete: the calendar still has events or tasks. Move them out and try again."),
			keyboards.CalendarCard(lang, id, "owner", false))
		return
	}
	h.showCalendars(chatID, messageID)
}

// startCalendarName opens the text step for a new name. An empty id means a
// new calendar rather than a rename.
func (h *Handler) startCalendarName(chatID int64, id, prompt string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = calendarFlowPrefix + "name:" + id
	h.sendHTMLWithKeyboard(chatID, prompt, keyboards.CalendarList(h.lang(chatID), nil, nil))
}

func (h *Handler) startCalendarEmail(chatID int64, id string) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = calendarFlowPrefix + "mail:" + id
	h.sendHTMLWithKeyboard(chatID, h.t(chatID,
		"Напиши email того, кого зовёшь.",
		"Write the email of the person you are inviting."),
		keyboards.CalendarInvite(h.lang(chatID), id))
}

// handleCalendarText reads the answer to a text step.
func (h *Handler) handleCalendarText(chatID int64, flow, text string) {
	lang := h.lang(chatID)
	us := h.store.GetOrCreate(chatID)
	rest := strings.TrimPrefix(flow, calendarFlowPrefix)
	kind, id, _ := strings.Cut(rest, ":")
	text = strings.TrimSpace(text)
	h.store.ClearFlow(chatID)

	// Any write below changes what the list says, so the cached list goes first.
	h.invalidateCalendars(chatID)

	switch kind {
	case "name":
		if id == "" {
			if _, err := h.api.CreateCalendar(us.AuthToken, text, ""); err != nil {
				h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
					"Не смог создать календарь.", "Could not create the calendar."),
					keyboards.CalendarList(lang, nil, nil))
				return
			}
		} else if err := h.api.UpdateCalendar(us.AuthToken, id, &text, nil); err != nil {
			h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
				"Не смог переименовать.", "Could not rename it."),
				keyboards.CalendarList(lang, nil, nil))
			return
		}
		h.showCalendars(chatID, 0)
	case "mail":
		if err := h.api.InviteByEmail(us.AuthToken, id, text, "editor"); err != nil {
			// 🔴 The likely failure is not a bug: a Telegram-only account has
			// no email, so there is nothing to invite by. Say the way out
			// instead of the error code.
			h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
				"У этого email нет аккаунта. Дай ссылку — она работает и без email.",
				"No account with that email. Send them the link instead — it needs no email."),
				keyboards.CalendarInvite(lang, id))
			return
		}
		h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
			"Пригласил. Ответ придёт ему в приложении.",
			"Invited. They will see it in the app."),
			keyboards.CalendarList(lang, nil, nil))
	}
}

func (h *Handler) setCalendarColour(chatID int64, messageID int, payload string) {
	hex, id, ok := strings.Cut(payload, "_")
	if !ok {
		return
	}
	colour := "#" + hex
	h.invalidateCalendars(chatID)
	if err := h.api.UpdateCalendar(h.store.GetOrCreate(chatID).AuthToken, id, nil, &colour); err != nil {
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Не смог поменять цвет.", "Could not change the colour."),
			keyboards.CalendarColours(h.lang(chatID), id))
		return
	}
	h.showCalendarCard(chatID, messageID, id)
}

// calendarListTTL is how long a fetched list is reused.
//
// 🔴 Short on purpose. The screens here call findCalendar two or three times
// per press — the card wants the name, the keyboard wants the role, the members
// screen wants both — and without this every one of those was its own HTTPS
// round trip to the API. Pressing a calendar cost three.
//
// Ten seconds is long enough to cover one press and everything it redraws, and
// short enough that a rename made on the web shows up on the next screen rather
// than after a restart. It is a cache for one interaction, not for a session.
const calendarListTTL = 10 * time.Second

// calendarList reads the calendars for this chat, reusing a very recent read.
func (h *Handler) calendarList(chatID int64) ([]api.CalendarDetail, error) {
	us := h.store.GetOrCreate(chatID)
	if us.CalendarsAt.After(time.Now().Add(-calendarListTTL)) && us.Calendars != nil {
		return us.Calendars, nil
	}
	cals, err := h.api.CalendarsFull(us.AuthToken)
	if err != nil {
		return nil, err
	}
	us.Calendars, us.CalendarsAt = cals, time.Now()
	return cals, nil
}

// invalidateCalendars drops the cache after a write.
//
// 🔴 Called by every mutation here. A cache that outlives the change it does
// not know about is worse than no cache: the user renames a calendar, the
// screen redraws from ten-second-old data, and the rename looks like it failed.
func (h *Handler) invalidateCalendars(chatID int64) {
	us := h.store.GetOrCreate(chatID)
	us.Calendars, us.CalendarsAt = nil, time.Time{}
	us.PersonalCalendar, us.PersonalCalendarKnown = "", false
}

func (h *Handler) findCalendar(chatID int64, id string) (api.CalendarDetail, bool) {
	cals, err := h.calendarList(chatID)
	if err != nil {
		return api.CalendarDetail{}, false
	}
	for _, c := range cals {
		if c.ID == id {
			return c, true
		}
	}
	return api.CalendarDetail{}, false
}

func roleName(lang i18n.Lang, role string) string {
	switch role {
	case "owner":
		return i18n.T(lang, "владелец", "owner")
	case "editor":
		return i18n.T(lang, "может менять", "editor")
	case "viewer":
		return i18n.T(lang, "только смотрит", "viewer")
	}
	return role
}

// myUserID reads and caches the caller's own user id.
//
// Cached on the chat because leaving a calendar needs it and a second round
// trip per press buys nothing: a user id does not change for the life of an
// account.
func (h *Handler) myUserID(chatID int64) string {
	us := h.store.GetOrCreate(chatID)
	if id, ok := us.FlowData["user_id"].(string); ok && id != "" {
		return id
	}
	id, err := h.api.MyUserID(us.AuthToken)
	if err != nil {
		return ""
	}
	us.FlowData["user_id"] = id
	return id
}

// botUsername is the @name this bot answers to.
//
// 🔴 Asked of Telegram, not of the configuration. getMe runs at startup and
// reports the name belonging to the token actually in use, so a dev bot cannot
// hand out a link that opens the production bot — which is the failure a
// hardcoded or mistyped name would produce, silently, in the one message whose
// whole job is to be forwarded to someone else.
func (h *Handler) botUsername() string {
	if h.bot != nil && h.bot.Self.UserName != "" {
		return h.bot.Self.UserName
	}
	return h.cfg.BotUsername
}

// askInviteLink asks before joining, and asks before onboarding.
//
// The token travels in callback_data rather than in the chat state: /start
// delivers it once, and a state that a later message could clear would lose it
// between the question and the answer. A 256-bit token is 43 base64url
// characters, so «cl_acc_» plus the token is 50 bytes — inside Telegram's
// 64-byte cap with room to spare.
func (h *Handler) askInviteLink(chatID int64, token string) {
	lang := h.lang(chatID)
	h.sendHTMLWithKeyboard(chatID, i18n.T(lang,
		"Тебя зовут в общий календарь в NeuroBoost. Принять?",
		"You are invited to a shared calendar in NeuroBoost. Accept?"),
		tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Принять", "✅ Accept"), "cl_acc_"+token),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✖ Нет", "✖ No"), "cl_dec"))))
}

// acceptInviteLink redeems the token, then hands a brand-new person to
// onboarding — in that order, so nothing is lost either way.
func (h *Handler) acceptInviteLink(chatID int64, messageID int, token string) {
	lang := h.lang(chatID)
	h.invalidateCalendars(chatID)
	c, err := h.api.AcceptInviteLink(h.store.GetOrCreate(chatID).AuthToken, token)
	if err != nil {
		h.editOrSend(chatID, messageID, i18n.T(lang,
			"Ссылка не сработала — возможно, её уже использовали. Попроси новую.",
			"That link did not work — it may already have been used. Ask for a new one."),
			keyboards.None())
		return
	}
	h.editOrSend(chatID, messageID, fmt.Sprintf(i18n.T(lang,
		"Готово — ты в календаре «%s».", "Done — you are in «%s»."),
		format.Escape(c.Name)), keyboards.None())

	if h.needsOnboarding(chatID) {
		h.startOnboarding(chatID, 0, "")
	}
}

// personalCalendarName is the calendar a new item lands in when none was named.
//
// Read once per chat and cached: every card prints it, and an HTTP round trip
// per keypress is what the Lang and TZ caches already exist to avoid. A failure
// to read is cached as «unknown» for this message only, so a blip does not
// pin the wrong answer for the session.
func (h *Handler) personalCalendarName(chatID int64) string {
	us := h.store.GetOrCreate(chatID)
	if us.PersonalCalendarKnown {
		return us.PersonalCalendar
	}
	cals, err := h.api.CalendarsFull(us.AuthToken)
	if err != nil {
		return ""
	}
	for _, c := range cals {
		if c.IsPersonal() {
			us.PersonalCalendar, us.PersonalCalendarKnown = c.Name, true
			return c.Name
		}
	}
	return ""
}

// calendarNameFor names the calendar an item is going to, falling back to the
// personal one when nothing was chosen — because that is where it will land.
func (h *Handler) calendarNameFor(chatID int64, chosen string) string {
	if chosen != "" {
		return chosen
	}
	return h.personalCalendarName(chatID)
}
