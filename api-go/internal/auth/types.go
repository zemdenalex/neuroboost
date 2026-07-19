package auth

import (
	"context"
	"time"
)

// User represents a user in the system
type User struct {
	ID          string                 `json:"id"`
	Email       *string                `json:"email,omitempty"`
	TgID        *int64                 `json:"tg_id,omitempty"`
	TgUsername  *string                `json:"tg_username,omitempty"`
	TgFirstName *string                `json:"tg_first_name,omitempty"`
	TgLastName  *string                `json:"tg_last_name,omitempty"`
	TgPhotoURL  *string                `json:"tg_photo_url,omitempty"`
	DisplayName *string                `json:"display_name,omitempty"`
	Timezone    string                 `json:"timezone"`
	Locale      string                 `json:"locale"`
	IsAdmin     bool                   `json:"is_admin"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	LastLoginAt *time.Time             `json:"last_login_at,omitempty"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      User   `json:"user"`
}

// TelegramLoginRequest represents the Telegram Login Widget data
type TelegramLoginRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// RegisterRequest for email/password registration
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// LoginRequest for email/password login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateUserRequest represents profile/settings update
type UpdateUserRequest struct {
	DisplayName *string                `json:"display_name,omitempty"`
	Timezone    *string                `json:"timezone,omitempty"`
	Locale      *string                `json:"locale,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UserSettings represents user preferences stored in settings JSONB
// This is a reference structure - actual storage is flexible JSONB
type UserSettings struct {
	// Layout preferences
	HeaderVariant string `json:"header_variant,omitempty"` // "horizontal" or "vertical"
	UIScale       int    `json:"ui_scale,omitempty"`       // 80-150 (percentage)

	// Work hours
	WorkDays  []string `json:"work_days,omitempty"`  // ["Mon", "Tue", "Wed", "Thu", "Fri"]
	WorkStart string   `json:"work_start,omitempty"` // "09:00"
	WorkEnd   string   `json:"work_end,omitempty"`   // "17:00"

	// Feature toggles
	Features map[string]bool `json:"features,omitempty"`

	// Notification preferences
	QuietHoursStart string `json:"quiet_hours_start,omitempty"` // "22:00"
	QuietHoursEnd   string `json:"quiet_hours_end,omitempty"`   // "08:00"
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
