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

// TelegramLogin exchanges a signed Login Widget payload for a JWT.
//
// The payload is built and signed by internal/auth; this only carries it. The
// endpoint is the same one the web widget posts to, so the bot gets an ordinary
// user session rather than a privileged one — and lands on the account the
// tg_id is already attached to, instead of creating a parallel one.
func (c *Client) TelegramLogin(payload any) (string, int64, error) {
	var resp struct {
		Data struct {
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"data"`
	}
	if err := c.post("/api/auth/telegram", "", payload, &resp); err != nil {
		return "", 0, err
	}
	if resp.Data.Token == "" {
		return "", 0, fmt.Errorf("login succeeded but returned no token")
	}
	return resp.Data.Token, resp.Data.ExpiresAt, nil
}

func (c *Client) GetEvents(token string, start, end string) ([]Event, error) {
	var resp struct {
		Data []Event `json:"data"`
	}
	params := url.Values{"start": {start}, "end": {end}}
	err := c.get("/api/events", token, params, &resp)
	return resp.Data, err
}

func (c *Client) CreateEvent(token string, req CreateEventReq) (*Event, error) {
	var resp struct {
		Data Event `json:"data"`
	}
	err := c.post("/api/events", token, req, &resp)
	return &resp.Data, err
}

func (c *Client) GetTasks(token string, status string) ([]Task, error) {
	var resp struct {
		Data []Task `json:"data"`
	}
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	err := c.get("/api/tasks", token, params, &resp)
	return resp.Data, err
}

func (c *Client) CreateTask(token string, req CreateTaskReq) (*Task, error) {
	var resp struct {
		Data Task `json:"data"`
	}
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

// PendingNotification mirrors the API's /api/svc payload.
type PendingNotification struct {
	ID         string `json:"id"`
	TgID       int64  `json:"tg_id"`
	Text       string `json:"text"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
}

// PendingNotifications claims the due notifications.
//
// Unlike every other call on this client it authenticates with a service
// token rather than a user JWT — it deliberately reads across all users, which
// is why the endpoint it hits is rate-limited and constant-time guarded on the
// API side.
func (c *Client) PendingNotifications(serviceToken string) ([]PendingNotification, error) {
	req, err := http.NewRequest("GET", c.base+"/api/svc/notifications/pending", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Service-Token", serviceToken)

	var resp struct {
		Data []PendingNotification `json:"data"`
	}
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// AckNotification records whether delivery succeeded.
func (c *Client) AckNotification(serviceToken, id string, delivered bool, sendErr string) error {
	body, err := json.Marshal(map[string]any{"delivered": delivered, "error": sendErr})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+"/api/svc/notifications/"+id+"/ack", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}

// NotificationAction forwards a button press from a notification.
//
// The service token authenticates the bot; tg_id tells the API which person
// pressed it, and the API checks that the reminder belongs to them. No user JWT
// is involved — the presser exists to the bot only as a Telegram id.
func (c *Client) NotificationAction(serviceToken string, tgID int64, reminderID, action string, minutes int) error {
	payload := map[string]any{"tg_id": tgID, "reminder_id": reminderID, "action": action}
	if minutes > 0 {
		payload["minutes"] = minutes
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+"/api/svc/notifications/action", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}
