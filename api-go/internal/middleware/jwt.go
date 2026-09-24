package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"neuroboost/api-go/internal/util"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// Claims represents the JWT payload
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
	TgID   int64  `json:"tg_id,omitempty"`
	jwt.RegisteredClaims
}

// bearerUser returns the user in a valid "Bearer <token>" header, or an
// error code for the strict middleware to answer with. Shared by both
// middlewares so a token cannot be valid for one and not the other.
func bearerUser(r *http.Request, secret string) (userID, code, message string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "MISSING_TOKEN", "Authorization header required"
	}

	// Extract token from "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", "INVALID_TOKEN_FORMAT", "Authorization header must be: Bearer <token>"
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", "INVALID_TOKEN", "Invalid or expired token"
	}
	return claims.UserID, "", ""
}

// JWTMiddleware validates JWT tokens and injects user_id into context
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, code, message := bearerUser(r, secret)
			if code != "" {
				util.RespondError(w, http.StatusUnauthorized, code, message)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTMiddleware is for endpoints anyone may call, such as feedback:
// a valid token attaches the user, anything else passes on anonymous. It never
// refuses, so a feedback form still works for someone whose session expired.
func OptionalJWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, code, _ := bearerUser(r, secret); code == "" && userID != "" {
				r = r.WithContext(context.WithValue(r.Context(), UserIDKey, userID))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext extracts user_id from request context
func UserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
