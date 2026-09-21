package api

// Task ↔ event, the two doors api-go opened on 21.09 (plan A1).

// ConvertReq asks POST /api/tasks/{id}/convert to put a task into the calendar.
type ConvertReq struct {
	Mode     string `json:"mode"`             // "move" | "link"
	Repeat   string `json:"repeat,omitempty"` // "series" | "once"; required by the API for a repeating task
	StartsAt string `json:"starts_at"`        // RFC3339
	EndsAt   string `json:"ends_at"`
	AllDay   bool   `json:"all_day,omitempty"`
}

// ConvertTask moves or links a task into the calendar.
func (c *Client) ConvertTask(token, taskID string, req ConvertReq) (*Event, error) {
	var resp struct {
		Data Event `json:"data"`
	}
	if err := c.post("/api/tasks/"+taskID+"/convert", token, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ToTaskReq asks POST /api/events/{id}/to-task.
type ToTaskReq struct {
	Mode       string `json:"mode"`
	Repeat     string `json:"repeat,omitempty"`
	Occurrence string `json:"occurrence,omitempty"`
	DryRun     bool   `json:"dry_run,omitempty"`
}

// PlannedTask is the task an event becomes; ID is empty on a dry run.
type PlannedTask struct {
	ID               string   `json:"id,omitempty"`
	Title            string   `json:"title"`
	Description      *string  `json:"description,omitempty"`
	Tags             []string `json:"tags"`
	DueDate          string   `json:"due_date"`
	EstimatedMinutes *int     `json:"estimated_minutes,omitempty"`
	Rrule            *string  `json:"rrule,omitempty"`
}

// ToTaskResult is the answer; Lost holds api-go's codes (start_time, color,
// location, reminders), which the bot words.
type ToTaskResult struct {
	Task    PlannedTask `json:"task"`
	Lost    []string    `json:"lost"`
	EventID *string     `json:"event_id"`
	DryRun  bool        `json:"dry_run"`
}

// EventToTask turns an event — the whole series, or the occurrence the id
// names — into a task. With DryRun nothing is written: the answer is the card.
//
// The id may be synthetic («uuid:YYYY-MM-DD»); api-go reads the day from it.
func (c *Client) EventToTask(token, eventID string, req ToTaskReq) (*ToTaskResult, error) {
	var resp struct {
		Data ToTaskResult `json:"data"`
	}
	if err := c.post("/api/events/"+eventID+"/to-task", token, req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
