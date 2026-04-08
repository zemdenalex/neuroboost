package reflections

import "time"

// Reflection represents a post-event reflection
type Reflection struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	EventID      string    `json:"event_id"`
	Energy       *int      `json:"energy,omitempty"`
	Mood         *int      `json:"mood,omitempty"`
	Focus        *int      `json:"focus,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	WasCompleted bool      `json:"was_completed"`
	WasOnTime    bool      `json:"was_on_time"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateRequest represents the request to create a reflection
type CreateRequest struct {
	EventID      string  `json:"event_id"`
	Energy       *int    `json:"energy,omitempty"`
	Mood         *int    `json:"mood,omitempty"`
	Focus        *int    `json:"focus,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	WasCompleted *bool   `json:"was_completed,omitempty"`
	WasOnTime    *bool   `json:"was_on_time,omitempty"`
}

// UpdateRequest represents the request to update a reflection
type UpdateRequest struct {
	Energy       *int    `json:"energy,omitempty"`
	Mood         *int    `json:"mood,omitempty"`
	Focus        *int    `json:"focus,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	WasCompleted *bool   `json:"was_completed,omitempty"`
	WasOnTime    *bool   `json:"was_on_time,omitempty"`
}
