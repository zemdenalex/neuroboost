package feedback

import (
	"context"
	"encoding/json"
	"net/http"
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

// Feedback represents a feedback entry
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

// UpdateRequest for updating feedback
type UpdateRequest struct {
	Status   *string `json:"status,omitempty"`
	Priority *string `json:"priority,omitempty"`
}

// Create handles POST /api/feedback
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate type
	if req.Type != "bug" && req.Type != "feature" && req.Type != "other" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_TYPE", "Type must be 'bug', 'feature', or 'other'")
		return
	}

	// Validate required fields
	if req.Title == "" || req.Description == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_FIELDS", "Title and description are required")
		return
	}

	// Get user ID if authenticated (optional)
	userID := middleware.UserIDFromContext(r.Context())
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Insert feedback
	var feedback Feedback
	err := h.db.Pool.QueryRow(r.Context(), `
		INSERT INTO feedback (user_id, type, title, description, page_url, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, title, description, page_url, user_agent, status, priority, created_at
	`, userIDPtr, req.Type, req.Title, req.Description, nullString(req.PageURL), nullString(req.UserAgent)).Scan(
		&feedback.ID, &feedback.UserID, &feedback.Type, &feedback.Title, &feedback.Description,
		&feedback.PageURL, &feedback.UserAgent, &feedback.Status, &feedback.Priority, &feedback.CreatedAt,
	)

	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create feedback")
		return
	}

	util.RespondJSON(w, http.StatusCreated, feedback)
}

// List handles GET /api/feedback
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
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

	// Get status filter
	status := r.URL.Query().Get("status")

	var rows pgx.Rows
	if status != "" {
		rows, err = h.db.Pool.Query(r.Context(), `
			SELECT id, user_id, type, title, description, page_url, user_agent, status, priority, created_at, resolved_at
			FROM feedback
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT 100
		`, status)
	} else {
		rows, err = h.db.Pool.Query(r.Context(), `
			SELECT id, user_id, type, title, description, page_url, user_agent, status, priority, created_at, resolved_at
			FROM feedback
			ORDER BY created_at DESC
			LIMIT 100
		`)
	}

	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "QUERY_ERROR", "Failed to fetch feedback")
		return
	}
	defer rows.Close()

	var feedbackList []Feedback
	for rows.Next() {
		var f Feedback
		err := rows.Scan(
			&f.ID, &f.UserID, &f.Type, &f.Title, &f.Description,
			&f.PageURL, &f.UserAgent, &f.Status, &f.Priority, &f.CreatedAt, &f.ResolvedAt,
		)
		if err != nil {
			continue
		}
		feedbackList = append(feedbackList, f)
	}

	if feedbackList == nil {
		feedbackList = []Feedback{}
	}

	util.RespondJSON(w, http.StatusOK, feedbackList)
}

// Update handles PATCH /api/feedback/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	feedbackID := chi.URLParam(r, "id")
	if feedbackID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Feedback ID is required")
		return
	}

	// Check if user is admin
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

	// Build update query
	if req.Status != nil {
		// Validate status
		validStatuses := map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true}
		if !validStatuses[*req.Status] {
			util.RespondError(w, http.StatusBadRequest, "INVALID_STATUS", "Invalid status value")
			return
		}

		resolvedAt := (*time.Time)(nil)
		if *req.Status == "resolved" || *req.Status == "closed" {
			now := time.Now()
			resolvedAt = &now
		}

		_, err = h.db.Pool.Exec(r.Context(), `
			UPDATE feedback SET status = $1, resolved_at = $2 WHERE id = $3
		`, *req.Status, resolvedAt, feedbackID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update status")
			return
		}
	}

	if req.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if !validPriorities[*req.Priority] {
			util.RespondError(w, http.StatusBadRequest, "INVALID_PRIORITY", "Invalid priority value")
			return
		}

		_, err = h.db.Pool.Exec(r.Context(), `
			UPDATE feedback SET priority = $1 WHERE id = $2
		`, *req.Priority, feedbackID)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update priority")
			return
		}
	}

	// Fetch updated feedback
	var f Feedback
	err = h.db.Pool.QueryRow(r.Context(), `
		SELECT id, user_id, type, title, description, page_url, user_agent, status, priority, created_at, resolved_at
		FROM feedback WHERE id = $1
	`, feedbackID).Scan(
		&f.ID, &f.UserID, &f.Type, &f.Title, &f.Description,
		&f.PageURL, &f.UserAgent, &f.Status, &f.Priority, &f.CreatedAt, &f.ResolvedAt,
	)

	if err != nil {
		util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Feedback not found")
		return
	}

	util.RespondJSON(w, http.StatusOK, f)
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
