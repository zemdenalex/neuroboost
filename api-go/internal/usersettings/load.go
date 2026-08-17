package usersettings

import (
	"context"

	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB follows the package-level-pool pattern the other packages use.
func InitDB(d *database.DB) { db = d }

// LoadReminders reads one user's reminder settings.
//
// It never returns an error: a profile we cannot read yields the defaults,
// because failing a whole event creation over an unreadable settings blob
// would be a worse outcome than applying the standard preset.
func LoadReminders(ctx context.Context, userID string) Reminders {
	if db == nil {
		return DefaultReminders()
	}
	var raw []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(settings, '{}') FROM "user" WHERE id = $1`, userID).Scan(&raw)
	if err != nil {
		return DefaultReminders()
	}
	return ParseReminders(raw)
}

// DefaultEventOffsets resolves the offsets a newly created event should get
// when the request omitted the field entirely.
//
// The backend applies the preset rather than the frontend, so an event created
// from the bot or an import gets reminders too — otherwise the setting would
// be a property of one form rather than of the user.
func DefaultEventOffsets(ctx context.Context, userID string) []int {
	s := LoadReminders(ctx, userID)
	return OffsetsForPreset(s, s.DefaultEventPreset)
}

// DefaultTaskOffsets is DefaultEventOffsets for tasks.
//
// This is what makes quick-add tasks reachable by reminders at all: quick-add
// sends only a title, so without this every task captured in one keystroke
// would be silently excluded from notifications.
func DefaultTaskOffsets(ctx context.Context, userID string) []int {
	s := LoadReminders(ctx, userID)
	return OffsetsForPreset(s, s.DefaultTaskPreset)
}
