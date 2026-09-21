package api

import "net/url"

// What statistics reads beyond events and tasks (spec 22.09 §5, §2).

// TaskOccurrence is one answered day of a repeating task.
type TaskOccurrence struct {
	TaskID     string `json:"task_id"`
	Occurrence string `json:"occurrence"`
	State      string `json:"state"`
}

// TaskOccurrences lists answered days in [from, to], dates YYYY-MM-DD. The API
// refuses more than 366 days; «Всё» asks year by year.
func (c *Client) TaskOccurrences(token, from, to string) ([]TaskOccurrence, error) {
	var resp struct {
		Data []TaskOccurrence `json:"data"`
	}
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	err := c.get("/api/tasks/occurrences", token, params, &resp)
	return resp.Data, err
}

// Reflection is what statistics needs of one: the day it was written.
type Reflection struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// Reflections lists every reflection of the caller.
func (c *Client) Reflections(token string) ([]Reflection, error) {
	var resp struct {
		Data []Reflection `json:"data"`
	}
	err := c.get("/api/reflections", token, nil, &resp)
	return resp.Data, err
}
