package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/config"
	"github.com/zemdenalex/neuroboost-bot/internal/i18n"
	"github.com/zemdenalex/neuroboost-bot/internal/keyboards"
	"github.com/zemdenalex/neuroboost-bot/internal/state"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

var statsMSK = func() *time.Location { l, _ := time.LoadLocation("Europe/Moscow"); return l }()

// Tue 22.09.2026 15:00 Moscow, and Sun 27.09 20:00 — the end of that week.
var statsTue = time.Date(2026, 9, 22, 15, 0, 0, 0, statsMSK)
var statsSun = time.Date(2026, 9, 27, 20, 0, 0, 0, statsMSK)

func weekScreen(entity string, now time.Time, d statsData) string {
	v := statsView{Period: statgrid.Week, Entity: entity}
	g := statgrid.Layout(statgrid.Week, now, 0, statsMSK, statgrid.Scale{Kind: scaleDay24}, 2026)
	return renderStatsScreen(i18n.RU, v, g, d, statgrid.Scale{Kind: scaleDay24}, statsMSK, now)
}

func mskISO(day, h, m int) string {
	return time.Date(2026, 9, day, h, m, 0, 0, statsMSK).Format(time.RFC3339)
}

func TestStatsViewRoundTripsThroughAButton(t *testing.T) {
	v := statsView{Period: statgrid.Week, Entity: "t", Offset: -3}
	if got, ok := parseStatsView(v.code()); !ok || got != v {
		t.Errorf("round trip: %+v %v", got, ok)
	}
	for _, bad := range []string{"x_a_0", "w_z_0", "w_a_x", "w_a", ""} {
		if _, ok := parseStatsView(bad); ok {
			t.Errorf("accepted %q", bad)
		}
	}
	if got, _ := parseStatsView("w_a_999"); got.Offset != maxStatsOffset {
		t.Errorf("offset not clamped: %d", got.Offset)
	}
	if got, _ := parseStatsView("a_a_5"); got.Offset != 0 {
		t.Errorf("«all» has no offset: %d", got.Offset)
	}
}

// Denis 20.09 / spec §7: two events at one time are booked once, planned twice.
func TestTheWeekShowsBusyHoursOnceAndPlannedTwice(t *testing.T) {
	text := weekScreen("e", statsTue, statsData{Events: []api.Event{
		{StartsAt: mskISO(22, 10, 0), EndsAt: mskISO(22, 11, 0)},
		{StartsAt: mskISO(22, 10, 30), EndsAt: mskISO(22, 11, 30)},
	}})
	if !strings.Contains(text, "Занято: 1,5 ч · В планах: 2 ч") {
		t.Errorf("screen:\n%s", text)
	}
}

// Week: a row per day, a bar per hour (Denis 21.09).
func TestTheWeekGridIsTheDaysAndHours(t *testing.T) {
	text := weekScreen("e", statsTue, statsData{Events: []api.Event{
		{StartsAt: mskISO(22, 12, 0), EndsAt: mskISO(22, 13, 0)},
	}})
	pre := text[strings.Index(text, "<pre>")+5 : strings.Index(text, "</pre>")]
	lines := strings.Split(strings.TrimRight(pre, "\n"), "\n")
	if len(lines) != 8 {
		t.Fatalf("pre has %d lines, want header + 7:\n%s", len(lines), pre)
	}
	tue := []rune(lines[2])
	if !strings.HasPrefix(lines[2], "вт ") || string(tue[3+12]) == "·" || string(tue[3+11]) != "·" {
		t.Errorf("tuesday line %q: want a bar at 12:00 and nothing at 11:00", lines[2])
	}
}

