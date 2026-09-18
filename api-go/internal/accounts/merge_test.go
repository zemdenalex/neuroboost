package accounts

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

// These tests need a real database. A merge is SQL — uniqueness, cascades,
// transaction boundaries — and none of that exists in a fake.
//
// 🔴 Run them with DATABASE_URL set AND with -count=1. Without the variable
// every test here skips and the package still prints ok; with a cached result
// the suite reports a pass it never performed. Both failures look identical to
// a green run, and both have already happened in this repository.
func openDB(t *testing.T) *database.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

// makeUser creates an account and removes it when the test ends.
func makeUser(t *testing.T, d *database.DB, tgID *int64, email *string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (tg_id, email, timezone, settings)
		 VALUES ($1, $2, 'Europe/Moscow', '{}'::jsonb) RETURNING id`,
		tgID, email).Scan(&id); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = $1`, id)
	})
	return id
}

func makePersonalCalendar(t *testing.T, d *database.DB, owner string) string {
	t.Helper()
	var id string
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Личный', 'personal') RETURNING id`,
		owner).Scan(&id); err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	return id
}

func makeEvent(t *testing.T, d *database.DB, user, calendar, title string) string {
	t.Helper()
	var id string
	start := time.Now().UTC().Add(time.Hour)
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		user, calendar, title, start, start.Add(time.Hour)).Scan(&id); err != nil {
		t.Fatalf("create event: %v", err)
	}
	return id
}

