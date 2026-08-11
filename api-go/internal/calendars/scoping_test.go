// api-go/internal/calendars/scoping_test.go
package calendars

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sqlBlockRe finds every backtick-delimited Go raw string in a source file
// that looks like a SQL statement. It is the unit both tests below scan:
// test 1 asks "does this block filter by user_id", test 2 asks "does this
// block insert into event/task without calendar_id".
var sqlBlockRe = regexp.MustCompile("(?s)`([^`]*)`")

// sqlKeywordRe recognizes a backtick string as SQL rather than some other
// raw string literal (a template, a help message, ...). Case-insensitive:
// this codebase's convention is uppercase SQL keywords, but the guard must
// not depend on every future query following that convention by hand.
var sqlKeywordRe = regexp.MustCompile(`(?i)\b(SELECT|INSERT|UPDATE|DELETE)\b`)

// userIDFilterRe matches `user_id = $1` and its dynamic-argument cousin
// `user_id = $%d` (built via fmt.Sprintf for a variable-length UPDATE) alike:
// the digit after `$` is deliberately not part of the pattern. Also matches
// the `user_id = ANY($1)` form — the array-membership counterpart to
// `calendar_id = ANY($1)` seen elsewhere in this codebase, and specifically
// the form most likely to be copy-pasted from that neighboring line without
// noticing it re-adds a user_id scope. Case-insensitive for the same reason
// as sqlKeywordRe above.
var userIDFilterRe = regexp.MustCompile(`(?i)user_id\s*=\s*(?:any\()?\$`)

// tableRefRe matches a SQL block that operates on the event or task table —
// `FROM event`, `UPDATE event`, `JOIN event`, and the task equivalents.
// `DELETE FROM event` is covered by the `FROM` case already. `\b` after the
// table name is load-bearing: it is what keeps `event_exception`,
// `task_dependency`, and `task_requirement` — different tables, legitimately
// scoped by user_id or anything else — from matching `event`/`task`.
// Case-insensitive: `select ... from task ...` is valid Go/SQL and must not
// slip past this guard just because it isn't uppercase.
var tableRefRe = regexp.MustCompile(`(?i)\b(?:FROM|UPDATE|JOIN)\s+(event|task)\b`)

// minSQLBlocksScanned is today's actual count of SQL-shaped backtick strings
// under api-go/internal, given headroom. It exists so that if a query ever
// moves out of a backtick string — into a named constant, a query builder,
// fmt.Sprintf with the SQL itself as an argument — both tests below still
// have something to fail on, instead of silently finding zero matches and
// going green having checked nothing.
//
// 🔴 Raise this as the codebase grows queries. Never lower it to make a
// failure go away — that defeats the reason it exists.
const minSQLBlocksScanned = 60

// walkGoFiles calls fn with the contents of every non-test .go file under
// api-go/internal.
func walkGoFiles(t *testing.T, fn func(path, src string)) {
	t.Helper()

	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fn(path, string(src))
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

// 🔴 One forgotten spot means someone else's data in someone else's browser,
// and there's no undoing that: once shown, it's been shown. So the rule is
// enforced mechanically, not by a reviewer's attention.
//
// The rule is narrower than "no query filters by user_id" — that would also
// flag planning/reflections/reminders' own tables, and their own tables are
// legitimately personal (out of P3 slice 1's scope; the spec's "Что НЕ
// входит" section says so explicitly). The actual rule: a query that reads
// or writes the event or task table may not gate that access on user_id —
// access to those two tables specifically comes from calendar membership.
// A block is forbidden only when it BOTH references event/task (tableRefRe)
// AND filters by user_id (userIDFilterRe). No allow-list is needed with this
// narrower rule: calendar_member and "user" table queries never match
// tableRefRe in the first place, since neither name is "event" or "task".
//
// Recursive over api-go/internal, not a fixed file list: a hardcoded list of
// two files is structurally blind to every other file, including the two
// this same fix round found (recurrence.go, occurrence.go).
func TestHandlersDoNotScopeQueriesByUserID(t *testing.T) {
	scanned := 0

	walkGoFiles(t, func(path, src string) {
		for _, m := range sqlBlockRe.FindAllString(src, -1) {
			block := m[1 : len(m)-1] // strip the surrounding backticks
			if !sqlKeywordRe.MatchString(block) {
				continue
			}
			scanned++

			if !tableRefRe.MatchString(block) || !userIDFilterRe.MatchString(block) {
				continue
			}
			t.Errorf(
				"%s scopes an event/task query by user_id = $. Access comes from calendar "+
					"membership: use calendars.CalendarIDsFor and `calendar_id = ANY($n)`.\n\tquery: %s",
				path, strings.TrimSpace(block))
		}
	})

	if scanned < minSQLBlocksScanned {
		t.Fatalf(
			"only scanned %d SQL blocks under api-go/internal, expected at least %d — "+
				"the regex likely stopped matching real queries (moved to a constant, a "+
				"builder, ...) and this test just checked nothing",
			scanned, minSQLBlocksScanned)
	}
}

// insertIntoRe finds `INSERT INTO event (...)` / `INSERT INTO task (...)`,
// column list included, however many lines it wraps across. `\b` after the
// table name is load-bearing: it is what keeps `event_exception` — a
// different table, never subject to this rule — from matching `event`.
// Column lists in this codebase never nest parens, so stopping at the first
// `)` is exact, not an approximation.
var insertIntoRe = regexp.MustCompile(`(?s)INSERT INTO (event|task)\b\s*\(([^)]*)\)`)

// minEventOrTaskInserts is today's actual count of INSERT INTO event/task
// statements under api-go/internal (6, measured 2026-08-11: events/handlers.go,
// events/occurrence.go, export/handlers.go x2, tasks/handlers.go x2), minus
// headroom. A fixed INSERT is still an INSERT — this count does not shrink
// as violations below get fixed, only grows as the codebase does.
//
// 🔴 Raise this as new INSERT INTO event/task call sites appear. Never lower
// it to make a failure go away.
const minEventOrTaskInserts = 4

// 🔴 event.calendar_id and task.calendar_id are NOT NULL (migration 000012).
// An INSERT into either table that omits the column does not silently scope
// wrong — it fails every time, at runtime, on the first request that hits it.
// occurrence.go's detachOccurrence shipped exactly that bug: the replacement
// row for a detached recurring occurrence had no calendar_id at all. Caught
// by hand once; this test is what stops the next one from needing that.
func TestInsertsIntoEventOrTaskSetCalendarID(t *testing.T) {
	found := 0

	walkGoFiles(t, func(path, src string) {
		for _, m := range insertIntoRe.FindAllStringSubmatch(src, -1) {
			found++
			table, cols := m[1], m[2]
			if !strings.Contains(cols, "calendar_id") {
				t.Errorf(
					"%s: INSERT INTO %s does not list calendar_id. That column is NOT NULL "+
						"(migration 000012) — this insert fails at runtime, every time, not just "+
						"for the wrong user. Add calendar_id to the column list and its value to "+
						"the args.",
					path, table)
			}
		}
	})

	if found < minEventOrTaskInserts {
		t.Fatalf(
			"only found %d INSERT INTO event/task statements under api-go/internal, "+
				"expected at least %d — the regex likely stopped matching real inserts and "+
				"this test just checked nothing",
			found, minEventOrTaskInserts)
	}
}
