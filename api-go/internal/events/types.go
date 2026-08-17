package events

import "time"

// Event represents a calendar event
type Event struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CalendarID  string    `json:"calendar_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	AllDay      bool      `json:"all_day"`
	Rrule       *string   `json:"rrule,omitempty"`
	Timezone    string    `json:"timezone"`
	Location    *string   `json:"location,omitempty"`
	Color       *string   `json:"color,omitempty"`
	Tags        []string  `json:"tags"`
	// ReminderOffsets holds minutes-before-start for each reminder. Stored on
	// the event rather than as pre-generated rows: a daily event with five
	// offsets would otherwise be 1825 rows a year.
	ReminderOffsets  []int     `json:"reminder_offsets"`
	TaskID           *string   `json:"task_id,omitempty"`
	IsWorkEvent      bool      `json:"is_work_event"`
	RecurringEventID *string   `json:"recurring_event_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// IsShared reports that somebody else can see this event: its calendar has
	// at least one other active member. Derived on read, never stored — a column
	// would go stale the moment someone accepts an invitation or leaves, and
	// nothing would say so.
	IsShared bool `json:"is_shared"`
	// AuthorName is set only when the event was created by somebody other than
	// the caller. It is therefore per-viewer and could not be a column even in
	// principle: the same row is "mine" to one member and "hers" to the other.
	AuthorName *string `json:"author_name,omitempty"`
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	Title string `json:"title"`
	// Which calendar the event goes in. Absent or empty means the author's
	// personal calendar — the behaviour every creation had before shared
	// calendars, and the one the bot and the importer still depend on.
	//
	// 🔴 Checked, never trusted: calendars.WritableIDFor verifies the caller is
	// an active owner or editor. The row's user_id records authorship, not
	// access, so an unchecked id would write into someone else's calendar with
	// nothing downstream objecting.
	CalendarID  *string  `json:"calendar_id,omitempty"`
	Description *string  `json:"description,omitempty"`
	StartsAt    string   `json:"starts_at"` // ISO 8601
	EndsAt      string   `json:"ends_at"`   // ISO 8601
	AllDay      bool     `json:"all_day"`
	Rrule       *string  `json:"rrule,omitempty"`
	Timezone    string   `json:"timezone,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Color       *string  `json:"color,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	// Pointer, not a bare slice: an absent field means "apply the user's
	// default preset", an explicitly empty array means "deliberately no
	// reminders". A []int cannot tell those apart.
	ReminderOffsets *[]int  `json:"reminder_offsets,omitempty"`
	TaskID          *string `json:"task_id,omitempty"`
	IsWorkEvent     *bool   `json:"is_work_event,omitempty"`
}

// UpdateEventRequest represents the request to update an event
type UpdateEventRequest struct {
	Title           *string  `json:"title,omitempty"`
	Description     *string  `json:"description,omitempty"`
	StartsAt        *string  `json:"starts_at,omitempty"`
	EndsAt          *string  `json:"ends_at,omitempty"`
	AllDay          *bool    `json:"all_day,omitempty"`
	Rrule           *string  `json:"rrule,omitempty"`
	Timezone        *string  `json:"timezone,omitempty"`
	Location        *string  `json:"location,omitempty"`
	Color           *string  `json:"color,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	ReminderOffsets *[]int   `json:"reminder_offsets,omitempty"`
	IsWorkEvent     *bool    `json:"is_work_event,omitempty"`

	// CalendarID moves the event to another calendar.
	//
	// 🔴 Two calendars are checked, not one. The destination is checked by
	// WritableIDFor below; the SOURCE is checked by updateEvent's WHERE
	// clause, which scopes to WritableIDsFor — without that, read access to a
	// calendar would be enough to move its events out of it, which is
	// deletion by another name as far as the other members are concerned.
	//
	// Applies to the whole series. See UpdateHandler for why scope=occurrence
	// is refused rather than quietly interpreted.
	CalendarID *string `json:"calendar_id,omitempty"`
}

// MoveEventRequest represents a drag-and-drop move operation
type MoveEventRequest struct {
	StartsAt string `json:"starts_at"` // New start time (ISO 8601)
	EndsAt   string `json:"ends_at"`   // New end time (ISO 8601)
}

// ResizeEventRequest represents a resize operation
type ResizeEventRequest struct {
	EndsAt string `json:"ends_at"` // New end time (ISO 8601)
}

// EventException represents an exception to a recurring event
type EventException struct {
	ID                 string    `json:"id"`
	EventID            string    `json:"event_id"`
	UserID             string    `json:"user_id"`
	Occurrence         time.Time `json:"occurrence"`
	Skipped            bool      `json:"skipped"`
	ReplacementEventID *string   `json:"replacement_event_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// AddExceptionRequest represents adding an exception to a recurring event
type AddExceptionRequest struct {
	Occurrence         string  `json:"occurrence"` // ISO 8601
	Skipped            bool    `json:"skipped"`
	ReplacementEventID *string `json:"replacement_event_id,omitempty"`
}

// ListEventsQuery represents query parameters for listing events
type ListEventsQuery struct {
	Start string // ISO 8601 - start of range
	End   string // ISO 8601 - end of range
}
