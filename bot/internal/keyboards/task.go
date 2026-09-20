package keyboards

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
)

// TaskCard is what a parsed line offers.
//
// 🔴 ✅ Создать is FIRST and always enabled. Denis, 18.08: "all of them
// shouldn't be required to create the task". The wizard is an offer underneath
// it, never a gate in front of it.
//
// switchable adds «сделать событием / заметкой». It is on when the card came
// from a line the user typed, so the kind can still be changed without typing
// it again — Denis, 17.09: the word «задача» should cost one action, not two.
func TaskCard(lang i18n.Lang, switchable bool) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Создать", "✅ Create"), "nt_save"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.T(lang, "📝 Подробнее (по шагам)", "📝 More (step by step)"), "nt_wizard"),
		),
	}
	if switchable {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📅 Сделать событием", "📅 Make it an event"), "qa_event"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "📝 Сделать заметкой", "📝 Make it a note"), "qa_note"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "🗑 Удалить черновик", "🗑 Delete draft"), "main_menu"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// WizardPriority, WizardDue and WizardEstimate are the three steps behind
// "📝 Подробнее". Each carries wizardEscapes() — Denis, 18.08: nothing is
// required, every step must offer both "skip this field" and "save now".
//
// Each also takes what the typed line (or an earlier wizard step) already
// answered for this field — nil when nothing is known yet. Spec, part 3: "a
// step whose value is already known shows it as the current value and lets
// you replace it." The matching button gets a "✓ " prefix; every button stays
// live, because replacing the known value is exactly what a tap here does.
func WizardPriority(lang i18n.Lang, current *int) tgbotapi.InlineKeyboardMarkup {
	label := func(p int) string {
		text := format.PriorityEmoji(p) + " " + format.PriorityLabel(lang, p)
		if current != nil && *current == p {
			return "✓ " + text
		}
		return text
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label(1), "nt_p_1"),
			tgbotapi.NewInlineKeyboardButtonData(label(2), "nt_p_2"),
			tgbotapi.NewInlineKeyboardButtonData(label(3), "nt_p_3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label(4), "nt_p_4"),
			tgbotapi.NewInlineKeyboardButtonData(label(5), "nt_p_5"),
			tgbotapi.NewInlineKeyboardButtonData(label(0), "nt_p_0"),
		),
		wizardEscapes(lang),
	)
}

// current is the offset ("0"/"1"/"7") the known due date matches, or "" when
// nothing is known, or when it is known but does not land on one of these
// three quick choices (the step's own text carries the exact date in that
// case — see newtask.go: wizardStepText).
func WizardDue(lang i18n.Lang, current string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		dueRow(lang, "nt_d_", current),
		wizardEscapes(lang),
	)
}

// current is the known estimate in minutes, or nil when nothing is known.
func WizardEstimate(lang i18n.Lang, current *int) tgbotapi.InlineKeyboardMarkup {
	marked := ""
	if current != nil {
		marked = strconv.Itoa(*current)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		estimateRow(lang, "nt_e_", marked),
		wizardEscapes(lang),
	)
}

// dueRow and estimateRow are the wizard's two data rows, factored out so the
// existing-task card (keyboards.go: TaskDue, TaskEstimate) offers exactly the
// same question and the same values under a different footer, instead of a
// second copy of the labels and offsets that drifts the first time one of
// them is edited. Only the callback prefix varies between those callers, which
// pass marked = "" — they are editing, not resuming a wizard, so there is no
// "current step value" to highlight.
func dueRow(lang i18n.Lang, prefix, marked string) []tgbotapi.InlineKeyboardButton {
	label := func(text, val string) string {
		if marked != "" && marked == val {
			return "✓ " + text
		}
		return text
	}
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "Сегодня", "Today"), "0"), prefix+"0"),
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "Завтра", "Tomorrow"), "1"), prefix+"1"),
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "Через неделю", "In a week"), "7"), prefix+"7"),
	)
}

func estimateRow(lang i18n.Lang, prefix, marked string) []tgbotapi.InlineKeyboardButton {
	label := func(text, val string) string {
		if marked != "" && marked == val {
			return "✓ " + text
		}
		return text
	}
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "15м", "15m"), "15"), prefix+"15"),
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "30м", "30m"), "30"), prefix+"30"),
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "1ч", "1h"), "60"), prefix+"60"),
		tgbotapi.NewInlineKeyboardButtonData(label(i18n.T(lang, "2ч", "2h"), "120"), prefix+"120"),
	)
}

// wizardEscapes is the row that makes "nothing is required" true. It is one
// function so a new step cannot be added without it.
func wizardEscapes(lang i18n.Lang) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "⏭ Пропустить", "⏭ Skip"), "nt_skip"),
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "✅ Создать сейчас", "✅ Create now"), "nt_save"),
	)
}

// RepeatCodes maps the wizard's repeat buttons to rules the API parses.
//
// One-letter codes on the wire, rules in one table: the handler reads this map
// rather than switching on the letters, so a button and its rule cannot be
// edited apart. "n" is «не повторять» — a real answer, which clears a repeat the
// typed line may have set.
var RepeatCodes = map[string]string{
	"n":  "",
	"d":  "FREQ=DAILY",
	"d2": "FREQ=DAILY;INTERVAL=2",
	"w":  "FREQ=WEEKLY",
	"m":  "FREQ=MONTHLY",
}

// WizardRepeat asks how often a task comes back. current is the rule already
// known ("" for none), marked the way the other steps mark their value.
//
// ⚠ No «свой вариант» button, deliberately: a custom period is TYPED — «раз в 3
// дня», «каждые 2 недели» — and the step says so. The parser has understood
// those since v0.4.11.2; a button that opens a second question to collect the
// same sentence is one more step, and Denis's rule is that corrections go toward
// fewer.
func WizardRepeat(lang i18n.Lang, current string) tgbotapi.InlineKeyboardMarkup {
	btn := func(text, code string) tgbotapi.InlineKeyboardButton {
		if RepeatCodes[code] == current && (current != "" || code == "n") {
			text = "✓ " + text
		}
		return tgbotapi.NewInlineKeyboardButtonData(text, "nt_r_"+code)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			btn(i18n.T(lang, "Каждый день", "Every day"), "d"),
			btn(i18n.T(lang, "Через день", "Every other day"), "d2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			btn(i18n.T(lang, "Каждую неделю", "Every week"), "w"),
			btn(i18n.T(lang, "Каждый месяц", "Every month"), "m"),
		),
		tgbotapi.NewInlineKeyboardRow(
			btn(i18n.T(lang, "Не повторять", "Do not repeat"), "n"),
		),
		wizardEscapes(lang),
	)
}
