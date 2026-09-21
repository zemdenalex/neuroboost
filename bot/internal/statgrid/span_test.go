package statgrid

import (
	"testing"
	"time"
)

func at(h, m int) time.Time { return time.Date(2026, 9, 21, h, m, 0, 0, time.UTC) }

// Spec §7, first line: 10:00–11:00 and 10:30–11:30 → busy 1.5 h, planned 2 h.
func TestOverlapsCountOnceInBusyAndTwiceInPlanned(t *testing.T) {
	spans := []Span{{at(10, 0), at(11, 0)}, {at(10, 30), at(11, 30)}}
	from, to := at(0, 0), at(23, 59)
	if b := Busy(spans, from, to); b != 90*time.Minute {
		t.Errorf("busy = %v, want 1h30m", b)
	}
	if p := Planned(spans, from, to); p != 2*time.Hour {
		t.Errorf("planned = %v, want 2h", p)
	}
}

func TestSpansAreClippedToTheCell(t *testing.T) {
	spans := []Span{{at(9, 30), at(12, 0)}}
	if b := Busy(spans, at(10, 0), at(11, 0)); b != time.Hour {
		t.Errorf("busy in 10–11 = %v, want 1h", b)
	}
	if b := Busy(spans, at(13, 0), at(14, 0)); b != 0 {
		t.Errorf("busy outside = %v", b)
	}
}

func TestContainedAndTouchingSpans(t *testing.T) {
	spans := []Span{{at(8, 0), at(12, 0)}, {at(9, 0), at(10, 0)}, {at(12, 0), at(13, 0)}}
	if b := Busy(spans, at(0, 0), at(23, 0)); b != 5*time.Hour {
		t.Errorf("busy = %v, want 5h (contained counts once, touching adds)", b)
	}
}
