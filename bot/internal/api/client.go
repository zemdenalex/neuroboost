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

// Event is as much of an event as the bot shows or edits.
//
// 🔴 Rrule, Tags, CalendarID and ReminderOffsets were absent until 16.09, and
// their absence was invisible: the bot only ever CREATED events, so nothing
// read them back. The moment it could edit one, a missing field here would have
// meant loading an event, saving it, and silently dropping its repeat.
type Event struct {
	ID          string `json:"id"`
	CalendarID  string `json:"calendar_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	AllDay      bool   `json:"all_day"`
	Color       string `json:"color"`

	Rrule           *string  `json:"rrule,omitempty"`
	Tags            []string `json:"tags"`
	ReminderOffsets []int    `json:"reminder_offsets"`
	IsShared        bool     `json:"is_shared"`

	// TaskID is the task this event gives time to, if linked. Occurrences of
	// a series carry the parent's — that is how the task list finds «📅 вт
	// 15:00» without a door of its own.
	TaskID *string `json:"task_id,omitempty"`
}

// CreateEventReq mirrors api-go's CreateEventRequest for the fields the bot
// sends.
//
// 🔴 Every optional field is a pointer or omitempty, and ReminderOffsets
// especially. POST /api/events applies the user's default preset when the key
// is ABSENT; an explicit empty array means "stay silent forever", which is the
// exact defect this product hit on 12.08. Those are different events, and a
// plain []int cannot tell them apart.
type CreateEventReq struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	StartsAt    string  `json:"starts_at"`
	EndsAt      string  `json:"ends_at"`
	AllDay      bool    `json:"all_day,omitempty"`
	Rrule       *string `json:"rrule,omitempty"`
	Colour      *string `json:"color,omitempty"`
	CalendarID  *string `json:"calendar_id,omitempty"`
	TaskID      *string `json:"task_id,omitempty"`

	Tags            []string `json:"tags,omitempty"`
	ReminderOffsets *[]int   `json:"reminder_offsets,omitempty"`
}

// Calendar is one of the user's calendars, as much of it as the bot needs to
// resolve a word like «работа» to an id.
type Calendar struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Colour   string `json:"color"`
	IsShared bool   `json:"is_shared"`
}

type Task struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Status           string   `json:"status"`
	Priority         int      `json:"priority"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	DueDate          string   `json:"due_date"`
	CompletedAt      string   `json:"completed_at"`
	Tags             []string `json:"tags"`

	// Rrule is set when the task repeats.
	//
	// 🔴 For a repeating task `Status` describes the SERIES — TODO means "still
	// running" — and what happened TODAY is OccurrenceState. Reading Status
	// alone shows «выпить таблетки» as outstanding all day after it was ticked.
	Rrule string `json:"rrule,omitempty"`
	// OccurrenceState is today's answer: "", "done" or "skipped".
	OccurrenceState string `json:"occurrence_state,omitempty"`
	// NagMinutes is how often an unanswered reminder comes back, 0 for never.
	NagMinutes int `json:"nag_minutes,omitempty"`

	// ActualMinutes is time logged against the task; statistics prefers it to
	// the estimate (spec 22.09 §2).
	ActualMinutes int `json:"actual_minutes,omitempty"`
	// RepeatAnchor is the first day of a series (RFC3339), from which its days
	// are counted.
	RepeatAnchor string `json:"repeat_anchor,omitempty"`
	// ParentID is set on a subtask: the task it belongs to.
	ParentID string `json:"parent_id,omitempty"`
	// CalendarID is the calendar the task lives in.
	CalendarID string `json:"calendar_id,omitempty"`
}

// Repeats reports whether this task is a series.
func (t Task) Repeats() bool { return t.Rrule != "" }

// AnsweredToday reports whether today's occurrence has been dealt with — done
// or deliberately skipped. Both mean "not outstanding right now".
func (t Task) AnsweredToday() bool {
	return t.Repeats() && t.OccurrenceState != ""
}

