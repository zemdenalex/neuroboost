package accounts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CalendarChoice is what happens when both accounts own a personal calendar.
//
// Denis, 17.09: «about the calendar ask too» — so this is never inferred. It
// arrives from a button he pressed in the bot.
type CalendarChoice string

const (
	// CalendarsMerge folds the absorbed personal calendar into the survivor's
	// and removes the empty shell.
	CalendarsMerge CalendarChoice = "merge"
	// CalendarsKeepBoth demotes the absorbed personal calendar to a shared one
	// named «Из Telegram» and keeps it beside the survivor's.
	//
	// ⚠ The spec says kind = 'regular'. There is no such kind: the CHECK on
	// `calendar` allows 'personal' and 'shared' only. 'shared' it is — a second
	// 'personal' row would violate idx_calendar_one_personal_per_owner anyway,
	// which is the whole reason this question is asked.
	CalendarsKeepBoth CalendarChoice = "keep_both"
)

// roleRank orders calendar roles so that the stronger one survives when both
// accounts are members of the same shared calendar.
func roleRank(role string) int {
	switch role {
	case "owner":
		return 3
	case "editor":
		return 2
	case "viewer":
		return 1
	}
	return 0
}

// Merge folds account `absorb` into account `keep` and deletes `absorb`.
//
// 🔴 One transaction, and it is irreversible. Seventeen of the foreign keys on
// "user" are ON DELETE CASCADE, so the final DELETE is only safe because step 6
// proves nothing points at `absorb` any more; a non-zero count rolls the whole
// thing back rather than deleting.
//
// There is deliberately no exported entry point that takes two user ids alone:
// callers go through a confirmed account_merge_request, because the request row
// is the evidence that one person controlled both sides at the same moment.
func Merge(ctx context.Context, pool *pgxpool.Pool, requestID string) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	var keep, absorb string
	var choice *string
	var status string
	// The request is locked, checked for freshness, and read in one statement:
	// two confirmations arriving together must not both proceed.
	err = tx.QueryRow(ctx, `
		SELECT status, keep_user_id,
		       CASE WHEN keep_user_id = site_user_id THEN tg_user_id ELSE site_user_id END,
		       calendar_choice
		  FROM account_merge_request
		 WHERE id = $1 AND expires_at > NOW()
		   FOR UPDATE`, requestID).Scan(&status, &keep, &absorb, &choice)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("merge request not found or expired")
	}
	if err != nil {
		return fmt.Errorf("load request: %w", err)
	}
	if status != "pending" {
		return fmt.Errorf("merge request is %s, not pending", status)
	}
	if keep == "" || absorb == "" {
		return fmt.Errorf("merge request has no answer yet: which account stays is unanswered")
	}
	if keep == absorb {
		return fmt.Errorf("merge request points both sides at one account")
	}

	// Lock both accounts in a stable order, so two merges touching the same
	// pair cannot deadlock each other.
	first, second := keep, absorb
	if second < first {
		first, second = second, first
	}
	var locked int
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM (
		    SELECT id FROM "user" WHERE id IN ($1, $2) ORDER BY id FOR UPDATE
		 ) s`, first, second).Scan(&locked); err != nil {
		return fmt.Errorf("lock accounts: %w", err)
	}
	if locked != 2 {
		return fmt.Errorf("one of the accounts is gone (locked %d of 2)", locked)
	}

	if err = mergePersonalCalendars(ctx, tx, keep, absorb, choice); err != nil {
		return err
	}
	if err = moveEverythingElse(ctx, tx, keep, absorb); err != nil {
		return err
	}
	if err = mergeSettings(ctx, tx, keep, absorb); err != nil {
		return err
	}
	if err = mergeIdentity(ctx, tx, keep, absorb); err != nil {
		return err
	}

	// 🔴 Step 6 of the spec, and the line this whole function stands on.
	if left, ferr := danglingReferences(ctx, tx, absorb); ferr != nil {
		return fmt.Errorf("count references: %w", ferr)
	} else if len(left) > 0 {
		return fmt.Errorf("refusing to delete: %v still reference the absorbed account", left)
	}

	if _, err = tx.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, absorb); err != nil {
		return fmt.Errorf("delete absorbed account: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE account_merge_request SET status = 'done', resolved_at = NOW() WHERE id = $1`,
		requestID); err != nil {
		return fmt.Errorf("close request: %w", err)
	}
	return nil
}

