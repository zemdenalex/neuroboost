package calendars

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Sharing a calendar — the tests that matter are the refusals.
//
// Every one of these is a way for one person to end up seeing another person's
// calendar, which is the worst thing this product can do. They run against a
// real Postgres (setupTestDB skips without DATABASE_URL); the pure access rule
// they sit on top of is AccessibleIDs, tested separately without a database.

// seedPerson makes another person to share with, and returns their email —
// which is what the invite path needs and what writable_ids_test.go's seedUser
// (kept, it has other callers) does not return.
func seedPerson(t *testing.T, label string) (id, email string) {
	t.Helper()
	ctx := context.Background()
	email = fmt.Sprintf("%s-%d@example.com", label, time.Now().UnixNano())
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("seed %s: %v", label, err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id) })
	return id, email
}

// TestInviteByEmailGrantsNothingUntilAccepted is the whole point of the
// 'invited' status: an invitation is an offer, not a grant.
func TestInviteByEmailGrantsNothingUntilAccepted(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Наши планы", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inviteeID, inviteeEmail := seedPerson(t, "invitee")

	mem, err := InviteByEmail(ctx, ownerID, c.ID, inviteeEmail, RoleEditor)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if mem.Status != StatusInvited {
		t.Fatalf("a fresh invitation must be 'invited', got %q", mem.Status)
	}

	// The claim: no read access yet.
	ids, err := CalendarIDsFor(ctx, inviteeID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	for _, id := range ids {
		if id == c.ID {
			t.Fatal("an invited user can already read the calendar — the invitation granted access")
		}
	}

	// The positive control. Without it, a CalendarIDsFor that returned nothing
	// at all — or an invite that silently did nothing — would pass the above.
	if err := RespondToInvitation(ctx, inviteeID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}
	ids, err = CalendarIDsFor(ctx, inviteeID)
	if err != nil {
		t.Fatalf("ids after accept: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == c.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("accepting the invitation did not grant access")
	}
}

// TestViewerCannotWrite pins the role split against the helper that enforces
// it. Read access and write access are different questions, and this is where
// they are answered differently for the same person.
func TestViewerCannotWrite(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Только посмотреть", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	viewerID, viewerEmail := seedPerson(t, "viewer")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, viewerEmail, RoleViewer); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := RespondToInvitation(ctx, viewerID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}

	readable, err := CalendarIDsFor(ctx, viewerID)
	if err != nil {
		t.Fatalf("readable: %v", err)
	}
	writable, err := WritableIDsFor(ctx, viewerID)
	if err != nil {
		t.Fatalf("writable: %v", err)
	}
	if !contains(readable, c.ID) {
		t.Fatal("a viewer must be able to READ the calendar")
	}
	if contains(writable, c.ID) {
		t.Fatal("a viewer must NOT be able to write to the calendar")
	}
}

// TestEditorCanWrite is the other half — without it, a WritableIDsFor that
// returned nothing would pass TestViewerCannotWrite.
func TestEditorCanWrite(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Вместе ведём", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	editorID, editorEmail := seedPerson(t, "editor")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, editorEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := RespondToInvitation(ctx, editorID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}

	writable, err := WritableIDsFor(ctx, editorID)
	if err != nil {
		t.Fatalf("writable: %v", err)
	}
	if !contains(writable, c.ID) {
		t.Fatal("an editor must be able to write to the calendar")
	}
}

// TestPersonalCalendarCannotBeShared is the refusal that protects everything a
// user never deliberately filed anywhere.
func TestPersonalCalendarCannotBeShared(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	personalID, err := PersonalIDFor(ctx, ownerID)
	if err != nil {
		t.Fatalf("personal: %v", err)
	}
	_, otherEmail := seedPerson(t, "other")

	if _, err := InviteByEmail(ctx, ownerID, personalID, otherEmail, RoleEditor); !errors.Is(err, ErrCalendarIsPersonalShare) {
		t.Fatalf("invite to personal: want ErrCalendarIsPersonalShare, got %v", err)
	}
	if _, err := CreateInviteLink(ctx, ownerID, personalID, RoleEditor); !errors.Is(err, ErrCalendarIsPersonalShare) {
		t.Fatalf("link for personal: want ErrCalendarIsPersonalShare, got %v", err)
	}
}

// TestOnlyOwnerCanInvite: an editor is a collaborator, not a co-owner. Without
// this, anyone let into a calendar could let anyone else in.
func TestOnlyOwnerCanInvite(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Общий", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	editorID, editorEmail := seedPerson(t, "editor")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, editorEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := RespondToInvitation(ctx, editorID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}

	_, strangerEmail := seedPerson(t, "stranger")
	if _, err := InviteByEmail(ctx, editorID, c.ID, strangerEmail, RoleEditor); !errors.Is(err, ErrNotCalendarOwner) {
		t.Fatalf("editor inviting: want ErrNotCalendarOwner, got %v", err)
	}
	if _, err := CreateInviteLink(ctx, editorID, c.ID, RoleEditor); !errors.Is(err, ErrNotCalendarOwner) {
		t.Fatalf("editor making a link: want ErrNotCalendarOwner, got %v", err)
	}
}

// TestStrangerInvitingGetsNotFound: a stranger must not learn the calendar
// exists — the same rule the rest of this package follows.
func TestStrangerInvitingGetsNotFound(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Чужой", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	strangerID, _ := seedPerson(t, "stranger")
	_, targetEmail := seedPerson(t, "target")

	if _, err := InviteByEmail(ctx, strangerID, c.ID, targetEmail, RoleEditor); !errors.Is(err, ErrCalendarNotFound) {
		t.Fatalf("stranger inviting: want ErrCalendarNotFound, got %v", err)
	}
}

// TestInviteLinkIsSingleUse. The link is a bearer credential; a reusable one
// forwarded in a chat is an open door.
func TestInviteLinkIsSingleUse(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "По ссылке", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inv, err := CreateInviteLink(ctx, ownerID, c.ID, RoleEditor)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if inv.Token == "" {
		t.Fatal("empty token")
	}
	// Two hours, and the assertion has slack for clock and round-trip only.
	if d := time.Until(inv.ExpiresAt); d > InviteTTL+time.Minute || d < InviteTTL-time.Minute {
		t.Errorf("link must live %v, expires in %v", InviteTTL, d)
	}

	firstID, _ := seedPerson(t, "first")
	got, err := AcceptInviteLink(ctx, firstID, inv.Token)
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if got.ID != c.ID || got.Status != StatusActive {
		t.Errorf("accepting a link must grant ACTIVE membership, got %s/%s", got.ID, got.Status)
	}

	secondID, _ := seedPerson(t, "second")
	if _, err := AcceptInviteLink(ctx, secondID, inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("second use: want ErrInviteInvalid, got %v", err)
	}
	// And the second person got nothing.
	ids, err := CalendarIDsFor(ctx, secondID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	if contains(ids, c.ID) {
		t.Fatal("a spent link still granted access")
	}
}

// TestExpiredInviteLinkIsRefused. Backdated in SQL rather than by waiting two
// hours — the expiry is a column comparison, and the clock is not the thing
// under test.
func TestExpiredInviteLinkIsRefused(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Просроченная", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inv, err := CreateInviteLink(ctx, ownerID, c.ID, RoleEditor)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := db.Pool.Exec(ctx,
		`UPDATE calendar_invite SET expires_at = NOW() - interval '1 minute' WHERE token = $1`,
		inv.Token); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	lateID, _ := seedPerson(t, "late")
	if _, err := AcceptInviteLink(ctx, lateID, inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("expired link: want ErrInviteInvalid, got %v", err)
	}
	ids, err := CalendarIDsFor(ctx, lateID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	if contains(ids, c.ID) {
		t.Fatal("an expired link still granted access")
	}
}

// TestInvitingAnActiveMemberDoesNotDemoteThem is the reason the upsert says DO
// NOTHING. DO UPDATE here would take an active member back to 'invited' — from
// a button labelled "invite" — and cut off their access without a word.
func TestInvitingAnActiveMemberDoesNotDemoteThem(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Уже вместе", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	memberID, memberEmail := seedPerson(t, "member")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, memberEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := RespondToInvitation(ctx, memberID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}

	if _, err := InviteByEmail(ctx, ownerID, c.ID, memberEmail, RoleViewer); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("re-invite: want ErrAlreadyMember, got %v", err)
	}

	writable, err := WritableIDsFor(ctx, memberID)
	if err != nil {
		t.Fatalf("writable: %v", err)
	}
	if !contains(writable, c.ID) {
		t.Fatal("re-inviting an active editor demoted them — access lost")
	}
}

// TestDeclineRemovesTheInvitation, and does not leave a row that later reads as
// membership.
func TestDeclineRemovesTheInvitation(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Отказ", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inviteeID, inviteeEmail := seedPerson(t, "decliner")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, inviteeEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := RespondToInvitation(ctx, inviteeID, c.ID, false); err != nil {
		t.Fatalf("decline: %v", err)
	}
	// Declining twice is not a silent success — there is nothing left to decline.
	if err := RespondToInvitation(ctx, inviteeID, c.ID, false); !errors.Is(err, ErrNoInvitation) {
		t.Fatalf("second decline: want ErrNoInvitation, got %v", err)
	}

	list, err := ListFor(ctx, inviteeID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, x := range list {
		if x.ID == c.ID {
			t.Fatal("a declined calendar is still listed")
		}
	}
}

// TestAcceptingWithoutAnInvitationIsRefused: the status='invited' condition in
// RespondToInvitation is authorisation, not a filter.
func TestAcceptingWithoutAnInvitationIsRefused(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Не приглашали", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	strangerID, _ := seedPerson(t, "uninvited")

	if err := RespondToInvitation(ctx, strangerID, c.ID, true); !errors.Is(err, ErrNoInvitation) {
		t.Fatalf("accepting nothing: want ErrNoInvitation, got %v", err)
	}
	ids, err := CalendarIDsFor(ctx, strangerID)
	if err != nil {
		t.Fatalf("ids: %v", err)
	}
	if contains(ids, c.ID) {
		t.Fatal("accepting a non-existent invitation granted access")
	}
}

// TestRemoveMemberAndLeave covers both directions plus the owner guard.
func TestRemoveMemberAndLeave(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Состав", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	aID, aEmail := seedPerson(t, "a")
	bID, bEmail := seedPerson(t, "b")
	for _, u := range []struct{ id, email string }{{aID, aEmail}, {bID, bEmail}} {
		if _, err := InviteByEmail(ctx, ownerID, c.ID, u.email, RoleEditor); err != nil {
			t.Fatalf("invite: %v", err)
		}
		if err := RespondToInvitation(ctx, u.id, c.ID, true); err != nil {
			t.Fatalf("accept: %v", err)
		}
	}

	// Owner removes A.
	if err := RemoveMember(ctx, ownerID, c.ID, aID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	ids, _ := CalendarIDsFor(ctx, aID)
	if contains(ids, c.ID) {
		t.Fatal("a removed member still has access")
	}

	// B removes themselves.
	if err := RemoveMember(ctx, bID, c.ID, bID); err != nil {
		t.Fatalf("leave: %v", err)
	}
	ids, _ = CalendarIDsFor(ctx, bID)
	if contains(ids, c.ID) {
		t.Fatal("a member who left still has access")
	}

	// The owner cannot leave or be removed: a calendar with no owner can never
	// be renamed, shared or deleted again.
	if err := RemoveMember(ctx, ownerID, c.ID, ownerID); !errors.Is(err, ErrCannotRemoveOwner) {
		t.Fatalf("owner leaving: want ErrCannotRemoveOwner, got %v", err)
	}
}

// TestNonOwnerCannotRemoveSomeoneElse — the authorisation on the other path.
func TestNonOwnerCannotRemoveSomeoneElse(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Кто кого", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	aID, aEmail := seedPerson(t, "a")
	bID, bEmail := seedPerson(t, "b")
	for _, u := range []struct{ id, email string }{{aID, aEmail}, {bID, bEmail}} {
		if _, err := InviteByEmail(ctx, ownerID, c.ID, u.email, RoleEditor); err != nil {
			t.Fatalf("invite: %v", err)
		}
		if err := RespondToInvitation(ctx, u.id, c.ID, true); err != nil {
			t.Fatalf("accept: %v", err)
		}
	}

	if err := RemoveMember(ctx, aID, c.ID, bID); !errors.Is(err, ErrNotCalendarOwner) {
		t.Fatalf("editor removing another: want ErrNotCalendarOwner, got %v", err)
	}
	ids, _ := CalendarIDsFor(ctx, bID)
	if !contains(ids, c.ID) {
		t.Fatal("the removal was refused but happened anyway")
	}
}

// TestPendingInviteeCannotReadTheRoster: an invitation is an offer. Until it is
// accepted the invitee learns nothing about who else is in there.
func TestPendingInviteeCannotReadTheRoster(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Состав скрыт", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inviteeID, inviteeEmail := seedPerson(t, "pending")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, inviteeEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}

	if _, err := ListMembers(ctx, inviteeID, c.ID); !errors.Is(err, ErrCalendarNotFound) {
		t.Fatalf("pending invitee reading roster: want ErrCalendarNotFound, got %v", err)
	}

	// Positive control: once accepted, they can.
	if err := RespondToInvitation(ctx, inviteeID, c.ID, true); err != nil {
		t.Fatalf("accept: %v", err)
	}
	members, err := ListMembers(ctx, inviteeID, c.ID)
	if err != nil {
		t.Fatalf("roster after accept: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("want owner + member, got %d", len(members))
	}
}

// TestInviteRejectsBadRole — owner is not grantable, nor is anything invented.
func TestInviteRejectsBadRole(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Роли", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, email := seedPerson(t, "role")

	for _, role := range []string{RoleOwner, "admin", "", "EDITOR"} {
		if _, err := InviteByEmail(ctx, ownerID, c.ID, email, role); !errors.Is(err, ErrInvalidRole) {
			t.Errorf("role %q: want ErrInvalidRole, got %v", role, err)
		}
	}
}

// TestInviteUnknownEmailIsRefused — v1 does not create shell accounts.
func TestInviteUnknownEmailIsRefused(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Нет такого", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := InviteByEmail(ctx, ownerID, c.ID, "nobody-here-at-all@example.com", RoleEditor); !errors.Is(err, ErrInviteeNotFound) {
		t.Fatalf("unknown email: want ErrInviteeNotFound, got %v", err)
	}
}

// TestInviteEmailIsCaseInsensitive — an address is not case sensitive in
// practice, and someone who registered with a capital would be unreachable.
func TestInviteEmailIsCaseInsensitive(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Регистр", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, email := seedPerson(t, "CaseTest")

	upper := ""
	for _, r := range email {
		if r >= 'a' && r <= 'z' {
			upper += string(r - 32)
		} else {
			upper += string(r)
		}
	}
	if upper == email {
		t.Fatal("the seeded address has no letters to upper-case; the test would prove nothing")
	}

	if _, err := InviteByEmail(ctx, ownerID, c.ID, upper, RoleEditor); err != nil {
		t.Fatalf("invite with a differently-cased address: %v", err)
	}
}

// TestInviteNotifiesOnlyTelegramUsers.
//
// Denis asked for the invitation to reach Telegram (17.08). The delivery
// pipeline addresses rows in `reminder`, so inviting enqueues one — but only
// when the invitee actually has a tg_id.
//
// 🔴 The second half is the one that matters. PendingHandler filters out rows
// whose user has no tg_id AT CLAIM TIME, so a row written for someone without
// Telegram is never delivered and never cleaned up: the queue fills with
// messages for people who cannot receive them. That is the state Lizok is in
// today — an account with an email and no Telegram.
func TestInviteNotifiesOnlyTelegramUsers(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	c, err := Create(ctx, ownerID, "Уведомить", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Someone without Telegram: no row.
	silentID, silentEmail := seedPerson(t, "no-telegram")
	if _, err := InviteByEmail(ctx, ownerID, c.ID, silentEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if n := countInviteRows(t, silentID, c.ID); n != 0 {
		t.Errorf("queued %d notifications for a user with no tg_id — they can never be delivered", n)
	}

	// Someone with Telegram: exactly one row, carrying the calendar.
	chattyID, chattyEmail := seedPerson(t, "telegram")
	if _, err := db.Pool.Exec(ctx,
		`UPDATE "user" SET tg_id = $2 WHERE id = $1`, chattyID, time.Now().UnixNano()%1_000_000_000); err != nil {
		t.Fatalf("set tg_id: %v", err)
	}
	if _, err := InviteByEmail(ctx, ownerID, c.ID, chattyEmail, RoleEditor); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if n := countInviteRows(t, chattyID, c.ID); n != 1 {
		t.Fatalf("want exactly one queued notification, got %d", n)
	}

	// And it says who and which calendar — a notification that reads "you were
	// invited" and nothing else makes the person open the app to find out what
	// they are answering, which is the chore this was meant to remove.
	var message string
	if err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(message, '') FROM reminder
		  WHERE user_id = $1 AND calendar_id = $2 AND source_kind = 'INVITE'`,
		chattyID, c.ID).Scan(&message); err != nil {
		t.Fatalf("read message: %v", err)
	}
	if !strings.Contains(message, "Уведомить") {
		t.Errorf("the notification does not name the calendar: %q", message)
	}
}

// TestTwoInvitationsForOnePersonBothSurvive is the reason migration 000015
// rebuilt idx_reminder_dedupe.
//
// An INVITE row has NULL for event_id, task_id, occurrence_start AND
// minutes_before. Under the old key — which does not include calendar_id and
// is NULLS NOT DISTINCT — two invitations to two different calendars for the
// same person collide, and the second one is silently swallowed by
// ON CONFLICT DO NOTHING. Nothing errors; the message simply never arrives.
func TestTwoInvitationsForOnePersonBothSurvive(t *testing.T) {
	ownerID, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	first, err := Create(ctx, ownerID, "Первый общий", nil)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := Create(ctx, ownerID, "Второй общий", nil)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	inviteeID, inviteeEmail := seedPerson(t, "two-invites")
	if _, err := db.Pool.Exec(ctx,
		`UPDATE "user" SET tg_id = $2 WHERE id = $1`, inviteeID, time.Now().UnixNano()%1_000_000_000+1); err != nil {
		t.Fatalf("set tg_id: %v", err)
	}

	for _, cal := range []Calendar{first, second} {
		if _, err := InviteByEmail(ctx, ownerID, cal.ID, inviteeEmail, RoleEditor); err != nil {
			t.Fatalf("invite to %s: %v", cal.Name, err)
		}
	}

	for _, cal := range []Calendar{first, second} {
		if n := countInviteRows(t, inviteeID, cal.ID); n != 1 {
			t.Errorf("calendar %q: want one notification, got %d", cal.Name, n)
		}
	}
}

func countInviteRows(t *testing.T, userID, calendarID string) int {
	t.Helper()
	var n int
	if err := db.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM reminder
		  WHERE user_id = $1 AND calendar_id = $2 AND source_kind = 'INVITE'`,
		userID, calendarID).Scan(&n); err != nil {
		t.Fatalf("count invite rows: %v", err)
	}
	return n
}
