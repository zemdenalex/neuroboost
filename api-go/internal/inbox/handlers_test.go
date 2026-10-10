package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/database"
)

const testToken = "inbox-test-token"

// inboxDB is a real database and the routes wired the way main.go wires them,
// so every test goes through the door the bot and the script use.
func inboxDB(t *testing.T) (http.Handler, string) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(d.Close)
	prefix := fmt.Sprintf("test-%d-", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(context.Background(), `DELETE FROM inbox_note WHERE client_key LIKE $1`, prefix+"%")
	})
	r := chi.NewRouter()
	NewHandler(d).Routes(r, testToken)
	return r, prefix
}

func do(t *testing.T, h http.Handler, method, path, body, token string, hdr map[string]string) (int, map[string]json.RawMessage) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Inbox-Token", token)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var env map[string]json.RawMessage
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env
}

func note(t *testing.T, env map[string]json.RawMessage) Note {
	t.Helper()
	var n Note
	if err := json.Unmarshal(env["data"], &n); err != nil {
		t.Fatalf("decode note: %v (%s)", err, env["data"])
	}
	return n
}

func TestInboxRejectsAMissingOrWrongToken(t *testing.T) {
	h, _ := inboxDB(t)
	for _, tok := range []string{"", "wrong"} {
		if code, _ := do(t, h, http.MethodGet, "/api/inbox", "", tok, nil); code != http.StatusUnauthorized {
			t.Errorf("token %q: got %d, want 401", tok, code)
		}
	}
	// Positive control: the same request with the right token is let in.
	if code, _ := do(t, h, http.MethodGet, "/api/inbox", "", testToken, nil); code != http.StatusOK {
		t.Errorf("right token: got %d, want 200", code)
	}
}

func TestInboxIsClosedWhenNoTokenIsConfigured(t *testing.T) {
	r := chi.NewRouter()
	NewHandler(nil).Routes(r, "")
	if code, _ := do(t, r, http.MethodGet, "/api/inbox", "", "", nil); code != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503", code)
	}
}

// A redelivered Telegram update carries the same client_key; it must find the
// note it already made, not make a twin.
func TestCreateIsIdempotentOnClientKey(t *testing.T) {
	h, p := inboxDB(t)
	body := fmt.Sprintf(`{"owner_tg_id":7,"type":"idea","projects":["#AvoCar","avocar","graph"],"text":"x","client_key":"%sa"}`, p)
	code, env := do(t, h, http.MethodPost, "/api/inbox", body, testToken, nil)
	if code != http.StatusCreated {
		t.Fatalf("first: %d", code)
	}
	first := note(t, env)
	if got := strings.Join(first.Projects, ","); got != "avocar,graph" {
		t.Errorf("projects = %q, want avocar,graph", got)
	}
	code, env = do(t, h, http.MethodPost, "/api/inbox", body, testToken, nil)
	if code != http.StatusOK || note(t, env).ID != first.ID {
		t.Errorf("second: code %d id %d, want 200 and id %d", code, note(t, env).ID, first.ID)
	}
}

func TestUpdateRetagsAppendsAndMovesStatus(t *testing.T) {
	h, p := inboxDB(t)
	_, env := do(t, h, http.MethodPost, "/api/inbox",
		fmt.Sprintf(`{"owner_tg_id":7,"text":"forwarded post","source":{"kind":"channel","name":"Ch","url":"https://t.me/ch/5"},"client_key":"%sb"}`, p), testToken, nil)
	n := note(t, env)
	if n.Type != "inbox" || n.Status != "new" || n.Source == nil || n.Source.URL != "https://t.me/ch/5" {
		t.Fatalf("created = %+v", n)
	}
	code, env := do(t, h, http.MethodPatch, fmt.Sprintf("/api/inbox/%d", n.ID),
		`{"type":"link","projects":["harness"],"append_text":"my comment","status":"discussed"}`, testToken, nil)
	if code != http.StatusOK {
		t.Fatalf("patch: %d", code)
	}
	u := note(t, env)
	if u.Type != "link" || u.Status != "discussed" || strings.Join(u.Projects, ",") != "harness" ||
		u.Text != "forwarded post\n\nmy comment" {
		t.Errorf("updated = %+v", u)
	}
	// A patch that names only the status leaves type, projects and text alone.
	_, env = do(t, h, http.MethodPatch, fmt.Sprintf("/api/inbox/%d", n.ID), `{"status":"done"}`, testToken, nil)
	if d := note(t, env); d.Type != "link" || len(d.Projects) != 1 || d.Text != u.Text || d.Status != "done" {
		t.Errorf("status-only patch changed more: %+v", d)
	}
	if code, _ := do(t, h, http.MethodPatch, fmt.Sprintf("/api/inbox/%d", n.ID), `{"status":"maybe"}`, testToken, nil); code != http.StatusBadRequest {
		t.Errorf("bad status: %d, want 400", code)
	}
}

