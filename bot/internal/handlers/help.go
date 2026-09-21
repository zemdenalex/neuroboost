package handlers

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/logsafe"
)

// handleHelp answers «ℹ️ Что это?» and its «« Назад» (spec 21.09 §C). Why the
// explanation is a message of its own rather than an edit: keyboards/help.go.
//
// It touches no flow. A help press in the middle of a path is a question about
// the path, not a step out of it.
func (h *Handler) handleHelp(chatID int64, messageID int, data string) {
	if data == "help_x" {
		// ⚠ Knowingly inert past 48 hours: Telegram refuses to delete an older
		// bot message, and the edit fallback editOrSend uses is bound by the same
		// window. Posting a new message instead would answer «Назад» with more
		// text, the opposite of what was pressed. An explanation that old is not
		// under anything the user is still doing.
		if _, err := h.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID)); err != nil {
			log.Printf("help in chat %d: could not delete the explanation: %s", chatID, logsafe.Redact(err))
		}
		return
	}
	text := helpText(h.lang(chatID), strings.TrimPrefix(data, "help_"))
	if text == "" {
		text = h.t(chatID, "Это объяснение устарело — открой экран заново.", "This explanation is out of date — open the screen again.")
	}
	h.sendHTMLWithKeyboard(chatID, text, keyboards.HelpBack(h.lang(chatID)))
}

// helpText is the one registry of screen explanations. Each answers three
// questions — what this is, what pressing does, how to undo it — and says
// nothing the code does not do: every «как отменить» here was checked against
// the handler or the schema it names.
//
// "" means no explanation; the registry test turns that red for any screen
// listed in keyboards.HelpScreens.
func helpText(lang i18n.Lang, screen string) string {
	switch screen {
	case keyboards.HelpToEvent:
		return i18n.T(lang,
			"ℹ️ <b>📅 В календарь</b>\n\nЗадача получает время: в календаре появляется событие, и напоминания приходят по нему.\n\nСпрошу, связать или перенести, потом когда. До подтверждения покажу, что чем станет.\n\n<b>Отменить:</b> ❌ Отмена на любом шаге ничего не меняет. Уже связал — удали событие, задача останется. Уже перенёс — в событии нажми ✅ Сделать задачей.",
			"ℹ️ <b>📅 To calendar</b>\n\nThe task gets a time: an event appears on the calendar, and reminders come from it.\n\nI ask whether to link or move, then when. Before you confirm, I show what becomes what.\n\n<b>Undo:</b> ❌ Cancel at any step changes nothing. Already linked — delete the event, the task stays. Already moved — on the event press ✅ Make it a task.")
	case keyboards.HelpToTask:
		return i18n.T(lang,
			"ℹ️ <b>✅ Сделать задачей</b>\n\nСобытие — когда, задача — что сделать. Из события получится задача с тем же названием.\n\nСпрошу, связать или перенести. Что не переедет в задачу, назову до подтверждения.\n\n<b>Отменить:</b> ❌ Отмена на любом шаге ничего не меняет. Уже перенёс — в задаче нажми 📅 В календарь.",
			"ℹ️ <b>✅ Make it a task</b>\n\nAn event is when, a task is what to do. The event becomes a task with the same title.\n\nI ask whether to link or move. Anything that will not carry over is named before you confirm.\n\n<b>Undo:</b> ❌ Cancel at any step changes nothing. Already moved — on the task press 📅 To calendar.")
	case keyboards.HelpLink:
		return i18n.T(lang,
			"ℹ️ <b>🔗 Связать или ➡️ Перенести</b>\n\nЗадача — что сделать. Событие — когда.\n\n🔗 <b>Связать</b> — живут оба: событие даёт время и напоминания, задача закрывается только своим «Готово».\n➡️ <b>Перенести</b> — остаётся одно из двух.\n\n<b>Отменить:</b> ❌ Отмена ничего не меняет. Связанное событие можно удалить — задача останется.",
			"ℹ️ <b>🔗 Link or ➡️ Move</b>\n\nA task is what to do. An event is when.\n\n🔗 <b>Link</b> — both live: the event carries the time and reminders; the task closes only with its own «Done».\n➡️ <b>Move</b> — one of the two remains.\n\n<b>Undo:</b> ❌ Cancel changes nothing. A linked event can be deleted — the task stays.")
	case keyboards.HelpRepeat:
		return i18n.T(lang,
			"ℹ️ <b>🔁 Повтор</b>\n\nЗадача, которая возвращается: каждый день, через день, раз в неделю или в месяц.\n\nВыберешь — задача станет серией: «Готово» отмечается за каждый день отдельно, и дни серии видны в статистике.\n\n<b>Отменить:</b> «Не повторять» — задача снова разовая.",
			"ℹ️ <b>🔁 Repeat</b>\n\nA task that comes back: every day, every other day, weekly or monthly.\n\nPick one and the task becomes a series: «Done» is marked for each day on its own, and the series days show in statistics.\n\n<b>Undo:</b> «Don't repeat» — the task is a one-off again.")
	case keyboards.HelpNag:
		return i18n.T(lang,
			"ℹ️ <b>🔔 Долбить</b>\n\nНапоминание, на которое не ответили, приходит снова — через выбранный промежуток.\n\nПовторяется, пока не нажмёшь кнопку под напоминанием, и не дольше конца того дня.\n\n<b>Отменить:</b> «Не долбить».",
			"ℹ️ <b>🔔 Nag</b>\n\nA reminder nobody answered comes again — after the gap you pick.\n\nIt repeats until you press a button under the reminder, and never past the end of that day.\n\n<b>Undo:</b> «Don't nag».")
	case keyboards.HelpPostpone:
		return i18n.T(lang,
			"ℹ️ <b>⏰ Отложить</b>\n\nДля повторяющейся задачи: пропустить ближайшие дни, не сдвигая ритм.\n\nВыберешь «2 дня» — два ближайших дня серии будут пропущены, дальше всё как было.\n\n<b>Отменить:</b> « Назад ничего не меняет. Уже пропущенные дни из бота не вернуть.",
			"ℹ️ <b>⏰ Postpone</b>\n\nFor a repeating task: skip the next days without moving the rhythm.\n\nPick «2 days» — the next two days of the series are skipped, and it goes on as before.\n\n<b>Undo:</b> « Back changes nothing. Days already skipped cannot be brought back from the bot.")
	case keyboards.HelpPriority:
		return i18n.T(lang,
			"ℹ️ <b>🔘 Символ приоритета</b>\n\nКак срочность видна в списках: кружками, точками с цифрой или только порядком.\n\nМеняется только вид. Сами задачи и их порядок остаются прежними.\n\n<b>Отменить:</b> выбери другой — здесь или потом в настройках.",
			"ℹ️ <b>🔘 Priority symbol</b>\n\nHow urgency shows in lists: circles, dots with a digit, or order only.\n\nOnly the look changes. The tasks and their order stay as they are.\n\n<b>Undo:</b> pick another — here or later in settings.")
	case keyboards.HelpUpdates:
		return i18n.T(lang,
			"ℹ️ <b>🔔 Обновления</b>\n\nРаз в релиз я присылаю, что нового в боте.\n\n🔕 — больше не присылаю. Напоминания о задачах и событиях приходят как прежде: это другое.\n\n<b>Отменить:</b> в настройках, «🔔 Снова присылать».",
			"ℹ️ <b>🔔 Updates</b>\n\nOnce per release I send what is new in the bot.\n\n🔕 — I stop sending them. Reminders about tasks and events keep coming: that is separate.\n\n<b>Undo:</b> in settings, «🔔 Send again».")
	}
	return ""
}
