package calendars

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sharing a calendar with another person.
//
// Two channels, both asked for by Denis on 17.08.2026:
//
//   - by email, for someone who already has an account. Creates the
//     calendar_member row straight away with status='invited'; the invitee sees
//     it in their own calendar list (ListFor deliberately does not filter by
//     status) and accepts in the app. No mail is sent — and none could be,
//     there is no mail sending anywhere in this codebase.
//   - by link, for someone who may not have an account yet. A token with a
//     two-hour life and a single use.
//
// 🔴 The access rule itself needed no changes. CalendarIDsFor has filtered on
// status='active' since slice 1, so an invited member sees nothing until they
// accept, and that is covered by tests written before this file existed. What
// is new here is only how a row gets created.

var (
	// ErrCalendarIsPersonalShare refuses to share the personal calendar.
	//
	// 🔴 This is the most dangerous operation in the file. A personal calendar
	// holds everything a user has ever written that was not deliberately put
	// somewhere else, and sharing it is one click away from sharing a life. If
	// you want to share, make a shared calendar — that is what they are for.
	ErrCalendarIsPersonalShare = errors.New("personal calendar cannot be shared")

	// ErrInviteeNotFound means no account with that email.
	//
	// The spec's v1 rule, kept: inviting someone who has not registered is
	// refused with a clear message rather than silently creating a shell
	// account. The invite LINK is the answer for that person.
	ErrInviteeNotFound = errors.New("no user with that email")

	// ErrAlreadyMember is a success-shaped failure: the person is already in.
	ErrAlreadyMember = errors.New("already a member of this calendar")

	// ErrInvalidRole rejects anything but editor/viewer.
	//
	// owner is excluded on purpose and not by omission: a calendar has exactly
	// one owner, its creator, and handing that over is a different operation
	// with different consequences (the owner can delete the calendar).
	ErrInvalidRole = errors.New("role must be editor or viewer")

	// ErrInviteInvalid covers unknown, expired and already-used tokens with one
	// error, deliberately. Telling the holder of a bad link which of the three
	// it is tells them whether the token ever existed.
	ErrInviteInvalid = errors.New("invitation link is not valid")

	// ErrNoInvitation is returned when accepting or declining something that is
	// not actually a pending invitation for this user.
	ErrNoInvitation = errors.New("no pending invitation for this calendar")

	// ErrCannotRemoveOwner guards the roster. Removing the owner would leave a
	// calendar nobody can rename, share or delete.
	ErrCannotRemoveOwner = errors.New("the owner cannot be removed")
)

// InviteTTL is how long an invite link lives. Denis's number, 17.08.
//
// 🔴 Short because the link IS the credential: anyone holding it becomes the
// invitee, and we cannot know who opened it. Two hours is long enough to paste
// into a chat and short enough that a forwarded message is usually already dead.
const InviteTTL = 2 * time.Hour

// Member is one person on a calendar, as the roster shows them.
type Member struct {
	UserID      string  `json:"user_id"`
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
}

// Invite is a link that grants membership when opened.
type Invite struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// validShareRole reports whether a role may be granted to someone else.
func validShareRole(role string) bool {
	return role == RoleEditor || role == RoleViewer
}

