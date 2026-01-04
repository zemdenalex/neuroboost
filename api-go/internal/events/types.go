package events

import "time"

// Event represents a calendar event
type Event struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      time.Time  `json:"ends_at"`
	AllDay      bool       `json:"all_day"`
	Rrule       *string    `json:"rrule,omitempty"`
	Timezone    string     `json:"timezone"`
	Location    *string    `json:"location,omitempty"`
	Color       *string    `json:"color,omitempty"`
	Tags        []string   `json:"tags"`
	TaskID      *string    `json:"task_id,omitempty"`
	IsWorkEvent bool       `json:"is_work_event"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	StartsAt    string   `json:"starts_at"` // ISO 8601
	EndsAt      string   `json:"ends_at"`   // ISO 8601
	AllDay      bool     `json:"all_day"`
	Rrule       *string  `json:"rrule,omitempty"`
	Timezone    string   `json:"timezone,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Color       *string  `json:"color,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	TaskID      *string  `json:"task_id,omitempty"`
	IsWorkEvent *bool    `json:"is_work_event,omitempty"`
}

// UpdateEventRequest represents the request to update an event
type UpdateEventRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	StartsAt    *string  `json:"starts_at,omitempty"`
	EndsAt      *string  `json:"ends_at,omitempty"`
	AllDay      *bool    `json:"all_day,omitempty"`
	Rrule       *string  `json:"rrule,omitempty"`
	Timezone    *string  `json:"timezone,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Color       *string  `json:"color,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsWorkEvent *bool    `json:"is_work_event,omitempty"`
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