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
	// ReminderOffsets holds minutes-before-due_date for each reminder.
	ReminderOffsets []int      `json:"reminder_offsets"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
	// Pointer, not a bare slice: an absent field means "apply the user's
	// default preset", an explicitly empty array means "deliberately no
	// reminders".
	ReminderOffsets *[]int `json:"reminder_offsets,omitempty"`
}

// BatchCreateRequest creates many tasks in one round-trip, so pasting a list
// of twenty is one request rather than twenty.
type BatchCreateRequest struct {
	Tasks []CreateTaskRequest `json:"tasks"`
}

// BatchRowError reports a single row that could not be created.
// Index is the row's position in the submitted Tasks slice.
type BatchRowError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchCreateResponse carries the tasks that were created plus the rows that
// failed. Rows are independent: one bad line does not discard a whole paste.
type BatchCreateResponse struct {
	Tasks  []Task          `json:"tasks"`
	Errors []BatchRowError `json:"errors"`
}

// MaxBatchTasks caps one batch request, guarding against an accidental paste
// of a very large document.
const MaxBatchTasks = 100

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
	ReminderOffsets  *[]int        `json:"reminder_offsets,omitempty"`
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
