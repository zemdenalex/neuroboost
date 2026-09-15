package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/format"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/parse"
)

// The list half of creation. Denis, item 5: several events or tasks in one
// message, with day headers that carry downward.
//
// 🔴 One card for the whole list, not one per entry. Eight confirmations in a
// row is a worse experience than no confirmation at all, and it would teach
// exactly the habit the card exists to prevent: pressing ✅ without reading.

func listOf(h *Handler, chatID int64) ([]*draftState, bool) {
	us := h.store.GetOrCreate(chatID)
	list, ok := us.FlowData["list"].([]*draftState)
	return list, ok && len(list) > 0
}

// askListOrSingle puts the question and goes no further.
func (h *Handler) askListOrSingle(chatID int64, text string) {
	us := h.store.GetOrCreate(chatID)
	us.FlowData["raw"] = text
	us.FlowStep = "list:confirm"
	n := len(parse.Entries(text))
	h.sendHTMLWithKeyboard(chatID,
		fmt.Sprintf("Это одна запись или список из %d?\n\n<i>Одной записью название будет целиком, со всеми строками.</i>", n),
		keyboards.ListConfirm(n))
}

// buildList turns the remembered text into one draft per entry.
func (h *Handler) buildList(chatID int64) []*draftState {
	us := h.store.GetOrCreate(chatID)
	raw, _ := us.FlowData["raw"].(string)
	now := time.Now().In(h.location())

	parsed := parse.ParseEventList(raw, now)
	list := make([]*draftState, 0, len(parsed))
	for _, p := range parsed {
		list = append(list, &draftState{Title: p.Title, D: p.Draft})
	}
	return list
}

// renderList writes the whole list as one card.
//
// Entries are numbered because the edit step asks for a number, and a list
// whose numbering exists only in the keyboard is a list nobody can point at.
func renderList(list []*draftState, now time.Time) string {
	var b strings.Builder
	b.WriteString("📋 <b>Список из " + strconv.Itoa(len(list)) + "</b>\n\n")
	for i, st := range list {
		fmt.Fprintf(&b, "<b>%d.</b> %s\n", i+1, strings.ReplaceAll(renderDraft(*st, now), "\n", "\n    "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (h *Handler) showList(chatID int64, messageID int) {
	list, ok := listOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return
	}
	us := h.store.GetOrCreate(chatID)
	us.FlowStep = "list"
	delete(us.FlowData, "draft")
	h.editOrSend(chatID, messageID, renderList(list, time.Now().In(h.location())), keyboards.ListCard(len(list)))
}

// createList writes every entry, and says by name which ones did not make it.
//
// 🔴 «Создано 5 из 8» is useless: the three that failed are the only ones the
// user now has to do something about, and a count does not say which they are.
func (h *Handler) createList(chatID int64, messageID int) {
	list, ok := listOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return
	}

	var made, failed []string
	var left []*draftState
	for _, st := range list {
		if err := h.createOne(chatID, *st); err != nil {
			failed = append(failed, format.Escape(st.Title)+" — "+format.Escape(err.Error()))
			left = append(left, st)
			continue
		}
		made = append(made, format.Escape(st.Title))
	}

	var b strings.Builder
	if len(made) > 0 {
		b.WriteString("✅ <b>Создано</b>\n• " + strings.Join(made, "\n• ") + "\n")
	}
	if len(failed) > 0 {
		b.WriteString("\n❌ <b>Не создано</b>\n• " + strings.Join(failed, "\n• ") +
			"\n\nОстались в черновике — нажми «Создать», чтобы попробовать ещё раз.")
	}

	us := h.store.GetOrCreate(chatID)
	if len(left) == 0 {
		h.store.ClearFlow(chatID)
		h.editOrSend(chatID, messageID, b.String(), keyboards.AgendaActions())
		return
	}
	// Keep what failed, so a retry is one button and not a retyped block.
	us.FlowData["list"] = left
	h.editOrSend(chatID, messageID, b.String(), keyboards.ListCard(len(left)))
}

// handleListCallback answers the list buttons. Returns false for anything it
// does not own.
func (h *Handler) handleListCallback(chatID int64, messageID int, data string) bool {
	us := h.store.GetOrCreate(chatID)

	switch {
	case data == "dr_one":
		raw, _ := us.FlowData["raw"].(string)
		st := h.parseIntoDraft(chatID, strings.ReplaceAll(raw, "\n", " "))
		us.FlowData["draft"] = &st
		delete(us.FlowData, "list")
		h.showCard(chatID, messageID)
		return true

	case data == "dr_many":
		list := h.buildList(chatID)
		if len(list) == 0 {
			h.lostDraft(chatID)
			return true
		}
		us.FlowData["list"] = list
		h.showList(chatID, messageID)
		return true

	case data == "dr_list":
		h.showList(chatID, messageID)
		return true

	case data == "dr_makeall":
		h.createList(chatID, messageID)
		return true

	case data == "dr_pick":
		list, ok := listOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return true
		}
		h.editOrSend(chatID, messageID, "Какую строку изменить?", keyboards.ListPick(len(list)))
		return true

	case strings.HasPrefix(data, "dr_item_"):
		list, ok := listOf(h, chatID)
		if !ok {
			h.lostDraft(chatID)
			return true
		}
		i, err := strconv.Atoi(strings.TrimPrefix(data, "dr_item_"))
		if err != nil || i < 0 || i >= len(list) {
			h.showList(chatID, messageID)
			return true
		}
		// The whole single-entry machinery works on FlowData["draft"], so
		// pointing it at this element makes every edit button apply to it
		// without a second implementation.
		us.FlowData["draft"] = list[i]
		h.showCard(chatID, messageID)
		return true
	}
	return false
}
