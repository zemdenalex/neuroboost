package handlers

import (
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// ⚙️ Настройки → 🔤 Ключевые слова.
//
// Denis, 15.09: «Ключевые слова не обязательно теги, они должны быть как
// триггеры… при добавлении слов выбирается что оно обозначает/заменяет то есть
// какую характеристику (тег/дата/цвет/календарь и тд)».
//
// Adding a word takes three steps — the word, the characteristic, the value —
// and the third is skipped for the two characteristics that have no value
// («весь день», «задача»).

func (h *Handler) handleKeywords(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	h.store.ClearFlow(chatID)

	vocab, err := h.api.BotKeywords(us.AuthToken)
	if err != nil {
		h.editOrSend(chatID, messageID, "Не удалось прочитать настройки.", keyboards.SettingsMenu())
		return
	}

	var b strings.Builder
	b.WriteString("🔤 <b>Свои слова</b>\n\n")
	if len(vocab) == 0 {
		b.WriteString("Пока ни одного.\n\nСлово, написанное в строке создания, задаёт " +
			"характеристику события — тег, цвет, календарь, дату — и в название не попадает.")
	}

	words := make([]string, 0, len(vocab))
	for w := range vocab {
		words = append(words, w)
	}
	// Sorted, so the list and its buttons stay in the same order between
	// renders: map iteration would reshuffle them, and the delete button under
	// a word would move while someone was reaching for it.
	sort.Strings(words)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, w := range words {
		kw := vocab[w]
		field, known := parse.FieldByName(kw.Field)
		label := kw.Field
		if known {
			label = parse.FieldLabel(field)
		}
		line := "• <code>" + format.Escape(w) + "</code> → " + format.Escape(label)
		if kw.Value != "" {
			line += " " + format.Escape(kw.Value)
		}
		b.WriteString(line + "\n")

		// 🔴 callback_data is capped at 64 BYTES and a Cyrillic letter costs
		// two. A button over the cap makes Telegram reject the WHOLE keyboard,
		// so the screen would fail rather than that one row — a word too long
		// to address gets no delete button, and the list says why.
		if data := "kw_del_" + w; len(data) <= 64 {
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("🗑 "+w, data),
			})
		} else {
			b.WriteString("   <i>(слишком длинное, чтобы удалить кнопкой)</i>\n")
		}
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "kw_add"),
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "settings_menu"),
	})

	h.editOrSend(chatID, messageID, b.String(), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) startKeywordFlow(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "keyword"
	us.FlowStep = "word"
	us.FlowData = map[string]any{}
	h.editOrSend(chatID, messageID,
		"Напиши <b>одно слово</b>. Потом выберешь, что оно означает.\n\n"+
			"Например: <code>созвон</code> — и дальше «Календарь», «Работа».",
		keyboards.TriggerCancel())
}

// handleKeywordInput takes the word, and later the value when the chosen
// characteristic needs one typed.
func (h *Handler) handleKeywordInput(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	switch us.FlowStep {
	case "word":
		word := strings.ToLower(strings.TrimSpace(text))
		if word == "" || strings.ContainsAny(word, " \n\t") {
			// ⚠ One word, because matching is per token: a two-word trigger
			// would never match anything and would look broken rather than
			// unsupported.
			h.sendText(chatID, "Нужно ровно одно слово, без пробелов.")
			return
		}
		us.FlowData["word"] = word
		us.FlowStep = "field"

		labels := make([]string, 0, len(parse.TriggerFields))
		names := make([]string, 0, len(parse.TriggerFields))
		for _, tf := range parse.TriggerFields {
			labels = append(labels, tf.Label)
			names = append(names, tf.Name)
		}
		h.sendHTMLWithKeyboard(chatID,
			"Что означает <code>"+format.Escape(word)+"</code>?",
			keyboards.TriggerFieldPicker(labels, names))

	case "value":
		field, _ := us.FlowData["field"].(string)
		h.saveKeyword(chatID, 0, field, strings.TrimSpace(text))

	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "Что-то пошло не так.", keyboards.SettingsMenu())
	}
}

