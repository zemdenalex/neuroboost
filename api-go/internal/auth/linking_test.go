package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

// The linking secrets are database behaviour — one live token per person,
// spent-once redemption, an attempt counter — so they are tested against a
// database rather than against a fake that would agree with whatever the code
// does.
//
// 🔴 -count=1, always. A cached pass of a DB test is a pass that never ran.
func linkingDB(t *testing.T) *database.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

func linkUser(t *testing.T, d *database.DB) string {
	t.Helper()
	var id string
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO "user" (timezone, settings) VALUES ('Europe/Moscow', '{}'::jsonb) RETURNING id`).
		Scan(&id); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, id)
	})
	return id
}

// 🔴 The secret must exist nowhere but in the message the person is reading.
// A stored token is a token that leaks with the backup.
func TestTheSecretIsNeverStored(t *testing.T) {
	d := linkingDB(t)
	h := &Handler{db: d}
	user := linkUser(t, d)

	token, err := newLoginToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := h.issue(context.Background(), user, loginLinkKind, token); err != nil {
		t.Fatalf("issue: %v", err)
	}

	var stored string
	if err := d.Pool.QueryRow(context.Background(),
		`SELECT token_hash FROM auth_link_token WHERE user_id = $1`, user).Scan(&stored); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored == token {
		t.Fatal("the token itself is in the database")
	}
	if stored != hashSecret(token) {
		t.Errorf("stored value is neither the token nor its hash: %q", stored)
	}
}

// Pressing the button twice must not leave two working links.
func TestIssuingReplacesTheLiveToken(t *testing.T) {
	d := linkingDB(t)
	h := &Handler{db: d}
	user := linkUser(t, d)
	ctx := context.Background()

	first, _ := newLoginToken()
	second, _ := newLoginToken()
	if err := h.issue(ctx, user, loginLinkKind, first); err != nil {
		t.Fatalf("issue first: %v", err)
	}
	if err := h.issue(ctx, user, loginLinkKind, second); err != nil {
		t.Fatalf("issue second: %v", err)
	}

	var live int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM auth_link_token
		  WHERE user_id = $1 AND kind = $2 AND redeemed_at IS NULL`, user, loginLinkKind).
		Scan(&live); err != nil {
		t.Fatalf("count: %v", err)
	}
	if live != 1 {
		t.Errorf("%d live links, expected 1 — revoking the one you can see would not revoke the others", live)
	}

	var hash string
	if err := d.Pool.QueryRow(ctx,
		`SELECT token_hash FROM auth_link_token
		  WHERE user_id = $1 AND redeemed_at IS NULL`, user).Scan(&hash); err != nil {
		t.Fatalf("read: %v", err)
	}
	if hash != hashSecret(second) {
		t.Errorf("the surviving link is not the newest one")
	}
	// The two kinds are independent: a login link must not revoke a link code.
	code, _ := newLinkCode()
	if err := h.issue(ctx, user, linkCodeKind, code); err != nil {
		t.Fatalf("issue code: %v", err)
	}
	var both int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM auth_link_token WHERE user_id = $1 AND redeemed_at IS NULL`, user).
		Scan(&both); err != nil {
		t.Fatalf("count both: %v", err)
	}
	if both != 2 {
		t.Errorf("%d live secrets, expected a login link and a code side by side", both)
	}
}

// An expired secret is not a secret. The lookup must not find it.
func TestAnExpiredTokenIsNotFound(t *testing.T) {
	d := linkingDB(t)
	h := &Handler{db: d}
	user := linkUser(t, d)
	ctx := context.Background()

	token, _ := newLoginToken()
	if err := h.issue(ctx, user, loginLinkKind, token); err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`UPDATE auth_link_token SET expires_at = NOW() - interval '1 minute' WHERE user_id = $1`,
		user); err != nil {
		t.Fatalf("age the token: %v", err)
	}

	var found int
	if err := d.Pool.QueryRow(ctx, `
		SELECT count(*) FROM auth_link_token
		 WHERE kind = $1 AND token_hash = $2 AND redeemed_at IS NULL AND expires_at > NOW()`,
		loginLinkKind, hashSecret(token)).Scan(&found); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found != 0 {
		t.Errorf("an expired link is still redeemable")
	}

	// ⚠ Positive control: without it, a lookup that is simply broken would
	// return zero here and the test would congratulate itself.
	fresh, _ := newLoginToken()
	if err := h.issue(ctx, user, loginLinkKind, fresh); err != nil {
		t.Fatalf("issue fresh: %v", err)
	}
	if err := d.Pool.QueryRow(ctx, `
		SELECT count(*) FROM auth_link_token
		 WHERE kind = $1 AND token_hash = $2 AND redeemed_at IS NULL AND expires_at > NOW()`,
		loginLinkKind, hashSecret(fresh)).Scan(&found); err != nil {
		t.Fatalf("lookup fresh: %v", err)
	}
	if found != 1 {
		t.Fatal("the lookup finds nothing at all — the test above proved nothing")
	}
}

// The link is one-shot. A second redemption of the same token finds no row to
// spend, whatever else happens.
func TestALinkIsSpentOnce(t *testing.T) {
	d := linkingDB(t)
	h := &Handler{db: d}
	user := linkUser(t, d)
	ctx := context.Background()

	token, _ := newLoginToken()
	if err := h.issue(ctx, user, loginLinkKind, token); err != nil {
		t.Fatalf("issue: %v", err)
	}

	spend := func() bool {
		var got string
		err := d.Pool.QueryRow(ctx, `
			UPDATE auth_link_token SET redeemed_at = NOW()
			 WHERE id = (SELECT id FROM auth_link_token
			              WHERE kind = $1 AND token_hash = $2
			                AND redeemed_at IS NULL AND expires_at > NOW()
			              FOR UPDATE SKIP LOCKED LIMIT 1)
			 RETURNING user_id`, loginLinkKind, hashSecret(token)).Scan(&got)
		return err == nil
	}

	if !spend() {
		t.Fatal("the link did not work even once")
	}
	if spend() {
		t.Error("the link worked a second time")
	}
}

// Five wrong guesses burn the code.
func TestACodeBurnsAfterFiveWrongGuesses(t *testing.T) {
	d := linkingDB(t)
	h := &Handler{db: d}
	user := linkUser(t, d)
	ctx := context.Background()

	code, _ := newLinkCode()
	if err := h.issue(ctx, user, linkCodeKind, code); err != nil {
		t.Fatalf("issue: %v", err)
	}
	for i := 0; i < linkCodeAttempts; i++ {
		h.burnAttempt(ctx)
	}

	var attempts int
	if err := d.Pool.QueryRow(ctx,
		`SELECT attempts FROM auth_link_token WHERE user_id = $1 AND kind = $2`, user, linkCodeKind).
		Scan(&attempts); err != nil {
		t.Fatalf("read attempts: %v", err)
	}
	if attempts < linkCodeAttempts {
		t.Errorf("attempts = %d after %d wrong guesses — guessing is not limited",
			attempts, linkCodeAttempts)
	}
}

// Six digits, always six, including the ones with leading zeros.
//
// ⚠ A code printed as "4821" instead of "004821" is a code the person cannot
// type back. %06d is doing real work here, not formatting.
func TestCodesAreAlwaysSixDigits(t *testing.T) {
	for i := 0; i < 200; i++ {
		c, err := newLinkCode()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if len(c) != 6 {
			t.Fatalf("code %q has %d digits", c, len(c))
		}
		for _, r := range c {
			if r < '0' || r > '9' {
				t.Fatalf("code %q is not digits", c)
			}
		}
	}
}

// Two links issued in the same moment must differ.
func TestTokensAreNotPredictable(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		tok, err := newLoginToken()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if seen[tok] {
			t.Fatalf("token repeated after %d draws", i)
		}
		seen[tok] = true
	}
	_ = time.Now
}
