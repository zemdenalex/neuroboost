package keyboards

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
)

// TaskCard is what a parsed line offers.
//
// 🔴 ✅ Создать is FIRST and always enabled. Denis, 18.08: "all of them
// shouldn't be required to create the task". The wizard is an offer underneath
// it, never a gate in front of it.
func TaskCard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Создать", "nt_save"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Подробнее (по шагам)", "nt_wizard"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "main_menu"),
		),
	)
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
func WizardPriority(current *int) tgbotapi.InlineKeyboardMarkup {
	label := func(p int) string {
		text := format.PriorityEmoji(p) + " " + format.PriorityLabel(p)
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
		wizardEscapes(),
	)
}

// current is the offset ("0"/"1"/"7") the known due date matches, or "" when
// nothing is known, or when it is known but does not land on one of these
// three quick choices (the step's own text carries the exact date in that
// case — see newtask.go: wizardStepText).
func WizardDue(current string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		dueRow("nt_d_", current),
		wizardEscapes(),
	)
}

// current is the known estimate in minutes, or nil when nothing is known.
func WizardEstimate(current *int) tgbotapi.InlineKeyboardMarkup {
	marked := ""
	if current != nil {
		marked = strconv.Itoa(*current)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		estimateRow("nt_e_", marked),
		wizardEscapes(),
	)
}

// dueRow and estimateRow are the wizard's two data rows, factored out so the
// existing-task card (keyboards.go: TaskDue, TaskEstimate) offers exactly the
// same question and the same values under a different footer, instead of a
// second copy of the labels and offsets that drifts the first time one of
// them is edited. Only the callback prefix varies between those callers, which
// pass marked = "" — they are editing, not resuming a wizard, so there is no
// "current step value" to highlight.
func dueRow(prefix, marked string) []tgbotapi.InlineKeyboardButton {
	label := func(text, val string) string {
		if marked != "" && marked == val {
			return "✓ " + text
		}
		return text
	}
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(label("Сегодня", "0"), prefix+"0"),
		tgbotapi.NewInlineKeyboardButtonData(label("Завтра", "1"), prefix+"1"),
		tgbotapi.NewInlineKeyboardButtonData(label("Через неделю", "7"), prefix+"7"),
	)
}

func estimateRow(prefix, marked string) []tgbotapi.InlineKeyboardButton {
	label := func(text, val string) string {
		if marked != "" && marked == val {
			return "✓ " + text
		}
		return text
	}
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(label("15м", "15"), prefix+"15"),
		tgbotapi.NewInlineKeyboardButtonData(label("30м", "30"), prefix+"30"),
		tgbotapi.NewInlineKeyboardButtonData(label("1ч", "60"), prefix+"60"),
		tgbotapi.NewInlineKeyboardButtonData(label("2ч", "120"), prefix+"120"),
	)
}

// wizardEscapes is the row that makes "nothing is required" true. It is one
// function so a new step cannot be added without it.
func wizardEscapes() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "nt_skip"),
		tgbotapi.NewInlineKeyboardButtonData("✅ Создать сейчас", "nt_save"),
	)
}
