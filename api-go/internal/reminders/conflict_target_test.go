package reminders

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The ON CONFLICT target in the snooze must name exactly the columns of the
// dedupe index — no more, no fewer.
//
// 🔴 It did not, and snooze answered 500 to every press. 000015 added
// `calendar_id` to idx_reminder_dedupe; the INSERT in action.go kept the old
// five-column target, and PostgreSQL replied «there is no unique or exclusion
// constraint matching the ON CONFLICT specification» (SQLSTATE 42P10). Measured
// from the dev API's own log on 18.09: six 500s in forty seconds, while the
// chat said only «⚠️ Не получилось — попробуй ещё раз». Production carries the
// same index and the same defect; it shows zero failures only because nobody
// had pressed the button there.
//
// 🔴 Why this test and not a database one: every DB-backed test in this package
// begins with `if dsn == "" { t.Skip() }`, and CI has no DATABASE_URL. A
// control that skips is a control that cannot fail, so the defect shipped past
// a green suite. This one reads the migration and the source, needs nothing,
// and cannot be skipped.
func TestSnoozeConflictTargetMatchesTheDedupeIndex(t *testing.T) {
	indexCols := dedupeIndexColumns(t)
	targetCols := snoozeConflictColumns(t)

	if len(indexCols) == 0 {
		t.Fatal("could not read the dedupe index from the migrations")
	}
	if len(targetCols) == 0 {
		t.Fatal("could not read the ON CONFLICT target from action.go")
	}
	if strings.Join(indexCols, ",") != strings.Join(targetCols, ",") {
		t.Errorf("ON CONFLICT does not match idx_reminder_dedupe — every snooze will be a 500\n index:  %v\n insert: %v",
			indexCols, targetCols)
	}
}

// dedupeIndexColumns reads the LAST definition of idx_reminder_dedupe in the
// migrations — the last one is the one in force.
func dedupeIndexColumns(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)

	re := regexp.MustCompile(`(?is)CREATE\s+UNIQUE\s+INDEX[^;]*?idx_reminder_dedupe[^(]*\((.*?)\)\s*(?:NULLS[^;]*)?;`)
	var cols []string
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if m := re.FindStringSubmatch(string(body)); m != nil {
			cols = splitColumns(m[1])
		}
	}
	return cols
}

func snoozeConflictColumns(t *testing.T) []string {
	t.Helper()
	body, err := os.ReadFile("action.go")
	if err != nil {
		t.Fatalf("read action.go: %v", err)
	}
	re := regexp.MustCompile(`(?is)ON\s+CONFLICT\s*\((.*?)\)\s*\n?\s*DO\s+UPDATE`)
	m := re.FindStringSubmatch(string(body))
	if m == nil {
		return nil
	}
	return splitColumns(m[1])
}

// splitColumns splits a column list at top-level commas, so COALESCE(a, b)
// stays one entry, and normalises whitespace.
func splitColumns(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, normalise(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, normalise(s[start:]))
	clean := out[:0]
	for _, c := range out {
		if c != "" {
			clean = append(clean, c)
		}
	}
	return clean
}

func normalise(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), "")
}
