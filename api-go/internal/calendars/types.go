package calendars

import (
	"errors"
	"fmt"
	"time"
)

// Calendar kinds.
const (
	KindPersonal = "personal"
	KindShared   = "shared"
)

// Calendar is one calendar as the API returns it. Role and status come from
// the requesting user's own calendar_member row, not from the calendar.
//
// Field names on the wire are snake_case, matching every other module in this
// API. The TypeScript type mirrors them verbatim: no conversion layer means no
// place for a conversion layer to disagree.
type Calendar struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color"`
	Kind      string    `json:"kind"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateFields carries a PATCH. A nil field means "leave unchanged" — clearing
// a colour back to null is not expressible in v1 and no caller needs it.
type UpdateFields struct {
	Name  *string
	Color *string
}

var (
	// ErrCalendarNotFound covers both "no such calendar" and "not a member of
	// it" on purpose: distinguishing them tells a stranger that the calendar
	// exists.
	ErrCalendarNotFound = errors.New("calendar not found")

	// ErrNotCalendarOwner is only ever returned to an actual member.
	ErrNotCalendarOwner = errors.New("not the calendar owner")

	// ErrCalendarIsPersonal guards the delete path. event.calendar_id
	// references calendar(id) with no ON DELETE clause, so deleting a personal
	// calendar either violates the foreign key (non-empty) or silently strips
	// the user of their own calendar (empty).
	ErrCalendarIsPersonal = errors.New("personal calendar cannot be deleted")

	// ErrInvalidName is returned by Create and Update when NormalizeName
	// rejects the supplied name (empty, or over maxNameLen runes). Exported so
	// the handler can classify it with errors.Is instead of falling through to
	// a generic 500 on a validation failure the store already caught.
	ErrInvalidName = errors.New("invalid calendar name")
)

// NotEmptyError reports what is still inside a calendar the caller tried to
// delete. The counts are the payload of the 409 the spec requires (§5.1).
type NotEmptyError struct {
	Events int
	Tasks  int
}

func (e *NotEmptyError) Error() string {
	return fmt.Sprintf("calendar not empty: %d events, %d tasks", e.Events, e.Tasks)
}
