// api-go/internal/calendars/writescoping_test.go
package calendars

import (
	"regexp"
	"strings"
	"testing"
)

// lineCommentRe strips `//` comments before the scan below.
//
// Not cosmetic: the first version of this test read comments as code and
// flagged reminders/action.go for the sentence EXPLAINING that it had just
// been moved off CalendarIDsFor. A guard that cannot tell a call from a
// mention of a call gets silenced by whoever meets it next, which costs more
// than the guard is worth.
var lineCommentRe = regexp.MustCompile(`(?m)//.*$`)

// A function that CHANGES an event or a task must scope by WritableIDsFor, not
// by CalendarIDsFor.
//
// 🔴 The difference is read access versus write access, and confusing them is
// silent. CalendarIDsFor answers "every calendar you may READ". Ten write
// helpers across events/ and tasks/ used it to build their `calendar_id =
// ANY($n)` clause, so being given read access to a shared calendar carried the
// right to edit, move, resize and delete everything in it — and to move an
// event out of it entirely, which the other members cannot tell apart from a
// deletion. Nothing objects at runtime: a row's user_id records who wrote it,
// not who may change it, and every query involved is internally consistent.
//
// Found on 2026-08-15 while adding calendar_id to PATCH /api/events/:id, which
// needed the source calendar checked as well as the destination. It is latent
// today — nothing creates a viewer membership until invitations land — so a
// DB-backed test proves only that today's code is right. This one keeps
// tomorrow's honest, which is the half that actually matters for a rule whose
// whole hazard is that it costs nothing until it costs everything.
//
// If a legitimate exception ever appears — a write that genuinely should reach
// every readable calendar — do not delete this test: add the function to the
// allowlist with the reason in the commit message.

// mutatesEventOrTaskRe matches a SQL block that CHANGES the event or task
// table. Reads are deliberately excluded: narrowing them would hide a shared
// calendar's events from the people it was shared with, and stop their
// reminders (reminders/scan.go scopes by CalendarIDsFor on purpose).
//
// `\b` after the table name keeps event_exception, task_dependency and
// task_requirement out — different tables with their own rules.
var mutatesEventOrTaskRe = regexp.MustCompile(`(?i)\b(?:UPDATE\s+(event|task)\b|DELETE\s+FROM\s+(event|task)\b|INSERT\s+INTO\s+(event|task)\b)`)

// minWriteFunctionsScanned is today's count of functions that both mutate
// event/task and resolve a calendar id list, with a small margin.
//
// 🔴 Same discipline as minSQLBlocksScanned: raise it as the codebase grows,
// never lower it to make a failure go away. Without a floor this test would
// pass by finding nothing the day these queries stop being backtick literals
// or the helper is renamed — green having checked nothing, which is the exact
// failure this file exists to prevent elsewhere.
//
// Measured at 9 on 2026-08-15 by setting this to 500 and reading the count
// back out of the failure message: 4 in events/handlers.go (updateEvent,
// deleteEvent, moveEvent, resizeEvent), 4 in tasks/handlers.go (updateTask,
// deleteTask, scheduleTask, logTaskTime), and reminders/action.go's
// ActionHandler — which no one had listed, and which this test found.
//
// The margin is 1 rather than the 3 used by minSQLBlocksScanned: there are
// nine functions, not eighty, so three would be a third of the population and
// would let an entire subsystem's worth of writes disappear unnoticed.
const minWriteFunctionsScanned = 8

func TestWritesScopeByWritableCalendars(t *testing.T) {
	var scanned int
	var offenders []string

	walkGoFiles(t, func(path, src string) {
		// Split on top-level func declarations. Crude, and adequate: every
		// query in this codebase lives inside one, and a split that occasionally
		// over-groups can only make this test stricter, never blinder.
		for _, fn := range strings.Split(lineCommentRe.ReplaceAllString(src, ""), "\nfunc ") {
			mutates := false
			for _, m := range sqlBlockRe.FindAllString(fn, -1) {
				block := m[1 : len(m)-1]
				if mutatesEventOrTaskRe.MatchString(block) {
					mutates = true
					break
				}
			}
			if !mutates {
				continue
			}
			// Only functions that resolve a calendar list at all are in scope —
			// a mutation reached through some other guard is not this test's
			// business.
			if !strings.Contains(fn, "CalendarIDsFor") && !strings.Contains(fn, "WritableIDsFor") {
				continue
			}
			scanned++

			if strings.Contains(fn, "CalendarIDsFor") {
				name := strings.SplitN(fn, "(", 2)[0]
				offenders = append(offenders, path+": "+strings.TrimSpace(name))
			}
		}
	})

	if scanned < minWriteFunctionsScanned {
		t.Fatalf("only %d write functions scanned, expected at least %d — the queries or the helper names changed shape and this guard is no longer looking at anything",
			scanned, minWriteFunctionsScanned)
	}

	if len(offenders) > 0 {
		t.Errorf("these functions change an event or task but scope by CalendarIDsFor (read access) instead of WritableIDsFor (write access):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