// requireShareableOwner is the gate every write in this file goes through:
// the caller must own the calendar, and the calendar must not be personal.
func requireShareableOwner(ctx context.Context, userID, calendarID string) error {
	kind, err := requireOwner(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if kind == KindPersonal {
		return ErrCalendarIsPersonalShare
	}
	return nil
}

// ListMembers returns everyone on a calendar, including invitees.
//
// Readable by any ACTIVE member, not only the owner: knowing who else can see
// your events is part of knowing what you have shared. An invited-but-not-yet-
// accepted member is deliberately NOT allowed to read the roster — they have
// not agreed to be here yet, and until they do they learn nothing about it.
func ListMembers(ctx context.Context, userID, calendarID string) ([]Member, error) {
	// membership() already filters on status='active', so a pending invitee
	// gets ErrNoRows here and is told the calendar does not exist — which is
	// the intended answer: they have not agreed to be here yet, and until they
	// do they learn nothing about who else is.
	if _, _, err := membership(ctx, userID, calendarID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCalendarNotFound
		}
		return nil, err
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT u.id::text, u.email, u.display_name, cm.role, cm.status
		   FROM calendar_member cm
		   JOIN "user" u ON u.id = cm.user_id
		  WHERE cm.calendar_id = $1
		  ORDER BY (cm.role = 'owner') DESC, cm.created_at`,
		calendarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Member{}
	for rows.Next() {
		var mem Member
		if err := rows.Scan(&mem.UserID, &mem.Email, &mem.DisplayName, &mem.Role, &mem.Status); err != nil {
			return nil, err
		}
		out = append(out, mem)
	}
	return out, rows.Err()
}

// InviteByEmail adds someone who already has an account, as status='invited'.
func InviteByEmail(ctx context.Context, ownerID, calendarID, email, role string) (Member, error) {
	if !validShareRole(role) {
		return Member{}, ErrInvalidRole
	}
	if err := requireShareableOwner(ctx, ownerID, calendarID); err != nil {
		return Member{}, err
	}

	// Case-insensitive, because an address is not case sensitive in practice
	// and a user who typed theirs with a capital would otherwise be
	// unreachable. email is UNIQUE in the schema, so at most one row matches.
	var inviteeID string
	err := db.Pool.QueryRow(ctx,
		`SELECT id::text FROM "user" WHERE lower(email) = lower($1)`,
		strings.TrimSpace(email)).Scan(&inviteeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrInviteeNotFound
	}
	if err != nil {
		return Member{}, err
	}

	// Inviting yourself is not an error worth a special path — you are already
	// the owner, so the "already a member" answer is the true one.
	tag, err := db.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (calendar_id, user_id) DO NOTHING`,
		calendarID, inviteeID, role, StatusInvited)
	if err != nil {
		return Member{}, err
	}
	// 🔴 DO NOTHING, not DO UPDATE — the opposite of PersonalIDFor's repair.
	// There, the row is the owner's own and only one state is legal. Here, an
	// existing row may be an ACTIVE membership, and overwriting it would demote
	// an active member back to "invited" and cut off their access, from a
	// button labelled "invite".
	if tag.RowsAffected() == 0 {
		return Member{}, ErrAlreadyMember
	}

	var mem Member
	if err := db.Pool.QueryRow(ctx,
		`SELECT u.id::text, u.email, u.display_name, cm.role, cm.status
		   FROM calendar_member cm JOIN "user" u ON u.id = cm.user_id
		  WHERE cm.calendar_id = $1 AND cm.user_id = $2`,
		calendarID, inviteeID,
	).Scan(&mem.UserID, &mem.Email, &mem.DisplayName, &mem.Role, &mem.Status); err != nil {
		return Member{}, err
	}

	// Tell them in Telegram, if we can. Not fatal: the invitation exists either
	// way and is visible in the app, so a failure to enqueue a message must not
	// undo it. Logged by the caller's error path only when it is fatal — here
	// the honest outcome is "invited, possibly unnoticed".
	if err := enqueueInviteNotification(ctx, ownerID, inviteeID, calendarID); err != nil {
		return mem, nil
	}
	return mem, nil
}

