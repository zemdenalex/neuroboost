package events

import (
	"strings"
	"time"
)

// instanceIDDateLayout must match the format expandRecurrence uses when it
// stamps synthetic instance IDs.
const instanceIDDateLayout = "2006-01-02"

// parseInstanceID splits a recurring-instance ID of the form
// "<parent uuid>:<YYYY-MM-DD>" into the parent event ID and the occurrence date.
//
// Anything not in that form is returned unchanged with isInstance false, so a
// caller can funnel every event ID through this without special-casing. That
// matters because the ID reaches a Postgres uuid cast: passing a synthetic ID
// through raises a cast error that surfaces to the client as a 500.
//
// Splitting on the last colon is safe — a UUID contains none.
func parseInstanceID(id string) (parentID string, occurrence time.Time, isInstance bool) {
	idx := strings.LastIndex(id, ":")
	if idx < 0 {
		return id, time.Time{}, false
	}

	date, err := time.Parse(instanceIDDateLayout, id[idx+1:])
	if err != nil {
		return id, time.Time{}, false
	}

	return id[:idx], date, true
}
