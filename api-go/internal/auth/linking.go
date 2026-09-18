package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// Linking email accounts and Telegram accounts, v0.4.11.5.
//
// Denis, 17.09: «dev website doesn't work with telegram login, I can't test it,
// we need to enable linking between email and telegram profiles… and in the
// website add telegram with bot asking if you're the one who's connecting the
// profile». Two directions, and the bot always asks.
//
// 🔴 Neither secret is ever written to a log, a response field it does not
// need, or an error message. Gotcha 14's lesson was not "redact when printing"
// — it was that redaction at the reader does not protect a value that crosses a
// process boundary. A value stored only as a hash cannot leak from anywhere.

const (
	linkTokenTTL      = 10 * time.Minute
	linkCodeAttempts  = 5
	loginLinkKind     = "LOGIN"
	linkCodeKind      = "LINK"
	mergeRequestTTL   = 10 * time.Minute
	loginLinkByteSize = 32
)

// hashSecret stores tokens and codes one-way.
//
// SHA-256 rather than bcrypt deliberately: these secrets live ten minutes, are
// high-entropy (or attempt-limited), and redeeming happens by lookup — a slow
// hash cannot be looked up, only compared row by row.
func hashSecret(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func newLoginToken() (string, error) {
	b := make([]byte, loginLinkByteSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newLinkCode returns six digits from a cryptographic source.
//
// ⚠ Six digits is a million possibilities, which is only safe because the code
// dies after five wrong attempts and after ten minutes. The attempt limit is
// the security property here; the length is only convenience.
func newLinkCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// issue replaces any live token of this kind for this user and returns the new
// secret. One live secret per person per kind: pressing the button twice must
// not leave two working links, or revoking the one you can see does nothing.
func (h *Handler) issue(ctx context.Context, userID, kind, secret string) error {
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM auth_link_token WHERE user_id = $1 AND kind = $2 AND redeemed_at IS NULL`,
		userID, kind); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO auth_link_token (user_id, kind, token_hash, expires_at)
		 VALUES ($1, $2, $3, NOW() + $4::interval)`,
		userID, kind, hashSecret(secret), fmt.Sprintf("%d seconds", int(linkTokenTTL.Seconds()))); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreateLoginLink hands a Telegram-only account a way onto the website.
//
// POST /api/auth/login-link — authenticated.
func (h *Handler) CreateLoginLink(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	token, err := newLoginToken()
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to create link")
		return
	}
	if err := h.issue(r.Context(), userID, loginLinkKind, token); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to create link")
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_in": int(linkTokenTTL.Seconds()),
	})
}

// RedeemLoginLink turns a one-shot link into a session.
//
// POST /api/auth/login-link/redeem — public; the token is the credential.
func (h *Handler) RedeemLoginLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	ctx := r.Context()
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sign in")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 🔴 Locked, matched and spent in one statement. Two redemptions arriving
	// together must not both succeed — a "one-shot" link that two people can
	// use is not one-shot, it is a race.
	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE auth_link_token
		   SET redeemed_at = NOW()
		 WHERE id = (SELECT id FROM auth_link_token
		              WHERE kind = $1 AND token_hash = $2
		                AND redeemed_at IS NULL AND expires_at > NOW()
		              FOR UPDATE SKIP LOCKED
		              LIMIT 1)
		 RETURNING user_id`, loginLinkKind, hashSecret(req.Token)).Scan(&userID)
	if err != nil {
		// Spent, expired and never-existed are one answer on purpose: telling
		// them apart tells a stranger which links were real.
		util.RespondError(w, http.StatusGone, "LINK_EXPIRED", "Эта ссылка уже использована или истекла")
		return
	}

	user, err := h.findUserByID(ctx, userID)
	if err != nil || user == nil {
		util.RespondError(w, http.StatusUnauthorized, "NOT_FOUND", "Account not found")
		return
	}
	token, expiresAt, err := h.generateJWT(user)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sign in")
		return
	}
	h.updateLastLogin(ctx, user.ID)
	util.RespondJSON(w, http.StatusOK, AuthResponse{Token: token, ExpiresAt: expiresAt, User: *user})
}

// CreateLinkCode gives the Telegram side a six-digit code to read out on the
// website.
//
// POST /api/auth/link-code — authenticated (the Telegram account).
func (h *Handler) CreateLinkCode(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	code, err := newLinkCode()
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to create code")
		return
	}
	if err := h.issue(r.Context(), userID, linkCodeKind, code); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to create code")
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]any{
		"code":       code,
		"expires_in": int(linkTokenTTL.Seconds()),
		"attempts":   linkCodeAttempts,
	})
}

// RedeemLinkCode opens a merge request. It merges nothing.
//
// POST /api/auth/link-code/redeem — authenticated (the website account).
//
// 🔴 The split is the design, not caution: the code proves control of the
// Telegram account and this session proves control of the email, but only the
// button in the bot proves the two are one person's intention. Denis asked for
// exactly that — «bot asking if you're the one who's connecting the profile».
func (h *Handler) RedeemLinkCode(w http.ResponseWriter, r *http.Request) {
	siteUserID := middleware.UserIDFromContext(r.Context())
	if siteUserID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	ctx := r.Context()
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to link")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Every live code of this kind is examined, so a wrong guess burns an
	// attempt on the code it was aimed at rather than failing anonymously.
	var tokenID, tgUserID, storedHash string
	var attempts int
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, token_hash, attempts
		  FROM auth_link_token
		 WHERE kind = $1 AND token_hash = $2
		   AND redeemed_at IS NULL AND expires_at > NOW()
		 FOR UPDATE`, linkCodeKind, hashSecret(req.Code)).Scan(&tokenID, &tgUserID, &storedHash, &attempts)
	if err != nil {
		// A miss cannot burn an attempt on a code we did not find, so the
		// protection that matters is the ten-minute life plus the limit below.
		h.burnAttempt(ctx)
		util.RespondError(w, http.StatusUnauthorized, "BAD_CODE", "Код не подошёл")
		return
	}
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashSecret(req.Code))) != 1 {
		util.RespondError(w, http.StatusUnauthorized, "BAD_CODE", "Код не подошёл")
		return
	}
	if attempts >= linkCodeAttempts {
		util.RespondError(w, http.StatusGone, "CODE_BURNED", "Код сгорел — запроси новый в боте")
		return
	}
	if tgUserID == siteUserID {
		util.RespondError(w, http.StatusConflict, "SAME_ACCOUNT", "Это один и тот же аккаунт")
		return
	}

	if _, err := tx.Exec(ctx,
		`UPDATE auth_link_token SET redeemed_at = NOW() WHERE id = $1`, tokenID); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to link")
		return
	}

	var requestID string
	err = tx.QueryRow(ctx, `
		INSERT INTO account_merge_request (site_user_id, tg_user_id, expires_at)
		VALUES ($1, $2, NOW() + $3::interval)
		RETURNING id`,
		siteUserID, tgUserID, fmt.Sprintf("%d seconds", int(mergeRequestTTL.Seconds()))).Scan(&requestID)
	if err != nil {
		// The partial unique index is the only thing that can reject this, and
		// it means a merge for one of these accounts is already waiting.
		util.RespondError(w, http.StatusConflict, "ALREADY_PENDING",
			"Заявка на объединение уже ждёт подтверждения в боте")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to link")
		return
	}
	util.RespondJSON(w, http.StatusOK, map[string]any{
		"request_id": requestID,
		"status":     "pending",
		"message":    "Подтверди объединение в боте",
	})
}

