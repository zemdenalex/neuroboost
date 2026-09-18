// Package accounts merges two user accounts into one.
//
// Kept out of internal/auth deliberately: auth is imported by middleware, and
// the merge needs calendars and settings. A leaf package is also the honest
// shape — the merge is a database operation with one entry point, not a request
// handler.
package accounts

// Handling says what the merge does with one foreign-key column on "user".
type Handling int

const (
	// Move rewrites user_id from the absorbed account to the survivor.
	Move Handling = iota
	// MoveOrDrop moves the row unless the survivor already has its own, in
	// which case the absorbed row is deleted: the column carries a uniqueness
	// constraint that two accounts cannot both satisfy.
	MoveOrDrop
	// Drop deletes the absorbed account's rows without moving them.
	Drop
	// Clear is handled by the database itself (ON DELETE SET NULL) and needs
	// no statement — it is listed so that "known" and "handled" stay the same
	// set.
	Clear
)

// Column is one foreign key pointing at "user"(id).
type Column struct {
	Table  string
	Column string
	How    Handling
	// Why is written for the person reading a red test, not for the compiler.
	Why string
}

// Columns is what the merge knows about. fk_test.go compares it against what
// the database actually has.
//
// 🔴 This list is the feature's safety net. Seventeen of these foreign keys are
// ON DELETE CASCADE, so a column missing from here is not a row left behind —
// it is somebody's events deleted silently when the absorbed account is
// removed. The spec was written believing there were 18 of them; by the time
// the code was written there were 19, because task_occurrence arrived
// overnight. That is precisely the drift the test exists to catch.
var Columns = []Column{
	{"alert_status", "user_id", MoveOrDrop, "user_id is the PRIMARY KEY — one row per person, so the survivor's wins"},
	{"calendar", "owner_id", Move, ""},
	{"calendar_invite", "created_by", Move, ""},
	{"calendar_invite", "used_by", Move, ""},
	{"calendar_member", "user_id", MoveOrDrop, "UNIQUE (calendar_id, user_id) — both accounts can be in one shared calendar"},
	{"event", "user_id", Move, ""},
	{"event_exception", "user_id", Move, ""},
	{"feedback", "user_id", Move, ""},
	{"need", "user_id", Move, "routes were removed in the 14.08 cleanup; the table and its rows were not"},
	{"opportunity", "user_id", Move, "same as need"},
	{"pattern_metrics", "user_id", Move, "same as need"},
	{"planning_edge", "user_id", Move, ""},
	{"planning_node", "user_id", Move, ""},
	{"reflection", "user_id", Move, ""},
	{"reminder", "user_id", Move, ""},
	{"task", "user_id", Move, ""},
	{"task_dependency", "user_id", Move, ""},
	{"task_occurrence", "user_id", Move, ""},
	{"task_requirement", "user_id", Move, ""},

	{"auth_link_token", "user_id", Drop, "a one-shot secret belongs to the account that asked for it and does not transfer"},
	{"account_merge_request", "site_user_id", Clear, "ON DELETE SET NULL keeps the record of the merge after the account is gone"},
	{"account_merge_request", "tg_user_id", Clear, "same"},
	{"account_merge_request", "keep_user_id", Clear, "same"},
}

// Key is how a column is named in messages and in the test's set comparison.
func (c Column) Key() string { return c.Table + "." + c.Column }
