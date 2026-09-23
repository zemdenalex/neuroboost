package api

import "net/url"

// «Задачи дня» (spec 2026-09-22): the bot's side of the D1 endpoints. Every
// day is a date in the user's zone, YYYY-MM-DD; the server computes done and
// level, the bot only draws them (spec §3).

// DayItem is one task promised for a day.
type DayItem struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
}

// Day is one day of the set: its target, whether it was taken, what is in it
// and how it went. Level is 0…5 — ⬛🟫🟥🟧🟨🟩.
type Day struct {
	Day       string    `json:"day"`
	Target    int       `json:"target"`
	Confirmed bool      `json:"confirmed"`
	Items     []DayItem `json:"items"`
	Done      int       `json:"done"`
	Level     int       `json:"level"`
}

// DayTasks reads every day of [from, to] (the server allows ≤ 62 days).
func (c *Client) DayTasks(token, from, to string) ([]Day, error) {
	var resp struct {
		Data []Day `json:"data"`
	}
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	err := c.get("/api/day-tasks", token, params, &resp)
	return resp.Data, err
}

// DayProposal is what the server would offer for the day. It writes nothing.
func (c *Client) DayProposal(token, day string) ([]DayItem, error) {
	var resp struct {
		Data []DayItem `json:"data"`
	}
	params := url.Values{}
	params.Set("day", day)
	err := c.get("/api/day-tasks/proposal", token, params, &resp)
	return resp.Data, err
}

// ConfirmDay takes the day with these tasks and answers with the day as it now
// is.
func (c *Client) ConfirmDay(token, day string, taskIDs []string) (Day, error) {
	if taskIDs == nil {
		taskIDs = []string{} // «take the day with nothing added» sends [], not null
	}
	var resp struct {
		Data Day `json:"data"`
	}
	err := c.post("/api/day-tasks/confirm", token, map[string]any{"day": day, "task_ids": taskIDs}, &resp)
	return resp.Data, err
}

// AddDayTask promises one task for a day.
func (c *Client) AddDayTask(token, day, taskID string) (Day, error) {
	var resp struct {
		Data Day `json:"data"`
	}
	err := c.post("/api/day-tasks", token, map[string]any{"day": day, "task_id": taskID}, &resp)
	return resp.Data, err
}

// RemoveDayTask takes a task out of a day. After 12:00 today, or on a past
// day, the server refuses with TOO_LATE.
func (c *Client) RemoveDayTask(token, day, taskID string) error {
	return c.del("/api/day-tasks/"+url.PathEscape(day)+"/"+url.PathEscape(taskID), token)
}
