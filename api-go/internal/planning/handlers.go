package planning

import (
	"context"
	"net/http"
	"sort"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/events"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/usersettings"
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

	// 🔴 The week is the USER's: Monday 00:00 where they are. It was cut at
	// Monday 00:00 UTC, which put a Moscow user's Monday 01:00 meeting into
	// the week before.
	loc := userLocation(r.Context(), userID)

	// Parse date param — accept "2026-04-07" or RFC3339
	dateParam := r.URL.Query().Get("date")
	var anchorDate time.Time
	var err error

	if dateParam == "" {
		anchorDate = time.Now().In(loc)
	} else {
		// Try date-only first (a day on the user's calendar), then RFC3339
		anchorDate, err = time.ParseInLocation("2006-01-02", dateParam, loc)
		if err != nil {
			anchorDate, err = time.Parse(time.RFC3339, dateParam)
			if err != nil {
				util.RespondError(w, http.StatusBadRequest, "INVALID_DATE", "Invalid date format. Use YYYY-MM-DD")
				return
			}
			anchorDate = anchorDate.In(loc)
		}
	}

	// Compute Monday of the week (ISO: Monday = 1)
	weekday := int(anchorDate.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> treat as 7 so Monday offset works
	}
	daysToMonday := weekday - 1
	weekStart := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day()-daysToMonday,
		0, 0, 0, 0, loc)
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
		// 🔴 Was the literal 40, "8h × 5 days", for as long as this endpoint has
		// existed. The user's own work_start/work_end/work_days were written by
		// the Settings page, copied into localStorage by AuthContext, and read
		// by nothing — this line is the reader they never had, and without it
		// the setting is decorative in every client.
		//
		// Defaults to the same 40 when the blob is empty or unreadable, so the
		// number does not move for anyone who has not set it.
		AvailableHours: usersettings.LoadWorkWeek(r.Context(), userID).Hours(),
	}

	util.RespondJSON(w, http.StatusOK, plan)
}

// listUnscheduledTasks returns tasks that are not DONE or SCHEDULED
func listUnscheduledTasks(ctx context.Context, userID string) ([]PlanningTask, error) {
	calIDs, err := calendars.CalendarIDsFor(ctx, userID)
	if err != nil {
		return nil, err
	}

	// An empty list is a legitimate "nothing visible", not an error:
	// ANY('{}') returns zero rows.
	// 🔴 A repeating task whose TODAY is already done is not unfinished work.
	// `status` stays TODO for the whole series, so without this join the planner
	// would keep offering «выпить таблетки» to be scheduled again, every time,
	// after it had been ticked.
	rows, err := db.Pool.Query(ctx, `
		SELECT t.id, t.title, t.priority, t.estimated_minutes,
		       CASE WHEN t.due_date IS NOT NULL THEN to_char(t.due_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       t.category
		FROM task t
		LEFT JOIN task_occurrence o
		       ON o.task_id = t.id
		      AND o.occurrence = (NOW() AT TIME ZONE COALESCE(
		      -- 🔴 The VIEWER's zone ($2), not the author's.
		      --
		      -- A shared calendar holds tasks written by other people, so
		      -- t.user_id is whoever created the series — not whoever is asking
		      -- what day it is. Reading their zone would tell a Moscow reader
		      -- whether the Tokyo day was done. Same class as the nag bug fixed
		      -- the same morning: a time that is valid, just not the reader's.
		            (SELECT timezone FROM "user" WHERE id = $2), 'Europe/Moscow'))::date
		WHERE t.calendar_id = ANY($1)
		  AND t.status NOT IN ('DONE', 'SCHEDULED', 'CANCELLED')
		  AND o.state IS NULL
		ORDER BY t.priority ASC, t.due_date ASC NULLS LAST, t.created_at DESC
	`, calIDs, userID)
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

// listWeekEvents returns the week's events and the hours they take.
//
// 🔴 Through events.ListExpanded, the same reader as GET /api/events: reading
// rows by starts_at counted a repeating series only in the week of its first
// row, so every later week of a weekly meeting planned as empty.
func listWeekEvents(ctx context.Context, userID string, weekStart, weekEnd time.Time) ([]PlanningEvent, float64, error) {
	list, err := events.ListExpanded(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, 0, err
	}

	week := []PlanningEvent{}
	var totalHours float64
	for _, e := range list {
		// ListExpanded answers what overlaps the range; the plan lists what
		// STARTS in the week, as it always has.
		if e.StartsAt.Before(weekStart) || !e.StartsAt.Before(weekEnd) {
			continue
		}
		week = append(week, PlanningEvent{
			ID: e.ID, Title: e.Title, StartsAt: e.StartsAt, EndsAt: e.EndsAt, AllDay: e.AllDay, Color: e.Color,
		})
		if !e.AllDay {
			totalHours += e.EndsAt.Sub(e.StartsAt).Hours()
		}
	}
	// ListExpanded keeps a series' occurrences together in parent order; the
	// plan is read by time.
	sort.SliceStable(week, func(i, j int) bool { return week[i].StartsAt.Before(week[j].StartsAt) })
	return week, totalHours, nil
}

// userLocation is the user's zone, Moscow when unset or unreadable (the same
// fallback the rest of the API uses).
func userLocation(ctx context.Context, userID string) *time.Location {
	var tz string
	if err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(timezone, 'Europe/Moscow') FROM "user" WHERE id = $1`, userID).Scan(&tz); err != nil {
		tz = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		if loc, err = time.LoadLocation("Europe/Moscow"); err != nil {
			return time.UTC
		}
	}
	return loc
}
