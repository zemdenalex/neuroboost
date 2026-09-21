package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// The «что нового» broadcast (spec 21.09 §D). Recipients and results go
// through the service token, like the notifier: the bot reads across users.

// Recipient is who gets a broadcast, and in which language.
type Recipient struct {
	TgID int64  `json:"tg_id"`
	Lang string `json:"lang"`
}

// BroadcastRecipients lists the subscribed, recently active users who have
// not got this version yet (not 200) and have not blocked the bot (not 403).
func (c *Client) BroadcastRecipients(serviceToken, version string, activeDays int) ([]Recipient, error) {
	q := url.Values{}
	q.Set("version", version)
	q.Set("active_days", strconv.Itoa(activeDays))
	req, err := http.NewRequest("GET", c.base+"/api/svc/broadcast/recipients?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	var resp struct {
		Data []Recipient `json:"data"`
	}
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// MarkBroadcast records one recipient's result — the per-recipient log that
// makes a retry reach only the people it missed.
func (c *Client) MarkBroadcast(serviceToken string, tgID int64, version string, code int) error {
	body, err := json.Marshal(map[string]any{"tg_id": tgID, "version": version, "code": code})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.base+"/api/svc/broadcast/mark", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Service-Token", serviceToken)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, nil)
}

// IsAdmin reports whether the signed-in user is an admin.
func (c *Client) IsAdmin(token string) (bool, error) {
	var resp struct {
		Data struct {
			IsAdmin bool `json:"is_admin"`
		} `json:"data"`
	}
	if err := c.get("/api/auth/me", token, nil, &resp); err != nil {
		return false, err
	}
	return resp.Data.IsAdmin, nil
}
