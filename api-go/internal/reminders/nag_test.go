package reminders

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
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

// 🔴 The end of the day belongs to the OWNER, not to UTC.
//
// This is the defect the first version shipped with: occurrence_start comes back
// from postgres in UTC, so building 23:59:59 from it produced the end of the UTC
// day. For a Moscow reader that is 02:59 the next morning — exactly the wake-up
// the guard exists to prevent — and for anyone west of UTC it cuts their evening
// short instead.
func TestNagStopsAtTheEndOfTheOWNERSDay(t *testing.T) {
	// An event at 20:00 Moscow == 17:00 UTC.
	occurrence := time.Date(2026, time.September, 18, 17, 0, 0, 0, time.UTC)

	// 22:30 Moscow — still the same evening, so nagging is fine.
	sameEvening := time.Date(2026, time.September, 18, 19, 30, 0, 0, time.UTC)
	if !WithinSameLocalDay(&occurrence, sameEvening, "Europe/Moscow") {
		t.Error("22:30 Moscow was refused, but it is the same evening")
	}

	// 01:30 Moscow the next day == 22:30 UTC the SAME day. Under the UTC bug
	// this passed, because it is still before 23:59:59 UTC.
	nextMorning := time.Date(2026, time.September, 18, 22, 30, 0, 0, time.UTC)
	if WithinSameLocalDay(&occurrence, nextMorning, "Europe/Moscow") {
		t.Error("01:30 the following morning in Moscow was allowed — that is the 2am wake-up")
	}

	// The mirror case, west of UTC: 21:00 New York on the 18th is 01:00 UTC on
	// the 19th. Under the UTC bug this was refused and the evening was cut off.
	nyOccurrence := time.Date(2026, time.September, 18, 23, 0, 0, 0, time.UTC) // 19:00 NY
	nyEvening := time.Date(2026, time.September, 19, 1, 0, 0, 0, time.UTC)     // 21:00 NY
	if !WithinSameLocalDay(&nyOccurrence, nyEvening, "America/New_York") {
		t.Error("21:00 New York was refused, but it is the same evening there")
	}
}

// A row about no particular day has no day to overrun. The nag count still
// bounds it.
func TestNagAllowsRowsWithNoOccurrence(t *testing.T) {
	far := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !WithinSameLocalDay(nil, far, "Europe/Moscow") {
		t.Error("a row with no occurrence was refused")
	}
}

// An unknown zone falls back to UTC rather than to the server's local time.
func TestNagFallsBackToUTCForAnUnknownZone(t *testing.T) {
	occurrence := time.Date(2026, time.September, 18, 12, 0, 0, 0, time.UTC)
	if !WithinSameLocalDay(&occurrence, occurrence.Add(time.Hour), "Mars/Olympus") {
		t.Error("an unknown zone refused a same-day nag")
	}
	if WithinSameLocalDay(&occurrence, occurrence.Add(24*time.Hour), "Mars/Olympus") {
		t.Error("an unknown zone allowed a nag a day later")
	}
}