// Denis 22.09: a closed task is its time; a series is «done N of M».
func TestTasksCountTheirTimeAndSeriesDays(t *testing.T) {
	text := weekScreen("t", statsSun, statsData{
		Tasks: []api.Task{
			{ID: "one", Status: "DONE", CompletedAt: mskISO(22, 11, 0), EstimatedMinutes: 90},
			{ID: "pills", Status: "TODO", Rrule: "FREQ=DAILY", RepeatAnchor: mskISO(21, 0, 0)},
			{ID: "open", Status: "TODO"},
		},
		Occ: []api.TaskOccurrence{
			{TaskID: "pills", Occurrence: "2026-09-21", State: "done"},
			{TaskID: "pills", Occurrence: "2026-09-22", State: "done"},
			{TaskID: "pills", Occurrence: "2026-09-24", State: "done"},
			{TaskID: "pills", Occurrence: "2026-09-25", State: "skipped"},
		},
	})
	for _, want := range []string{"Закрыто: 1", "Открыто: 1", "сделано 3 из 7"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	// Tuesday: 90 min closed + a 30-minute series day = 2 h.
	if !strings.Contains(text, "вт ") || !strings.Contains(text, " 2 ч") {
		t.Errorf("tuesday total missing:\n%s", text)
	}
}

func TestReflectionsInAWeekAreTicks(t *testing.T) {
	text := weekScreen("r", statsTue, statsData{Refl: []api.Reflection{{CreatedAt: mskISO(21, 12, 0)}}})
	if !strings.Contains(text, "пн ✓") || !strings.Contains(text, "вт ·") || strings.Contains(text, "<pre>") {
		t.Errorf("screen:\n%s", text)
	}
}

func TestAnEmptyPeriodSaysSo(t *testing.T) {
	if text := weekScreen("a", statsTue, statsData{}); !strings.Contains(text, "Пока пусто") {
		t.Errorf("screen:\n%s", text)
	}
}

func TestEveryStatsButtonFits(t *testing.T) {
	for _, kb := range []struct {
		name   string
		period string
		off    int
	}{{"week", "w", -maxStatsOffset}, {"all", "a", 0}} {
		n := eachButton(keyboards.StatsNav(i18n.RU, kb.period, "r", kb.off, scaleLabel(i18n.RU, scalePeak)), func(data string) {
			if len(data) > 64 {
				t.Errorf("%s: %q is %d bytes", kb.name, data, len(data))
			}
			if _, ok := parseStatsView(strings.TrimPrefix(strings.TrimPrefix(data, "stsc_"), "st_")); !ok && data != "main_menu" {
				t.Errorf("%s: %q does not parse back", kb.name, data)
			}
		})
		if n == 0 {
			t.Errorf("%s: no buttons", kb.name)
		}
	}
}

// Through the handler: the data is read, and 📏 remembers the next scale.
func TestTheScaleButtonRemembersTheNextScale(t *testing.T) {
	var patched string
	start := time.Now().Add(10 * time.Minute).UTC()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			patched = string(b)
			_, _ = w.Write([]byte(`{"data":{}}`))
		case r.URL.Path == "/api/auth/me":
			_, _ = w.Write([]byte(`{"data":{"settings":{"bot":{"lang":"ru"}}}}`))
		case r.URL.Path == "/api/events":
			_, _ = w.Write([]byte(`{"data":[{"id":"e","starts_at":"` + start.Format(time.RFC3339) +
				`","ends_at":"` + start.Add(time.Hour).Format(time.RFC3339) + `"}]}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	bot, fake := newFakeTelegram(t)
	h := New(bot, api.NewClient(srv.URL), state.NewStore(), config.Config{})

	h.handleStatsScale(940, 0, statsView{Period: statgrid.Week, Entity: "a"})
	if !strings.Contains(patched, `"stats_scale":"work"`) || !strings.Contains(patched, `"lang":"ru"`) {
		t.Errorf("PATCH = %s", patched)
	}
	// ⚠ The event starts ten minutes from now: the week holds it unless the
	// test runs in the last ten minutes of a Sunday.
	if text := fake.last(t).Text; !strings.Contains(text, "Занято: 1 ч") {
		t.Errorf("screen:\n%s", text)
	}
}
