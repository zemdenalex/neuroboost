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

// JWTMiddleware validates JWT tokens and injects user_id into context
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				util.RespondError(w, http.StatusUnauthorized, "MISSING_TOKEN", "Authorization header required")
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				util.RespondError(w, http.StatusUnauthorized, "INVALID_TOKEN_FORMAT", "Authorization header must be: Bearer <token>")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				util.RespondError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
				return
			}

			// Inject user_id into context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
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
