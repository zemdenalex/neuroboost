package reminders

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// 🔴 A nag must MOVE a reminder, never insert one.
//
// The dedupe index is (user_id, source_kind, COALESCE(event_id, task_id),
// calendar_id, occurrence_start, minutes_before) — 000015:34 — and `remind_at`
// is not in it. Rows for 09:00 and 09:10 of the same occurrence differ only in a
// column outside the key, so a second INSERT collides and the second nag
// silently does not exist.
//
// A source scan rather than a database test, and for a reason that cost a
// release: every DB-backed test in this package begins with
// `if dsn == "" { t.Skip() }`, and CI has no DATABASE_URL for most of them. The
// snooze that answered 500 to every user from 000015 until 18.09 shipped past a
// green suite for exactly that reason. This one cannot skip.
func TestNaggingMovesRowsAndNeverInsertsThem(t *testing.T) {
	body, err := os.ReadFile("nag.go")
	if err != nil {
		t.Fatalf("read nag.go: %v", err)
	}
	src := string(body)

	if !strings.Contains(src, "UPDATE reminder") {
		t.Error("nag.go does not update reminder at all — it cannot be re-arming anything")
	}

	// INSERT in any form, including ON CONFLICT: the conflict target would have
	// to name `remind_at`, and it cannot, because the index does not.
	if regexp.MustCompile(`(?i)insert\s+into\s+reminder`).MatchString(src) {
		t.Error("nag.go INSERTs into reminder; the unique index will reject the second nag of an occurrence")
	}
}

// The cap exists because a bot that nags forever gets muted, and a muted bot
// delivers nothing at all — including the reminder that mattered.
func TestNaggingIsBounded(t *testing.T) {
	if maxNags < 1 || maxNags > 12 {
		t.Errorf("maxNags = %d; outside anything defensible for one morning", maxNags)
	}

	body, err := os.ReadFile("nag.go")
	if err != nil {
		t.Fatalf("read nag.go: %v", err)
	}
	src := string(body)

	for _, guard := range []string{
		"nag_count < $1",        // the cap is applied in the query
		"answered_at IS NULL",   // an answer stops it
		"status = 'SENT'",       // only things actually delivered
		"nag_count = nag_count", // and the count goes up, or the cap never bites
	} {
		if !strings.Contains(src, guard) {
			t.Errorf("nag.go is missing the guard %q", guard)
		}
	}

	// 🔴 The re-check on the UPDATE. Between the SELECT and the write the user
	// may have answered; re-arming a reminder they just dismissed is the most
	// annoying thing this code could do.
	updateIdx := strings.Index(src, "UPDATE reminder")
	if updateIdx < 0 {
		t.Fatal("no UPDATE found")
	}
	tail := src[updateIdx:]
	if !strings.Contains(tail[:min(len(tail), 400)], "answered_at IS NULL") {
		t.Error("the UPDATE does not re-check answered_at; an answer racing the scan would be ignored")
	}
}

// Every action must record that it was answered, or the nag never stops.
//
// ⚠ ActionAck used to change nothing at all — action.go said so in a comment:
// "Nothing to change — the row is already SENT". That was true while a reminder
// arrived once. It stopped being true when nagging existed.
func TestEveryAnswerIsRecorded(t *testing.T) {
	body, err := os.ReadFile("action.go")
	if err != nil {
		t.Fatalf("read action.go: %v", err)
	}
	src := string(body)

	if !strings.Contains(src, "answered_at = NOW()") {
		t.Error("action.go never sets answered_at; every answered reminder would keep coming back")
	}

	// Set before the switch that DISPATCHES the actions, so no branch can
	// forget it. If it moved inside a case, the branches that did not get it
	// would nag for ever.
	//
	// ⚠ LastIndex, not Index: there are two `switch req.Action` blocks in this
	// file and the first one is in ValidateAction, well above the handler. The
	// first version of this check used Index and failed against correct code —
	// a reminder that a probe can be wrong about the thing it is probing.
	mark := strings.Index(src, "answered_at = NOW()")
	sw := strings.LastIndex(src, "switch req.Action {")
	if mark < 0 || sw < 0 || mark > sw {
		t.Error("answered_at is set inside the action switch; a branch that forgets it nags for ever")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
