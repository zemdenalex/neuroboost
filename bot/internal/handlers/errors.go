package handlers

import (
	"log"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
)

// One sentence a person can act on, in their language.
//
// 🔴 Written after 21.09, when the chat showed Denis this:
//
//	❌ Не получилось: API error 400: {"error":{"code":"NOT_AN_OCCURRENCE", …}}
//
// and, an hour later, during the М9 outage:
//
//	❌ Не удалось создать: Post "https://dev.neuroboost.website/api/tasks":
//	context deadline exceeded (Client.Timeout exceeded while awaiting headers)
//
// Both are true. Neither tells the reader whether to press again, fix the
// input, or go and have a coffee — and the second one is the diagnosis we
// ourselves spent an hour on. The detail belongs in the log, where it is kept;
// the chat gets the answer to «what do I do now».
func (h *Handler) errorText(chatID int64, err error) string {
	if err == nil {
		return ""
	}

	// Kept here, and only here — through the redactor.
	//
	// 🔴 Gotcha 14: a Telegram error carries the URL it was talking to, and
	// that URL contains the bot token. This is a NEW logging site, and the
	// lesson from 16.08 is that redaction belongs where the secret can be
	// born, not on whoever happens to read the file later.
	log.Printf("chat %d: %s", chatID, logsafe.Redact(err))

	switch api.CodeOf(err) {
	case "NOT_AN_OCCURRENCE":
		// 🔴 Since PressedDay this is nearly unreachable for a button — it now
		// resolves to the next day of the series. What is left is a series that
		// has genuinely run out, and that is what this says.
		return h.t(chatID,
			"Эта серия уже закончилась — закрывать в ней нечего.",
			"This series has already ended — there is no day left to close.")
	case "NOT_RECURRING":
		return h.t(chatID,
			"Эта задача не повторяется, так что «на сегодня» к ней не применить.",
			"This task does not repeat, so there is no single day to close.")
	case "NOT_FOUND":
		return h.t(chatID,
			"Не нашёл — возможно, это уже удалено.",
			"Not found — it may already be gone.")
	case "NOT_AUTHENTICATED", "INVALID_TOKEN":
		return h.t(chatID,
			"Сессия истекла. Нажми /start — я войду заново.",
			"The session expired. Press /start and I'll sign in again.")
	case "VALIDATION_ERROR", "INVALID_REQUEST", "INVALID_DATE", "INVALID_REPEAT":
		return h.t(chatID,
			"Так не получится — проверь, что написано.",
			"That didn't parse — check what you wrote.")
	case "FORBIDDEN", "NO_ACCESS":
		return h.t(chatID,
			"Сюда нет доступа — календарь чужой и только для чтения.",
			"No access here — that calendar is somebody else's, read-only.")
	}

	// Not one of ours: a timeout, a dead host, a proxy page. On 21.09 this was
	// the whole machine being cut off for 75 minutes, and the chat said
	// «context deadline exceeded» five times in a row.
	return h.t(chatID,
		"Сервер не ответил. Ничего не потеряно — попробуй ещё раз через минуту.",
		"The server didn't answer. Nothing is lost — try again in a minute.")
}
