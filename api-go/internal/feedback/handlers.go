package feedback

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

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// Feedback represents a backlog item (bug, feature, design, improvement, idea, other)
type Feedback struct {
	ID          string     `json:"id"`
	UserID      *string    `json:"user_id,omitempty"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	PageURL     *string    `json:"page_url,omitempty"`
	UserAgent   *string    `json:"user_agent,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AdminNotes  string     `json:"admin_notes"`
	Tags        []string   `json:"tags"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// CreateRequest for creating feedback
type CreateRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PageURL     string `json:"page_url,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
}

// UpdateRequest for updating feedback (admin)
type UpdateRequest struct {
	Status     *string   `json:"status,omitempty"`
	Priority   *string   `json:"priority,omitempty"`
	AdminNotes *string   `json:"admin_notes,omitempty"`
	Tags       *[]string `json:"tags,omitempty"`
}

// ImportItem represents a single item in a bulk import
type ImportItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Source      string   `json:"source"`
}

// ImportRequest for bulk importing backlog items
type ImportRequest struct {
	Items []ImportItem `json:"items"`
}

var validTypes = map[string]bool{
	"bug": true, "feature": true, "design": true,
	"improvement": true, "idea": true, "other": true,
}

var validStatuses = map[string]bool{
	"open": true, "triaged": true, "approved": true, "denied": true,
	"in_progress": true, "resolved": true, "closed": true,
}

var validPriorities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

var validSortColumns = map[string]string{
	"created_at": "created_at",
	"priority":   "priority",
	"status":     "status",
	"type":       "type",
	"title":      "title",
}

// allColumns is the SELECT column list for feedback queries
const allColumns = `id, user_id, type, title, description, page_url, user_agent, status, priority, admin_notes, tags, source, created_at, resolved_at`

func scanFeedback(row pgx.Row) (Feedback, error) {
	var f Feedback
	err := row.Scan(
		&f.ID, &f.UserID, &f.Type, &f.Title, &f.Description,
		&f.PageURL, &f.UserAgent, &f.Status, &f.Priority,
		&f.AdminNotes, &f.Tags, &f.Source, &f.CreatedAt, &f.ResolvedAt,
	)
	if f.Tags == nil {
		f.Tags = []string{}
	}
	return f, err
}

func scanFeedbackRows(rows pgx.Rows) []Feedback {
	var list []Feedback
	for rows.Next() {
		var f Feedback
		err := rows.Scan(
			&f.ID, &f.UserID, &f.Type, &f.Title, &f.Description,
			&f.PageURL, &f.UserAgent, &f.Status, &f.Priority,
			&f.AdminNotes, &f.Tags, &f.Source, &f.CreatedAt, &f.ResolvedAt,
		)
		if err != nil {
			continue
		}
		if f.Tags == nil {
			f.Tags = []string{}
		}
		list = append(list, f)
	}
	if list == nil {
		list = []Feedback{}
	}
	return list
}

