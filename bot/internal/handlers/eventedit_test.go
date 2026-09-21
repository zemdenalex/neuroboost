package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

// Each prefix routes where it says.
//
// ⚠ This test was written believing the prefixes nested — that "evdy_" starts
// with "evd_" and the order of the checks was the whole correctness. Sabotage
// disproved it: the trailing underscore separates them, so reversing the order
// changes nothing. The test is kept because it pins the routing itself, and
// because the underscore that makes them safe is easy to drop.
func TestEventRoutePrefixesAreMutuallyExclusive(t *testing.T) {
	cases := []struct {
		data string
		kind string
		id   string
	}{
		{"event_pick", eventPick, ""},
		{"ev_abc-123", eventOpen, "abc-123"},
		{"eve_abc-123", eventEdit, "abc-123"},
		{"evd_abc-123", eventDeleteAsk, "abc-123"},
		{"evdy_abc-123", eventDelete, "abc-123"},
	}
	for _, c := range cases {
		kind, id, ok := eventRoute(c.data)
		if !ok {
			t.Errorf("%q: not routed", c.data)
			continue
		}
		if kind != c.kind {
			t.Errorf("%q routed to %q, want %q", c.data, kind, c.kind)
		}
		if id != c.id {
			t.Errorf("%q gave id %q, want %q", c.data, id, c.id)
		}
	}
}

// A prefix with nothing after it addresses no event. Routing it anyway would
// fetch "" and report the event missing, which reads as data loss rather than
// as a malformed button.
func TestEventRouteRefusesAnEmptyID(t *testing.T) {
	for _, data := range []string{"ev_", "eve_", "evd_", "evdy_", "ev", "other"} {
		if kind, id, ok := eventRoute(data); ok {
			t.Errorf("%q routed to %q/%q, want a refusal", data, kind, id)
		}
	}
}

func moscow(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skip("no tzdata for Europe/Moscow")
	}
	return loc
}

// 🔴 An event loaded into the card and saved again must come back as the SAME
// instants. The draft holds a local day plus offsets while the API speaks in
// instants, so the conversion crosses the timezone twice — and a mistake there
// shifts every event by the offset the moment it is opened, which reads as the
// bot quietly corrupting data.
func TestDraftFromEventRoundTripsTheTimes(t *testing.T) {
	loc := moscow(t)

	ev := api.Event{
		ID:       "e1",
		Title:    "Оркестр",
		StartsAt: "2026-09-16T11:00:00Z", // 14:00 Moscow
		EndsAt:   "2026-09-16T12:00:00Z", // 15:00 Moscow
	}

	st := draftFromEvent(ev, loc)
	if st.EventID != "e1" {
		t.Errorf("EventID = %q — without it the card would CREATE a second event", st.EventID)
	}
	if st.D.Day.Day() != 16 || st.D.Day.Month() != time.September {
		t.Errorf("day = %s, want 16 September local", st.D.Day.Format("2 Jan"))
	}
	if st.D.Start != 14*time.Hour {
		t.Errorf("Start = %v, want 14:00 local", st.D.Start)
	}
	if !st.D.HasEnd || st.D.End != 15*time.Hour {
		t.Errorf("End = %v (hasEnd %v), want 15:00 local", st.D.End, st.D.HasEnd)
	}

	start, end := draftBounds(st)
	if got := start.UTC().Format(time.RFC3339); got != ev.StartsAt {
		t.Errorf("start came back as %s, want %s", got, ev.StartsAt)
	}
	if got := end.UTC().Format(time.RFC3339); got != ev.EndsAt {
		t.Errorf("end came back as %s, want %s", got, ev.EndsAt)
	}
}

// Everything the card can show survives the load. A field dropped here is a
// field silently erased on the next save — the whole event is sent back.
func TestDraftFromEventKeepsEveryEditableField(t *testing.T) {
	loc := moscow(t)
	rrule := "FREQ=WEEKLY"

	ev := api.Event{
		ID:              "e2",
		CalendarID:      "cal-7",
		Title:           "Оркестр",
		StartsAt:        "2026-09-16T11:00:00Z",
		EndsAt:          "2026-09-16T12:00:00Z",
		Color:           "blue",
		Rrule:           &rrule,
		Tags:            []string{"музыка"},
		ReminderOffsets: []int{15},
	}

	st := draftFromEvent(ev, loc)
	if st.CalendarID != "cal-7" {
		t.Errorf("CalendarID = %q — saving would move the event to the personal calendar", st.CalendarID)
	}
	if st.D.Repeat != "FREQ=WEEKLY" {
		t.Errorf("Repeat = %q — saving would turn a weekly event into a one-off", st.D.Repeat)
	}
	if st.D.Colour != "blue" {
		t.Errorf("Colour = %q", st.D.Colour)
	}
	if len(st.D.Tags) != 1 || st.D.Tags[0] != "музыка" {
		t.Errorf("Tags = %v", st.D.Tags)
	}
	if st.ReminderOffsets == nil || len(*st.ReminderOffsets) != 1 || (*st.ReminderOffsets)[0] != 15 {
		t.Errorf("ReminderOffsets = %v — nil would silently fall back to the preset", st.ReminderOffsets)
	}
	// 🔴 RepeatAsked must stay false. An existing event already answered the
	// frequency question; asking again on every open would be the card refusing
	// to save until the user re-answered something they never changed.
	if st.D.RepeatAsked {
		t.Error("RepeatAsked is true on an event that already has a rule")
	}
}