// CreateTaskReq mirrors api-go's CreateTaskRequest for the fields the bot uses.
//
// 🔴 Every optional field is a pointer with omitempty, and Priority especially:
// 0 is Buffer, a priority a user can pick, so a plain int cannot express "not
// stated". The API treats an absent priority as its own default; it treats 0 as
// Buffer. Those are different tasks.
type CreateTaskReq struct {
	Title            string   `json:"title"`
	Priority         *int     `json:"priority,omitempty"`
	Status           string   `json:"status,omitempty"`
	DueDate          *string  `json:"due_date,omitempty"` // ISO 8601
	EstimatedMinutes *int     `json:"estimated_minutes,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	// Rrule makes the task repeat. omitempty matters: a one-off task must send
	// no key at all, not an empty rule.
	//
	// 🔴 Absent until 20.09 — which is why «повторяющиеся задачи» shipped as a
	// release note and could not be created by anyone.
	Rrule *string `json:"rrule,omitempty"`
	// NagMinutes repeats an unanswered reminder every N minutes.
	NagMinutes *int `json:"nag_minutes,omitempty"`
	// ParentID makes the task a subtask.
	ParentID *string `json:"parent_id,omitempty"`
	// CalendarID picks the calendar. 🔴 Without it the API files the task in
	// the author's personal calendar: a subtask must send its task's.
	CalendarID *string `json:"calendar_id,omitempty"`
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
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		// The body carries the code — TOO_LATE, NOT_FOUND — and a caller that
		// answers the user by code cannot do without it (day-tasks, 23.09).
		body, _ := io.ReadAll(resp.Body)
		return errorFromBody(resp.StatusCode, body)
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
		return errorFromBody(resp.StatusCode, body)
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

// Calendars lists the calendars this user can write to.
//
// Needed for the keyword «календарь работа» and for a bare word that happens to
// name a calendar. The bot could not read this list at all before 15.09, which
// is why an event from the bot always landed in the personal calendar.
func (c *Client) Calendars(token string) ([]Calendar, error) {
	var resp struct {
		Data []Calendar `json:"data"`
	}
	err := c.get("/api/calendars", token, nil, &resp)
	return resp.Data, err
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

// ScheduleTask turns a task into a calendar block.
//
// The body is ScheduleTaskRequest (api-go/internal/tasks/types.go:128). Until
// 18.08 this method sent {"start_time", "estimated_minutes"} — fields that
// endpoint has never decoded — and had no callers, so nothing ever found out.
// all_day is sent explicitly rather than left to the zero value: the wire
// format is the contract, not Go's defaults.
func (c *Client) ScheduleTask(token string, id string, startsAt, endsAt string) error {
	body := map[string]any{"starts_at": startsAt, "ends_at": endsAt, "all_day": false}
	return c.post("/api/tasks/"+id+"/schedule", token, body, nil)
}

// Settings are a JSONB blob on the user row, and PATCH /api/auth/me REPLACES
// it wholesale (auth/handlers.go:266 — `settings = $N`, not a jsonb merge).
//
// 🔴 That makes a partial write destructive. A bot sending {"work_start":"08:00"}
// would erase header_variant, ui_scale, work_days, features and quiet hours in
// one call, and the user would find out days later in the web app. The web
// client survives this only because it always holds the whole object in memory
// and sends all of it back.
//
// So the bot reads first and merges. MergeSettings is separate from the request
// so the merge itself can be tested without a server: it is the part that has to
// be right, and "did not lose the keys it never knew about" is not observable in
// the bot's own UI.

// MySettings returns the current settings blob for this user.
func (c *Client) MySettings(token string) (map[string]any, error) {
	var resp struct {
		Data struct {
			Settings map[string]any `json:"settings"`
		} `json:"data"`
	}
	if err := c.get("/api/auth/me", token, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data.Settings == nil {
		return map[string]any{}, nil
	}
	return resp.Data.Settings, nil
}

// MergeSettings lays a patch over the current blob without dropping anything
// else. Shallow by design: every field this bot touches is a scalar or a whole
// array, and a deep merge would make removing an element impossible.
func MergeSettings(current, patch map[string]any) map[string]any {
	merged := make(map[string]any, len(current)+len(patch))
	for k, v := range current {
		merged[k] = v
	}
	for k, v := range patch {
		merged[k] = v
	}
	return merged
}

// PatchSettings reads, merges and writes back. The read-modify-write race is
// real but narrow, and losing a concurrent edit is a smaller harm than the
// guaranteed erasure a blind write causes.
func (c *Client) PatchSettings(token string, patch map[string]any) (map[string]any, error) {
	current, err := c.MySettings(token)
	if err != nil {
		return nil, err
	}
	merged := MergeSettings(current, patch)
	if err := c.patch("/api/auth/me", token, map[string]any{"settings": merged}, nil); err != nil {
		return nil, err
	}
	return merged, nil
}

// PlanningTask is one task the planner has not placed yet.
type PlanningTask struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Priority         int    `json:"priority"`
	EstimatedMinutes *int   `json:"estimated_minutes,omitempty"`
}

// WeekPlanResult mirrors planning.WeekPlan, narrowed to what the bot draws.
//
// Only four of its fields are read. The rest — week_start, week_end,
// week_events — are answered by the endpoint and deliberately not decoded: the
// bot shows a load figure and a list of unplaced work, and pulling the whole
// week's events across to count hours the server already counted would be a
// second source of truth for the same number.
type WeekPlanResult struct {
	UnscheduledTasks []PlanningTask `json:"unscheduled_tasks"`
	ScheduledHours   float64        `json:"scheduled_hours"`
	AvailableHours   float64        `json:"available_hours"`
}

// WeekPlan asks the API what this week already looks like.
func (c *Client) WeekPlan(token string) (WeekPlanResult, error) {
	var resp struct {
		Data WeekPlanResult `json:"data"`
	}
	err := c.get("/api/planning/week", token, nil, &resp)
	return resp.Data, err
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
	// Lang is the recipient's language, sent by the API because the notifier
	// cannot ask: it runs on the service token and knows this person only as a
	// Telegram id.
	Lang string `json:"lang"`
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
func (c *Client) NotificationAction(serviceToken string, tgID int64, reminderID, action string, minutes int) (ActionOutcome, error) {
	var out ActionOutcome
	payload := map[string]any{"tg_id": tgID, "reminder_id": reminderID, "action": action}
	if minutes > 0 {
		payload["minutes"] = minutes
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}
	req, err := http.NewRequest("POST", c.base+"/api/svc/notifications/action", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	req.Header.Set("Content-Type", "application/json")
	var wrapper struct {
		Data ActionOutcome `json:"data"`
	}
	if err := c.do(req, &wrapper); err != nil {
		return out, err
	}
	return wrapper.Data, nil
}

// ActionOutcome is what the API says happened.
//
// ⚠ It used to say nothing — the call returned only an error, and the chat
// reported success by assumption. That was harmless while every button meant
// one thing; it stopped being harmless when a press could lead to a SECOND
// question (the personal calendars) which the bot would then never ask.
type ActionOutcome struct {
	// Ask names a follow-up question the bot must put to the person. Empty
	// means the action is finished.
	Ask string `json:"ask"`
	// Merged is true when two accounts have just become one.
	Merged bool `json:"merged"`
	// Already is true when the answer arrived after the question was settled.
	Already bool `json:"already"`
}

// CreateLoginLink asks for a one-shot link onto the website.
//
// 🔴 Runs on the USER's session, not the service token: the link signs somebody
// in, so the request must come from that somebody.
func (c *Client) CreateLoginLink(token string) (string, int, error) {
	var resp struct {
		Data struct {
			Token     string `json:"token"`
			ExpiresIn int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := c.post("/api/auth/login-link", token, map[string]any{}, &resp); err != nil {
		return "", 0, err
	}
	return resp.Data.Token, resp.Data.ExpiresIn, nil
}

// CreateLinkCode asks for the six digits to type on the website.
func (c *Client) CreateLinkCode(token string) (string, int, error) {
	var resp struct {
		Data struct {
			Code      string `json:"code"`
			ExpiresIn int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := c.post("/api/auth/link-code", token, map[string]any{}, &resp); err != nil {
		return "", 0, err
	}
	return resp.Data.Code, resp.Data.ExpiresIn, nil
}

// MarkOccurrence answers for ONE day of a repeating task: «сделал»,
// «пропустить», or «отложить на N дней».
//
// 🔴 Not UpdateTask with status=DONE. For a series, task.status describes the
// whole thing — DONE means «больше не повторять» — while a tick means «сделал
// сегодня». The endpoint has existed since 18.09 and had no caller until 20.09,
// so «✅ Готово» on «пить таблетки» ended the series for good the first time it
// was pressed.
//
// state and postponeDays are mutually exclusive; the API refuses both rather
// than guessing, because they mean opposite things.
// OccurrenceResult is which day the server actually acted on.
//
// 🔴 Read rather than assumed. A press with no date means «the day of the
// series I am looking at», and since 21.09 that is not always today: a task
// due tomorrow starts its series tomorrow. The bot used to answer «Сделано на
// сегодня» from a constant, which would now be a lie on exactly the task that
// made this necessary.
type OccurrenceResult struct {
	Occurrence    string `json:"occurrence"`
	State         string `json:"state"`
	PostponedDays int    `json:"postponed_days"`
	Closed        int    `json:"closed"`
	From          string `json:"from"`
}

func (c *Client) MarkOccurrence(token, taskID, state string, postponeDays int) (OccurrenceResult, error) {
	body := map[string]any{}
	if state != "" {
		body["state"] = state
	}
	if postponeDays > 0 {
		body["postpone_days"] = postponeDays
	}
	// 🔴 The envelope. `c.do` decodes the body as given and unwraps nothing —
	// every other call in this file declares its own `Data` field, and the
	// first version of this one did not. It would have decoded to a zero
	// value, Occurrence would have been "", and closedDayText would have
	// printed «✅ Сделано на сегодня» on the task due tomorrow: the exact lie
	// this whole change exists to stop, arrived at from the other side.
	var resp struct {
		Data OccurrenceResult `json:"data"`
	}
	err := c.post("/api/tasks/"+taskID+"/occurrences", token, body, &resp)
	return resp.Data, err
}
