package parse

import (
	"testing"
	"time"
)

// Denis, 17.09: «должен воспринимать 3 или 4 цифры в начале или в конце события
// как время: 1330 — 13:30, 900 — 9:00, 0100 — 01:00, 1 — 1:00».
func TestCompactClockTimes(t *testing.T) {
	cases := map[string]time.Duration{
		"1330 обед":    13*time.Hour + 30*time.Minute,
		"900 зарядка":  9 * time.Hour,
		"0100 подъём":  1 * time.Hour,
		"1 подъём":     1 * time.Hour,
		"созвон в 930": 9*time.Hour + 30*time.Minute,
	}
	for line, want := range cases {
		p := ParseLine(line, quickNow())
		if !p.Draft.HasTime || p.Draft.Start != want {
			t.Errorf("%q: start %v, want %v (title %q)", line, p.Draft.Start, want, p.Title)
		}
		if !p.Draft.IsUncertain(FieldTime) {
			t.Errorf("%q: a compact time is a loose reading and must carry ⚠", line)
		}
	}
	// As the end of a range too.
	p := ParseLine("оркестр 1450-1800", quickNow())
	if !p.Draft.HasEnd || p.Draft.Start != 14*time.Hour+50*time.Minute || p.Draft.End != 18*time.Hour {
		t.Errorf("«1450-1800» → %v–%v", p.Draft.Start, p.Draft.End)
	}
}

// Numbers that are not clock times stay out of the time field.
func TestCompactClockLeavesOtherNumbersAlone(t *testing.T) {
	for _, line := range []string{"2530 шагов", "отжаться 1330 раз", "9999 шагов"} {
		p := ParseLine(line, quickNow())
		if p.Draft.HasTime {
			t.Errorf("%q read as %v", line, p.Draft.Start)
		}
	}
}