// handleKeywordCallback answers the field and value pickers. Returns false for
// anything it does not own.
func (h *Handler) handleKeywordCallback(chatID int64, messageID int, data string) bool {
	us := h.store.GetOrCreate(chatID)

	switch {
	case strings.HasPrefix(data, "kwf_"):
		name := strings.TrimPrefix(data, "kwf_")
		field, ok := parse.FieldByName(name)
		if !ok || us.CurrentFlow != "keyword" {
			h.handleKeywords(chatID, messageID)
			return true
		}
		us.FlowData["field"] = name
		word, _ := us.FlowData["word"].(string)

		switch field {
		case parse.FieldAllDay, parse.FieldKind:
			// No value to give: the characteristic IS the answer.
			h.saveKeyword(chatID, messageID, name, "")
		case parse.FieldColour:
			h.editOrSend(chatID, messageID, "Какой цвет у <code>"+format.Escape(word)+"</code>?",
				keyboards.TriggerColourPicker())
		case parse.FieldRepeat:
			h.editOrSend(chatID, messageID, "Как часто повторять?", keyboards.TriggerFreqPicker())
		default:
			us.FlowStep = "value"
			h.editOrSend(chatID, messageID, valuePrompt(field, word), keyboards.TriggerCancel())
		}
		return true

	case strings.HasPrefix(data, "kwv_col_"):
		h.saveKeyword(chatID, messageID, "colour", strings.TrimPrefix(data, "kwv_col_"))
		return true

	case strings.HasPrefix(data, "kwv_freq_"):
		h.saveKeyword(chatID, messageID, "repeat", "FREQ="+strings.TrimPrefix(data, "kwv_freq_"))
		return true
	}
	return false
}

func valuePrompt(field parse.Field, word string) string {
	switch field {
	case parse.FieldTag:
		return "Каким тегом? Напиши тег — или отправь <code>" + format.Escape(word) + "</code>, чтобы тег назывался так же."
	case parse.FieldCalendar:
		return "В какой календарь? Напиши его название ровно так, как оно в приложении."
	case parse.FieldDay:
		return "Какой день? Например: <code>завтра</code>, <code>среда</code>, <code>следующий понедельник</code>.\n\n" +
			"⚠ Сохраню фразу, а не дату — «завтра» останется завтрашним днём и через неделю."
	case parse.FieldTime:
		return "Какое время? Например: <code>14:00</code> или <code>14:00-15:30</code>."
	}
	return "Какое значение?"
}

func (h *Handler) saveKeyword(chatID int64, messageID int, field, value string) {
	us := h.store.GetOrCreate(chatID)
	word, _ := us.FlowData["word"].(string)
	if word == "" || field == "" {
		h.handleKeywords(chatID, messageID)
		return
	}

	if err := h.api.SetBotKeyword(us.AuthToken, word, field, value); err != nil {
		// 🔴 Nothing was written — SetBotKeyword refuses to patch a blob it
		// could not read first, because merging onto an empty map erases the
		// rest of the user's settings (gotcha 21).
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, "❌ Не удалось сохранить: "+format.Escape(err.Error())+
			"\n\nНастройки при этом не изменились.", keyboards.SettingsMenu())
		return
	}

	h.store.ClearFlow(chatID)
	h.handleKeywords(chatID, messageID)
}

func (h *Handler) handleKeywordDelete(chatID int64, messageID int, word string) {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetBotKeyword(us.AuthToken, word, "", ""); err != nil {
		h.editOrSend(chatID, messageID, "❌ Не удалось удалить: "+format.Escape(err.Error()),
			keyboards.SettingsMenu())
		return
	}
	h.handleKeywords(chatID, messageID)
}
