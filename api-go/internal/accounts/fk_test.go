package accounts

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	"neuroboost/api-go/internal/database"
)

// Does the merge know about every foreign key that points at "user"?
//
// 🔴 The question is not academic. Seventeen of these keys are ON DELETE
// CASCADE, and the merge ends by deleting the absorbed account. A table this
// list has never heard of does not produce an error, a warning, or an orphan —
// it produces a silent DELETE of that person's rows at the end of a successful
// merge.
//
// Written against information_schema rather than against a list of table names,
// for the same reason as schema_shape_test.go (gotcha 18): a check built from
// the same assumption as the code cannot disagree with it.
//
// Shown red before it was believed: adding a table with a user_id column to the
// test database and re-running names it in the failure. See merge_test.go,
// TestAnUnknownUserTableIsCaught.
func TestMergeKnowsEveryForeignKeyOnUser(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()

	inDB, err := userForeignKeys(context.Background(), d)
	if err != nil {
		t.Fatalf("read foreign keys: %v", err)
	}
	// 🔴 An empty answer is not an answer: a query that matched nothing would
	// make this test pass while proving nothing at all.
	if len(inDB) == 0 {
		t.Fatal("no foreign keys on \"user\" found — the query is wrong, not the schema")
	}

	known := map[string]bool{}
	for _, c := range Columns {
		known[c.Key()] = true
	}
	have := map[string]bool{}
	for _, k := range inDB {
		have[k] = true
	}

	var missing, extra []string
	for _, k := range inDB {
		if !known[k] {
			missing = append(missing, k)
		}
	}
	for _, c := range Columns {
		if !have[c.Key()] {
			extra = append(extra, c.Key())
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("the database has foreign keys on \"user\" that the merge does not handle:\n  %s\n"+
			"Add them to accounts.Columns. Until then a merge deletes these rows by cascade.",
			strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("accounts.Columns names foreign keys that no longer exist:\n  %s",
			strings.Join(extra, "\n  "))
	}
}

// userForeignKeys asks the database which columns reference "user"(id).
func userForeignKeys(ctx context.Context, d *database.DB) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT c.conrelid::regclass::text || '.' || a.attname
		  FROM pg_constraint c
		  JOIN unnest(c.conkey) WITH ORDINALITY k(attnum, ord) ON true
		  JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
		 WHERE c.contype = 'f' AND c.confrelid = '"user"'::regclass
		 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
