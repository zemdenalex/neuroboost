package handlers

// What a month cell shows besides the date (spec 2026-09-22 §6; Denis 24.09:
// in D3). The date always stays: a cell without it cannot be picked.
const (
	cellBoth   = "both"
	cellColour = "colour"
	cellBar    = "bar"
)

var cellKinds = map[string]bool{cellBoth: true, cellColour: true, cellBar: true}

// calendarCell is the chat's choice, cached. A failed read is not cached and
// draws both, as before the choice existed.
func (h *Handler) calendarCell(chatID int64) string {
	us := h.store.GetOrCreate(chatID)
	if us.CalendarCellKnown {
		return us.CalendarCell
	}
	v, err := h.api.BotSetting(us.AuthToken, "calendar_cell")
	if err != nil {
		return cellBoth
	}
	if !cellKinds[v] {
		v = cellBoth
	}
	us.CalendarCell, us.CalendarCellKnown = v, true
	return v
}

// setCalendarCell writes the choice and keeps the cache in step.
func (h *Handler) setCalendarCell(chatID int64, v string) error {
	us := h.store.GetOrCreate(chatID)
	if err := h.api.SetBotSetting(us.AuthToken, "calendar_cell", v); err != nil {
		return err
	}
	us.CalendarCell, us.CalendarCellKnown = v, true
	return nil
}

// applyCell drops what the choice hides. With day tasks off there is no
// colour, and the choice row is hidden too: the bar stays whatever was chosen,
// or «colour only» would leave a bare calendar with no way back.
func applyCell(cell string, dayOn bool, levels map[string]int, colours map[string]string) (map[string]int, map[string]string) {
	if !dayOn {
		return levels, nil
	}
	switch cell {
	case cellColour:
		return nil, colours
	case cellBar:
		return levels, nil
	}
	return levels, colours
}
