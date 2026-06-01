package tasks

import "time"

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusTodo       TaskStatus = "TODO"
	StatusInProgress TaskStatus = "IN_PROGRESS"
	StatusScheduled  TaskStatus = "SCHEDULED"
	StatusDone       TaskStatus = "DONE"
	StatusCancelled  TaskStatus = "CANCELLED"
)

// TaskCategory represents the category/urgency of a task
type TaskCategory string

const (
	CategoryEmergency    TaskCategory = "EMERGENCY"
	CategoryAsap         TaskCategory = "ASAP"
	CategoryMustToday    TaskCategory = "MUST_TODAY"
	CategoryDeadlineSoon TaskCategory = "DEADLINE_SOON"
	CategoryIfPossible   TaskCategory = "IF_POSSIBLE"
	CategoryBuffer       TaskCategory = "BUFFER"
)

// Task represents a task in the system
type Task struct {
	ID               string        `json:"id"`
	UserID           string        `json:"user_id"`
	Title            string        `json:"title"`
	Description      *string       `json:"description,omitempty"`
	Status           TaskStatus    `json:"status"`
	Category         *TaskCategory `json:"category,omitempty"`
	Priority         int           `json:"priority"`
	EstimatedMinutes *int          `json:"estimated_minutes,omitempty"`
	ActualMinutes    int           `json:"actual_minutes"`
	DueDate          *time.Time    `json:"due_date,omitempty"`
	Tags             []string      `json:"tags"`
	Contexts         []string      `json:"contexts"`
	Energy           *int          `json:"energy,omitempty"`
	ParentID         *string       `json:"parent_id,omitempty"`
	CompletedAt      *time.Time    `json:"completed_at,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// CreateTaskRequest represents the request to create a task
type CreateTaskRequest struct {
	Title            string        `json:"title"`
	Description      *string       `json:"description,omitempty"`
	Status           *TaskStatus   `json:"status,omitempty"`
	Category         *TaskCategory `json:"category,omitempty"`
	Priority         *int          `json:"priority,omitempty"`
	EstimatedMinutes *int          `json:"estimated_minutes,omitempty"`
	DueDate          *string       `json:"due_date,omitempty"` // ISO 8601
	Tags             []string      `json:"tags,omitempty"`
	Contexts         []string      `json:"contexts,omitempty"`
	Energy           *int          `json:"energy,omitempty"`
	ParentID         *string       `json:"parent_id,omitempty"`
}

// UpdateTaskRequest represents the request to update a task
type UpdateTaskRequest struct {
	Title            *string       `json:"title,omitempty"`
	Description      *string       `json:"description,omitempty"`
	Status           *TaskStatus   `json:"status,omitempty"`
	Category         *TaskCategory `json:"category,omitempty"`
	Priority         *int          `json:"priority,omitempty"`
	EstimatedMinutes *int          `json:"estimated_minutes,omitempty"`
	DueDate          *string       `json:"due_date,omitempty"`
	Tags             []string      `json:"tags,omitempty"`
	Contexts         []string      `json:"contexts,omitempty"`
	Energy           *int          `json:"energy,omitempty"`
	ParentID         *string       `json:"parent_id,omitempty"`
}

// LogTimeRequest adds (or, with a negative value, removes) focused minutes
// to a task's actual_minutes total. Used by the Pomodoro timer + undo.
type LogTimeRequest struct {
	Minutes int `json:"minutes"`
}

// ScheduleTaskRequest represents scheduling a task as an event
type ScheduleTaskRequest struct {
	StartsAt string  `json:"starts_at"` // ISO 8601
	EndsAt   string  `json:"ends_at"`   // ISO 8601
	AllDay   bool    `json:"all_day"`
	Color    *string `json:"color,omitempty"`
}

// ListTasksQuery represents query parameters for listing tasks
type ListTasksQuery struct {
	Status   *TaskStatus   `json:"status,omitempty"`
	Category *TaskCategory `json:"category,omitempty"`
	Priority *int          `json:"priority,omitempty"`
	Context  *string       `json:"context,omitempty"`
}