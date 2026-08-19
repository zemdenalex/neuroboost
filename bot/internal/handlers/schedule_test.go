package handlers

import (
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("tzdata for %s unavailable: %v", name, err)
	}
	return loc
}

func TestScheduleStartResolvesEachSlot(t *testing.T) {
	loc := mustLoad(t, "Europe/Moscow")
	// A Tuesday afternoon, and deliberately not on a round minute: "Сейчас" has
	// to shed the seconds, and a fixture that had none could not show it.
	now := time.Date(2026, 8, 18, 14, 37, 41, 500_000_000, loc)

	cases := []struct {
		slot string
		want time.Time
	}{
		{"now", time.Date(2026, 8, 18, 14, 37, 0, 0, loc)},
		{"hour", time.Date(2026, 8, 18, 15, 37, 0, 0, loc)},
		{"eve", time.Date(2026, 8, 18, 19, 0, 0, 0, loc)},
		{"tmr", time.Date(2026, 8, 19, 9, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		got, ok := scheduleStart(c.slot, now, loc)
		if !ok {
			t.Errorf("%s: refused a slot it should accept", c.slot)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("%s: got %s, want %s", c.slot, got, c.want)
		}
	}
}

func TestEveningAfterSevenMeansTomorrowEvening(t *testing.T) {
	// The rule the event flow already follows. Without it, "сегодня вечером"
	// pressed at 21:00 schedules a block two hours in the past, which the API
	// accepts without complaint — nothing downstream would have objected.
	loc := mustLoad(t, "Europe/Moscow")
	now := time.Date(2026, 8, 18, 21, 0, 0, 0, loc)

	got, ok := scheduleStart("eve", now, loc)
	if !ok {
		t.Fatal("eve refused")
	}
	want := time.Date(2026, 8, 19, 19, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("got %s, want %s", got, want)
	}
	if !got.After(now) {
		t.Errorf("scheduled %s, which is in the past relative to %s", got, now)
	}
}

func TestEveningExactlyAtSevenIsTomorrow(t *testing.T) {
	// The boundary the `!start.After(now)` spelling decides. At exactly 19:00
	// there is no evening left to schedule into.
	loc := mustLoad(t, "Europe/Moscow")
	now := time.Date(2026, 8, 18, 19, 0, 0, 0, loc)
	got, _ := scheduleStart("eve", now, loc)
	if got.Day() != 19 {
		t.Errorf("got %s, want the 19th", got)
	}
}

func TestUnknownSlotIsRefusedNotDefaulted(t *testing.T) {
	// Callback data comes from the user's client. A slot this bot does not know
	// must produce nothing at all — defaulting to "now" would let a crafted or
	// stale button write an arbitrary time into someone's calendar.
	loc := mustLoad(t, "Europe/Moscow")
	for _, slot := range []string{"", "NOW", "yesterday", "now ", "tmr;drop"} {
		if _, ok := scheduleStart(slot, time.Now(), loc); ok {
			t.Errorf("slot %q was accepted", slot)
		}
	}
}

func TestParsePlanCallback(t *testing.T) {
	const uuid = "8f14e45f-ceea-467a-9575-0f0e2d4a2f1b"

	id, slot, minutes, ok := parsePlanCallback(uuid + "_eve_120")
	if !ok {
		t.Fatal("a well-formed callback was rejected")
	}
	if id != uuid || slot != "eve" || minutes != 120 {
		t.Errorf("got (%q, %q, %d)", id, slot, minutes)
	}

	// Cutting from the right is the whole point: an id carrying underscores
	// still resolves to the id, not to its first segment.
	id, slot, minutes, ok = parsePlanCallback("legacy_task_77_now_30")
	if !ok || id != "legacy_task_77" || slot != "now" || minutes != 30 {
		t.Errorf("right-cut failed: (%q, %q, %d, %v)", id, slot, minutes, ok)
	}

	for _, bad := range []string{
		"",                   // nothing
		"abc",                // no separators
		uuid + "_eve",        // no duration
		uuid + "_soon_30",    // slot this bot does not know
		uuid + "_eve_0",      // a zero-length block
		uuid + "_eve_-30",    // negative
		uuid + "_eve_2000",   // longer than a day
		uuid + "_eve_thirty", // not a number
		"_eve_30",            // no task
	} {
		if _, _, _, ok := parsePlanCallback(bad); ok {
			t.Errorf("accepted malformed callback %q", bad)
		}
	}
}

// scheduleKeyboards is every keyboard this feature can put on screen, built
// with a real 36-byte UUID rather than "id" — the short fixture is what lets a
// callback-data budget look fine in a test and fail in Telegram.
func scheduleKeyboards() map[string]tgbotapi.InlineKeyboardMarkup {
	const uuid = "8f14e45f-ceea-467a-9575-0f0e2d4a2f1b"
	return map[string]tgbotapi.InlineKeyboardMarkup{
		"TaskActions":          keyboards.TaskActions(uuid),
		"TaskScheduleWhen":     keyboards.TaskScheduleWhen(uuid),
		"TaskScheduleDuration": keyboards.TaskScheduleDuration(uuid, "eve"),
		"TaskDue":              keyboards.TaskDue(uuid),
		"TaskEstimate":         keyboards.TaskEstimate(uuid),
	}
}

func eachButton(kb tgbotapi.InlineKeyboardMarkup, fn func(data string)) int {
	var n int
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == nil {
				continue
			}
			n++
			fn(*b.CallbackData)
		}
	}
	return n
}

// Telegram rejects callback_data over 64 bytes, and rejects it at send time:
// the keyboard simply does not appear, with nothing in the log that reads as
// "too long". A UUID spends 36 of the 64 before any prefix of ours.
func TestScheduleCallbackDataFitsTelegramsLimit(t *testing.T) {
	for name, kb := range scheduleKeyboards() {
		seen := eachButton(kb, func(data string) {
			if len(data) > 64 {
				t.Errorf("%s: callback_data %q is %d bytes, Telegram allows 64", name, data, len(data))
			}
			if data == "" {
				t.Errorf("%s: a button carries empty callback_data", name)
			}
		})
		// The floor. A keyboard that produced no buttons at all would satisfy
		// every assertion above by having nothing to check.
		if seen == 0 {
			t.Errorf("%s: no buttons with callback data — this test proved nothing", name)
		}
	}
}

// Every button must match a prefix the router actually dispatches. A keyboard
// whose callback nothing handles is this bot's oldest failure mode: the user
// taps, the spinner stops, and nothing happens — indistinguishable from a slow
// network. handler.go is the list; keep them in step.
func TestEveryScheduleButtonHasAPrefixTheRouterKnows(t *testing.T) {
	routed := []string{
		"task_sched_", "task_when_", "task_plan_",
		"task_action_", "task_done_", "task_delete_", "top_tasks",
		"task_due_set_", "task_due_", "task_est_set_", "task_est_", "task_tag_",
	}
	for name, kb := range scheduleKeyboards() {
		eachButton(kb, func(data string) {
			for _, p := range routed {
				if strings.HasPrefix(data, p) {
					return
				}
			}
			t.Errorf("%s: %q matches no routed prefix", name, data)
		})
	}
}
