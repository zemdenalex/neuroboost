package api

// Editing an event, which the bot could not do until 16.09.
//
// Denis: «нет возможности редактировать событие, задачи можно выбрать и
// редактировать, а события нет». Creation had a confirmation card with nine
// editable fields, and none of it was reachable once the event existed.

// UpdateEventReq mirrors api-go's UpdateEventRequest.
//
// 🔴 Every field is a pointer and omitempty, and that is the whole contract of
// a PATCH: an ABSENT field is left alone, a present one is written. A plain
// string would send "" for every field the user did not touch and blank the
// event — which is how a partial update quietly becomes a destructive one.
type UpdateEventReq struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	StartsAt    *string `json:"starts_at,omitempty"`
	EndsAt      *string `json:"ends_at,omitempty"`
	AllDay      *bool   `json:"all_day,omitempty"`
	Rrule       *string `json:"rrule,omitempty"`
	Colour      *string `json:"color,omitempty"`
	CalendarID  *string `json:"calendar_id,omitempty"`

	Tags            []string `json:"tags,omitempty"`
	ReminderOffsets *[]int   `json:"reminder_offsets,omitempty"`
}

func (c *Client) GetEvent(token, id string) (*Event, error) {
	var resp struct {
		Data Event `json:"data"`
	}
	err := c.get("/api/events/"+id, token, nil, &resp)
	return &resp.Data, err
}

// UpdateEvent writes the changed fields.
//
// ⚠ No `scope` parameter is sent, so a repeating event is changed as a WHOLE
// SERIES. That is the API's default and the only mode it accepts for a calendar
// change (it refuses scope=occurrence outright rather than guessing). The card
// says so before it saves; offering "this one only" from a bot that cannot then
// honour it for every field would be a button that lies.
func (c *Client) UpdateEvent(token, id string, req UpdateEventReq) error {
	return c.patch("/api/events/"+id, token, req, nil)
}

func (c *Client) DeleteEvent(token, id string) error {
	return c.del("/api/events/"+id, token)
}