// mergePersonalCalendars answers the second question Denis asked to be asked.
func mergePersonalCalendars(ctx context.Context, tx pgx.Tx, keep, absorb string, choice *string) error {
	var keepCal, absorbCal *string
	if err := tx.QueryRow(ctx,
		`SELECT (SELECT id FROM calendar WHERE owner_id = $1 AND kind = 'personal'),
		        (SELECT id FROM calendar WHERE owner_id = $2 AND kind = 'personal')`,
		keep, absorb).Scan(&keepCal, &absorbCal); err != nil {
		return fmt.Errorf("find personal calendars: %w", err)
	}
	if absorbCal == nil {
		return nil // nothing to decide
	}
	if keepCal == nil {
		// The survivor has none: the absorbed one simply becomes theirs, and no
		// unique index is in the way.
		_, err := tx.Exec(ctx, `UPDATE calendar SET owner_id = $1 WHERE id = $2`, keep, *absorbCal)
		return err
	}

	// Both exist. Only now does the answer matter — and its absence is an error
	// rather than a default, because a default here silently picks for somebody.
	if choice == nil {
		return fmt.Errorf("both accounts have a personal calendar and the question was not answered")
	}
	switch CalendarChoice(*choice) {
	case CalendarsMerge:
		for _, table := range []string{"event", "task", "reminder"} {
			if _, err := tx.Exec(ctx,
				fmt.Sprintf(`UPDATE %s SET calendar_id = $1 WHERE calendar_id = $2`, table),
				*keepCal, *absorbCal); err != nil {
				return fmt.Errorf("move %s into the surviving calendar: %w", table, err)
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM calendar_member WHERE calendar_id = $1`, *absorbCal); err != nil {
			return fmt.Errorf("clear members of the absorbed calendar: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM calendar WHERE id = $1`, *absorbCal); err != nil {
			return fmt.Errorf("delete the absorbed calendar: %w", err)
		}
	case CalendarsKeepBoth:
		if _, err := tx.Exec(ctx,
			`UPDATE calendar SET owner_id = $1, kind = 'shared', name = 'Из Telegram' WHERE id = $2`,
			keep, *absorbCal); err != nil {
			return fmt.Errorf("keep both calendars: %w", err)
		}
	default:
		return fmt.Errorf("unknown calendar choice %q", *choice)
	}
	return nil
}

// moveEverythingElse walks the foreign-key list, so a column added to fk.go is
// handled here without anybody remembering to.
func moveEverythingElse(ctx context.Context, tx pgx.Tx, keep, absorb string) error {
	for _, c := range Columns {
		switch c.How {
		case Clear:
			// The database does it (ON DELETE SET NULL).
		case Drop:
			if _, err := tx.Exec(ctx,
				fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`, c.Table, c.Column), absorb); err != nil {
				return fmt.Errorf("drop %s: %w", c.Key(), err)
			}
		case MoveOrDrop:
			if err := moveOrDrop(ctx, tx, c, keep, absorb); err != nil {
				return err
			}
		case Move:
			if _, err := tx.Exec(ctx,
				fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`, c.Table, c.Column, c.Column),
				keep, absorb); err != nil {
				return fmt.Errorf("move %s: %w", c.Key(), err)
			}
		}
	}
	return nil
}