// burnAttempt charges a wrong guess against every live code.
//
// ⚠ Against EVERY live code, because a miss by definition does not identify one.
// Without this the five-attempt limit would only apply to somebody typing their
// own code wrong, and not at all to somebody guessing.
func (h *Handler) burnAttempt(ctx context.Context) {
	_, _ = h.db.Pool.Exec(ctx,
		`UPDATE auth_link_token SET attempts = attempts + 1
		  WHERE kind = $1 AND redeemed_at IS NULL AND expires_at > NOW()`, linkCodeKind)
}

// SetCredentials gives an account that arrived from Telegram a way to sign in
// without it.
//
// POST /api/auth/credentials — authenticated.
//
// 🔴 Only for an account that has no email yet. Changing an existing email is a
// different operation with different consequences (it moves where password
// resets go), and the spec puts it out of scope on purpose — so this refuses
// rather than quietly doing it.
func (h *Handler) SetCredentials(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	// Same two rules as Register, deliberately identical: an account that can be
	// created one way and not the other is a rule nobody can state.
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		util.RespondError(w, http.StatusBadRequest, "INVALID_EMAIL", "Valid email is required")
		return
	}
	if len(req.Password) < 8 {
		util.RespondError(w, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters")
		return
	}

	ctx := r.Context()
	var existingEmail *string
	if err := h.db.Pool.QueryRow(ctx, `SELECT email FROM "user" WHERE id = $1`, userID).
		Scan(&existingEmail); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read account")
		return
	}
	if existingEmail != nil && *existingEmail != "" {
		util.RespondError(w, http.StatusConflict, "EMAIL_SET",
			"У этого аккаунта уже есть email")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "HASH_ERROR", "Failed to process password")
		return
	}

	// 🔴 The UNIQUE index decides, not a prior SELECT: between a check and a
	// write another request can take the address. The 409 below is the spec's
	// path 2 — «у этого email есть аккаунт, войди в него и привяжи Telegram
	// кодом» — and it is reached by the database refusing, not by guessing.
	tag, err := h.db.Pool.Exec(ctx,
		`UPDATE "user" SET email = $1, password_hash = $2, updated_at = NOW()
		  WHERE id = $3 AND email IS NULL`, req.Email, string(hashed), userID)
	if err != nil {
		util.RespondError(w, http.StatusConflict, "EMAIL_EXISTS",
			"У этого email уже есть аккаунт — войди в него и привяжи Telegram кодом")
		return
	}
	if tag.RowsAffected() == 0 {
		util.RespondError(w, http.StatusConflict, "EMAIL_SET", "У этого аккаунта уже есть email")
		return
	}

	user, err := h.findUserByID(ctx, userID)
	if err != nil || user == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read account")
		return
	}
	util.RespondJSON(w, http.StatusOK, user)
}
