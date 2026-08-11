package calendars

import (
	"context"

	"neuroboost/api-go/internal/database"
)

var db *database.DB

// InitDB sets the database connection for the calendars package.
func InitDB(d *database.DB) { db = d }

// CalendarIDsFor returns every calendar the user may read.
//
// 🔴 This is the ONLY sanctioned way to scope a query for events or tasks. A
// handler that filters by user_id instead is a bug, and scoping_test.go fails
// the build for it.
//
// Filtering happens in Go rather than SQL so the rule lives in AccessibleIDs,
// where it is tested. The row count here is a handful per user.
func CalendarIDsFor(ctx context.Context, userID string) ([]string, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT calendar_id::text, status FROM calendar_member WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := []Membership{}
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.CalendarID, &m.Status); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return AccessibleIDs(memberships), nil
}

// PersonalIDFor returns the user's own calendar, which is where anything
// created without an explicit calendar goes.
func PersonalIDFor(ctx context.Context, userID string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`SELECT id::text FROM calendar WHERE owner_id = $1 AND kind = 'personal' LIMIT 1`,
		userID).Scan(&id)
	return id, err
}
