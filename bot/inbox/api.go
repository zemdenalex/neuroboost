package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// API talks to /api/inbox with X-Inbox-Token.
type API struct {
	Base  string
	Token string
	HTTP  *http.Client
}

// Note is the part of the API's note the bot shows back.
type Note struct {
	ID       int64    `json:"id"`
	Type     string   `json:"type"`
	Projects []string `json:"projects"`
	Status   string   `json:"status"`
	Text     string   `json:"text"`
}

func (a *API) client() *http.Client {
	if a.HTTP != nil {
		return a.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (a *API) do(ctx context.Context, method, path string, body io.Reader, hdr map[string]string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.Base+path, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Inbox-Token", a.Token)
	if _, ok := hdr["Content-Type"]; !ok {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := a.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, bytes.TrimSpace(raw))
	}
	if out != nil {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return resp.StatusCode, err
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// Create saves a draft; created is false when the client_key was seen before.
func (a *API) Create(ctx context.Context, owner int64, d Draft, via string) (n Note, created bool, err error) {
	body, _ := json.Marshal(map[string]any{
		"owner_tg_id": owner, "type": d.Type, "projects": d.Projects, "text": d.Text,
		"source": d.Source, "via": via, "client_key": d.ClientKey,
	})
	code, err := a.do(ctx, http.MethodPost, "/api/inbox", bytes.NewReader(body), nil, &n)
	return n, code == http.StatusCreated, err
}

// Patch changes type, projects or status; nil fields are left alone.
func (a *API) Patch(ctx context.Context, id int64, fields map[string]any) (Note, error) {
	body, _ := json.Marshal(fields)
	var n Note
	_, err := a.do(ctx, http.MethodPatch, "/api/inbox/"+strconv.FormatInt(id, 10), bytes.NewReader(body), nil, &n)
	return n, err
}

func (a *API) Get(ctx context.Context, id int64) (Note, error) {
	var n Note
	_, err := a.do(ctx, http.MethodGet, "/api/inbox/"+strconv.FormatInt(id, 10), nil, nil, &n)
	return n, err
}

// AddMedia uploads a file; data == nil records only that it existed.
func (a *API) AddMedia(ctx context.Context, id int64, m Media, data []byte) error {
	hdr := map[string]string{
		"X-Media-Kind": m.Kind, "X-File-Unique-Id": m.UniqueID,
		"X-File-Name": m.FileName, "X-File-Size": strconv.FormatInt(m.Size, 10),
		"Content-Type": m.Mime,
	}
	if m.Mime == "" {
		hdr["Content-Type"] = "application/octet-stream"
	}
	if data == nil {
		hdr["X-Media-Omitted"] = "1"
	}
	_, err := a.do(ctx, http.MethodPost, "/api/inbox/"+strconv.FormatInt(id, 10)+"/media", bytes.NewReader(data), hdr, nil)
	return err
}

// CountNew returns how many new notes there are, by type.
func (a *API) CountNew(ctx context.Context) (map[string]int, error) {
	var ns []Note
	if _, err := a.do(ctx, http.MethodGet, "/api/inbox?status=new", nil, nil, &ns); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, n := range ns {
		out[n.Type]++
	}
	return out, nil
}
