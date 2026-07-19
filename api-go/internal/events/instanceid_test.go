package events

import (
	"testing"
	"time"
)

// Recurring instances are emitted with a synthetic ID of the form
// "<parent uuid>:<YYYY-MM-DD>" (see expandRecurrence). Every mutation handler
// previously passed that straight into a Postgres uuid cast, which fails and
// surfaces as a 500 — leaving recurring events effectively read-only (R1).
func TestParseInstanceID(t *testing.T) {
	const parent = "0f4b1a2c-3d4e-4f50-8a1b-2c3d4e5f6a7b"

	tests := []struct {
		name           string
		id             string
		wantParent     string
		wantOccurrence time.Time
		wantIsInstance bool
	}{
		{
			name:           "synthetic instance id splits into parent and occurrence",
			id:             parent + ":2026-07-21",
			wantParent:     parent,
			wantOccurrence: time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
			wantIsInstance: true,
		},
		{
			name:           "plain uuid is not an instance and passes through",
			id:             parent,
			wantParent:     parent,
			wantOccurrence: time.Time{},
			wantIsInstance: false,
		},
		{
			name:           "unparseable date suffix is not treated as an instance",
			id:             parent + ":not-a-date",
			wantParent:     parent + ":not-a-date",
			wantOccurrence: time.Time{},
			wantIsInstance: false,
		},
		{
			name:           "empty id passes through untouched",
			id:             "",
			wantParent:     "",
			wantOccurrence: time.Time{},
			wantIsInstance: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotParent, gotOccurrence, gotIsInstance := parseInstanceID(tc.id)

			if gotParent != tc.wantParent {
				t.Errorf("parent = %q, want %q", gotParent, tc.wantParent)
			}
			if !gotOccurrence.Equal(tc.wantOccurrence) {
				t.Errorf("occurrence = %v, want %v", gotOccurrence, tc.wantOccurrence)
			}
			if gotIsInstance != tc.wantIsInstance {
				t.Errorf("isInstance = %v, want %v", gotIsInstance, tc.wantIsInstance)
			}
		})
	}
}

// The parser must round-trip whatever expandRecurrence produces, so the two
// cannot drift apart.
func TestParseInstanceIDRoundTripsExpandedInstance(t *testing.T) {
	const parent = "0f4b1a2c-3d4e-4f50-8a1b-2c3d4e5f6a7b"
	start := dt(2026, time.July, 20, 9, 0)

	ev := mkEvent(parent, start, time.Hour, "FREQ=DAILY")
	instances := expandRecurrence(ev, start, start.AddDate(0, 0, 3), nil)
	if len(instances) == 0 {
		t.Fatal("expected at least one instance")
	}

	for _, inst := range instances {
		gotParent, gotOccurrence, isInstance := parseInstanceID(inst.ID)
		if !isInstance {
			t.Errorf("id %q not recognised as an instance", inst.ID)
			continue
		}
		if gotParent != parent {
			t.Errorf("parent = %q, want %q", gotParent, parent)
		}
		wantDate := inst.StartsAt.UTC().Truncate(24 * time.Hour)
		if !gotOccurrence.Equal(wantDate) {
			t.Errorf("occurrence = %v, want %v (from %v)", gotOccurrence, wantDate, inst.StartsAt)
		}
	}
}
