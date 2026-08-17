package reminders

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"neuroboost/api-go/internal/database"
)

// Does the `reminder` table actually have the columns this package queries?
//
// 🔴 On 2026-08-18 production answered no, and nothing knew. 000001_baseline
// declares the table with CREATE TABLE IF NOT EXISTS; production already had a
// `reminder` table from before the migration system, with a `method` column and
// without minutes_before/channel/status/message/sent_at. The baseline created
// nothing and recorded success. Five migrations later 000010 indexed
// minutes_before, failed, left schema_migrations dirty at 10, and golang-migrate
// then refused to start the API at all. Production was down.
//
// Every test in this repository passed throughout, because every test database
// is built by running the migrations from scratch — and from scratch the
// baseline DOES create the table correctly. The divergence was invisible to
// exactly the thing meant to catch it.
//
// So this test asks the database what it has rather than trusting the chain
// that built it. It is cheap, and against a restored production dump it is the
// only thing here that would have gone red.
func TestReminderTableHasEveryColumnThisPackageQueries(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	ctx := context.Background()

	rows, err := d.Pool.Query(ctx,
		`SELECT column_name FROM information_schema.columns
		  WHERE table_schema = 'public' AND table_name = 'reminder'`)
	if err != nil {
		t.Fatalf("read columns: %v", err)
	}
	defer rows.Close()

	present := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan: %v", err)
		}
		present[c] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	// The floor. Without it, a query that returned nothing — wrong schema, wrong
	// database, a typo in the table name — would satisfy every check below by
	// finding no columns to complain about, and this test would guard nothing.
	if len(present) == 0 {
		t.Fatal("no columns found for `reminder` — the table is missing, and this test proved nothing")
	}

	// Written out rather than parsed from the SQL strings: a scan of the source
	// would be built from the same mistaken picture as the code, which is the
	// failure mode that let a silent switch ship in the bot. This list is the
	// contract, maintained by hand, and it is short.
	required := []string{
		"user_id", "event_id", "task_id", "calendar_id",
		"source_kind", "occurrence_start", "minutes_before",
		"remind_at", "channel", "status", "message", "sent_at", "attempts",
	}

	var missing []string
	for _, c := range required {
		if !present[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		t.Fatalf(
			"`reminder` is missing %s — the migration chain reported success against a table it did not create (see migration 000016)",
			strings.Join(missing, ", "),
		)
	}
}

// The repair itself, exercised against the shape production actually had.
//
// This builds the legacy table under a different name, applies the five
// statements from 000016 to it, and requires the result to carry what the code
// needs. It is the only place the pre-migration-system schema is written down.
func TestTheRepairFixesTheLegacyProductionShape(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	ctx := context.Background()

	const table = "reminder_legacy_shape_probe"
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table))
	})

	// Production's actual columns on 2026-08-18, read off the live database:
	// id, user_id, event_id, remind_at, method, created_at. Note `method` where
	// the baseline promised minutes_before/channel/status/message/sent_at.
	if _, err := d.Pool.Exec(ctx, fmt.Sprintf(`
		DROP TABLE IF EXISTS %s;
		CREATE TABLE %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			event_id UUID,
			remind_at TIMESTAMPTZ NOT NULL,
			method TEXT DEFAULT 'PUSH',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, table, table)); err != nil {
		t.Fatalf("seed the legacy shape: %v", err)
	}

	// Sanity: the fixture really is the broken shape. Otherwise the repair below
	// would be applied to an already-correct table and prove nothing.
	var hasMinutesBefore bool
	if err := d.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns
		   WHERE table_name = $1 AND column_name = 'minutes_before')`, table).Scan(&hasMinutesBefore); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if hasMinutesBefore {
		t.Fatal("the legacy fixture already has minutes_before — it is not the shape that broke production")
	}

	// The five statements of 000016, verbatim in effect.
	for _, stmt := range []string{
		`ADD COLUMN IF NOT EXISTS minutes_before INTEGER`,
		`ADD COLUMN IF NOT EXISTS channel TEXT DEFAULT 'TELEGRAM'`,
		`ADD COLUMN IF NOT EXISTS status  TEXT DEFAULT 'PENDING'`,
		`ADD COLUMN IF NOT EXISTS message TEXT`,
		`ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ`,
	} {
		if _, err := d.Pool.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s %s`, table, stmt)); err != nil {
			t.Fatalf("repair %q: %v", stmt, err)
		}
	}

	for _, c := range []string{"minutes_before", "channel", "status", "message", "sent_at"} {
		var ok bool
		if err := d.Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns
			   WHERE table_name = $1 AND column_name = $2)`, table, c).Scan(&ok); err != nil {
			t.Fatalf("verify %s: %v", c, err)
		}
		if !ok {
			t.Errorf("the repair did not add %s", c)
		}
	}
}
