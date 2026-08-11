// api-go/internal/calendars/scoping_test.go
package calendars

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 🔴 Одно забытое место = чужие данные в чужом браузере, и откатить это нельзя:
// показанное однажды показано. Поэтому запрет проверяется механически, а не
// вниманием ревьюера.
//
// Ищется буквально `user_id = $` — то есть ограничение ВЫБОРКИ по автору.
// INSERT, который пишет user_id как колонку авторства, под шаблон не попадает
// и остаётся разрешённым.
func TestHandlersDoNotScopeQueriesByUserID(t *testing.T) {
	forbidden := []byte("user_id = $")

	for _, path := range []string{
		"../events/handlers.go",
		"../tasks/handlers.go",
	} {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		if bytes.Contains(src, forbidden) {
			t.Errorf(
				"%s scopes a query by %q. Access comes from calendar membership: "+
					"use calendars.CalendarIDsFor and `calendar_id = ANY($n)`.",
				path, forbidden)
		}
	}
}

// insertIntoRe finds `INSERT INTO event (...)` / `INSERT INTO task (...)`,
// column list included, however many lines it wraps across. `\b` after the
// table name is load-bearing: it is what keeps `event_exception` — a
// different table, never subject to this rule — from matching `event`.
// Column lists in this codebase never nest parens, so stopping at the first
// `)` is exact, not an approximation.
var insertIntoRe = regexp.MustCompile(`(?s)INSERT INTO (event|task)\b\s*\(([^)]*)\)`)

// 🔴 event.calendar_id and task.calendar_id are NOT NULL (migration 000012).
// An INSERT into either table that omits the column does not silently scope
// wrong — it fails every time, at runtime, on the first request that hits it.
// occurrence.go's detachOccurrence shipped exactly that bug: the replacement
// row for a detached recurring occurrence had no calendar_id at all. Caught
// by hand once; this test is what stops the next one from needing that.
func TestInsertsIntoEventOrTaskSetCalendarID(t *testing.T) {
	root := ".."

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

		for _, m := range insertIntoRe.FindAllStringSubmatch(string(src), -1) {
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
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}
