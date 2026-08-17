package calendars

import (
	"context"
	"strings"
)

// SharedIDs returns the subset of calendarIDs that more than one active member
// can see.
//
// 🔴 "Shared" here means somebody else actually sees it — two or more ACTIVE
// members — not merely kind='shared'. A calendar the owner created and invited
// nobody to, or one whose single invitation is still unanswered, is private in
// every sense that matters to the person looking at the grid. Marking it would
// promise an audience that does not exist yet.
//
// The consequence is that the flag flips on its own the moment an invitation is
// accepted or a member leaves, which is the whole reason it is computed on read
// instead of stored.
func SharedIDs(ctx context.Context, calendarIDs []string) (map[string]bool, error) {
	shared := map[string]bool{}
	if len(calendarIDs) == 0 {
		return shared, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT calendar_id::text
		FROM calendar_member
		WHERE calendar_id = ANY($1) AND status = 'active'
		GROUP BY calendar_id
		HAVING count(*) > 1`, calendarIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		shared[id] = true
	}
	return shared, rows.Err()
}

// DisplayNames maps user ids to a short label fit for a calendar block.
func DisplayNames(ctx context.Context, userIDs []string) (map[string]string, error) {
	names := map[string]string{}
	if len(userIDs) == 0 {
		return names, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id::text, display_name, tg_first_name, tg_username, email
		FROM "user" WHERE id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var displayName, tgFirstName, tgUsername, email *string
		if err := rows.Scan(&id, &displayName, &tgFirstName, &tgUsername, &email); err != nil {
			return nil, err
		}
		names[id] = DisplayLabel(displayName, tgFirstName, tgUsername, email)
	}
	return names, rows.Err()
}

// DisplayLabel picks the shortest identifying name a user has.
//
// Every one of these columns is nullable — a Telegram-only account has no
// email, an email account has no tg_first_name, and display_name is optional
// for both — so the chain has to end in a constant rather than in "".
//
// ⚠ The email is deliberately reduced to its local part. ListMembers renders
// the full address, and that is right in a roster where you are deciding who to
// remove. In a calendar block it is wrong twice: it is the longest string a
// user has, and an unbroken 30-character address is exactly what pushed the
// /profile header past 375px (learning-fixture-data-can-disarm-a-control).
func DisplayLabel(displayName, tgFirstName, tgUsername, email *string) string {
	for _, candidate := range []*string{displayName, tgFirstName, tgUsername} {
		if v := trimmed(candidate); v != "" {
			return v
		}
	}
	if v := trimmed(email); v != "" {
		if local, _, found := strings.Cut(v, "@"); found && local != "" {
			return local
		}
		return v
	}
	return "Участник"
}

func trimmed(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
