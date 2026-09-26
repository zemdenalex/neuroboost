package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/release"
)

// The «что нового» broadcast and the way out of it (spec 21.09 §D).
//
// 🔴 21.09 the broadcast was a script on the bot host: it read the token, and
// reported «200: 1 · 403: 1 · сеть: 1» — a tally from which nobody could tell
// whom to retry without sending someone a duplicate. Now it is a command, only
// for an admin, that shows a dry run first and records every recipient's
// result; the next run skips whoever got it or blocked the bot.
//
// Denis 21.09: «только тем, кто пользовался ботом в последние 10 дней» and
// «надо будет сделать кнопку отписаться от обновлений».

// broadcastActiveDays is Denis's «последние 10 дней».
const broadcastActiveDays = 10

// broadcastText is one release's message in the recipient's language — the
// same notes as the «Что нового» screen, so the two never say different things.
func broadcastText(lang i18n.Lang, n release.Note) string {
	body := n.RU
	if lang == i18n.EN {
		body = n.EN
	}
	return fmt.Sprintf(i18n.T(lang, "🆕 <b>Что нового в NeuroBoost</b> · %s\n\n%s", "🆕 <b>What's new in NeuroBoost</b> · %s\n\n%s"),
		n.Version, body)
}

// idHash is how a recipient appears in the log: never the id itself.
func idHash(tgID int64) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(tgID, 10)))
	return hex.EncodeToString(sum[:4])
}

// requireAdmin answers «недоступна» to anyone else, asking the service nothing.
func (h *Handler) requireAdmin(chatID int64) bool {
	us := h.store.GetOrCreate(chatID)
	if ok, err := h.api.IsAdmin(us.AuthToken); err != nil || !ok {
		h.sendText(chatID, h.t(chatID, "Команда недоступна.", "Command not available."))
		return false
	}
	return true
}

// handleBroadcastCommand is /broadcast: a dry run — who and what — and a button.
// Nothing is sent from here.
func (h *Handler) handleBroadcastCommand(chatID int64) {
	if !h.requireAdmin(chatID) {
		return
	}
	note := release.Latest()
	list, err := h.api.BroadcastRecipients(h.cfg.ServiceToken, note.Version, broadcastActiveDays)
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получил список: ", "❌ Could not get the list: ")+h.errorText(chatID, err))
		return
	}
	ru, en := 0, 0
	for _, r := range list {
		if i18n.Parse(r.Lang) == i18n.EN {
			en++
		} else {
			ru++
		}
	}
	text := fmt.Sprintf(h.t(chatID,
		"📣 <b>Рассылка %s</b>: пробный прогон, ничего не отправлено\n\nПолучателей: %d (ru %d · en %d): активные за %d дней, подписанные, ещё не получившие.\n\n<b>RU</b>\n%s\n\n<b>EN</b>\n%s",
		"📣 <b>Broadcast %s</b>: dry run, nothing sent\n\nRecipients: %d (ru %d · en %d): active in %d days, subscribed, not yet reached.\n\n<b>RU</b>\n%s\n\n<b>EN</b>\n%s"),
		note.Version, len(list), ru, en, broadcastActiveDays,
		broadcastText(i18n.RU, note), broadcastText(i18n.EN, note))
	h.sendHTMLWithKeyboard(chatID, text, keyboards.BroadcastConfirm(h.lang(chatID), note.Version, len(list)))
}

// handleBroadcastGo sends — only on the admin's second, separate press.
func (h *Handler) handleBroadcastGo(chatID int64, messageID int, version string) {
	if !h.requireAdmin(chatID) {
		return
	}
	note := release.Latest()
	if version != note.Version {
		// A button from an older dry run must not send a different release.
		h.editOrSend(chatID, messageID, h.t(chatID,
			"Эта кнопка от прошлой версии, запусти /broadcast заново.",
			"That button is from an older version; run /broadcast again."), keyboards.None())
		return
	}
	list, err := h.api.BroadcastRecipients(h.cfg.ServiceToken, note.Version, broadcastActiveDays)
	if err != nil {
		h.sendText(chatID, h.t(chatID, "❌ Не получил список: ", "❌ Could not get the list: ")+h.errorText(chatID, err))
		return
	}

	var delivered, blocked, missed, unrecorded int
	for _, r := range list {
		lang := i18n.Parse(r.Lang)
		msg := tgbotapi.NewMessage(r.TgID, broadcastText(lang, note))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = keyboards.BroadcastFooter(lang, note.OfferPriority)

		code := http200
		if _, err := h.bot.Send(msg); err != nil {
			code = 0 // the network, or anything without a Telegram code
			var te *tgbotapi.Error
			if errors.As(err, &te) {
				code = te.Code
			}
		}
		switch code {
		case http200:
			delivered++
		case 403:
			blocked++
		default:
			missed++
		}
		if err := h.api.MarkBroadcast(h.cfg.ServiceToken, r.TgID, note.Version, code); err != nil {
			// Unrecorded = would be sent again next run. Said, not hidden.
			unrecorded++
		}
		log.Printf("broadcast %s %s %d", note.Version, idHash(r.TgID), code)
	}

	if unrecorded > 0 {
		h.sendText(chatID, fmt.Sprintf(h.t(chatID,
			"⚠ Результат не записался у %d: повторный запуск может прислать им ещё раз.",
			"⚠ %d results were not recorded: a second run may send to them again."), unrecorded))
	}
	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
		"📣 <b>Рассылка %s отправлена</b>\n\nДошло: %d · заблокировали бота: %d · не дошло: %d\n\nПовторный /broadcast пошлёт только тем, кому не дошло.",
		"📣 <b>Broadcast %s sent</b>\n\nDelivered: %d · blocked the bot: %d · missed: %d\n\nRunning /broadcast again reaches only the missed ones."),
		note.Version, delivered, blocked, missed), keyboards.None())
}

const http200 = 200

// handleUpdatesSet turns the broadcast on or off for the person who pressed.
// on == "" only shows the screen.
func (h *Handler) handleUpdatesSet(chatID int64, messageID int, on string) {
	us := h.store.GetOrCreate(chatID)
	if on == "on" || on == "off" {
		if err := h.api.SetBotSetting(us.AuthToken, "updates", on); err != nil {
			h.sendText(chatID, h.t(chatID, "❌ Не сохранилось: ", "❌ Not saved: ")+h.errorText(chatID, err))
			return
		}
	}
	state, _ := h.api.BotSetting(us.AuthToken, "updates")
	subscribed := !strings.EqualFold(state, "off")
	text := h.t(chatID,
		"🔔 <b>Обновления</b>\n\nРаз в релиз я присылаю, что нового. Сейчас: <b>присылаю</b>.",
		"🔔 <b>Updates</b>\n\nOnce per release I send what is new. Now: <b>on</b>.")
	if !subscribed {
		text = h.t(chatID,
			"🔕 <b>Обновления</b>\n\nБольше не пришлю, что нового. Посмотреть можно в любой момент в «Что нового».",
			"🔕 <b>Updates</b>\n\nI will not send what is new any more. It is always there in «What's new».")
	}
	h.editOrSend(chatID, messageID, text, keyboards.UpdatesToggle(h.lang(chatID), subscribed))
}
