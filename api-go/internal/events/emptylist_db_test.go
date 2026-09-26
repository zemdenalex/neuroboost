package events

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
)

// Gotcha 6: arrays from the API are [], never null. ListExpanded (4097870,
// shared by GET /api/events and planning) built its result with `var`, so a
// week without events answered {"data":null} — CI run 36222776573 caught it on
// the e2e account, whose calendar is empty.
func TestAnEmptyRangeIsAnEmptyListNotNull(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed test")
	}
	d, err := database.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer d.Close()
	InitDB(d)
	calendars.InitDB(d)
	ctx := context.Background()

	userID := seedUser(t, ctx, d, "empty")
	start := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	list, err := ListExpanded(ctx, userID, start, start.Add(7*24*time.Hour))
	if err != nil {
		t.Fatalf("ListExpanded: %v", err)
	}
	raw, _ := json.Marshal(list)
	if string(raw) != "[]" {
		t.Errorf("an empty week encodes as %s, want []", raw)
	}
}
