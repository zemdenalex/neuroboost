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
// raw string literal (a template, a help message, ...). Uppercase only —
// every query in this codebase is written in uppercase SQL keywords.
var sqlKeywordRe = regexp.MustCompile(`\b(SELECT|INSERT|UPDATE|DELETE)\b`)

// userIDFilterRe matches `user_id = $1` and its dynamic-argument cousin
// `user_id = $%d` (built via fmt.Sprintf for a variable-length UPDATE) alike:
// the digit after `$` is deliberately not part of the pattern.
var userIDFilterRe = regexp.MustCompile(`user_id\s*=\s*\$`)

// minSQLBlocksScanned is today's actual count of SQL-shaped backtick strings
// under api-go/internal (67, measured 2026-08-11), given headroom. It exists
// so that if a query ever moves out of a backtick string — into a named
// constant, a query builder, fmt.Sprintf with the SQL itself as an argument —
// both tests below still have something to fail on, instead of silently
// finding zero matches and going green having checked nothing.
//
// 🔴 Raise this as the codebase grows queries. Never lower it to make a
// failure go away — that defeats the reason it exists.
const minSQLBlocksScanned = 40

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

// userIDFilterAllowed reports whether a SQL block that filters by user_id is
// legitimately allowed to. Kept deliberately narrow — one entry per reason,
// not one entry per file — so a new violation in an already-forgiven file
// still gets caught.
func userIDFilterAllowed(block string) (ok bool, why string) {
	// calendar_member IS the membership table calendars.CalendarIDsFor reads
	// to answer "which calendars can this user see" — there is no calendar_id
	// to scope it by, filtering by user_id here is the whole point of the query.
	if strings.Contains(block, "calendar_member") {
		return true, "calendar_member: this is the membership lookup itself"
	}
	// The "user" table is the account row itself, not calendar-scoped data —
	// events and tasks live in calendars, a person does not.
	if strings.Contains(block, `"user"`) {
		return true, `"user" table: the account row itself, not calendar-scoped data`
	}
	return false, ""
}

// 🔴 Одно забытое место = чужие данные в чужом браузере, и откатить это нельзя:
// показанное однажды показано. Поэтому запрет проверяется механически, а не
// вниманием ревьюера.
//
// Ищется буквально `user_id = $` — то есть ограничение ВЫБОРКИ по автору.
// INSERT, который пишет user_id как колонку авторства, под шаблон не попадает
// и остаётся разрешённым (userIDFilterRe requires `=`, INSERT column lists
// don't have one).
//
// Recursive over api-go/internal, not a fixed file list: a hardcoded list of
// two files is structurally blind to every other file, including the two
// this same fix round found (recurrence.go, occurrence.go). The allow-list
// above is what keeps it from also flagging the one place user_id-filtering
// is correct.
func TestHandlersDoNotScopeQueriesByUserID(t *testing.T) {
	scanned := 0

	walkGoFiles(t, func(path, src string) {
		for _, m := range sqlBlockRe.FindAllString(src, -1) {
			block := m[1 : len(m)-1] // strip the surrounding backticks
			if !sqlKeywordRe.MatchString(block) {
				continue
			}
			scanned++

			if !userIDFilterRe.MatchString(block) {
				continue
			}
			if ok, _ := userIDFilterAllowed(block); ok {
				continue
			}
			t.Errorf(
				"%s scopes a query by user_id = $. Access comes from calendar membership: "+
					"use calendars.CalendarIDsFor and `calendar_id = ANY($n)`.\n\tquery: %s",
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