// moveOrDrop handles the tables where both accounts may already hold a row
// that a unique constraint says only one of them may hold.
func moveOrDrop(ctx context.Context, tx pgx.Tx, c Column, keep, absorb string) error {
	switch c.Table {
	case "alert_status":
		// user_id is the PRIMARY KEY: the survivor's row wins outright.
		if _, err := tx.Exec(ctx, `DELETE FROM alert_status WHERE user_id = $1`, absorb); err != nil {
			return fmt.Errorf("alert_status: %w", err)
		}
	case "calendar_member":
		// In a calendar both belong to, the stronger role survives.
		if _, err := tx.Exec(ctx, `
			UPDATE calendar_member k
			   SET role = a.role
			  FROM calendar_member a
			 WHERE a.user_id = $2 AND k.user_id = $1
			   AND k.calendar_id = a.calendar_id
			   AND $3::jsonb ->> a.role IS NOT NULL
			   AND ($3::jsonb ->> a.role)::int > ($3::jsonb ->> k.role)::int`,
			keep, absorb, roleRankJSON()); err != nil {
			return fmt.Errorf("calendar_member role: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM calendar_member a
			 WHERE a.user_id = $2
			   AND EXISTS (SELECT 1 FROM calendar_member k
			                WHERE k.user_id = $1 AND k.calendar_id = a.calendar_id)`,
			keep, absorb); err != nil {
			return fmt.Errorf("calendar_member dedupe: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE calendar_member SET user_id = $1 WHERE user_id = $2`, keep, absorb); err != nil {
			return fmt.Errorf("calendar_member move: %w", err)
		}
	case "day_commitment", "day_commitment_day":
		// «Задачи дня»: where both accounts hold the same key, the survivor's
		// row wins; the rest move. The key columns other than user_id are the
		// ones the two rows are compared on.
		on := "k.day = a.day"
		if c.Table == "day_commitment" {
			on += " AND k.task_id = a.task_id"
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
			DELETE FROM %[1]s a
			 WHERE a.user_id = $2
			   AND EXISTS (SELECT 1 FROM %[1]s k WHERE k.user_id = $1 AND %[2]s)`, c.Table, on),
			keep, absorb); err != nil {
			return fmt.Errorf("%s dedupe: %w", c.Table, err)
		}
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`UPDATE %s SET user_id = $1 WHERE user_id = $2`, c.Table), keep, absorb); err != nil {
			return fmt.Errorf("%s move: %w", c.Table, err)
		}
	default:
		return fmt.Errorf("no MoveOrDrop rule for %s — add one before merging", c.Key())
	}
	return nil
}

// roleRankJSON hands the ordering to SQL so the comparison happens in one
// statement rather than in a read-modify-write loop.
func roleRankJSON() string {
	b, _ := json.Marshal(map[string]int{
		"owner": roleRank("owner"), "editor": roleRank("editor"), "viewer": roleRank("viewer"),
	})
	return string(b)
}

// mergeSettings keeps the survivor's answers and adopts the absorbed account's
// only where the survivor has none.
//
// 🔴 Read-merge-write over the whole blob, never a partial write: PATCH
// /api/auth/me replaces `settings` wholesale (gotcha 21), and the same trap
// applies to any writer. A failed read must not become a write over an empty
// map — so a read error returns instead of continuing.
func mergeSettings(ctx context.Context, tx pgx.Tx, keep, absorb string) error {
	var keepRaw, absorbRaw []byte
	if err := tx.QueryRow(ctx,
		`SELECT (SELECT COALESCE(settings, '{}') FROM "user" WHERE id = $1),
		        (SELECT COALESCE(settings, '{}') FROM "user" WHERE id = $2)`,
		keep, absorb).Scan(&keepRaw, &absorbRaw); err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	keepMap, absorbMap := map[string]any{}, map[string]any{}
	if err := json.Unmarshal(keepRaw, &keepMap); err != nil {
		return fmt.Errorf("surviving settings are not an object: %w", err)
	}
	if err := json.Unmarshal(absorbRaw, &absorbMap); err != nil {
		return fmt.Errorf("absorbed settings are not an object: %w", err)
	}

	for k, v := range absorbMap {
		if _, taken := keepMap[k]; !taken {
			keepMap[k] = v
		}
	}
	// bot.keywords is a dictionary, not a scalar: the union is more useful than
	// either side, and on a collision the survivor's word wins.
	if a, ok := absorbMap["bot.keywords"].(map[string]any); ok {
		merged := map[string]any{}
		for k, v := range a {
			merged[k] = v
		}
		if s, ok := keepMap["bot.keywords"].(map[string]any); ok {
			for k, v := range s {
				merged[k] = v
			}
		}
		keepMap["bot.keywords"] = merged
	}

	out, err := json.Marshal(keepMap)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE "user" SET settings = $1 WHERE id = $2`, out, keep)
	return err
}

// mergeIdentity gives the survivor both identities.
//
// Order matters: the absorbed account releases tg_id and email first, because
// both are UNIQUE and the survivor cannot take them while they are held.
// Which side each identity comes from does not depend on which account
// survived — that is the point of merging rather than choosing.
func mergeIdentity(ctx context.Context, tx pgx.Tx, keep, absorb string) error {
	var tgID *int64
	var tgUsername, email, passwordHash *string
	if err := tx.QueryRow(ctx,
		`SELECT tg_id, tg_username, email, password_hash FROM "user" WHERE id = $1`,
		absorb).Scan(&tgID, &tgUsername, &email, &passwordHash); err != nil {
		return fmt.Errorf("read absorbed identity: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE "user" SET tg_id = NULL, email = NULL WHERE id = $1`, absorb); err != nil {
		return fmt.Errorf("release absorbed identity: %w", err)
	}

	if tgID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE "user" SET tg_id = $1, tg_username = COALESCE(tg_username, $2)
			 WHERE id = $3 AND tg_id IS NULL`, *tgID, tgUsername, keep); err != nil {
			return fmt.Errorf("adopt telegram identity: %w", err)
		}
	}
	if email != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE "user" SET email = $1, password_hash = COALESCE(password_hash, $2)
			 WHERE id = $3 AND email IS NULL`, *email, passwordHash, keep); err != nil {
			return fmt.Errorf("adopt email identity: %w", err)
		}
	}
	return nil
}

// danglingReferences asks every known foreign key whether it still points at
// the account about to be deleted.
//
// Built from the same list the merge walks, and that list is held true by
// TestMergeKnowsEveryForeignKeyOnUser — without that test this check would
// confirm only what the merge already believes.
func danglingReferences(ctx context.Context, tx pgx.Tx, absorb string) ([]string, error) {
	var left []string
	for _, c := range Columns {
		if c.How == Clear {
			// These are the merge request's own columns; they point at the
			// account by design until the DELETE turns them into NULL.
			continue
		}
		var n int
		if err := tx.QueryRow(ctx,
			fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, c.Table, c.Column),
			absorb).Scan(&n); err != nil {
			return nil, fmt.Errorf("%s: %w", c.Key(), err)
		}
		if n > 0 {
			left = append(left, fmt.Sprintf("%s (%d)", c.Key(), n))
		}
	}
	return left, nil
}