// Create handles POST /api/feedback
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if !validTypes[req.Type] {
		util.RespondError(w, http.StatusBadRequest, "INVALID_TYPE", "Invalid feedback type")
		return
	}

	if req.Title == "" || req.Description == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_FIELDS", "Title and description are required")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	row := h.db.Pool.QueryRow(r.Context(), `
		INSERT INTO feedback (user_id, type, title, description, page_url, user_agent, source)
		VALUES ($1, $2, $3, $4, $5, $6, 'user')
		RETURNING `+allColumns,
		userIDPtr, req.Type, req.Title, req.Description,
		nullString(req.PageURL), nullString(req.UserAgent),
	)

	feedback, err := scanFeedback(row)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create feedback")
		return
	}

	util.RespondJSON(w, http.StatusCreated, feedback)
}

// List handles GET /api/feedback with filtering and sorting
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
		return
	}

	isAdmin, err := h.isUserAdmin(r.Context(), userID)
	if err != nil || !isAdmin {
		util.RespondError(w, http.StatusForbidden, "NOT_ADMIN", "Admin access required")
		return
	}

	q := r.URL.Query()

	// Build dynamic WHERE clause
	var conditions []string
	var args []interface{}
	argIdx := 1

	if v := q.Get("status"); v != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v := q.Get("type"); v != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v := q.Get("priority"); v != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v := q.Get("source"); v != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v := q.Get("search"); v != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", argIdx))
		args = append(args, "%"+v+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Sort
	orderCol := "created_at"
	if col, ok := validSortColumns[q.Get("sort_by")]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if q.Get("sort_dir") == "asc" {
		orderDir = "ASC"
	}

	query := fmt.Sprintf("SELECT %s FROM feedback %s ORDER BY %s %s LIMIT 500",
		allColumns, where, orderCol, orderDir)

	rows, err := h.db.Pool.Query(r.Context(), query, args...)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "QUERY_ERROR", "Failed to fetch feedback")
		return
	}
	defer rows.Close()

	util.RespondJSON(w, http.StatusOK, scanFeedbackRows(rows))
}

// Update handles PATCH /api/feedback/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	feedbackID := chi.URLParam(r, "id")
	if feedbackID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Feedback ID is required")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
		return
	}

	isAdmin, err := h.isUserAdmin(r.Context(), userID)
	if err != nil || !isAdmin {
		util.RespondError(w, http.StatusForbidden, "NOT_ADMIN", "Admin access required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Build dynamic update
	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Status != nil {
		if !validStatuses[*req.Status] {
			util.RespondError(w, http.StatusBadRequest, "INVALID_STATUS", "Invalid status value")
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++

		// Set resolved_at when marking as resolved or closed
		if *req.Status == "resolved" || *req.Status == "closed" {
			now := time.Now()
			setClauses = append(setClauses, fmt.Sprintf("resolved_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
		}
	}

	if req.Priority != nil {
		if !validPriorities[*req.Priority] {
			util.RespondError(w, http.StatusBadRequest, "INVALID_PRIORITY", "Invalid priority value")
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}

	if req.AdminNotes != nil {
		setClauses = append(setClauses, fmt.Sprintf("admin_notes = $%d", argIdx))
		args = append(args, *req.AdminNotes)
		argIdx++
	}

	if req.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *req.Tags)
		argIdx++
	}

	if len(setClauses) == 0 {
		util.RespondError(w, http.StatusBadRequest, "NO_UPDATES", "No fields to update")
		return
	}

	args = append(args, feedbackID)
	query := fmt.Sprintf("UPDATE feedback SET %s WHERE id = $%d RETURNING %s",
		strings.Join(setClauses, ", "), argIdx, allColumns)

	row := h.db.Pool.QueryRow(r.Context(), query, args...)
	f, err := scanFeedback(row)
	if err != nil {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Feedback not found")
		return
	}

	util.RespondJSON(w, http.StatusOK, f)
}

// Import handles POST /api/feedback/import — bulk import backlog items
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
		return
	}

	isAdmin, err := h.isUserAdmin(r.Context(), userID)
	if err != nil || !isAdmin {
		util.RespondError(w, http.StatusForbidden, "NOT_ADMIN", "Admin access required")
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if len(req.Items) == 0 {
		util.RespondError(w, http.StatusBadRequest, "EMPTY_IMPORT", "No items to import")
		return
	}

	count := 0
	for _, item := range req.Items {
		if item.Title == "" {
			continue
		}
		if !validTypes[item.Type] {
			item.Type = "other"
		}
		if !validStatuses[item.Status] {
			item.Status = "open"
		}
		if !validPriorities[item.Priority] {
			item.Priority = "medium"
		}
		if item.Source == "" {
			item.Source = "imported"
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}

		resolvedAt := (*time.Time)(nil)
		if item.Status == "resolved" || item.Status == "closed" {
			now := time.Now()
			resolvedAt = &now
		}

		_, err := h.db.Pool.Exec(r.Context(), `
			INSERT INTO feedback (type, title, description, status, priority, tags, source, resolved_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, item.Type, item.Title, item.Description, item.Status, item.Priority, item.Tags, item.Source, resolvedAt)

		if err != nil {
			continue
		}
		count++
	}

	util.RespondJSON(w, http.StatusCreated, map[string]int{"count": count})
}

// Helper functions

func (h *Handler) isUserAdmin(ctx context.Context, userID string) (bool, error) {
	var isAdmin bool
	err := h.db.Pool.QueryRow(ctx, `SELECT COALESCE(is_admin, FALSE) FROM "user" WHERE id = $1`, userID).Scan(&isAdmin)
	return isAdmin, err
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