func TestDraftFromEventHandlesAllDay(t *testing.T) {
	loc := moscow(t)
	st := draftFromEvent(api.Event{
		ID: "e3", Title: "Анализы", AllDay: true,
		StartsAt: "2026-09-16T21:00:00Z", EndsAt: "2026-09-17T21:00:00Z",
	}, loc)

	if !st.D.AllDay {
		t.Error("AllDay was lost")
	}
	if st.D.HasTime {
		t.Error("an all-day event came back with a time")
	}
	// 21:00Z is midnight in Moscow on the 17th — the local day, not the UTC one.
	if st.D.Day.Day() != 17 {
		t.Errorf("day = %s, want the 17th local", st.D.Day.Format("2 Jan"))
	}
}

// An event the card cannot place in time still opens. Refusing to render it
// would hide the one row the user most needs to fix.
func TestDraftFromEventSurvivesAnUnparseableTime(t *testing.T) {
	st := draftFromEvent(api.Event{ID: "e4", Title: "Что-то", StartsAt: "не время"}, time.UTC)
	if st.EventID != "e4" || st.Title != "Что-то" {
		t.Errorf("got %+v", st)
	}
	if st.D.HasDay || st.D.HasTime {
		t.Error("a day or time was invented from an unparseable value")
	}
}

func TestEventPickLabelShowsWhenAndWhat(t *testing.T) {
	loc := moscow(t)
	got := eventPickLabel(api.Event{Title: "Оркестр", StartsAt: "2026-09-16T11:00:00Z"}, loc)
	if got != "16.09 14:00 · Оркестр" {
		t.Errorf("label = %q", got)
	}

	// A title too long for a button is cut, not wrapped: Telegram wraps it
	// badly and an unreadable button is no better than no button.
	long := eventPickLabel(api.Event{
		Title:    "Очень длинное название события которое никуда не поместится",
		StartsAt: "2026-09-16T11:00:00Z",
	}, loc)
	if len([]rune(long)) > 45 {
		t.Errorf("label is %d runes: %q", len([]rune(long)), long)
	}
}

// The words on the card are the only thing telling the user whether they are
// about to add a second event or change the one they opened.
func TestEditingCardSaysSaveAndNeverCreate(t *testing.T) {
	kb := keyboards.EventEditor(i18n.RU, "e1", "e1")
	var labels, data []string
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			labels = append(labels, b.Text)
			if b.CallbackData != nil {
				data = append(data, *b.CallbackData)
			}
		}
	}
	joined := ""
	for _, l := range labels {
		joined += l + " "
	}
	if !strings.Contains(joined, "Сохранить") {
		t.Errorf("the editing card does not say «Сохранить»: %v", labels)
	}
	if strings.Contains(joined, "Создать") {
		t.Errorf("the editing card offers «Создать» — it would read as making a second event: %v", labels)
	}
	// The fields are ON this screen since 17.09 (Denis: two extra confirmations
	// removed), and leaving without saving is a button of its own.
	if !containsAll(data, "dr_ok", "dre_title", "dre_date", "event_pick") {
		t.Errorf("the editing card does not carry the card's own callbacks: %v", data)
	}
}

func containsAll(have []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, h := range have {
			if h == w {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// 🔴 The underscore is what keeps the four prefixes apart. Without it, "evd"
// would be a prefix of "evdy" and the order of the checks WOULD decide the
// behaviour. This asserts the property directly rather than trusting the shape
// of the strings.
func TestNoEventPrefixIsAPrefixOfAnother(t *testing.T) {
	prefixes := []string{"ev_", "eve_", "evd_", "evdy_"}
	for _, a := range prefixes {
		for _, b := range prefixes {
			if a == b {
				continue
			}
			if strings.HasPrefix(b, a) {
				t.Errorf("%q is a prefix of %q — the order of the checks now decides "+
					"which handler runs, and one of them becomes unreachable", a, b)
			}
		}
	}
}

// 🔴 A recurring event comes back from the list as `<uuid>:YYYY-MM-DD`, and
// GET /api/events/{id} does not understand that form — it casts the id to a
// uuid and fails. Denis hit it on 16.09: «С повторяющимися пишет ❌ Не удалось
// открыть событие».
func TestSplitInstanceIDSeparatesTheSeriesFromTheOccurrence(t *testing.T) {
	parent, occ, isInstance := splitInstanceID("11111111-2222-3333-4444-555555555555:2026-09-23")
	if !isInstance {
		t.Fatal("an occurrence id was not recognised — the bot would ask the API for a uuid it cannot parse")
	}
	if parent != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("parent = %q", parent)
	}
	if occ != "2026-09-23" {
		t.Errorf("occurrence = %q", occ)
	}
}

// A plain id passes through untouched, so every caller can funnel ids through
// this without asking which kind it holds.
func TestSplitInstanceIDLeavesAPlainIDAlone(t *testing.T) {
	for _, id := range []string{
		"11111111-2222-3333-4444-555555555555",
		"",
		"no-colon-here",
		// A colon with something that is not a date after it is not an
		// occurrence — splitting on it would invent a parent that does not exist.
		"11111111-2222-3333-4444-555555555555:notadate",
	} {
		parent, occ, isInstance := splitInstanceID(id)
		if isInstance {
			t.Errorf("%q was read as an occurrence (parent %q, occ %q)", id, parent, occ)
		}
		if parent != id {
			t.Errorf("%q came back as %q", id, parent)
		}
	}
}
