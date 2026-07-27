package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"neuroboost/api-go/internal/config"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

type Handler struct {
	db  *database.DB
	cfg *config.Config
}

func NewHandler(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// TelegramLogin handles Telegram Login Widget authentication
func (h *Handler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	var req TelegramLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Verify Telegram hash
	if !h.verifyTelegramAuth(req) {
		util.RespondError(w, http.StatusUnauthorized, "INVALID_HASH", "Telegram authentication failed")
		return
	}

	// Check auth_date is not too old (allow 24 hours)
	if time.Now().Unix()-req.AuthDate > 86400 {
		util.RespondError(w, http.StatusUnauthorized, "AUTH_EXPIRED", "Authentication data expired")
		return
	}

	ctx := r.Context()

	// Try to find existing user by tg_id
	user, err := h.findUserByTgID(ctx, req.ID)
	if err != nil && err != pgx.ErrNoRows {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Database error")
		return
	}

	if user == nil {
		// Create new user
		user, err = h.createUserFromTelegram(ctx, req)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "CREATE_USER_ERROR", "Failed to create user")
			return
		}
	} else {
		// Update existing user's Telegram info
		err = h.updateTelegramInfo(ctx, user.ID, req)
		if err != nil {
			// Non-fatal, continue with login
		}
	}

	// Update last login
	h.updateLastLogin(ctx, user.ID)

	// Generate JWT
	token, expiresAt, err := h.generateJWT(user)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	util.RespondJSON(w, http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// Register handles email/password registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate email
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		util.RespondError(w, http.StatusBadRequest, "INVALID_EMAIL", "Valid email is required")
		return
	}

	// Validate password
	if len(req.Password) < 8 {
		util.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters")
		return
	}

	ctx := r.Context()

	// Check if email already exists
	existing, _ := h.findUserByEmail(ctx, req.Email)
	if existing != nil {
		util.RespondError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already registered")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to process password")
		return
	}

	// Create user
	user, err := h.createUserWithEmail(ctx, req.Email, string(hashedPassword), req.Name)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "CREATE_USER_ERROR", "Failed to create user")
		return
	}

	// Generate JWT
	token, expiresAt, err := h.generateJWT(user)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	util.RespondJSON(w, http.StatusCreated, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// Login handles email/password login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	ctx := r.Context()

	// Find user by email
	user, passwordHash, err := h.findUserWithPasswordByEmail(ctx, req.Email)
	if err != nil || user == nil {
		util.RespondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Verify password
	if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		util.RespondError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Update last login
	h.updateLastLogin(ctx, user.ID)

	// Generate JWT
	token, expiresAt, err := h.generateJWT(user)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	util.RespondJSON(w, http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// Me returns the current authenticated user
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	user, err := h.findUserByID(r.Context(), userID)
	if err != nil || user == nil {
		util.RespondError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	util.RespondJSON(w, http.StatusOK, user)
}

// UpdateMe updates the current user's profile and settings
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	ctx := r.Context()

	// Build update query dynamically based on provided fields
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.DisplayName != nil {
		updates = append(updates, fmt.Sprintf("display_name = $%d", argNum))
		args = append(args, *req.DisplayName)
		argNum++
	}

	if req.Timezone != nil {
		// Accept any valid IANA timezone (validated against the system tz database)
		// rather than a hand-maintained allowlist that drifted out of sync with the UI.
		if !validTimezone(*req.Timezone) {
			util.RespondError(w, http.StatusBadRequest, "INVALID_TIMEZONE", "Invalid timezone")
			return
		}
		updates = append(updates, fmt.Sprintf("timezone = $%d", argNum))
		args = append(args, *req.Timezone)
		argNum++
	}

	if req.Locale != nil {
		validLocales := map[string]bool{"ru": true, "en": true}
		if !validLocales[*req.Locale] {
			util.RespondError(w, http.StatusBadRequest, "INVALID_LOCALE", "Invalid locale")
			return
		}
		updates = append(updates, fmt.Sprintf("locale = $%d", argNum))
		args = append(args, *req.Locale)
		argNum++
	}

	if req.Settings != nil {
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			util.RespondError(w, http.StatusBadRequest, "INVALID_SETTINGS", "Invalid settings format")
			return
		}
		updates = append(updates, fmt.Sprintf("settings = $%d", argNum))
		args = append(args, settingsJSON)
		argNum++
	}

	if len(updates) == 0 {
		util.RespondError(w, http.StatusBadRequest, "NO_UPDATES", "No fields to update")
		return
	}

	// Add updated_at and user_id
	updates = append(updates, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE "user" SET %s
		WHERE id = $%d
	`, strings.Join(updates, ", "), argNum)

	_, err := h.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "UPDATE_ERROR", "Failed to update user")
		return
	}

	// Return updated user
	user, err := h.findUserByID(ctx, userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch updated user")
		return
	}

	util.RespondJSON(w, http.StatusOK, user)
}

// Logout handles logout (client-side token removal)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// For now, logout is handled client-side by removing the token
	// In the future, we could implement token blacklisting
	util.RespondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Helper methods

func (h *Handler) verifyTelegramAuth(req TelegramLoginRequest) bool {
	if h.cfg.TelegramBotToken == "" {
		return false
	}

	// Create data-check-string
	data := map[string]string{
		"id":         strconv.FormatInt(req.ID, 10),
		"first_name": req.FirstName,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
	}
	if req.LastName != "" {
		data["last_name"] = req.LastName
	}
	if req.Username != "" {
		data["username"] = req.Username
	}
	if req.PhotoURL != "" {
		data["photo_url"] = req.PhotoURL
	}

	// Sort keys and create check string
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, data[k]))
	}
	dataCheckString := strings.Join(parts, "\n")

	// Create secret key (SHA256 of bot token)
	secretKey := sha256.Sum256([]byte(h.cfg.TelegramBotToken))

	// Calculate HMAC-SHA256
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	return calculatedHash == req.Hash
}

func (h *Handler) generateJWT(user *User) (string, int64, error) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	var tgID int64
	if user.TgID != nil {
		tgID = *user.TgID
	}

	var email string
	if user.Email != nil {
		email = *user.Email
	}

	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  email,
		TgID:   tgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt.Unix(), nil
}

// Database methods

func (h *Handler) findUserByTgID(ctx context.Context, tgID int64) (*User, error) {
	var user User
	var email, tgUsername, tgFirstName, tgLastName, tgPhotoURL, displayName *string
	var tgIDPtr *int64
	var lastLoginAt *time.Time
	var isAdmin *bool
	var settingsJSON []byte

	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, email, tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url, 
		       display_name, COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'), 
		       COALESCE(is_admin, FALSE), created_at, last_login_at, COALESCE(settings, '{}')
		FROM "user" WHERE tg_id = $1
	`, tgID).Scan(
		&user.ID, &email, &tgIDPtr, &tgUsername, &tgFirstName, &tgLastName, &tgPhotoURL,
		&displayName, &user.Timezone, &user.Locale, &isAdmin, &user.CreatedAt, &lastLoginAt, &settingsJSON,
	)

	if err != nil {
		return nil, err
	}

	user.Email = email
	user.TgID = tgIDPtr
	user.TgUsername = tgUsername
	user.TgFirstName = tgFirstName
	user.TgLastName = tgLastName
	user.TgPhotoURL = tgPhotoURL
	user.DisplayName = displayName
	user.LastLoginAt = lastLoginAt
	if isAdmin != nil {
		user.IsAdmin = *isAdmin
	}

	// Parse settings JSON
	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &user.Settings)
	}

	return &user, nil
}

