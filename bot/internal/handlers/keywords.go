package handlers

import (
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// ⚙️ Настройки → 🔤 Ключевые слова.
//
// Denis, 15.09: «также должна быть возможность создавать свои слова для тегов».
// A word added here becomes a tag whenever it appears in a creation line.

func (h *Handler) handleKeywords(chatID int64, messageID int) {
	us := h.store.GetOrCreate(chatID)
	vocab, err := h.api.BotKeywords(us.AuthToken)
	if err != nil {
		h.editOrSend(chatID, messageID, "Не удалось прочитать настройки.", keyboards.SettingsMenu())
		return
	}

	var b strings.Builder
	b.WriteString("🔤 <b>Свои слова</b>\n\n")
	if len(vocab) == 0 {
		b.WriteString("Пока ни одного.\n\nСлово, написанное в строке создания, станет тегом " +
			"и в название не попадёт.")
	} else {
		b.WriteString("Слово → тег:\n")
	}

	words := make([]string, 0, len(vocab))
	for w := range vocab {
		words = append(words, w)
	}
	// Sorted, so the list and its buttons stay in the same order between
	// renders: map iteration would reshuffle them and the delete button under
	// a word would move while someone was reaching for it.
	sort.Strings(words)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, w := range words {
		b.WriteString("• <code>" + format.Escape(w) + "</code> → " + format.Escape(vocab[w]) + "\n")
		// 🔴 callback_data is capped at 64 BYTES, and a Cyrillic letter costs
		// two. A button over the cap is rejected by Telegram when the keyboard
		// is sent — the whole screen fails, not just that row — so a word too
		// long to address simply gets no delete button, and the list says so.
		if data := "kw_del_" + w; len(data) <= 64 {
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("🗑 "+w, data),
			})
		} else {
			b.WriteString("   <i>(слишком длинное, чтобы удалить кнопкой — замени его на пустой тег)</i>\n")
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
	h.editOrSend(chatID, messageID,
		"Напиши <code>слово = тег</code>.\n\nНапример: <code>спорт = здоровье</code>.\n"+
			"Одно слово — <code>спорт</code> — станет тегом с тем же именем.", keyboards.SettingsMenu())
}

func (h *Handler) handleKeywordInput(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)

	word, tag := splitKeyword(text)
	if word == "" {
		h.sendText(chatID, "Не понял. Напиши «слово = тег» или просто одно слово.")
		return
	}

	if err := h.api.SetBotKeyword(us.AuthToken, word, tag); err != nil {
		// 🔴 Nothing was written — SetBotKeyword refuses to patch a blob it
		// could not read first, because merging onto an empty map erases the
		// rest of the user's settings (gotcha 21).
		h.store.ClearFlow(chatID)
		h.sendHTMLWithKeyboard(chatID, "❌ Не удалось сохранить: "+format.Escape(err.Error())+
			"\n\nНастройки при этом не изменились.", keyboards.SettingsMenu())
		return
	}

	h.store.ClearFlow(chatID)
	h.sendHTML(chatID, "✅ <code>"+format.Escape(word)+"</code> → "+format.Escape(tag))
	h.handleKeywords(chatID, 0)
}

func (h *Handler) handleKeywordDelete(chatID int64, messageID int, word string) {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetBotKeyword(us.AuthToken, word, ""); err != nil {
		h.editOrSend(chatID, messageID, "❌ Не удалось удалить: "+format.Escape(err.Error()),
			keyboards.SettingsMenu())
		return
	}
	h.handleKeywords(chatID, messageID)
}

// splitKeyword reads «слово = тег», and «слово» on its own as a word that is
// its own tag.
//
// ⚠ Only the FIRST separator splits: a tag may contain an equals sign, and a
// word may not, so everything after the first one belongs to the tag.
func splitKeyword(text string) (string, string) {
	text = strings.TrimSpace(text)
	for _, sep := range []string{"=", "→", "->", ":"} {
		if i := strings.Index(text, sep); i >= 0 {
			word := strings.ToLower(strings.TrimSpace(text[:i]))
			tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(text[i:], sep)))
			if word == "" || tag == "" {
				return "", ""
			}
			return word, tag
		}
	}
	// One word, no separator: it tags itself. Two or more words with no
	// separator is ambiguous and is refused rather than guessed.
	if fields := strings.Fields(text); len(fields) == 1 {
		w := strings.ToLower(fields[0])
		return w, w
	}
	return "", ""
}
