package handlers

import (
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/parse"
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
		h.editOrSend(chatID, messageID, h.t(chatID, "Не удалось прочитать настройки.", "Could not read your settings."), keyboards.SettingsMenu(h.lang(chatID)))
		return
	}

	var b strings.Builder
	b.WriteString(h.t(chatID, "🔤 <b>Свои слова</b>\n\n", "🔤 <b>Your words</b>\n\n"))
	if len(vocab) == 0 {
		b.WriteString(h.t(chatID, "Пока ни одного.\n\nСлово, написанное в строке создания, задаёт характеристику события (тег, цвет, календарь, дату) и в название не попадает.", "None yet.\n\nA word written in a creation line sets one characteristic of the event (a tag, a colour, a calendar, a date) and stays out of the title."))
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
			b.WriteString(h.t(chatID, "   <i>(слишком длинное, чтобы удалить кнопкой)</i>\n", "   <i>(too long to delete with a button)</i>\n"))
		}
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(h.t(chatID, "➕ Добавить", "➕ Add"), "kw_add"),
		tgbotapi.NewInlineKeyboardButtonData(h.t(chatID, "⬅️ Назад", "⬅️ Back"), "settings_menu"),
	})

	h.editOrSend(chatID, messageID, b.String(), tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) startKeywordFlow(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	us.CurrentFlow = "keyword"
	us.FlowStep = "word"
	us.FlowData = map[string]any{}
	h.editOrSend(chatID, messageID,
		h.t(chatID, "Напиши <b>одно слово</b>. Потом выберешь, что оно означает.\n\nНапример: <code>созвон</code>, и дальше «Календарь», «Работа».", "Write <b>one word</b>. Then you'll pick what it means.\n\nFor example: <code>созвон</code>, then «Calendar», «Работа»."),
		keyboards.TriggerCancel(h.lang(chatID)))
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
			h.sendText(chatID, h.t(chatID, "Нужно ровно одно слово, без пробелов.", "One word exactly, no spaces."))
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
			h.t(chatID, "Что означает <code>", "What does <code>")+format.Escape(word)+h.t(chatID, "</code>?", "</code> mean?"),
			keyboards.TriggerFieldPicker(h.lang(chatID), labels, names))

	case "value":
		field, _ := us.FlowData["field"].(string)
		h.saveKeyword(chatID, 0, field, strings.TrimSpace(text))

	default:
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, h.t(chatID, "Что-то пошло не так.", "Something went wrong."), keyboards.SettingsMenu(h.lang(chatID)))
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
			h.editOrSend(chatID, messageID, h.t(chatID, "Какой цвет у <code>", "What colour is <code>")+format.Escape(word)+h.t(chatID, "</code>?", "</code>?"),
				keyboards.TriggerColourPicker(h.lang(chatID)))
		case parse.FieldRepeat:
			h.editOrSend(chatID, messageID, h.t(chatID, "Как часто повторять?", "How often should it repeat?"), keyboards.TriggerFreqPicker(h.lang(chatID)))
		default:
			us.FlowStep = "value"
			h.editOrSend(chatID, messageID, valuePrompt(h.lang(chatID), field, word), keyboards.TriggerCancel(h.lang(chatID)))
		}
		return true

	case strings.HasPrefix(data, "kwv_col_"):
		h.saveKeyword(chatID, messageID, "colour", strings.TrimPrefix(data, "kwv_col_"))
		return true

	case strings.HasPrefix(data, "kwv_freq_"):
		h.saveKeyword(chatID, messageID, "repeat", parse.FreqRule(strings.TrimPrefix(data, "kwv_freq_")))
		return true
	}
	return false
}

func valuePrompt(lang i18n.Lang, field parse.Field, word string) string {
	switch field {
	case parse.FieldTag:
		return i18n.T(lang, "Каким тегом? Напиши тег или отправь <code>", "Which tag? Write it or send <code>") + format.Escape(word) + i18n.T(lang, "</code>, чтобы тег назывался так же.", "</code> to name the tag after the word.")
	case parse.FieldCalendar:
		return i18n.T(lang, "В какой календарь? Напиши его название ровно так, как оно в приложении.", "Which calendar? Write its name exactly as it is in the app.")
	case parse.FieldDay:
		return i18n.T(lang, "Какой день? Например: <code>завтра</code>, <code>среда</code>, <code>следующий понедельник</code>.\n\n⚠ Сохраню фразу, а не дату: «завтра» останется завтрашним днём и через неделю.", "Which day? For example: <code>завтра</code>, <code>среда</code>, <code>следующий понедельник</code>.\n\n⚠ I store the phrase, not the date: «завтра» still means tomorrow a week from now.")
	case parse.FieldTime:
		return i18n.T(lang, "Какое время? Например: <code>14:00</code> или <code>14:00-15:30</code>.", "What time? For example: <code>14:00</code> or <code>14:00-15:30</code>.")
	}
	return i18n.T(lang, "Какое значение?", "What value?")
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
		h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не удалось сохранить: ", "❌ Could not save: ")+format.Escape(h.errorText(chatID, err))+
			h.t(chatID, "\n\nНастройки при этом не изменились.", "\n\nNothing was changed."), keyboards.SettingsMenu(h.lang(chatID)))
		return
	}

	h.store.ClearFlow(chatID)
	h.handleKeywords(chatID, messageID)
}

func (h *Handler) handleKeywordDelete(chatID int64, messageID int, word string) {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetBotKeyword(us.AuthToken, word, "", ""); err != nil {
		h.editOrSend(chatID, messageID, h.t(chatID, "❌ Не удалось удалить: ", "❌ Could not delete: ")+format.Escape(h.errorText(chatID, err)),
			keyboards.SettingsMenu(h.lang(chatID)))
		return
	}
	h.handleKeywords(chatID, messageID)
}
