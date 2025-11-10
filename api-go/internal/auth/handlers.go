package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	dbGlobal   *sql.DB
	botToken   string
	secureProd bool
)

// Init wires package-level dependencies (simple & avoids import cycles).
func Init(db *sql.DB) {
	dbGlobal = db
	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	env := os.Getenv("APP_ENV")
	secureProd = env == "production"
}

// ---------- Utilities ----------

type jsonError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, jsonError{Error: msg})
}

func setAuthCookie(w http.ResponseWriter, token string) {
	c := &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		// Expires is optional when MaxAge is set; add for some clients:
		Expires: time.Now().Add(30 * 24 * time.Hour),
	}
	http.SetCookie(w, c)
}

func clearAuthCookie(w http.ResponseWriter) {
	c := &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, c)
}

// ---------- Handlers ----------

// POST /api/auth/login
// Body: LoginRequest JSON
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if dbGlobal == nil {
		writeError(w, http.StatusInternalServerError, "auth not initialized")
		return
	}
	if botToken == "" {
		writeError(w, http.StatusInternalServerError, "telegram bot token not configured")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Build authData map<string,string> exactly as received
	authData := map[string]string{
		"id":         strconv.FormatInt(req.ID, 10),
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"username":   req.Username,
		"photo_url":  req.PhotoURL,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
		"hash":       req.Hash,
	}

	ok, err := VerifyTelegramAuth(authData, botToken)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "telegram auth verification failed")
		return
	}

	// Upsert user by telegram_id
	u, err := GetUserByTelegramID(dbGlobal, req.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u, err = CreateUser(dbGlobal, req.ID, req.Username, req.FirstName, req.LastName, req.PhotoURL)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to create user")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "user lookup error")
			return
		}
	} else {
		_ = UpdateUser(dbGlobal, u.ID, map[string]interface{}{
			"username":   req.Username,
			"first_name": req.FirstName,
			"last_name":  req.LastName,
			"photo_url":  req.PhotoURL,
			"last_login": time.Now(),
		})
		// refresh from DB is optional; existing u is fine for JWT
	}

	token, err := GenerateToken(u.ID, u.TelegramID, req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	setAuthCookie(w, token)

	resp := map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"telegram_id": u.TelegramID,
			"username":    req.Username,
			"first_name":  req.FirstName,
			"last_name":   req.LastName,
			"photo_url":   req.PhotoURL,
			"timezone":    u.Timezone,
		},
		"token": token, // provided for convenience; cookie is authoritative
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/auth/logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /api/auth/me
func MeHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok || u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"telegram_id": u.TelegramID,
			"username":    u.Username,
			"first_name":  u.FirstName,
			"last_name":   u.LastName,
			"photo_url":   u.PhotoURL,
			"timezone":    u.Timezone,
		},
	})
}

// POST /api/auth/refresh
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil || c.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	newToken, err := RefreshToken(c.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	if newToken != c.Value {
		setAuthCookie(w, newToken)
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": newToken})
}