// enqueueInviteNotification puts an invitation into the P2 delivery pipeline.
//
// 🔴 It writes to `reminder` with raw SQL instead of calling the reminders
// package, and that is deliberate: reminders imports calendars (for the write
// scoping on the "done" button), so calendars importing reminders would be an
// import cycle. A single INSERT is a much smaller price than inverting that
// dependency, and the notification pipeline is a queue — writing a row into it
// is its public interface as much as any function would be.
//
// 🔴 Only when the invitee actually has a tg_id. PendingHandler filters those
// rows out at claim time, so without this check the row would sit PENDING for
// ever, never delivered and never cleaned up — a queue that quietly fills with
// messages for people who cannot receive them. Lizok is exactly that case
// today: an account with an email and no Telegram.
func enqueueInviteNotification(ctx context.Context, inviterID, inviteeID, calendarID string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO reminder (user_id, source_kind, calendar_id, remind_at, status, channel, message)
		SELECT $2, 'INVITE', $3, NOW(), 'PENDING', 'TELEGRAM',
		       '📅 ' || COALESCE(inviter.display_name, inviter.email, 'Кто-то') ||
		       ' приглашает вас в календарь «' || c.name || '»'
		  FROM "user" invitee, "user" inviter, calendar c
		 WHERE invitee.id = $2 AND inviter.id = $1 AND c.id = $3
		   AND invitee.tg_id IS NOT NULL
		ON CONFLICT DO NOTHING`,
		inviterID, inviteeID, calendarID)
	return err
}

// RespondToInvitation turns a pending invitation into membership, or removes it.
//
// One function for accept and decline because they share every check and differ
// in one statement — split, the checks drift.
func RespondToInvitation(ctx context.Context, userID, calendarID string, accept bool) error {
	var sql string
	if accept {
		sql = `UPDATE calendar_member SET status = $3
		        WHERE calendar_id = $1 AND user_id = $2 AND status = 'invited'`
	} else {
		sql = `DELETE FROM calendar_member
		        WHERE calendar_id = $1 AND user_id = $2 AND status = 'invited'`
	}

	args := []any{calendarID, userID}
	if accept {
		args = append(args, StatusActive)
	}

	tag, err := db.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	// 🔴 The status='invited' condition is the authorisation, not a filter. An
	// UPDATE without it would let anyone who knows a calendar id promote their
	// own membership — and there is no membership to promote, so it would do
	// nothing; but written as an INSERT it would be a self-service grant. Kept
	// as an UPDATE over an existing invited row for exactly that reason.
	if tag.RowsAffected() == 0 {
		return ErrNoInvitation
	}
	return nil
}

// RemoveMember takes someone off a calendar, or takes yourself off it.
//
// Two callers in one: the owner removing another member, and a member leaving.
// They are the same operation with different authorisation, and keeping them
// together means the owner-protection below cannot be bypassed by using the
// other path.
func RemoveMember(ctx context.Context, actorID, calendarID, targetID string) error {
	leaving := actorID == targetID

	if !leaving {
		if err := requireShareableOwner(ctx, actorID, calendarID); err != nil {
			return err
		}
	}

	var role string
	err := db.Pool.QueryRow(ctx,
		`SELECT role FROM calendar_member WHERE calendar_id = $1 AND user_id = $2`,
		calendarID, targetID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCalendarNotFound
	}
	if err != nil {
		return err
	}
	// Covers the owner leaving as well as the owner being removed: a calendar
	// without an owner can never be renamed, shared or deleted again.
	if role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	if _, err := db.Pool.Exec(ctx,
		`DELETE FROM calendar_member WHERE calendar_id = $1 AND user_id = $2`,
		calendarID, targetID); err != nil {
		return err
	}
	return nil
}

// CreateInviteLink mints a single-use token for a calendar.
func CreateInviteLink(ctx context.Context, ownerID, calendarID, role string) (Invite, error) {
	if !validShareRole(role) {
		return Invite{}, ErrInvalidRole
	}
	if err := requireShareableOwner(ctx, ownerID, calendarID); err != nil {
		return Invite{}, err
	}

	token, err := newToken()
	if err != nil {
		return Invite{}, err
	}

	var inv Invite
	if err := db.Pool.QueryRow(ctx,
		`INSERT INTO calendar_invite (calendar_id, token, role, created_by, expires_at)
		 VALUES ($1, $2, $3, $4, NOW() + $5::interval)
		 RETURNING token, role, expires_at`,
		calendarID, token, role, ownerID, InviteTTL.String(),
	).Scan(&inv.Token, &inv.Role, &inv.ExpiresAt); err != nil {
		return Invite{}, err
	}
	return inv, nil
}

// AcceptInviteLink consumes a token and makes the caller an ACTIVE member.
//
// Active immediately, unlike an email invite: opening the link IS the
// acceptance. Asking someone to confirm twice adds a screen and no safety.
func AcceptInviteLink(ctx context.Context, userID, token string) (Calendar, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return Calendar{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 🔴 One statement claims the token: the UPDATE both checks validity and
	// marks it used, so two people opening the same link at the same moment
	// cannot both succeed. Checking first and updating after would leave exactly
	// that window open, and a link is a thing people forward.
	var calendarID, role string
	err = tx.QueryRow(ctx,
		`UPDATE calendar_invite
		    SET used_at = NOW(), used_by = $2
		  WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING calendar_id::text, role`,
		token, userID).Scan(&calendarID, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Calendar{}, ErrInviteInvalid
	}
	if err != nil {
		return Calendar{}, err
	}

	// DO NOTHING for the same reason as InviteByEmail: an existing row may be
	// the owner's own, and a link must never demote them.
	if _, err := tx.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (calendar_id, user_id) DO NOTHING`,
		calendarID, userID, role, StatusActive); err != nil {
		return Calendar{}, err
	}

	var c Calendar
	if err := tx.QueryRow(ctx,
		`SELECT c.id::text, c.name, c.color, c.kind, cm.role, cm.status, c.created_at
		   FROM calendar c JOIN calendar_member cm ON cm.calendar_id = c.id
		  WHERE c.id = $1 AND cm.user_id = $2`,
		calendarID, userID,
	).Scan(&c.ID, &c.Name, &c.Color, &c.Kind, &c.Role, &c.Status, &c.CreatedAt); err != nil {
		return Calendar{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Calendar{}, err
	}
	return c, nil
}

// newToken returns a URL-safe secret with 256 bits of entropy.
//
// crypto/rand, not math/rand: this value is the only thing standing between a
// stranger and someone's calendar. Errors are returned rather than ignored —
// a token from a failed read would be predictable, and "the link did not work"
// is a far better outcome than a guessable one.
func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