func (h *Handler) findUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	var emailPtr, tgUsername, tgFirstName, tgLastName, tgPhotoURL, displayName *string
	var tgIDPtr *int64
	var lastLoginAt *time.Time
	var isAdmin *bool
	var settingsJSON []byte

	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, email, tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url,
		       display_name, COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'),
		       COALESCE(is_admin, FALSE), created_at, last_login_at, COALESCE(settings, '{}')
		FROM "user" WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&user.ID, &emailPtr, &tgIDPtr, &tgUsername, &tgFirstName, &tgLastName, &tgPhotoURL,
		&displayName, &user.Timezone, &user.Locale, &isAdmin, &user.CreatedAt, &lastLoginAt, &settingsJSON,
	)

	if err != nil {
		return nil, err
	}

	user.Email = emailPtr
	user.TgID = tgIDPtr
	user.TgUsername = tgUsername
	user.TgFirstName = tgFirstName
	user.TgLastName = tgLastName
	user.TgPhotoURL = tgPhotoURL
	user.DisplayName = displayName
	user.LastLoginAt = lastLoginAt
	if isAdmin != nil {
		user.IsAdmin = *isAdmin
	}

	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &user.Settings)
	}

	return &user, nil
}

func (h *Handler) findUserWithPasswordByEmail(ctx context.Context, email string) (*User, string, error) {
	var user User
	var emailPtr, tgUsername, tgFirstName, tgLastName, tgPhotoURL, displayName, passwordHash *string
	var tgIDPtr *int64
	var lastLoginAt *time.Time
	var isAdmin *bool
	var settingsJSON []byte

	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url,
		       display_name, COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'),
		       COALESCE(is_admin, FALSE), created_at, last_login_at, COALESCE(settings, '{}')
		FROM "user" WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&user.ID, &emailPtr, &passwordHash, &tgIDPtr, &tgUsername, &tgFirstName, &tgLastName, &tgPhotoURL,
		&displayName, &user.Timezone, &user.Locale, &isAdmin, &user.CreatedAt, &lastLoginAt, &settingsJSON,
	)

	if err != nil {
		return nil, "", err
	}

	user.Email = emailPtr
	user.TgID = tgIDPtr
	user.TgUsername = tgUsername
	user.TgFirstName = tgFirstName
	user.TgLastName = tgLastName
	user.TgPhotoURL = tgPhotoURL
	user.DisplayName = displayName
	user.LastLoginAt = lastLoginAt
	if isAdmin != nil {
		user.IsAdmin = *isAdmin
	}

	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &user.Settings)
	}

	var hash string
	if passwordHash != nil {
		hash = *passwordHash
	}

	return &user, hash, nil
}

