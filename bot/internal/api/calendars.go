package api

import "strings"

// CalendarDetail is a calendar as /api/calendars actually returns it.
//
// 🔴 The names here are copied from api-go/internal/calendars/types.go:21, not
// invented. An unknown JSON key decodes to the zero value in Go and reports no
// error at all, so a misspelt tag would make every shared calendar look
// personal, forever, with nothing in any log.
//
// `role` and `status` come from the caller's own calendar_member row, not from
// the calendar — two people looking at the same calendar see different values.
type CalendarDetail struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Color  *string `json:"color"`
	Kind   string  `json:"kind"`
	Role   string  `json:"role"`
	Status string  `json:"status"`
}

// IsPersonal reports the one calendar that can be neither deleted nor left.
func (c CalendarDetail) IsPersonal() bool { return c.Kind == "personal" }

// IsOwner reports whether the caller may rename, recolour, invite and delete.
func (c CalendarDetail) IsOwner() bool { return c.Role == "owner" }

// ColourOrEmpty flattens the nullable colour. The personal calendar is created
// colourless by design, so null is a normal value and not a missing one.
func (c CalendarDetail) ColourOrEmpty() string {
	if c.Color == nil {
		return ""
	}
	return *c.Color
}

// Member is one person on a calendar.
//
// Email and DisplayName are both nullable: a Telegram-only account has no
// email at all, and that person is precisely who the invite link exists for.
type Member struct {
	UserID      string  `json:"user_id"`
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
}

// Label is what to print for a member: a name, else an email, else something
// that is not "<nil>".
func (m Member) Label() string {
	if m.DisplayName != nil && strings.TrimSpace(*m.DisplayName) != "" {
		return *m.DisplayName
	}
	if m.Email != nil && strings.TrimSpace(*m.Email) != "" {
		return *m.Email
	}
	return "—"
}

// CalendarsFull lists every calendar the caller can see, with their role.
func (c *Client) CalendarsFull(token string) ([]CalendarDetail, error) {
	var resp struct {
		Data []CalendarDetail `json:"data"`
	}
	if err := c.get("/api/calendars", token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CreateCalendar makes a new shared calendar.
func (c *Client) CreateCalendar(token, name, colour string) (*CalendarDetail, error) {
	body := map[string]any{"name": name}
	if colour != "" {
		body["color"] = colour
	}
	var resp struct {
		Data CalendarDetail `json:"data"`
	}
	if err := c.post("/api/calendars", token, body, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateCalendar sends ONLY the fields that changed.
//
// 🔴 Both are pointers because the API's updateRequest holds *string for both
// (api-go/internal/calendars/handlers.go:20): an absent key means "leave it",
// a present empty one means "make it empty". A struct with plain strings would
// blank the name on every colour change and vice versa.
func (c *Client) UpdateCalendar(token, id string, name, colour *string) error {
	body := map[string]any{}
	if name != nil {
		body["name"] = *name
	}
	if colour != nil {
		body["color"] = *colour
	}
	return c.patch("/api/calendars/"+id, token, body, nil)
}

// DeleteCalendar removes a calendar the caller owns.
//
// The API refuses a non-empty one with 409 and the counts inside
// (calendars.NotEmptyError), and refuses the personal one outright — both
// arrive here as a plain error, so the caller must say so in words rather than
// claim success.
func (c *Client) DeleteCalendar(token, id string) error {
	return c.del("/api/calendars/"+id, token)
}

// CalendarMembers lists everyone on a calendar, owner first.
func (c *Client) CalendarMembers(token, id string) ([]Member, error) {
	var resp struct {
		Data []Member `json:"data"`
	}
	if err := c.get("/api/calendars/"+id+"/members", token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// InviteByEmail invites someone who already has an email on their account.
//
// 🔴 This path cannot reach a Telegram-only user: their `email` is NULL, and
// the handler rejects an empty one with 400 (members_handlers.go:55). For them
// the answer is CalendarInviteLink.
func (c *Client) InviteByEmail(token, id, email, role string) error {
	return c.post("/api/calendars/"+id+"/invites", token,
		map[string]any{"email": email, "role": role}, nil)
}

// CalendarInviteLink mints a link token that grants membership when redeemed.
func (c *Client) CalendarInviteLink(token, id, role string) (string, error) {
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := c.post("/api/calendars/"+id+"/invite-links", token,
		map[string]any{"role": role}, &resp); err != nil {
		return "", err
	}
	return resp.Data.Token, nil
}

// AcceptInviteLink redeems a link token for the calling user.
func (c *Client) AcceptInviteLink(token, linkToken string) (*CalendarDetail, error) {
	var resp struct {
		Data CalendarDetail `json:"data"`
	}
	if err := c.post("/api/calendars/invite-links/accept", token,
		map[string]any{"token": linkToken}, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// LeaveCalendar removes a member; passing the caller's own id is how leaving
// works.
//
// RemoveMember (api-go/internal/calendars/members.go:284) opens with
// `leaving := actorID == targetID` and only demands ownership for anyone else —
// and refuses the owner either way, because a calendar with no owner can never
// be renamed, shared or deleted again.
func (c *Client) LeaveCalendar(token, id, userID string) error {
	return c.del("/api/calendars/"+id+"/members/"+userID, token)
}

// MyUserID returns the caller's own user id.
//
// Leaving a calendar is a DELETE on /members/{userId} with your OWN id — the
// API distinguishes leaving from removing by comparing the two — so the bot
// has to know who it is. Nothing else in the bot needed this until now.
func (c *Client) MyUserID(token string) (string, error) {
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.get("/api/auth/me", token, nil, &resp); err != nil {
		return "", err
	}
	return resp.Data.ID, nil
}
