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

// «Today» belongs to whoever is ASKING, not to whoever wrote the task.
//
// 🔴 A task lives in a calendar, and a shared calendar holds tasks written by
// other people. Resolving the day from `t.user_id` asks the AUTHOR what day it
// is — so a reader in Moscow looking at a series written by someone in Tokyo
// would be told whether the TOKYO day was done.
//
// Nobody is affected today: production has 0 shared calendars. But sharing is a
// shipped feature being tested right now, so this is a defect waiting for its
// first user, and it needs a shared calendar plus two timezones to reproduce at
// runtime — which is exactly the kind of thing no one reproduces by accident.
// Hence a source scan.
//
// ⚠ It is the same shape as the nag bug fixed the same morning: a time that is
// perfectly valid, just not the reader's.
func TestTodayIsResolvedInTheViewersZone(t *testing.T) {
	readers := []string{"handlers.go", "../planning/handlers.go"}

	found := 0
	for _, rel := range readers {
		body, err := os.ReadFile(filepath.Clean(rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		src := string(body)
		for i, line := range strings.Split(src, "\n") {
			if !strings.Contains(line, "AT TIME ZONE") {
				continue
			}
			found++
			// The zone must come from a parameter (the viewer), never from the
			// row's own author column.
			//
			// ⚠ The window has to reach the whole statement. The first version
			// looked 300 bytes ahead and the explanatory comment above the
			// lookup pushed it out of range — so the sabotage passed and the
			// guard proved nothing. 1500 covers the longest of these queries.
			start := indexOfLine(src, i)
			window := src[start:]
			if len(window) > 1500 {
				window = window[:1500]
			}
			if strings.Contains(window, "id = t.user_id") {
				t.Errorf("%s:%d resolves the day from the task's AUTHOR (t.user_id), not the viewer", rel, i+1)
			}
		}
	}

	// The floor: no occurrence joins found means the queries moved and this
	// guard is looking at nothing.
	if found == 0 {
		t.Fatal("no AT TIME ZONE found in any reader; this test proved nothing")
	}
}

func indexOfLine(src string, line int) int {
	idx := 0
	for i := 0; i < line; i++ {
		n := strings.IndexByte(src[idx:], '\n')
		if n < 0 {
			return idx
		}
		idx += n + 1
	}
	return idx
}
