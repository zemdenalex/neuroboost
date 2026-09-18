package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// Callback prefixes for the calendar screens.
//
// 🔴 The budget is Telegram's 64-byte callback_data cap minus a 36-byte UUID,
// which leaves 27 for the prefix. The longest here is «cl_leave_» at ten. A
// prefix that outgrows the budget does not fail at the button — Telegram
// refuses the entire message, so the screen simply never appears.
const (
	calList   = "cls"
	calNew    = "cl_new"
	calOpen   = "cl_"       // + id
	calName   = "cl_name_"  // + id
	calColour = "cl_col_"   // + id
	calMem    = "cl_mem_"   // + id
	calInvite = "cl_inv_"   // + id
	calLink   = "cl_link_"  // + id
	calMail   = "cl_mail_"  // + id
	calLeave  = "cl_leave_" // + id
	calDelete = "cl_del_"   // + id
)

// CalendarList renders one button per calendar, plus «создать» and «назад».
func CalendarList(lang i18n.Lang, names, ids []string) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(ids)+1)
	for i, id := range ids {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(names[i], calOpen+id)))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Создать", "➕ New"), calNew),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), "main_menu")))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CalendarCard offers exactly the actions the API will accept for this role.
//
// 🔴 Three shapes, not one shape with disabled buttons. The owner of a shared
// calendar may rename, recolour, invite and delete; a member may only look and
// leave; the personal calendar may be renamed but neither left (an owner may
// not leave — api-go calendars/members.go:304) nor deleted (ErrCalendarIsPersonal
// guards that path). A button the API refuses is a button that teaches
// distrust, and «есть кнопка, но она не работает» was three separate defects
// in one evening on 17.09.
func CalendarCard(lang i18n.Lang, id, role string, personal bool) tgbotapi.InlineKeyboardMarkup {
	owner := role == "owner"
	rows := [][]tgbotapi.InlineKeyboardButton{}

	if owner {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✏️ Имя", "✏️ Name"), calName+id),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🎨 Цвет", "🎨 Colour"), calColour+id)))
	}

	second := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "👥 Участники", "👥 Members"), calMem+id),
	}
	if owner {
		second = append(second,
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔗 Пригласить", "🔗 Invite"), calInvite+id))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(second...))

	switch {
	case personal:
		// Neither leaving nor deleting is possible, so neither is offered.
	case owner:
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Удалить", "🗑 Delete"), calDelete+id)))
	default:
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🚪 Выйти", "🚪 Leave"), calLeave+id)))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), calList)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CalendarInvite offers both ways of inviting.
//
// Both, because they reach different people: the email path cannot reach a
// Telegram-only account at all (their email is NULL and the handler rejects an
// empty one), and the link path is the only one that works for someone who has
// never opened the web app.
func CalendarInvite(lang i18n.Lang, id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔗 Ссылкой", "🔗 By link"), calLink+id),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✉️ По email", "✉️ By email"), calMail+id)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), calOpen+id)))
}

// calendarPalette is the colour choice offered for a calendar.
//
// Hex, not palette names: /api/calendars validates against a strict
// `^#[0-9a-fA-F]{6}$` (api-go calendars/handlers.go:26), while the draft card's
// colours are names the event API resolves itself. Two different vocabularies
// for two different endpoints — reusing draftColours here would send «blue» to
// a regexp that wants «#3b82f6».
// calendarPalette is the colour choice offered for a calendar.
//
// Hex, not palette names: /api/calendars validates against a strict
// `^#[0-9a-fA-F]{6}$` (api-go calendars/handlers.go:26), while the draft card's
// colours are names the event API resolves itself. Two different vocabularies
// for two different endpoints — reusing draftColours here would send «blue» to
// a regexp that wants «#3b82f6».
//
// Built through i18n.T rather than as a table of bare strings, the same shape
// draftColours uses: the scan that keeps untranslated text out of the bot reads
// call sites, and a literal in a package-level var is invisible to it in one
// language and shipped in the other.
func calendarPalette(lang i18n.Lang) []struct{ Label, Hex string } {
	return []struct{ Label, Hex string }{
		{i18n.T(lang, "синий", "blue"), "3b82f6"},
		{i18n.T(lang, "фиолетовый", "violet"), "7c3aed"},
		{i18n.T(lang, "зелёный", "green"), "22c55e"},
		{i18n.T(lang, "красный", "red"), "ef4444"},
		{i18n.T(lang, "янтарный", "amber"), "f59e0b"},
		{i18n.T(lang, "голубой", "cyan"), "06b6d4"},
		{i18n.T(lang, "розовый", "pink"), "ec4899"},
		{i18n.T(lang, "серый", "slate"), "64748b"},
	}
}

// CalendarColours offers the palette. The callback carries the hex WITHOUT the
// leading «#» — it is re-added on the way out, because a «#» inside
// callback_data is legal but easy to lose in a URL-ish string.
func CalendarColours(lang i18n.Lang, id string) tgbotapi.InlineKeyboardMarkup {
	palette := calendarPalette(lang)
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(palette); i += 2 {
		end := i + 2
		if end > len(palette) {
			end = len(palette)
		}
		row := []tgbotapi.InlineKeyboardButton{}
		for _, c := range palette[i:end] {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				c.Label, "cl_setcol_"+c.Hex+"_"+id))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Назад", "⬅️ Back"), calOpen+id)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// CalendarConfirm asks before something irreversible, naming the calendar.
//
// 🔴 The name is in the button, not only in the text above it. A bare «Да» is
// answerable without reading, which is the whole failure mode a confirmation
// exists to prevent.
func CalendarConfirm(lang i18n.Lang, id, name, action string) tgbotapi.InlineKeyboardMarkup {
	var label string
	if action == "leave" {
		label = i18n.T(lang, "🚪 Да, выйти из «"+name+"»", "🚪 Yes, leave «"+name+"»")
	} else {
		label = i18n.T(lang, "🗑 Да, удалить «"+name+"»", "🗑 Yes, delete «"+name+"»")
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "cl_"+action+"ok_"+id)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ Отмена", "⬅️ Cancel"), calOpen+id)))
}

// FeedbackKinds is the two things a person might want to say.
func FeedbackKinds(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🐞 Ошибка", "🐞 Bug"), "fb_bug"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "💡 Идея", "💡 Idea"), "fb_idea")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu")))
}

// WhatsNew shows the newest release and offers the older ones.
func WhatsNew(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📜 Прошлые версии", "📜 Older versions"), "whatsnew_all")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Меню", "« Menu"), "main_menu")))
}
