package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB sets the database connection for the tasks package
func InitDB(database *database.DB) {
	db = database
}

// ListHandler returns all tasks for a user
func ListHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")
	context := r.URL.Query().Get("context")

	tasks, err := listTasks(r.Context(), userID, status, category, context)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch tasks")
		return
	}

	util.RespondJSON(w, http.StatusOK, tasks)
}

// CreateHandler creates a new task
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate required fields
	if req.Title == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_TITLE", "Title is required")
		return
	}

	// Parse due date if provided
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_DUE_DATE", "Invalid due date format")
			return
		}
		dueDate = &t
	}

	// Set defaults
	status := StatusTodo
	if req.Status != nil {
		status = *req.Status
	}

	priority := 3
	if req.Priority != nil {
		priority = *req.Priority
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	contexts := req.Contexts
	if contexts == nil {
		contexts = []string{}
	}

	task, err := createTask(r.Context(), userID, req, status, priority, dueDate, tags, contexts)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create task")
		return
	}

	util.RespondJSON(w, http.StatusCreated, task)
}

// GetHandler returns a single task
func GetHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	task, err := getTask(r.Context(), userID, taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch task")
		return
	}

	util.RespondJSON(w, http.StatusOK, task)
}

// UpdateHandler updates a task
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	task, err := updateTask(r.Context(), userID, taskID, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update task")
		return
	}

	util.RespondJSON(w, http.StatusOK, task)
}

// DeleteHandler deletes a task
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	err := deleteTask(r.Context(), userID, taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "DELETE_ERROR", "Failed to delete task")
		return
	}

	util.RespondJSON(w, http.StatusOK, map[string]string{"message": "Task deleted"})
}

// ScheduleHandler schedules a task as an event
func ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Task ID is required")
		return
	}

	var req ScheduleTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_START", "Invalid start date format")
		return
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_END", "Invalid end date format")
		return
	}

	event, err := scheduleTask(r.Context(), userID, taskID, startsAt, endsAt, req.AllDay, req.Color)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Task not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "SCHEDULE_ERROR", "Failed to schedule task")
		return
	}

	util.RespondJSON(w, http.StatusCreated, event)
}

// Database operations

func listTasks(ctx context.Context, userID, status, category, taskContext string) ([]Task, error) {
	query := `
		SELECT id, user_id, title, description, status, category, priority,
		       estimated_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		       energy, parent_id, completed_at, created_at, updated_at, actual_minutes
		FROM task
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argNum := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argNum)
		args = append(args, category)
		argNum++
	}

	if taskContext != "" {
		query += fmt.Sprintf(" AND $%d = ANY(contexts)", argNum)
		args = append(args, taskContext)
		argNum++
	}

	query += " ORDER BY priority ASC, due_date ASC NULLS LAST, created_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var tags, contexts []string
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Category,
			&t.Priority, &t.EstimatedMinutes, &t.DueDate, &tags, &contexts,
			&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
		)
		if err != nil {
			return nil, err
		}
		t.Tags = tags
		t.Contexts = contexts
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []Task{}
	}

	return tasks, nil
}

func createTask(ctx context.Context, userID string, req CreateTaskRequest, status TaskStatus, priority int, dueDate *time.Time, tags, contexts []string) (*Task, error) {
	var t Task
	var resultTags, resultContexts []string

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO task (user_id, title, description, status, category, priority, estimated_minutes, due_date, tags, contexts, energy, parent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, user_id, title, description, status, category, priority,
		          estimated_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		          energy, parent_id, completed_at, created_at, updated_at, actual_minutes
	`, userID, req.Title, req.Description, status, req.Category, priority, req.EstimatedMinutes, dueDate, tags, contexts, req.Energy, req.ParentID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Category,
		&t.Priority, &t.EstimatedMinutes, &t.DueDate, &resultTags, &resultContexts,
		&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
	)

	if err != nil {
		return nil, err
	}

	t.Tags = resultTags
	t.Contexts = resultContexts
	return &t, nil
}

