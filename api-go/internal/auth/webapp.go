package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"neuroboost/api-go/internal/util"
)

// webAppMaxAge bounds how old initData may be, the same day as the Login Widget.
const webAppMaxAge = 24 * time.Hour

// verifyWebAppInitData checks the initData string a Telegram Mini App receives
// (core.telegram.org/bots/webapps, "Validating data received via the Mini App").
//
// 🔴 This is NOT the Login Widget scheme. There the secret is SHA256(token);
// here it is HMAC-SHA256 keyed with the constant "WebAppData" over the token,
// and the checked fields are whatever Telegram sent (query_id, user, chat_*,
// start_param…), not a fixed list. verifyTelegramAuth cannot be reused.
// The check is local: the server never calls Telegram, which matters because
// Telegram is unreachable from the main host (CLAUDE.md gotcha 19).
func verifyWebAppInitData(initData, botToken string, now time.Time) (TelegramLoginRequest, error) {
	var out TelegramLoginRequest
	if botToken == "" {
		return out, errors.New("bot token not configured")
	}
	values, err := url.ParseQuery(initData)
	if err != nil {
		return out, errors.New("initData is not a query string")
	}
	hash := values.Get("hash")
	if hash == "" {
		return out, errors.New("no hash")
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+values.Get(k))
	}

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(lines, "\n")))
	want := mac.Sum(nil)
	got, err := hex.DecodeString(hash)
	if err != nil || !hmac.Equal(got, want) {
		return out, errors.New("bad signature")
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || now.Sub(time.Unix(authDate, 0)) > webAppMaxAge {
		return out, errors.New("initData expired")
	}

	var u struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		PhotoURL  string `json:"photo_url"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &u); err != nil || u.ID == 0 {
		return out, errors.New("no user in initData")
	}
	return TelegramLoginRequest{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Username:  u.Username,
		PhotoURL:  u.PhotoURL,
		AuthDate:  authDate,
	}, nil
}

// TelegramWebAppLoginRequest is the body of POST /api/auth/telegram-webapp:
// the raw window.Telegram.WebApp.initData string, untouched.
type TelegramWebAppLoginRequest struct {
	InitData string `json:"init_data"`
}

// TelegramWebApp signs a user in from inside the Telegram Mini App. Same user
// lookup as TelegramLogin (by tg_id, created on first visit), different proof.
func (h *Handler) TelegramWebApp(w http.ResponseWriter, r *http.Request) {
	var req TelegramWebAppLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InitData == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	tg, err := verifyWebAppInitData(req.InitData, h.cfg.TelegramBotToken, time.Now())
	if err != nil {
		util.RespondError(w, http.StatusUnauthorized, "INVALID_INIT_DATA", "Telegram Mini App authentication failed")
		return
	}

	ctx := r.Context()
	user, err := h.findUserByTgID(ctx, tg.ID)
	if err != nil && err != pgx.ErrNoRows {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Database error")
		return
	}
	if user == nil {
		user, err = h.createUserFromTelegram(ctx, tg)
		if err != nil {
			util.RespondError(w, http.StatusInternalServerError, "CREATE_USER_ERROR", "Failed to create user")
			return
		}
	} else {
		// Non-fatal, as in TelegramLogin: a stale name must not block sign-in.
		_ = h.updateTelegramInfo(ctx, user.ID, tg)
	}
	h.updateLastLogin(ctx, user.ID)

	token, expiresAt, err := h.generateJWT(user)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}
	util.RespondJSON(w, http.StatusOK, AuthResponse{Token: token, ExpiresAt: expiresAt, User: *user})
}