// makeRequest writes the confirmed merge request that Merge works from.
func makeRequest(t *testing.T, d *database.DB, site, tg, keep string, choice *string) string {
	t.Helper()
	var id string
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO account_merge_request (site_user_id, tg_user_id, keep_user_id, calendar_choice, expires_at)
		 VALUES ($1, $2, $3, $4, NOW() + interval '10 minutes') RETURNING id`,
		site, tg, keep, choice).Scan(&id); err != nil {
		t.Fatalf("create merge request: %v", err)
	}
	return id
}

func count(t *testing.T, d *database.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := d.Pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// The plain case: the site account keeps its email, gains the Telegram identity
// and every event the Telegram account had.
func TestMergeCarriesDataAndBothIdentities(t *testing.T) {
	d := openDB(t)
	ctx := context.Background()

	tgID := int64(time.Now().UnixNano() % 1_000_000_000)
	email := fmt.Sprintf("merge-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)

	siteCal := makePersonalCalendar(t, d, site)
	tgCal := makePersonalCalendar(t, d, tg)
	makeEvent(t, d, site, siteCal, "site event")
	makeEvent(t, d, tg, tgCal, "telegram event")

	choice := string(CalendarsMerge)
	req := makeRequest(t, d, site, tg, site, &choice)

	if err := Merge(ctx, d.Pool, req); err != nil {
		t.Fatalf("merge: %v", err)
	}

	if n := count(t, d, `SELECT count(*) FROM "user" WHERE id = $1`, tg); n != 0 {
		t.Errorf("absorbed account still exists")
	}
	if n := count(t, d, `SELECT count(*) FROM event WHERE user_id = $1`, site); n != 2 {
		t.Errorf("survivor has %d events, expected both", n)
	}
	// 🔴 The point of merging rather than choosing: the survivor holds both
	// identities regardless of which side survived.
	var gotTg *int64
	var gotEmail *string
	if err := d.Pool.QueryRow(ctx, `SELECT tg_id, email FROM "user" WHERE id = $1`, site).
		Scan(&gotTg, &gotEmail); err != nil {
		t.Fatalf("read survivor: %v", err)
	}
	if gotTg == nil || *gotTg != tgID {
		t.Errorf("survivor did not adopt tg_id: %v", gotTg)
	}
	if gotEmail == nil || *gotEmail != email {
		t.Errorf("survivor lost its email: %v", gotEmail)
	}
	// 'merge' folds one personal calendar into the other and removes the shell.
	if n := count(t, d, `SELECT count(*) FROM calendar WHERE id = $1`, tgCal); n != 0 {
		t.Errorf("the absorbed personal calendar was left behind")
	}
	if n := count(t, d, `SELECT count(*) FROM event WHERE calendar_id = $1`, siteCal); n != 2 {
		t.Errorf("events did not move into the surviving calendar: %d", n)
	}
	if n := count(t, d,
		`SELECT count(*) FROM account_merge_request WHERE id = $1 AND status = 'done'`, req); n != 1 {
		t.Errorf("the request was not closed")
	}
	// ⚠ The record of the merge outlives the account it deleted — SET NULL, not
	// CASCADE. Without this the only row explaining where somebody's data went
	// would be deleted by the merge itself.
	if n := count(t, d, `SELECT count(*) FROM account_merge_request WHERE id = $1`, req); n != 1 {
		t.Errorf("the merge deleted its own record")
	}
}

// «Оставить оба» keeps the absorbed personal calendar as a shared one.
func TestKeepBothLeavesTheSecondCalendarVisible(t *testing.T) {
	d := openDB(t)
	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 1
	email := fmt.Sprintf("keepboth-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)
	makePersonalCalendar(t, d, site)
	tgCal := makePersonalCalendar(t, d, tg)

	choice := string(CalendarsKeepBoth)
	req := makeRequest(t, d, site, tg, site, &choice)
	if err := Merge(context.Background(), d.Pool, req); err != nil {
		t.Fatalf("merge: %v", err)
	}

	var kind, name string
	var owner string
	if err := d.Pool.QueryRow(context.Background(),
		`SELECT kind, name, owner_id FROM calendar WHERE id = $1`, tgCal).Scan(&kind, &name, &owner); err != nil {
		t.Fatalf("the second calendar is gone: %v", err)
	}
	// 🔴 Not 'personal': idx_calendar_one_personal_per_owner allows one per
	// owner, so a second personal calendar is not a choice the database offers.
	if kind != "shared" {
		t.Errorf("kind = %q, expected shared", kind)
	}
	if owner != site {
		t.Errorf("the kept calendar did not change owner")
	}
	if name == "Личный" {
		t.Errorf("the kept calendar was not renamed, so the person now has two «Личный»")
	}
}

// Both accounts in one shared calendar: the stronger role survives, and there
// is exactly one membership row afterwards.
func TestSharedCalendarKeepsTheStrongerRole(t *testing.T) {
	d := openDB(t)
	ctx := context.Background()
	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 2
	email := fmt.Sprintf("roles-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)
	owner := makeUser(t, d, nil, nil)

	var cal string
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Общий', 'shared') RETURNING id`,
		owner).Scan(&cal); err != nil {
		t.Fatalf("create shared calendar: %v", err)
	}
	for u, role := range map[string]string{site: "viewer", tg: "editor"} {
		if _, err := d.Pool.Exec(ctx,
			`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, $3, 'active')`,
			cal, u, role); err != nil {
			t.Fatalf("add member: %v", err)
		}
	}

	req := makeRequest(t, d, site, tg, site, nil)
	if err := Merge(ctx, d.Pool, req); err != nil {
		t.Fatalf("merge: %v", err)
	}

	var role string
	if err := d.Pool.QueryRow(ctx,
		`SELECT role FROM calendar_member WHERE calendar_id = $1 AND user_id = $2`, cal, site).
		Scan(&role); err != nil {
		t.Fatalf("membership lost: %v", err)
	}
	if role != "editor" {
		t.Errorf("role = %q — the merge downgraded the person", role)
	}
	// Two rows went in, one must come out: UNIQUE (calendar_id, user_id) means
	// the survivor cannot be in the calendar twice.
	if n := count(t, d, `SELECT count(*) FROM calendar_member WHERE calendar_id = $1`, cal); n != 1 {
		t.Errorf("%d membership rows, expected exactly one after the merge", n)
	}
}

