package events

import (
	"testing"
	"time"
)

func TestValidateTimeRange(t *testing.T) {
	base := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr bool
	}{
		{"end after start is valid", base, base.Add(time.Hour), false},
		{"one-second range is valid", base, base.Add(time.Second), false},
		{"end equal to start is invalid (zero length)", base, base, true},
		{"end before start is invalid (inverted)", base, base.Add(-time.Hour), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateTimeRange(c.start, c.end)
			if c.wantErr && err == nil {
				t.Errorf("validateTimeRange(%v, %v): expected error, got nil", c.start, c.end)
			}
			if !c.wantErr && err != nil {
				t.Errorf("validateTimeRange(%v, %v): expected nil, got %v", c.start, c.end, err)
			}
		})
	}
}
