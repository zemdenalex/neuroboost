package handlers

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
)

func TestRangeIsSaneRefusesAnInvertedDay(t *testing.T) {
	// The web app accepts these (WorkHoursSection.tsx says so in its own
	// comment). Buttons cannot mean "overnight shift", so here they are mis-taps.
	for _, c := range []struct {
		start, end string
		want       bool
	}{
		{"09:00", "17:00", true},
		{"06:00", "22:00", true},
		{"17:00", "09:00", false},
		{"09:00", "09:00", false}, // a zero-length working day
		{"12:00", "12:00", false},
	} {
		if got := rangeIsSane(c.start, c.end); got != c.want {
			t.Errorf("%s–%s: got %v, want %v", c.start, c.end, got, c.want)
		}
	}
}

func TestSettingStringFallsBackOnEveryShapeTheBlobCanTake(t *testing.T) {
	// settings is JSONB decoded into map[string]any: a number stays a float64,
	// a null becomes nil, and an absent key is missing entirely. All three have
	// to produce the default rather than a panic or an empty label.
	settings := map[string]any{
		"work_start": "08:30",
		"work_end":   "",
		"ui_scale":   float64(120),
		"stray":      nil,
	}
	if got := settingString(settings, "work_start", "09:00"); got != "08:30" {
		t.Errorf("stored value ignored: %q", got)
	}
	if got := settingString(settings, "work_end", "17:00"); got != "17:00" {
		t.Errorf("empty string should fall back, got %q", got)
	}
	if got := settingString(settings, "absent", "17:00"); got != "17:00" {
		t.Errorf("missing key: %q", got)
	}
	if got := settingString(settings, "ui_scale", "17:00"); got != "17:00" {
		t.Errorf("a non-string must fall back, got %q", got)
	}
	if got := settingString(settings, "stray", "17:00"); got != "17:00" {
		t.Errorf("a null must fall back, got %q", got)
	}
	if got := settingString(nil, "work_start", "09:00"); got != "09:00" {
		t.Errorf("a nil blob must fall back, got %q", got)
	}
}

// Every hour the settings screen offers must round-trip through the callback
// parser the router hands it. The two lists live in settings.go and the
// keyboard renders whatever it is given, so nothing but a test connects them.
func TestEveryOfferedHourParsesBackToItself(t *testing.T) {
	check := func(kind string, kb tgbotapi.InlineKeyboardMarkup, want []int) {
		var seen []int
		for _, row := range kb.InlineKeyboard {
			for _, b := range row {
				if b.CallbackData == nil {
					continue
				}
				data := *b.CallbackData
				if !strings.HasPrefix(data, "wh_") {
					continue
				}
				rest := strings.TrimPrefix(data, "wh_")
				which, hourStr, ok := strings.Cut(rest, "_")
				if !ok || which != kind {
					t.Errorf("%s: %q does not parse as a %s hour", kind, data, kind)
					continue
				}
				var h int
				for _, c := range hourStr {
					if c < '0' || c > '9' {
						t.Errorf("%s: %q carries a non-numeric hour", kind, data)
						h = -1
						break
					}
					h = h*10 + int(c-'0')
				}
				if h >= 0 {
					seen = append(seen, h)
				}
				if len(data) > 64 {
					t.Errorf("%s: %q exceeds Telegram's 64 bytes", kind, data)
				}
			}
		}
		if len(seen) != len(want) {
			t.Fatalf("%s: keyboard offered %d hours, the handler knows %d", kind, len(seen), len(want))
		}
		for i, h := range want {
			if seen[i] != h {
				t.Errorf("%s: position %d is %d, want %d", kind, i, seen[i], h)
			}
		}
	}

	// 🔴 Written out by hand, NOT derived from startHours/endHours. The first
	// version of this test passed `startHours` as both the input and the
	// expectation, so it compared the lists to themselves: adding an hour to
	// the handler changed both sides at once and the test stayed green. A
	// control built from the thing it checks cannot fail for the bug it names.
	check("start", keyboards.WorkHoursStart(startHours), []int{6, 7, 8, 9, 10, 11, 12})
	check("end", keyboards.WorkHoursEnd(endHours), []int{14, 15, 16, 17, 18, 19, 20, 21, 22})
}

// The two lists must overlap in a way that leaves a legal day available, or a
// user who picks the latest start is told "выбери другое" with nothing to pick.
func TestEveryStartHasSomeLegalEnd(t *testing.T) {
	if len(startHours) == 0 || len(endHours) == 0 {
		t.Fatal("an empty hour list would make every assertion below vacuous")
	}
	for _, s := range startHours {
		var any bool
		for _, e := range endHours {
			if rangeIsSane(hhmm(s), hhmm(e)) {
				any = true
				break
			}
		}
		if !any {
			t.Errorf("start %02d:00 has no valid end on offer", s)
		}
	}
}

func hhmm(h int) string {
	return string(rune('0'+h/10)) + string(rune('0'+h%10)) + ":00"
}
