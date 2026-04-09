package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	base       string
	httpClient *http.Client
}

func NewClient(base string) *Client {
	return &Client{
		base:       base,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Event struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	AllDay      bool   `json:"all_day"`
	Color       string `json:"color"`
}

type CreateEventReq struct {
	Title    string `json:"title"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type Task struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Status           string `json:"status"`
	Priority         int    `json:"priority"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	DueDate          string `json:"due_date"`
	CompletedAt      string `json:"completed_at"`
}

type CreateTaskReq struct {
	Title    string `json:"title"`
	Priority int    `json:"priority"`
	Status   string `json:"status"`
}

type CreateFeedbackReq struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (c *Client) get(path string, token string, params url.Values, result any) error {
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, result)
}

func (c *Client) post(path string, token string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, result)
}

func (c *Client) patch(path string, token string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PATCH", c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, result)
}

func (c *Client) del(path string, token string) error {
	req, err := http.NewRequest("DELETE", c.base+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) GetEvents(token string, start, end string) ([]Event, error) {
	var resp struct{ Data []Event `json:"data"` }
	params := url.Values{"start": {start}, "end": {end}}
	err := c.get("/api/events", token, params, &resp)
	return resp.Data, err
}

func (c *Client) CreateEvent(token string, req CreateEventReq) (*Event, error) {
	var resp struct{ Data Event `json:"data"` }
	err := c.post("/api/events", token, req, &resp)
	return &resp.Data, err
}

func (c *Client) GetTasks(token string, status string) ([]Task, error) {
	var resp struct{ Data []Task `json:"data"` }
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	err := c.get("/api/tasks", token, params, &resp)
	return resp.Data, err
}

func (c *Client) CreateTask(token string, req CreateTaskReq) (*Task, error) {
	var resp struct{ Data Task `json:"data"` }
	err := c.post("/api/tasks", token, req, &resp)
	return &resp.Data, err
}

func (c *Client) UpdateTask(token string, id string, updates map[string]any) error {
	return c.patch("/api/tasks/"+id, token, updates, nil)
}

func (c *Client) DeleteTask(token string, id string) error {
	return c.del("/api/tasks/"+id, token)
}

func (c *Client) ScheduleTask(token string, id string, startTime string, durationMin int) error {
	body := map[string]any{"start_time": startTime, "estimated_minutes": durationMin}
	return c.post("/api/tasks/"+id+"/schedule", token, body, nil)
}

func (c *Client) SubmitFeedback(token string, req CreateFeedbackReq) error {
	return c.post("/api/feedback", token, req, nil)
}