func getTask(ctx context.Context, userID, taskID string) (*Task, error) {
	var t Task
	var tags, contexts []string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, title, description, status, category, priority,
		       estimated_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		       energy, parent_id, completed_at, created_at, updated_at, actual_minutes
		FROM task
		WHERE id = $1 AND user_id = $2
	`, taskID, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Category,
		&t.Priority, &t.EstimatedMinutes, &t.DueDate, &tags, &contexts,
		&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
	)

	if err != nil {
		return nil, err
	}

	t.Tags = tags
	t.Contexts = contexts
	return &t, nil
}

func updateTask(ctx context.Context, userID, taskID string, req UpdateTaskRequest) (*Task, error) {
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *req.Title)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++

		// Set completed_at when status changes to DONE
		if *req.Status == StatusDone {
			updates = append(updates, fmt.Sprintf("completed_at = $%d", argNum))
			args = append(args, time.Now())
			argNum++
		} else if *req.Status == StatusTodo || *req.Status == StatusInProgress {
			updates = append(updates, "completed_at = NULL")
		}
	}
	if req.Category != nil {
		updates = append(updates, fmt.Sprintf("category = $%d", argNum))
		args = append(args, *req.Category)
		argNum++
	}
	if req.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argNum))
		args = append(args, *req.Priority)
		argNum++
	}
	if req.EstimatedMinutes != nil {
		updates = append(updates, fmt.Sprintf("estimated_minutes = $%d", argNum))
		args = append(args, *req.EstimatedMinutes)
		argNum++
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			updates = append(updates, "due_date = NULL")
		} else {
			t, err := time.Parse(time.RFC3339, *req.DueDate)
			if err != nil {
				return nil, err
			}
			updates = append(updates, fmt.Sprintf("due_date = $%d", argNum))
			args = append(args, t)
			argNum++
		}
	}
	if req.Tags != nil {
		updates = append(updates, fmt.Sprintf("tags = $%d", argNum))
		args = append(args, req.Tags)
		argNum++
	}
	if req.Contexts != nil {
		updates = append(updates, fmt.Sprintf("contexts = $%d", argNum))
		args = append(args, req.Contexts)
		argNum++
	}
	if req.Energy != nil {
		updates = append(updates, fmt.Sprintf("energy = $%d", argNum))
		args = append(args, *req.Energy)
		argNum++
	}
	if req.ParentID != nil {
		updates = append(updates, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, *req.ParentID)
		argNum++
	}

	if len(updates) == 0 {
		return getTask(ctx, userID, taskID)
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, taskID, userID)

	query := fmt.Sprintf(`
		UPDATE task SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, title, description, status, category, priority,
		          estimated_minutes, due_date, COALESCE(tags, '{}'), COALESCE(contexts, '{}'),
		          energy, parent_id, completed_at, created_at, updated_at, actual_minutes
	`, strings.Join(updates, ", "), argNum, argNum+1)

	var t Task
	var tags, contexts []string

	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Category,
		&t.Priority, &t.EstimatedMinutes, &t.DueDate, &tags, &contexts,
		&t.Energy, &t.ParentID, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ActualMinutes,
	)

	if err != nil {
		return nil, err
	}

	t.Tags = tags
	t.Contexts = contexts
	return &t, nil
}

func deleteTask(ctx context.Context, userID, taskID string) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM task WHERE id = $1 AND user_id = $2
	`, taskID, userID)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// ScheduledEvent represents the event created from a task
type ScheduledEvent struct {
	ID       string    `json:"id"`
	TaskID   string    `json:"task_id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	AllDay   bool      `json:"all_day"`
	Color    *string   `json:"color,omitempty"`
}

func scheduleTask(ctx context.Context, userID, taskID string, startsAt, endsAt time.Time, allDay bool, color *string) (*ScheduledEvent, error) {
	// First get the task to use its title
	task, err := getTask(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	// Create event from task
	var event ScheduledEvent
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO event (user_id, title, starts_at, ends_at, all_day, task_id, color, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'Europe/Moscow')
		RETURNING id, task_id, title, starts_at, ends_at, all_day, color
	`, userID, task.Title, startsAt, endsAt, allDay, taskID, color).Scan(
		&event.ID, &event.TaskID, &event.Title, &event.StartsAt, &event.EndsAt, &event.AllDay, &event.Color,
	)

	if err != nil {
		return nil, err
	}

	// Update task status to SCHEDULED
	_, err = db.Pool.Exec(ctx, `
		UPDATE task SET status = 'SCHEDULED', updated_at = NOW() WHERE id = $1 AND user_id = $2
	`, taskID, userID)

	if err != nil {
		// Non-fatal, event was created
	}

	return &event, nil
}