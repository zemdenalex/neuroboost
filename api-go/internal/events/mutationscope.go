package events

import "time"

// mutationMode says how a mutation should be applied to a recurring event.
type mutationMode int

const (
	// mutatePlain is an ordinary, non-recurring event: act on the row directly.
	mutatePlain mutationMode = iota
	// mutateOccurrence changes a single occurrence, leaving the series intact.
	mutateOccurrence
	// mutateSeries changes the parent event and therefore every occurrence.
	mutateSeries
)

// Scope values accepted on the wire (query parameter `scope`).
const (
	scopeOccurrence = "occurrence"
	scopeSeries     = "series"
)

// resolveMutation decides which event row a mutation targets and how broadly it
// applies, given the raw ID from the URL and the requested scope.
//
// An unrecognised or absent scope deliberately falls back to the *narrowest*
// change. Defaulting to the series would mean a dropped query parameter
// silently rewrites every occurrence — the failure mode users cannot undo.
func resolveMutation(rawID, scope string) (targetID string, occurrence time.Time, mode mutationMode) {
	parentID, occ, isInstance := parseInstanceID(rawID)
	if !isInstance {
		// Scope is meaningless without a series to scope over.
		return parentID, time.Time{}, mutatePlain
	}

	if scope == scopeSeries {
		return parentID, time.Time{}, mutateSeries
	}

	return parentID, occ, mutateOccurrence
}
