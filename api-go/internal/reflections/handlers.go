package reflections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

var db *database.DB

// InitDB sets the database connection for the reflections package
func InitDB(database *database.DB) {
	db = database
}

// validateScale checks that a value is between 1 and 10 (inclusive)
func validateScale(val *int, field string) error {
	if val != nil && (*val < 1 || *val > 10) {
		return fmt.Errorf("%s must be between 1 and 10", field)
	}
	return nil
}

// CreateHandler handles POST /api/reflections (event_id in body)
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

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.EventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_EVENT_ID", "Event ID is required")
		return
	}

	// Validate scale fields
	if err := validateScale(req.Energy, "energy"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_ENERGY", err.Error())
		return
	}
	if err := validateScale(req.Mood, "mood"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_MOOD", err.Error())
		return
	}
	if err := validateScale(req.Focus, "focus"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_FOCUS", err.Error())
		return
	}

	reflection, err := upsertReflection(r.Context(), userID, req.EventID, req)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create reflection")
		return
	}

	util.RespondJSON(w, http.StatusCreated, reflection)
}

// CreateForEventHandler handles POST /api/events/{id}/reflection (event_id from URL)
func CreateForEventHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Event ID is required")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Override event_id from URL
	req.EventID = eventID

	// Validate scale fields
	if err := validateScale(req.Energy, "energy"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_ENERGY", err.Error())
		return
	}
	if err := validateScale(req.Mood, "mood"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_MOOD", err.Error())
		return
	}
	if err := validateScale(req.Focus, "focus"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_FOCUS", err.Error())
		return
	}

	reflection, err := upsertReflection(r.Context(), userID, eventID, req)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create reflection")
		return
	}

	util.RespondJSON(w, http.StatusCreated, reflection)
}

// ListHandler handles GET /api/reflections
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

	eventID := r.URL.Query().Get("event_id")

	reflections, err := listReflections(r.Context(), userID, eventID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch reflections")
		return
	}

	util.RespondJSON(w, http.StatusOK, reflections)
}

// UpdateHandler handles PATCH /api/reflections/{id}
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

	reflectionID := chi.URLParam(r, "id")
	if reflectionID == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Reflection ID is required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate scale fields
	if err := validateScale(req.Energy, "energy"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_ENERGY", err.Error())
		return
	}
	if err := validateScale(req.Mood, "mood"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_MOOD", err.Error())
		return
	}
	if err := validateScale(req.Focus, "focus"); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_FOCUS", err.Error())
		return
	}

	reflection, err := updateReflection(r.Context(), userID, reflectionID, req)
	if err != nil {
		if err == pgx.ErrNoRows {
			util.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Reflection not found")
			return
		}
		util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update reflection")
		return
	}

	util.RespondJSON(w, http.StatusOK, reflection)
}

// Database operations

func upsertReflection(ctx context.Context, userID, eventID string, req CreateRequest) (*Reflection, error) {
	wasCompleted := true
	if req.WasCompleted != nil {
		wasCompleted = *req.WasCompleted
	}

	wasOnTime := true
	if req.WasOnTime != nil {
		wasOnTime = *req.WasOnTime
	}

	var ref Reflection
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO reflection (user_id, event_id, energy, mood, focus, notes, was_completed, was_on_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, event_id) DO UPDATE SET
			energy = COALESCE(EXCLUDED.energy, reflection.energy),
			mood = COALESCE(EXCLUDED.mood, reflection.mood),
			focus = COALESCE(EXCLUDED.focus, reflection.focus),
			notes = COALESCE(EXCLUDED.notes, reflection.notes),
			was_completed = EXCLUDED.was_completed,
			was_on_time = EXCLUDED.was_on_time
		RETURNING id, user_id, event_id, energy, mood, focus, notes, was_completed, was_on_time, created_at
	`, userID, eventID, req.Energy, req.Mood, req.Focus, req.Notes, wasCompleted, wasOnTime).Scan(
		&ref.ID, &ref.UserID, &ref.EventID, &ref.Energy, &ref.Mood, &ref.Focus,
		&ref.Notes, &ref.WasCompleted, &ref.WasOnTime, &ref.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &ref, nil
}

func listReflections(ctx context.Context, userID, eventID string) ([]Reflection, error) {
	query := `
		SELECT id, user_id, event_id, energy, mood, focus, notes, was_completed, was_on_time, created_at
		FROM reflection
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argNum := 2

	if eventID != "" {
		query += fmt.Sprintf(" AND event_id = $%d", argNum)
		args = append(args, eventID)
		argNum++
	}

	_ = argNum // suppress unused warning

	query += " ORDER BY created_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reflections []Reflection
	for rows.Next() {
		var ref Reflection
		err := rows.Scan(
			&ref.ID, &ref.UserID, &ref.EventID, &ref.Energy, &ref.Mood, &ref.Focus,
			&ref.Notes, &ref.WasCompleted, &ref.WasOnTime, &ref.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reflections = append(reflections, ref)
	}

	if reflections == nil {
		reflections = []Reflection{}
	}

	return reflections, nil
}

func updateReflection(ctx context.Context, userID, reflectionID string, req UpdateRequest) (*Reflection, error) {
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Energy != nil {
		updates = append(updates, fmt.Sprintf("energy = $%d", argNum))
		args = append(args, *req.Energy)
		argNum++
	}
	if req.Mood != nil {
		updates = append(updates, fmt.Sprintf("mood = $%d", argNum))
		args = append(args, *req.Mood)
		argNum++
	}
	if req.Focus != nil {
		updates = append(updates, fmt.Sprintf("focus = $%d", argNum))
		args = append(args, *req.Focus)
		argNum++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argNum))
		args = append(args, *req.Notes)
		argNum++
	}
	if req.WasCompleted != nil {
		updates = append(updates, fmt.Sprintf("was_completed = $%d", argNum))
		args = append(args, *req.WasCompleted)
		argNum++
	}
	if req.WasOnTime != nil {
		updates = append(updates, fmt.Sprintf("was_on_time = $%d", argNum))
		args = append(args, *req.WasOnTime)
		argNum++
	}

	if len(updates) == 0 {
		return getReflection(ctx, userID, reflectionID)
	}

	args = append(args, reflectionID, userID)

	query := fmt.Sprintf(`
		UPDATE reflection SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, event_id, energy, mood, focus, notes, was_completed, was_on_time, created_at
	`, strings.Join(updates, ", "), argNum, argNum+1)

	var ref Reflection
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&ref.ID, &ref.UserID, &ref.EventID, &ref.Energy, &ref.Mood, &ref.Focus,
		&ref.Notes, &ref.WasCompleted, &ref.WasOnTime, &ref.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &ref, nil
}

func getReflection(ctx context.Context, userID, reflectionID string) (*Reflection, error) {
	var ref Reflection
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, event_id, energy, mood, focus, notes, was_completed, was_on_time, created_at
		FROM reflection
		WHERE id = $1 AND user_id = $2
	`, reflectionID, userID).Scan(
		&ref.ID, &ref.UserID, &ref.EventID, &ref.Energy, &ref.Mood, &ref.Focus,
		&ref.Notes, &ref.WasCompleted, &ref.WasOnTime, &ref.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &ref, nil
}
