package events

import (
	"testing"
	"time"
)

// A mutation arriving for a recurring event has to answer two questions before
// it touches the database: which row does it target, and does it change the one
// occurrence or the whole series. Getting this wrong is destructive — editing
// one instance must never silently rewrite every occurrence.
func TestResolveMutation(t *testing.T) {
	const parent = "0f4b1a2c-3d4e-4f50-8a1b-2c3d4e5f6a7b"
	occ := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		rawID          string
		scope          string
		wantTarget     string
		wantOccurrence time.Time
		wantMode       mutationMode
	}{
		{
			name:       "ordinary event is unaffected by scope",
			rawID:      parent,
			scope:      "",
			wantTarget: parent,
			wantMode:   mutatePlain,
		},
		{
			name:       "ordinary event ignores a series scope it cannot use",
			rawID:      parent,
			scope:      "series",
			wantTarget: parent,
			wantMode:   mutatePlain,
		},
		{
			name:           "instance with occurrence scope targets the parent and records the date",
			rawID:          parent + ":2026-07-21",
			scope:          "occurrence",
			wantTarget:     parent,
			wantOccurrence: occ,
			wantMode:       mutateOccurrence,
		},
		{
			name:       "instance with series scope edits the whole series",
			rawID:      parent + ":2026-07-21",
			scope:      "series",
			wantTarget: parent,
			wantMode:   mutateSeries,
		},
		{
			// The destructive default would be series. An absent or unrecognised
			// scope must fall back to the narrowest possible change.
			name:           "instance defaults to the single occurrence when scope is absent",
			rawID:          parent + ":2026-07-21",
			scope:          "",
			wantTarget:     parent,
			wantOccurrence: occ,
			wantMode:       mutateOccurrence,
		},
		{
			name:           "instance defaults to the single occurrence when scope is unrecognised",
			rawID:          parent + ":2026-07-21",
			scope:          "everything",
			wantTarget:     parent,
			wantOccurrence: occ,
			wantMode:       mutateOccurrence,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTarget, gotOccurrence, gotMode := resolveMutation(tc.rawID, tc.scope)

			if gotTarget != tc.wantTarget {
				t.Errorf("target = %q, want %q", gotTarget, tc.wantTarget)
			}
			if !gotOccurrence.Equal(tc.wantOccurrence) {
				t.Errorf("occurrence = %v, want %v", gotOccurrence, tc.wantOccurrence)
			}
			if gotMode != tc.wantMode {
				t.Errorf("mode = %v, want %v", gotMode, tc.wantMode)
			}
		})
	}
}
