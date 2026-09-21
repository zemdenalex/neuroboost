package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/database"
)

// Recipients and marks, against a real database through the handlers.
// ⚠ Needs DATABASE_URL and -count=1.

func testDB(t *testing.T) (*database.DB, context.Context) {
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
	InitDB(d)
	return d, context.Background()
}

// seed makes a Telegram user with the given settings and last login.
func seed(t *testing.T, ctx context.Context, d *database.DB, tgID int64, settings string, lastLogin time.Duration) {
	t.Helper()
	if _, err := d.Pool.Exec(ctx, `
		INSERT INTO "user" (email, tg_id, tg_auth_date, settings)
		VALUES ($1, $2, now() - $3::interval, $4::jsonb)`,
		fmt.Sprintf("bc-%d@example.com", tgID), tgID, fmt.Sprintf("%d seconds", int(lastLogin.Seconds())), settings); err != nil {
		t.Fatalf("seed %d: %v", tgID, err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE tg_id = $1`, tgID) })
}

func recipients(t *testing.T, query string) (int, []Recipient) {
	t.Helper()
	rec := httptest.NewRecorder()
	RecipientsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/svc/broadcast/recipients?"+query, nil))
	var got struct {
		Data []Recipient `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	return rec.Code, got.Data
}

func has(list []Recipient, id int64) bool {
	for _, r := range list {
		if r.TgID == id {
			return true
		}
	}
	return false
}

func TestRecipientsAreTheSubscribedAndRecentOnly(t *testing.T) {
	d, ctx := testDB(t)
	base := time.Now().UnixNano()%1_000_000_000 + 7_000_000_000
	day := 24 * time.Hour
	seed(t, ctx, d, base+1, `{"bot":{"lang":"en"}}`, 2*day)             // yes, en
	seed(t, ctx, d, base+2, `{"bot":{"updates":"off"}}`, 2*day)         // unsubscribed
	seed(t, ctx, d, base+3, `{"bot":{"updates":"on"}}`, 2*day)          // came back
	seed(t, ctx, d, base+4, `{}`, 11*day)                               // not active
	seed(t, ctx, d, base+5, `{"bot":{"broadcasts":{"v9":"200"}}}`, day) // delivered
	seed(t, ctx, d, base+6, `{"bot":{"broadcasts":{"v9":"403"}}}`, day) // blocked the bot
	seed(t, ctx, d, base+7, `{"bot":{"broadcasts":{"v9":"0"}}}`, day)   // network: retry
	seed(t, ctx, d, base+8, `{"bot":{"broadcasts":{"v8":"200"}}}`, day) // other version

	code, list := recipients(t, "version=v9&active_days=10")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	for id, want := range map[int64]bool{
		base + 1: true, base + 2: false, base + 3: true, base + 4: false,
		base + 5: false, base + 6: false, base + 7: true, base + 8: true,
	} {
		if has(list, id) != want {
			t.Errorf("tg %d: in list = %v, want %v", id-base, !want, want)
		}
	}
	for _, r := range list {
		if r.TgID == base+1 && r.Lang != "en" {
			t.Errorf("lang = %q, want en", r.Lang)
		}
	}
}

func TestRecipientsRefuseABadQuery(t *testing.T) {
	testDB(t)
	for _, q := range []string{"active_days=10", "version=v9&active_days=0", "version=v9&active_days=91", "version=v9&active_days=x"} {
		if code, _ := recipients(t, q); code != http.StatusBadRequest {
			t.Errorf("%q → %d, want 400", q, code)
		}
	}
}

// A mark is the per-recipient log (spec §D1): it must not touch anything else
// in the settings blob, and it must take the recipient out of the next run.
func TestAMarkIsLoggedAndKeepsTheRest(t *testing.T) {
	d, ctx := testDB(t)
	id := time.Now().UnixNano()%1_000_000_000 + 8_000_000_000
	seed(t, ctx, d, id, `{"work_start":"09:00","bot":{"lang":"ru"}}`, time.Hour)

	body, _ := json.Marshal(map[string]any{"tg_id": id, "version": "v9", "code": 200})
	rec := httptest.NewRecorder()
	MarkHandler(rec, httptest.NewRequest(http.MethodPost, "/api/svc/broadcast/mark", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("mark: %d %s", rec.Code, rec.Body.String())
	}
	var settings string
	_ = d.Pool.QueryRow(ctx, `SELECT settings::text FROM "user" WHERE tg_id = $1`, id).Scan(&settings)
	for _, want := range []string{`"v9": "200"`, `"lang": "ru"`, `"work_start": "09:00"`} {
		if !bytes.Contains([]byte(settings), []byte(want)) {
			t.Errorf("settings %s lost or lack %s", settings, want)
		}
	}
	if _, list := recipients(t, "version=v9&active_days=10"); has(list, id) {
		t.Error("a delivered recipient is still listed")
	}
}
