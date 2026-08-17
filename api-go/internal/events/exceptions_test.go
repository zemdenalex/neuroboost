package events

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"neuroboost/api-go/internal/database"
)

// fetchExceptions used to return a bare []time.Time and answer nil on any
// failure. Since nil means "this series has no skipped occurrences", a
// momentary database failure put every deleted occurrence back on the user's
// calendar — with nothing in the log to say so.
//
// The fix is the signature: an error is now returned, and ListHandler refuses
// the request instead of silently redrawing deleted occurrences.
func TestFetchExceptionsReportsFailureInsteadOfClaimingThereAreNone(t *testing.T) {
	unreachable := lazyBrokenDB(t)

	// db is package-level; every other test in this package is pure, and one
	// that silently inherited a broken pool would be maddening to diagnose.
	previous := db
	db = unreachable
	t.Cleanup(func() { db = previous })

	got, err := fetchExceptions(context.Background(), []string{"11111111-1111-1111-1111-111111111111"}, "22222222-2222-2222-2222-222222222222")

	if err == nil {
		t.Fatal("a failing query reported success; a deleted occurrence would reappear on the calendar")
	}
	if got != nil {
		t.Errorf("exceptions = %v, want nil alongside the error", got)
	}
}

// lazyBrokenDB builds a pool that CONNECTS ON FIRST USE and always fails.
//
// 🔴 Do not reach for database.New here: it pings during construction, so an
// unreachable DSN fails immediately and the t.Skip that used to guard it turned
// this whole test into a silent no-op. It skipped in CI and locally, and it was
// reported as covering the refusal path when it covered nothing.
// pgxpool.NewWithConfig does not dial until the first query, which is exactly
// the failure being tested: a pool that looks fine and fails when used.
func lazyBrokenDB(t *testing.T) *database.DB {
	t.Helper()
	// Port 1 is reserved and never listening.
	cfg, err := pgxpool.ParseConfig("postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("could not parse the test DSN: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("could not build a lazy pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &database.DB{Pool: pool}
}