func (h *Handler) findUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	var email, tgUsername, tgFirstName, tgLastName, tgPhotoURL, displayName *string
	var tgIDPtr *int64
	var lastLoginAt *time.Time
	var isAdmin *bool
	var settingsJSON []byte

	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, email, tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url,
		       display_name, COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'),
		       COALESCE(is_admin, FALSE), created_at, last_login_at, COALESCE(settings, '{}')
		FROM "user" WHERE id = $1
	`, id).Scan(
		&user.ID, &email, &tgIDPtr, &tgUsername, &tgFirstName, &tgLastName, &tgPhotoURL,
		&displayName, &user.Timezone, &user.Locale, &isAdmin, &user.CreatedAt, &lastLoginAt, &settingsJSON,
	)

	if err != nil {
		return nil, err
	}

	user.Email = email
	user.TgID = tgIDPtr
	user.TgUsername = tgUsername
	user.TgFirstName = tgFirstName
	user.TgLastName = tgLastName
	user.TgPhotoURL = tgPhotoURL
	user.DisplayName = displayName
	user.LastLoginAt = lastLoginAt
	if isAdmin != nil {
		user.IsAdmin = *isAdmin
	}

	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &user.Settings)
	}

	return &user, nil
}

func (h *Handler) createUserFromTelegram(ctx context.Context, req TelegramLoginRequest) (*User, error) {
	var user User
	authTime := time.Unix(req.AuthDate, 0)

	err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO "user" (tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url, tg_auth_date, display_name, last_login_at, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), '{}')
		RETURNING id, tg_id, tg_username, tg_first_name, tg_last_name, tg_photo_url, display_name,
		          COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'), COALESCE(is_admin, FALSE), created_at
	`, req.ID, nullString(req.Username), req.FirstName, nullString(req.LastName), nullString(req.PhotoURL), authTime, req.FirstName).Scan(
		&user.ID, &user.TgID, &user.TgUsername, &user.TgFirstName, &user.TgLastName, &user.TgPhotoURL,
		&user.DisplayName, &user.Timezone, &user.Locale, &user.IsAdmin, &user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	user.Settings = make(map[string]interface{})
	return &user, nil
}

func (h *Handler) createUserWithEmail(ctx context.Context, email, passwordHash, name string) (*User, error) {
	var user User
	displayName := name
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO "user" (email, password_hash, display_name, last_login_at, settings)
		VALUES ($1, $2, $3, NOW(), '{}')
		RETURNING id, email, display_name, COALESCE(timezone, 'Europe/Moscow'), COALESCE(locale, 'ru'), COALESCE(is_admin, FALSE), created_at
	`, email, passwordHash, displayName).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.Timezone, &user.Locale, &user.IsAdmin, &user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	user.Settings = make(map[string]interface{})
	return &user, nil
}

func (h *Handler) updateTelegramInfo(ctx context.Context, userID string, req TelegramLoginRequest) error {
	authTime := time.Unix(req.AuthDate, 0)
	_, err := h.db.Pool.Exec(ctx, `
		UPDATE "user" SET
			tg_username = $2,
			tg_first_name = $3,
			tg_last_name = $4,
			tg_photo_url = $5,
			tg_auth_date = $6,
			updated_at = NOW()
		WHERE id = $1
	`, userID, nullString(req.Username), req.FirstName, nullString(req.LastName), nullString(req.PhotoURL), authTime)
	return err
}

func (h *Handler) updateLastLogin(ctx context.Context, userID string) {
	h.db.Pool.Exec(ctx, `UPDATE "user" SET last_login_at = NOW() WHERE id = $1`, userID)
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