func TestListFiltersByStatusTypeAndProject(t *testing.T) {
	h, p := inboxDB(t)
	mk := func(k, typ, proj string) int64 {
		_, env := do(t, h, http.MethodPost, "/api/inbox",
			fmt.Sprintf(`{"owner_tg_id":7,"type":"%s","projects":["%s"],"client_key":"%s%s"}`, typ, proj, p, k), testToken, nil)
		return note(t, env).ID
	}
	idea := mk("l1", "idea", "avocar")
	refl := mk("l2", "reflection", "graph")
	done := mk("l3", "idea", "avocar")
	do(t, h, http.MethodPatch, fmt.Sprintf("/api/inbox/%d", done), `{"status":"done"}`, testToken, nil)

	list := func(q string) []int64 {
		_, env := do(t, h, http.MethodGet, fmt.Sprintf("/api/inbox?after_id=%d&%s", idea-1, q), "", testToken, nil)
		var ns []Note
		if err := json.Unmarshal(env["data"], &ns); err != nil {
			t.Fatal(err)
		}
		var ids []int64
		for _, n := range ns {
			ids = append(ids, n.ID)
		}
		return ids
	}
	if got := list("status=new&type=idea"); fmt.Sprint(got) != fmt.Sprint([]int64{idea}) {
		t.Errorf("new ideas = %v, want [%d]", got, idea)
	}
	if got := list("project=graph"); fmt.Sprint(got) != fmt.Sprint([]int64{refl}) {
		t.Errorf("graph = %v, want [%d]", got, refl)
	}
	if got := list("status=new,done&project=avocar"); fmt.Sprint(got) != fmt.Sprint([]int64{idea, done}) {
		t.Errorf("avocar all = %v, want [%d %d]", got, idea, done)
	}
}

func TestMediaIsStoredReturnedAndMayBeMetadataOnly(t *testing.T) {
	h, p := inboxDB(t)
	_, env := do(t, h, http.MethodPost, "/api/inbox", fmt.Sprintf(`{"owner_tg_id":7,"client_key":"%sm"}`, p), testToken, nil)
	n := note(t, env)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/inbox/%d/media", n.ID), bytes.NewReader([]byte("OggS-voice")))
	req.Header.Set("X-Inbox-Token", testToken)
	req.Header.Set("Content-Type", "audio/ogg")
	req.Header.Set("X-Media-Kind", "voice")
	req.Header.Set("X-File-Unique-Id", "u1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	if code, _ := do(t, h, http.MethodPost, fmt.Sprintf("/api/inbox/%d/media", n.ID), "", testToken,
		map[string]string{"X-Media-Kind": "video", "X-File-Unique-Id": "u2", "X-Media-Omitted": "1", "X-File-Size": "50000000"}); code != http.StatusCreated {
		t.Fatalf("metadata-only: %d", code)
	}

	_, env = do(t, h, http.MethodGet, fmt.Sprintf("/api/inbox/%d", n.ID), "", testToken, nil)
	got := note(t, env)
	if len(got.Media) != 2 || !got.Media[0].Stored || got.Media[1].Stored || got.Media[1].Size != 50000000 {
		t.Fatalf("media = %+v", got.Media)
	}
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/inbox/media/%d", got.Media[0].ID), nil)
	req.Header.Set("X-Inbox-Token", testToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != "OggS-voice" || rec.Header().Get("Content-Type") != "audio/ogg" {
		t.Errorf("download = %q %q", rec.Body.String(), rec.Header().Get("Content-Type"))
	}
	if code, _ := do(t, h, http.MethodGet, fmt.Sprintf("/api/inbox/media/%d", got.Media[1].ID), "", testToken, nil); code != http.StatusNotFound {
		t.Errorf("metadata-only download: %d, want 404", code)
	}
	if code, _ := do(t, h, http.MethodPost, "/api/inbox/999999999/media", "x", testToken,
		map[string]string{"X-Media-Kind": "photo", "X-File-Unique-Id": "u3"}); code != http.StatusNotFound {
		t.Errorf("media for a missing note: %d, want 404", code)
	}
}
