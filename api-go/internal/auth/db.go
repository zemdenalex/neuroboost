package auth

import (
        "context"
        "database/sql"
        "errors"
        "fmt"
        "time"
)

const dbTimeout = 5 * time.Second

// GetUserByTelegramID fetches a user by their Telegram ID.  Returns sql.ErrNoRows if
// no user exists with the given ID.
func GetUserByTelegramID(db *sql.DB, telegramID int64) (*User, error) {
        ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
        defer cancel()

        const q = `
SELECT id, telegram_id, username, first_name, last_name, photo_url, timezone, created_at, updated_at
FROM users
WHERE telegram_id = $1
LIMIT 1;`

        var u User
        err := db.QueryRowContext(ctx, q, telegramID).Scan(
                &u.ID, &u.TgID, &u.TgUsername, &u.TgFirstName, &u.TgLastName, &u.TgPhotoURL, &u.Timezone, &u.CreatedAt, &u.LastLoginAt,
        )
        if err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, sql.ErrNoRows
                }
                return nil, fmt.Errorf("get user by telegram id: %w", err)
        }
        return &u, nil
}

// GetUserByID fetches a user by their primary key.  Returns sql.ErrNoRows if
// not found.
func GetUserByID(db *sql.DB, userID int64) (*User, error) {
        ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
        defer cancel()

        const q = `
SELECT id, telegram_id, username, first_name, last_name, photo_url, timezone, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;`

        var u User
        err := db.QueryRowContext(ctx, q, userID).Scan(
                &u.ID, &u.TgID, &u.TgUsername, &u.TgFirstName, &u.TgLastName, &u.TgPhotoURL, &u.Timezone, &u.CreatedAt, &u.LastLoginAt,
        )
        if err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, sql.ErrNoRows
                }
                return nil, fmt.Errorf("get user by id: %w", err)
        }
        return &u, nil
}

// CreateUser inserts a new user with the provided Telegram details.  It
// returns the newly created user record.
func CreateUser(db *sql.DB, telegramID int64, username, firstName, lastName, photoURL string) (*User, error) {
        ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
        defer cancel()

        const q = `
INSERT INTO users (telegram_id, username, first_name, last_name, photo_url, timezone, last_login)
VALUES ($1, $2, $3, $4, $5, 'UTC', NOW())
RETURNING id, telegram_id, username, first_name, last_name, photo_url, timezone, created_at, updated_at;`

        var u User
        err := db.QueryRowContext(ctx, q, telegramID, username, firstName, lastName, photoURL).Scan(
                &u.ID, &u.TgID, &u.TgUsername, &u.TgFirstName, &u.TgLastName, &u.TgPhotoURL, &u.Timezone, &u.CreatedAt, &u.LastLoginAt,
        )
        if err != nil {
                return nil, fmt.Errorf("create user: %w", err)
        }
        return &u, nil
}

// UpdateUser accepts a small map and builds a parameterized update.  Supported keys:
// username, first_name, last_name, photo_url, timezone, last_login.  It will
// always update the updated_at timestamp.  Any keys not in the allowed
// list are ignored.
func UpdateUser(db *sql.DB, userID int64, updates map[string]interface{}) error {
        if len(updates) == 0 {
                return nil
        }
        ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
        defer cancel()

        allowed := map[string]bool{
                "username":   true,
                "first_name": true,
                "last_name":  true,
                "photo_url":  true,
                "timezone":   true,
                "last_login": true,
        }

        set := make([]string, 0, len(updates)+1)
        args := make([]any, 0, len(updates)+1)
        i := 1
        for k, v := range updates {
                if !allowed[k] {
                        continue
                }
                set = append(set, fmt.Sprintf("%s = $%d", k, i))
                args = append(args, v)
                i++
        }
        if len(set) == 0 {
                return nil
        }
        // Always bump updated_at; string literal avoids unnecessary Sprintf
        set = append(set, "updated_at = NOW()")

        q := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d;`, stringsJoin(set, ", "), i)
        args = append(args, userID)

        stmt, err := db.PrepareContext(ctx, q)
        if err != nil {
                return fmt.Errorf("prepare update: %w", err)
        }
        defer stmt.Close()

        if _, err := stmt.ExecContext(ctx, args...); err != nil {
                return fmt.Errorf("exec update: %w", err)
        }
        return nil
}

// stringsJoin is a local helper to avoid importing strings in multiple files.
func stringsJoin(parts []string, sep string) string {
        switch len(parts) {
        case 0:
                return ""
        case 1:
                return parts[0]
        }
        n := len(sep) * (len(parts) - 1)
        for i := 0; i < len(parts); i++ {
                n += len(parts[i])
        }
        b := make([]byte, 0, n)
        for i := 0; i < len(parts); i++ {
                if i > 0 {
                        b = append(b, sep...)
                }
                b = append(b, parts[i]...)
        }
        return string(b)
}