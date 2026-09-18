package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every place that reads tasks has to know that some of them repeat.
//
// 🔴 This is the real cost of recurring tasks, and it is not the migration.
// `task.status` now describes the SERIES, not today: a repeating task stays
// TODO while its day is already done. Any query that filters on status without
// consulting task_occurrence will therefore show a finished thing as
// outstanding, or an outstanding thing as finished — quietly, in one screen at a
// time.
//
// So this test reads the source of every known reader and fails when a query
// touches `task` without either joining the occurrence table or saying, in
// words, that it does not need to. The point is the EIGHTH reader: the one
// written in three months by somebody who never read the plan.
//
// ⚠ A source scan, not a behaviour test, and deliberately so: the failure is
// "someone added a query that forgot", which has no runtime symptom until a user
// notices their pills marked undone. There is nothing to execute.
func TestEveryTaskReaderKnowsAboutRecurrence(t *testing.T) {
	readers := []string{
		"handlers.go",
		"../reminders/scan.go",
		"../planning/handlers.go",
		"../export/queries.go",
		"../calendars/crud.go",
	}

	// A query may opt out by saying why, on the line above or inside the query.
	// The words are a contract: an opt-out without a reason is not an opt-out.
	const optOut = "recurrence-agnostic"

	found := 0
	for _, rel := range readers {
		body, err := os.ReadFile(filepath.Clean(rel))
		if err != nil {
			t.Fatalf("read %s: %v — the reader list is stale, which is worse than a failure", rel, err)
		}
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			if !strings.Contains(line, "FROM task") {
				continue
			}
			// `FROM task_occurrence` and `FROM task_dependency` are other tables.
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "FROM task_") {
				continue
			}
			found++

			// Look at the whole statement and the comment above it.
			from := i - 12
			if from < 0 {
				from = 0
			}
			to := i + 25
			if to > len(lines) {
				to = len(lines)
			}
			block := strings.Join(lines[from:to], "\n")

			if !strings.Contains(block, "task_occurrence") && !strings.Contains(block, optOut) {
				t.Errorf("%s:%d reads `task` without joining task_occurrence and without saying %q:\n    %s",
					rel, i+1, optOut, trimmed)
			}
		}
	}

	// 🔴 The floor. Without it, a typo in every path — or a refactor that moved
	// the queries — would leave this test finding nothing to complain about and
	// passing, which is the shape of a control that cannot fail.
	if found < 5 {
		t.Fatalf("only %d task queries found across %d readers; the list is wrong and this test proved nothing",
			found, len(readers))
	}
}
