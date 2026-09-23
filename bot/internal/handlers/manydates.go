package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Several dates in one line — «созвон 14.10 16.10».
//
// 🔴 Denis, 17.09: «если просто 2 даты написано, то он должен спросить, имел ли
// я в виду с … по …, или одну из этих дат, или обе… и если дат больше чем 2, то
// предложить промежуток от самой ранней до самой поздней или одинаковые события
// в эти даты с возможностью множественного выбора». Two answers, both explicit:
// one event across the span, or one event per ticked date.

// allDates is every date the line named, earliest first.
func allDates(st *draftState) []time.Time {
	dates := append([]time.Time{st.D.Day}, st.D.MoreDays...)
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func dateLabels(dates []time.Time) []string {
	out := make([]string, 0, len(dates))
	for _, d := range dates {
		out = append(out, d.Format("02.01"))
	}
	return out
}

func (h *Handler) askManyDates(chatID int64, messageID int) {
	st, ok := draftOf(h, chatID)
	if !ok {
		h.lostDraft(chatID)
		return
	}
	labels := dateLabels(allDates(st))
	h.store.GetOrCreate(chatID).FlowStep = "dates"

	h.editOrSend(chatID, messageID, fmt.Sprintf(h.t(chatID,
		"Нашёл несколько дат: %s.\n\nОдно событие на весь промежуток или по одному на каждую выбранную дату?",
		"Several dates here: %s.\n\nOne event across the whole span, or one on each date you tick?"),
		strings.Join(labels, ", ")),
		keyboards.ManyDates(h.lang(chatID), labels[0], labels[len(labels)-1]))
}

// handleDatesCallback answers the dr_d* buttons. Returns false for anything else.
func (h *Handler) handleDatesCallback(chatID int64, messageID int, data string) bool {
	st, ok := draftOf(h, chatID)
	if !ok {
		return false
	}
	us := h.store.GetOrCreate(chatID)
	dates := allDates(st)

	switch {
	case data == "dr_dspan":
		st.D.Day, st.D.EndDay = dates[0], dates[len(dates)-1]
		st.D.MoreDays = nil
		if !st.D.HasTime {
			st.D.AllDay = true
		}
		h.showCard(chatID, messageID)
		return true

	case data == "dr_dpick":
		chosen, okChosen := us.FlowData["dates_chosen"].([]bool)
		if !okChosen || len(chosen) != len(dates) {
			// Everything ticked to begin with: the user wrote every one of
			// these dates.
			chosen = make([]bool, len(dates))
			for i := range chosen {
				chosen[i] = true
			}
			us.FlowData["dates_chosen"] = chosen
		}
		h.showDatePicker(chatID, messageID, dates, chosen)
		return true

	case strings.HasPrefix(data, "dr_dtog_"):
		i, err := strconv.Atoi(strings.TrimPrefix(data, "dr_dtog_"))
		chosen, _ := us.FlowData["dates_chosen"].([]bool)
		if err != nil || i < 0 || i >= len(chosen) {
			return true
		}
		chosen[i] = !chosen[i]
		h.showDatePicker(chatID, messageID, dates, chosen)
		return true

	case data == "dr_dmake":
		chosen, _ := us.FlowData["dates_chosen"].([]bool)
		list := make([]*draftState, 0, len(dates))
		for i, day := range dates {
			if i < len(chosen) && !chosen[i] {
				continue
			}
			one := *st
			one.D.Day, one.D.MoreDays, one.D.EndDay = day, nil, time.Time{}
			list = append(list, &one)
		}
		switch len(list) {
		case 0:
			h.editOrSend(chatID, messageID, h.t(chatID,
				"Ни одной даты не отмечено.", "No dates are ticked."),
				keyboards.DatePicker(h.lang(chatID), dateLabels(dates), chosen))
		case 1:
			// One date ticked is an ordinary single event.
			us.FlowData["draft"] = list[0]
			h.showCard(chatID, messageID)
		default:
			// The list card is the same one a multi-line block gets: one
			// confirmation for all of them.
			delete(us.FlowData, "draft")
			us.FlowData["list"] = list
			h.showList(chatID, messageID)
		}
		return true
	}
	return false
}

func (h *Handler) showDatePicker(chatID int64, messageID int, dates []time.Time, chosen []bool) {
	h.editOrSend(chatID, messageID, h.t(chatID,
		"Отметь даты, на каждую создам такое же событие.",
		"Tick the dates; I will create the same event on each."),
		keyboards.DatePicker(h.lang(chatID), dateLabels(dates), chosen))
}
