package planning

import (
	"context"
	"net/http"
	"time"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB sets the database connection for the planning package
func InitDB(database *database.DB) {
	db = database
}

// WeekPlan is the response for GET /api/planning/week
type WeekPlan struct {
	WeekStart        time.Time       `json:"week_start"`
	WeekEnd          time.Time       `json:"week_end"`
	UnscheduledTasks []PlanningTask  `json:"unscheduled_tasks"`
	WeekEvents       []PlanningEvent `json:"week_events"`
	ScheduledHours   float64         `json:"scheduled_hours"`
	AvailableHours   float64         `json:"available_hours"`
}

// PlanningTask is a trimmed task representation for the planning view
type PlanningTask struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Priority         int     `json:"priority"`
	EstimatedMinutes *int    `json:"estimated_minutes,omitempty"`
	DueDate          *string `json:"due_date,omitempty"`
	Category         *string `json:"category,omitempty"`
}

// PlanningEvent is a trimmed event representation for the planning view
type PlanningEvent struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	AllDay   bool      `json:"all_day"`
	Color    *string   `json:"color,omitempty"`
}

// GetWeekHandler returns the weekly plan for a given date
// GET /api/planning/week?date=2026-04-07
func GetWeekHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	// Parse date param — accept "2026-04-07" or RFC3339
	dateParam := r.URL.Query().Get("date")
	var anchorDate time.Time
	var err error

	if dateParam == "" {
		anchorDate = time.Now().UTC()
	} else {
		// Try date-only first, then RFC3339
		anchorDate, err = time.Parse("2006-01-02", dateParam)
		if err != nil {
			anchorDate, err = time.Parse(time.RFC3339, dateParam)
			if err != nil {
				util.RespondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid date format. Use YYYY-MM-DD")
				return
			}
		}
		anchorDate = anchorDate.UTC()
	}

	// Compute Monday of the week (ISO: Monday = 1)
	weekday := int(anchorDate.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> treat as 7 so Monday offset works
	}
	daysToMonday := weekday - 1
	weekStart := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day()-daysToMonday,
		0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)

	tasks, err := listUnscheduledTasks(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch tasks")
		return
	}

	events, scheduledHours, err := listWeekEvents(r.Context(), userID, weekStart, weekEnd)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch events")
		return
	}

	plan := WeekPlan{
		WeekStart:        weekStart,
		WeekEnd:          weekEnd,
		UnscheduledTasks: tasks,
		WeekEvents:       events,
		ScheduledHours:   scheduledHours,
		AvailableHours:   40, // Default: 8h × 5 days
	}

	util.RespondJSON(w, http.StatusOK, plan)
}

// listUnscheduledTasks returns tasks that are not DONE or SCHEDULED
func listUnscheduledTasks(ctx context.Context, userID string) ([]PlanningTask, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, priority, estimated_minutes,
		       CASE WHEN due_date IS NOT NULL THEN to_char(due_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       category
		FROM task
		WHERE user_id = $1
		  AND status NOT IN ('DONE', 'SCHEDULED', 'CANCELLED')
		ORDER BY priority ASC, due_date ASC NULLS LAST, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []PlanningTask
	for rows.Next() {
		var t PlanningTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Priority, &t.EstimatedMinutes, &t.DueDate, &t.Category); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []PlanningTask{}
	}

	return tasks, nil
}

// listWeekEvents returns events within the week and computes total scheduled hours
func listWeekEvents(ctx context.Context, userID string, weekStart, weekEnd time.Time) ([]PlanningEvent, float64, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, starts_at, ends_at, all_day, color
		FROM event
		WHERE user_id = $1
		  AND starts_at >= $2
		  AND starts_at < $3
		ORDER BY starts_at ASC
	`, userID, weekStart, weekEnd)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []PlanningEvent
	var totalHours float64

	for rows.Next() {
		var e PlanningEvent
		if err := rows.Scan(&e.ID, &e.Title, &e.StartsAt, &e.EndsAt, &e.AllDay, &e.Color); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
		if !e.AllDay {
			totalHours += e.EndsAt.Sub(e.StartsAt).Hours()
		}
	}

	if events == nil {
		events = []PlanningEvent{}
	}

	return events, totalHours, nil
}
