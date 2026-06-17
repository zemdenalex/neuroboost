package export

import (
	"encoding/json"
	"net/http"
	"time"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB sets the database connection for the export package
func InitDB(database *database.DB) {
	db = database
}

// ExportPayload is the top-level structure returned by ExportHandler
type ExportPayload struct {
	Version    string     `json:"version"`
	ExportedAt time.Time  `json:"exported_at"`
	Events     []EventRow `json:"events"`
	Tasks      []TaskRow  `json:"tasks"`
	Settings   any        `json:"settings"`
}

// EventRow holds the columns exported from the event table
type EventRow struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
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
	TaskID      *string   `json:"task_id,omitempty"`
	IsWorkEvent bool      `json:"is_work_event"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskRow holds the columns exported from the task table
type TaskRow struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	Status           string     `json:"status"`
	Category         *string    `json:"category,omitempty"`
	Priority         int        `json:"priority"`
	EstimatedMinutes *int       `json:"estimated_minutes,omitempty"`
	ActualMinutes    int        `json:"actual_minutes"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	Tags             []string   `json:"tags"`
	Contexts         []string   `json:"contexts"`
	Energy           *int       `json:"energy,omitempty"`
	ParentID         *string    `json:"parent_id,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ImportRequest is the body accepted by ImportHandler — same shape as ExportPayload
type ImportRequest struct {
	Version string     `json:"version"`
	Events  []EventRow `json:"events"`
	Tasks   []TaskRow  `json:"tasks"`
}

// ImportResult reports how many records were inserted and how many were skipped
// because they failed per-row validation (e.g. a corrupted or hand-edited export).
type ImportResult struct {
	EventsImported int `json:"events_imported"`
	TasksImported  int `json:"tasks_imported"`
	EventsSkipped  int `json:"events_skipped"`
	TasksSkipped   int `json:"tasks_skipped"`
}

// ExportHandler returns all events and tasks for the authenticated user
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	events, err := queryEvents(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch events")
		return
	}

	tasks, err := queryTasks(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch tasks")
		return
	}

	payload := ExportPayload{
		Version:    "0.4.8",
		ExportedAt: time.Now().UTC(),
		Events:     events,
		Tasks:      tasks,
		Settings:   nil,
	}

	util.RespondJSON(w, http.StatusOK, payload)
}

// ImportHandler accepts an export JSON body and upserts records for the authenticated user
func ImportHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	eventsImported := 0
	eventsSkipped := 0
	for _, ev := range req.Events {
		// Enforce the same per-row rules as the Create handler — never silently
		// persist an invalid (e.g. inverted-time) row from a tampered export.
		if validateImportEvent(ev) != "" {
			eventsSkipped++
			continue
		}

		tags := ev.Tags
		if tags == nil {
			tags = []string{}
		}

		result, err := db.Pool.Exec(r.Context(), `
			INSERT INTO event (
				id, user_id, title, description, starts_at, ends_at, all_day, rrule,
				timezone, location, color, tags, task_id, is_work_event, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13, $14, $15, $16
			)
			ON CONFLICT (id) DO NOTHING
		`,
			ev.ID, userID, ev.Title, ev.Description, ev.StartsAt, ev.EndsAt, ev.AllDay, ev.Rrule,
			ev.Timezone, ev.Location, ev.Color, tags, ev.TaskID, ev.IsWorkEvent, ev.CreatedAt, ev.UpdatedAt,
		)
		if err == nil && result.RowsAffected() > 0 {
			eventsImported++
		}
	}

	tasksImported := 0
	tasksSkipped := 0
	for _, tk := range req.Tasks {
		if validateImportTask(tk) != "" {
			tasksSkipped++
			continue
		}

		tags := tk.Tags
		if tags == nil {
			tags = []string{}
		}
		contexts := tk.Contexts
		if contexts == nil {
			contexts = []string{}
		}

		result, err := db.Pool.Exec(r.Context(), `
			INSERT INTO task (
				id, user_id, title, description, status, category, priority,
				estimated_minutes, due_date, tags, contexts, energy, parent_id,
				completed_at, created_at, updated_at, actual_minutes
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12, $13,
				$14, $15, $16, $17
			)
			ON CONFLICT (id) DO NOTHING
		`,
			tk.ID, userID, tk.Title, tk.Description, tk.Status, tk.Category, tk.Priority,
			tk.EstimatedMinutes, tk.DueDate, tags, contexts, tk.Energy, tk.ParentID,
			tk.CompletedAt, tk.CreatedAt, tk.UpdatedAt, tk.ActualMinutes,
		)
		if err == nil && result.RowsAffected() > 0 {
			tasksImported++
		}
	}

	util.RespondJSON(w, http.StatusOK, ImportResult{
		EventsImported: eventsImported,
		TasksImported:  tasksImported,
		EventsSkipped:  eventsSkipped,
		TasksSkipped:   tasksSkipped,
	})
}
