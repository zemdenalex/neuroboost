package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/zemdenalex/neuroboost-bot/parse"
)

// «Напоминать до срока» lives behind its own door on the API —
// /api/{tasks|events}/{id}/remind-rule — so kind is "tasks" or "events".

// RemindRule returns the item's rule, or nil when it has none.
func (c *Client) RemindRule(token, kind, id string) (*parse.RemindRule, error) {
	var resp struct {
		Data *parse.RemindRule `json:"data"`
	}
	err := c.get("/api/"+kind+"/"+id+"/remind-rule", token, nil, &resp)
	return resp.Data, err
}

// SetRemindRule stores the rule; nil removes it.
func (c *Client) SetRemindRule(token, kind, id string, rule *parse.RemindRule) error {
	path := "/api/" + kind + "/" + id + "/remind-rule"
	if rule == nil {
		return c.del(path, token)
	}
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(req, nil)
}
