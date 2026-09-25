package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/events"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/usersettings"
)

// The first tests of this package (docs/agents/queue.md «Постоянная работа»),
// through GET /api/planning/week against a real database.
//
// ⚠ Needs DATABASE_URL and -count=1. Without the first it skips and still
// prints ok; without the second Go serves a cached result.

type fixture struct {
	d        *database.DB
	user     string
	calendar string
}

func setup(t *testing.T, timezone string) fixture {
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
	calendars.InitDB(d)
	events.InitDB(d)
	usersettings.InitDB(d)

	ctx := context.Background()
	var user, cal string
	email := fmt.Sprintf("planning-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, timezone) VALUES ($1, $2) RETURNING id`, email, timezone).Scan(&user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, user) })
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO calendar (owner_id, name, kind) VALUES ($1, 'Личный', 'personal') RETURNING id`,
		user).Scan(&cal); err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	if _, err := d.Pool.Exec(ctx,
		`INSERT INTO calendar_member (calendar_id, user_id, role, status) VALUES ($1, $2, 'owner', 'active')`,
		cal, user); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	return fixture{d: d, user: user, calendar: cal}
}

func (f fixture) event(t *testing.T, title string, start time.Time, minutes int, rrule string) {
	t.Helper()
	var rr *string
	if rrule != "" {
		rr = &rrule
	}
	if _, err := f.d.Pool.Exec(context.Background(),
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, rrule) VALUES ($1, $2, $3, $4, $5, $6)`,
		f.user, f.calendar, title, start, start.Add(time.Duration(minutes)*time.Minute), rr); err != nil {
		t.Fatalf("seed event %s: %v", title, err)
	}
}

func (f fixture) week(t *testing.T, date string) WeekPlan {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/planning/week?date="+date, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, f.user))
	rec := httptest.NewRecorder()
	GetWeekHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("week %s: status %d %s", date, rec.Code, rec.Body.String())
	}
	var got struct {
		Data WeekPlan `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return got.Data
}

// 🔴 A weekly meeting set up a month ago is on this week's plan too. The
// planner read event rows by starts_at, so a series counted only in the week
// of its first row: «запланировано» said 0 h for every week after it.
func TestARepeatingEventCountsInEveryWeekItOccurs(t *testing.T) {
	f := setup(t, "UTC")
	// Wednesday 10:00 UTC, four weeks before the week asked about.
	f.event(t, "планёрка", time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC), 60, "FREQ=WEEKLY")

	plan := f.week(t, "2026-09-21")
	if len(plan.WeekEvents) != 1 {
		t.Fatalf("week of 21.09 has %d events, want 1 (the series' 23.09 occurrence)", len(plan.WeekEvents))
	}
	if plan.ScheduledHours != 1 {
		t.Errorf("scheduled %v h, want 1", plan.ScheduledHours)
	}
}

// 🔴 The week is the USER's week. It was cut at Monday 00:00 UTC, so for a
// Moscow user a Monday 01:00 meeting fell into the week before.
func TestTheWeekIsCutInTheUsersZone(t *testing.T) {
	f := setup(t, "Europe/Moscow")
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}
	f.event(t, "ночной созвон", time.Date(2026, 9, 21, 1, 0, 0, 0, moscow), 30, "")

	if plan := f.week(t, "2026-09-21"); len(plan.WeekEvents) != 1 {
		t.Errorf("week of 21.09 has %d events, want 1: Monday 01:00 Moscow is that week", len(plan.WeekEvents))
	}
	if plan := f.week(t, "2026-09-14"); len(plan.WeekEvents) != 0 {
		t.Errorf("week of 14.09 has %d events, want 0", len(plan.WeekEvents))
	}
}

// Positive control for both: a plain event mid-week counts once, an all-day
// one is listed but adds no hours, and a day outside the week is not there.
func TestAPlainWeekCountsTimedEventsOnly(t *testing.T) {
	f := setup(t, "UTC")
	f.event(t, "врач", time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC), 90, "")
	f.event(t, "следующая неделя", time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC), 60, "")
	if _, err := f.d.Pool.Exec(context.Background(),
		`INSERT INTO event (user_id, calendar_id, title, starts_at, ends_at, all_day) VALUES ($1, $2, 'отпуск', $3, $4, true)`,
		f.user, f.calendar, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed all-day: %v", err)
	}

	plan := f.week(t, "2026-09-23")
	if len(plan.WeekEvents) != 2 {
		t.Errorf("%d events, want 2", len(plan.WeekEvents))
	}
	if plan.ScheduledHours != 1.5 {
		t.Errorf("scheduled %v h, want 1.5", plan.ScheduledHours)
	}
	if got := plan.WeekStart.Format("2006-01-02"); got != "2026-09-21" {
		t.Errorf("week starts %s, want the Monday 2026-09-21", got)
	}
}
