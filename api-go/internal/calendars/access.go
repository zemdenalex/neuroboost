// Package calendars owns the answer to one question: which calendars may a
// user read? Everything about events and tasks scopes itself through this.
package calendars

// Membership statuses.
const (
	StatusInvited = "invited"
	StatusActive  = "active"
)

// Member roles.
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// Membership is one calendar_member row, reduced to what access depends on.
type Membership struct {
	CalendarID string
	Status     string
}

// AccessibleIDs returns the calendars a user may read.
//
// Pure on purpose: the project has no database test harness, and this is the
// rule that must never be wrong — an error here shows one person another
// person's calendar. Keeping it separate from the query makes it testable.
//
// Returns an empty slice, never nil: the result goes into `calendar_id = ANY($1)`,
// where nil would read as "no condition" instead of "nothing matches".
func AccessibleIDs(memberships []Membership) []string {
	ids := make([]string, 0, len(memberships))
	for _, m := range memberships {
		if m.Status == StatusActive {
			ids = append(ids, m.CalendarID)
		}
	}
	return ids
}
