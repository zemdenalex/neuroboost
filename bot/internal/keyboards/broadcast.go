package keyboards

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// BroadcastConfirm is under the dry run: sending is a second, separate press.
func BroadcastConfirm(lang i18n.Lang, version string, n int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf(i18n.T(lang, "📣 Отправить %d", "📣 Send %d"), n), "bc_go_"+version),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "❌ Отмена", "❌ Cancel"), "main_menu"),
		),
	)
}

// UpdatesOff sits under every broadcast (spec 21.09 §D1).
func UpdatesOff(lang i18n.Lang) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔕 Не присылать обновления", "🔕 Stop sending updates"), "upd_off")))
}

// UpdatesToggle is the Settings screen's one button — the opposite of now.
func UpdatesToggle(lang i18n.Lang, subscribed bool) tgbotapi.InlineKeyboardMarkup {
	btn := tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔕 Не присылать", "🔕 Stop sending"), "upd_off")
	if !subscribed {
		btn = tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🔔 Снова присылать", "🔔 Send again"), "upd_on")
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btn),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "« Настройки", "« Settings"), "settings_menu")),
	)
}
