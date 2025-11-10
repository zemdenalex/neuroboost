package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	PhotoURL   string    `json:"photo_url"`
	Timezone   string    `json:"timezone"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Claims struct {
	UserID     int64  `json:"user_id"`
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// ---- Request context helpers (to avoid import cycles) ----

type ctxUserKeyType struct{}

var ctxUserKey ctxUserKeyType

func ContextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	v := ctx.Value(ctxUserKey)
	if v == nil {
		return nil, false
	}
	u, ok := v.(*User)
	return u, ok
}