// 🔴 The destructive path's guard. A merge that only proves its happy path
// proves nothing about the operation that deletes somebody's account.
//
// The sabotage is a table with ON DELETE RESTRICT, which the merge's own list
// does not know about — so the final DELETE fails, and the whole transaction
// must roll back with the absorbed account and its data intact.
func TestAFailedMergeLeavesBothAccountsUntouched(t *testing.T) {
	d := openDB(t)
	ctx := context.Background()

	if _, err := d.Pool.Exec(ctx, `
		CREATE TABLE merge_rollback_probe (
		    id      SERIAL PRIMARY KEY,
		    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT
		)`); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	defer func() {
		if _, err := d.Pool.Exec(context.Background(), `DROP TABLE merge_rollback_probe`); err != nil {
			t.Errorf("probe table left behind — it will fail the FK test: %v", err)
		}
	}()

	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 3
	email := fmt.Sprintf("rollback-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)
	tgCal := makePersonalCalendar(t, d, tg)
	makeEvent(t, d, tg, tgCal, "must survive")
	if _, err := d.Pool.Exec(ctx, `INSERT INTO merge_rollback_probe (user_id) VALUES ($1)`, tg); err != nil {
		t.Fatalf("seed probe row: %v", err)
	}

	req := makeRequest(t, d, site, tg, site, nil)
	if err := Merge(ctx, d.Pool, req); err == nil {
		t.Fatal("the merge deleted an account that something still referenced")
	}

	if n := count(t, d, `SELECT count(*) FROM "user" WHERE id = $1`, tg); n != 1 {
		t.Errorf("the absorbed account is gone after a failed merge")
	}
	if n := count(t, d, `SELECT count(*) FROM event WHERE user_id = $1`, tg); n != 1 {
		t.Errorf("the event moved even though the merge failed — the rollback did not happen")
	}
	var gotTg *int64
	if err := d.Pool.QueryRow(ctx, `SELECT tg_id FROM "user" WHERE id = $1`, tg).Scan(&gotTg); err != nil {
		t.Fatalf("read absorbed account: %v", err)
	}
	if gotTg == nil {
		t.Errorf("the account lost its tg_id and can no longer be logged into — rollback incomplete")
	}
	if n := count(t, d,
		`SELECT count(*) FROM account_merge_request WHERE id = $1 AND status = 'pending'`, req); n != 1 {
		t.Errorf("the request was closed despite the failure")
	}
}

// A request nobody answered must not merge anything: which account survives is
// Denis's decision, and an unanswered question is not a default.
func TestAnUnansweredRequestDoesNotMerge(t *testing.T) {
	d := openDB(t)
	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 4
	email := fmt.Sprintf("unanswered-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)

	var req string
	if err := d.Pool.QueryRow(context.Background(),
		`INSERT INTO account_merge_request (site_user_id, tg_user_id, expires_at)
		 VALUES ($1, $2, NOW() + interval '10 minutes') RETURNING id`, site, tg).Scan(&req); err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := Merge(context.Background(), d.Pool, req); err == nil {
		t.Fatal("merged without an answer to «какой аккаунт оставить»")
	}
	if n := count(t, d, `SELECT count(*) FROM "user" WHERE id = $1`, tg); n != 1 {
		t.Errorf("an unanswered request deleted an account")
	}
}

// A merge runs once. The second confirmation must find nothing to do.
func TestAMergeCannotRunTwice(t *testing.T) {
	d := openDB(t)
	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 5
	email := fmt.Sprintf("twice-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)
	req := makeRequest(t, d, site, tg, site, nil)

	if err := Merge(context.Background(), d.Pool, req); err != nil {
		t.Fatalf("first merge: %v", err)
	}
	if err := Merge(context.Background(), d.Pool, req); err == nil {
		t.Fatal("the same request merged a second time")
	}
}

// 🔴 The invariant that the calendars package's static check can no longer see.
//
// calendar_member.role is the truth and calendar.owner_id is a cache; writing
// one without the other produces a divergence that raises no error and that no
// single-sided test can notice. The merge writes owner_id in
// mergePersonalCalendars and moves the membership row later in
// moveEverythingElse, so the two agree only because of the order of those two
// steps — and order is precisely what a grep over SQL literals cannot check.
// This test is what makes that ordering load-bearing instead of lucky.
func TestOwnerAndMembershipStayInStep(t *testing.T) {
	d := openDB(t)
	ctx := context.Background()
	tgID := int64(time.Now().UnixNano()%1_000_000_000) + 6
	email := fmt.Sprintf("instep-%d@example.test", tgID)
	site := makeUser(t, d, nil, &email)
	tg := makeUser(t, d, &tgID, nil)

	// Only the Telegram side has a personal calendar, so the merge moves its
	// ownership rather than folding it away.
	tgCal := makePersonalCalendar(t, d, tg)
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, 'owner', 'active')`,
		tgCal, tg); err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	req := makeRequest(t, d, site, tg, site, nil)
	if err := Merge(ctx, d.Pool, req); err != nil {
		t.Fatalf("merge: %v", err)
	}

	var owner, memberRole string
	var member string
	if err := d.Pool.QueryRow(ctx, `
		SELECT c.owner_id, m.user_id, m.role
		  FROM calendar c
		  JOIN calendar_member m ON m.calendar_id = c.id AND m.role = 'owner'
		 WHERE c.id = $1`, tgCal).Scan(&owner, &member, &memberRole); err != nil {
		t.Fatalf("the calendar lost its owner row: %v", err)
	}
	if owner != member {
		t.Errorf("calendar.owner_id = %s but the owner membership belongs to %s — "+
			"requireOwner and PersonalIDFor now disagree and nothing will say so", owner, member)
	}
	if owner != site {
		t.Errorf("ownership did not move to the surviving account")
	}
}
