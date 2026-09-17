package keyboards

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// QuickAddKind is the question a line typed from nowhere gets: what is it?
//
// Its prefix, qa_, belongs to quick add alone. It is asked before the card's
// dr_ callbacks, and a shared prefix would route by declaration order.
//
// taskFirst puts Task before Event when the line said «задача».
func QuickAddKind(lang i18n.Lang, taskFirst bool) tgbotapi.InlineKeyboardMarkup {
	event := tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Событие", "📅 Event"), "qa_event")
	task := tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📋 Задача", "📋 Task"), "qa_task")
	first := tgbotapi.NewInlineKeyboardRow(event, task)
	if taskFirst {
		first = tgbotapi.NewInlineKeyboardRow(task, event)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		first,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📝 Заметка", "📝 Note"), "qa_note"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "qa_cancel"),
		),
	)
}

// OnboardNext is where onboarding hands over: the three places a new user goes
// first, as buttons.
func OnboardNext(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "➕ Создать", "➕ Create"), "create_menu"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗓 Календарь", "🗓 Calendar"), "cal_open"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⚙️ Настройки", "⚙️ Settings"), "settings_menu"),
		),
	)
}

// GuideMore opens the full vocabulary under a short guide. The prefix is the
// guide's own: dr_ belongs to the confirmation card.
func GuideMore(lang i18n.Lang, kind string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📖 Все слова и примеры", "📖 All words and examples"), "guide_full_"+kind),
		),
	)
}

// BackToList is the way back to an open event list.
func BackToList(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⬅️ К списку", "⬅️ Back to list"), "dr_list"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Отменить", "🗑 Cancel"), "dr_cancel"),
		),
	)
}

// None is «no buttons» in a form Telegram accepts.
//
// 🔴 tgbotapi.NewInlineKeyboardMarkup() with no rows serialises as
// {"inline_keyboard":null}, and Telegram refuses it — «field "inline_keyboard"
// must be of type Array». Refused as an edit AND as the fallback send, so the
// screen simply never changed: «Другое…», «📖» and «Переписать» were dead on
// Denis's phone on 17.09 while their tests passed. An empty array is accepted,
// and on an edit it removes the old buttons.
func None() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
}
