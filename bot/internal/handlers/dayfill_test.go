package handlers

import (
	"testing"
	"time"

	"github.com/zemdenalex/neuroboost-bot/internal/api"
	"github.com/zemdenalex/neuroboost-bot/internal/statgrid"
)

// Spec §4: the month calendar shows how full a day is, measured exactly as
// statistics measures it — the same busy time, the same scale.
func TestADaysFillFollowsTheScale(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, statsMSK)
	to := from.AddDate(0, 1, 0)
	events := []api.Event{{StartsAt: mskISO(22, 8, 0), EndsAt: mskISO(22, 20, 0)}}

	if l := dayLevels(events, from, to, statgrid.Scale{Kind: scaleDay24}, statsMSK)["2026-09-22"]; l != 4 {
		t.Errorf("24h scale: level %d, want 4 (12 of 24 hours)", l)
	}
	work := statgrid.Scale{Kind: scaleWork, WorkStart: 8, WorkEnd: 20}
	if l := dayLevels(events, from, to, work, statsMSK)["2026-09-22"]; l != 8 {
		t.Errorf("work scale: level %d, want 8", l)
	}
}

// An all-day event books no hours, but a day with one is not an empty day —
// a holiday must not look like a free Tuesday.
func TestAnAllDayEventMarksItsDay(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, statsMSK)
	// 🔴 The shape dev actually stores (checked 22.09): LOCAL midnight as an
	// instant — 21:00Z the day before, for Moscow. Reading the date off the
	// string marked the day before the holiday; the first fixture here was UTC
	// midnight and could not tell the two apart.
	levels := dayLevels([]api.Event{{StartsAt: "2026-09-22T21:00:00Z", EndsAt: "2026-09-23T21:00:00Z", AllDay: true}},
		from, from.AddDate(0, 1, 0), statgrid.Scale{Kind: scaleDay24}, statsMSK)
	if levels["2026-09-23"] != 1 {
		t.Errorf("all-day day level = %d, want 1 (levels %v)", levels["2026-09-23"], levels)
	}
	for _, other := range []string{"2026-09-22", "2026-09-24"} {
		if _, ok := levels[other]; ok {
			t.Errorf("the all-day event marked %s too", other)
		}
	}
}

func TestAnEmptyDayHasNoLevel(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, statsMSK)
	if levels := dayLevels(nil, from, from.AddDate(0, 1, 0), statgrid.Scale{Kind: scaleDay24}, statsMSK); len(levels) != 0 {
		t.Errorf("levels = %v", levels)
	}
}
