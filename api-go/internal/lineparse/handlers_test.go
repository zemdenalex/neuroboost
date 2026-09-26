package lineparse

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

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"neuroboost/api-go/internal/calendars"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

const testSecret = "lineparse-test-secret"

// A fixed clock: «завтра» must mean the same day on every run.
// 07:00 UTC on Saturday 26.09.2026 is 10:00 in Moscow.
var fixedNow = time.Date(2026, time.September, 26, 7, 0, 0, 0, time.UTC)

// seedUser is a user in Moscow with the given settings blob and a «Работа»
// calendar next to the personal one.
func seedUser(t *testing.T, settings string) (userID, workCalendarID string) {
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

	ctx := context.Background()
	email := fmt.Sprintf("lineparse-%d@example.com", time.Now().UnixNano())
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO "user" (email, timezone, settings) VALUES ($1, 'Europe/Moscow', $2::jsonb) RETURNING id`,
		email, settings).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(ctx, `DELETE FROM calendar WHERE id IN
			(SELECT calendar_id FROM calendar_member WHERE user_id = $1)`, userID)
		_, _ = d.Pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID)
	})
	work, err := calendars.Create(ctx, userID, "Работа", nil)
	if err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	return userID, work.ID
}

// router is the protected group exactly as main.go builds it: the JWT
// middleware in front of Register. A test that called the handler directly
// would prove nothing about the door.
func router() http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware(testSecret))
		Register(r)
	})
	return r
}

func token(t *testing.T, userID string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &middleware.Claims{
		UserID:           userID,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	s, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func post(t *testing.T, bearer string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/parse", bytes.NewReader(b))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	router().ServeHTTP(rec, req)
	return rec
}

func answer(t *testing.T, rec *httptest.ResponseRecorder) Answer {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data Answer `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return got.Data
}

func withClock(t *testing.T) {
	t.Helper()
	prev := now
	now = func() time.Time { return fixedNow }
	t.Cleanup(func() { now = prev })
}

func TestParseNeedsAToken(t *testing.T) {
	if rec := post(t, "", map[string]string{"text": "молоко"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d, want 401", rec.Code)
	}
}

func TestParseTimedLineIsAnEventInTheUsersZone(t *testing.T) {
	withClock(t)
	user, _ := seedUser(t, `{}`)
	got := answer(t, post(t, token(t, user), map[string]string{"text": "стоматолог завтра 15:00 напомни за час"}))

	if got.Kind != KindEvent || got.Title != "стоматолог" {
		t.Fatalf("kind %q title %q", got.Kind, got.Title)
	}
	// 15:00 in Moscow (UTC+3) is 12:00 UTC; no end given → one hour.
	if got.StartsAt == nil || *got.StartsAt != "2026-09-27T12:00:00Z" ||
		got.EndsAt == nil || *got.EndsAt != "2026-09-27T13:00:00Z" {
		t.Errorf("when = %v–%v", deref(got.StartsAt), deref(got.EndsAt))
	}
	if got.ReminderOffsets == nil || len(*got.ReminderOffsets) != 1 || (*got.ReminderOffsets)[0] != 60 {
		t.Errorf("reminders = %v, want [60]", got.ReminderOffsets)
	}
	if got.Timezone != "Europe/Moscow" {
		t.Errorf("timezone = %q", got.Timezone)
	}
}

func TestParseLineWithoutTimeIsAPlainTask(t *testing.T) {
	withClock(t)
	user, _ := seedUser(t, `{}`)
	got := answer(t, post(t, token(t, user), map[string]string{"text": "купить молоко завтра !1 30м #дом"}))

	if got.Kind != KindTask || got.Title != "купить молоко" {
		t.Fatalf("kind %q title %q", got.Kind, got.Title)
	}
	if got.Priority == nil || *got.Priority != 1 || got.EstimatedMinutes == nil || *got.EstimatedMinutes != 30 {
		t.Errorf("priority %v estimate %v", got.Priority, got.EstimatedMinutes)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "дом" {
		t.Errorf("tags = %v", got.Tags)
	}
	// Due tomorrow, at local midnight in Moscow.
	if got.DueDate == nil || *got.DueDate != "2026-09-26T21:00:00Z" {
		t.Errorf("due = %v", deref(got.DueDate))
	}
	// «Not stated» stays null, not zero: 0 is Buffer, a real priority.
	plain := answer(t, post(t, token(t, user), map[string]string{"text": "позвонить маме"}))
	if plain.Priority != nil || plain.DueDate != nil || plain.Kind != KindTask {
		t.Errorf("plain line: %+v", plain)
	}
}

func TestParseUsesTheUsersOwnWordsAndCalendars(t *testing.T) {
	withClock(t)
	user, work := seedUser(t, `{"bot":{"keywords":{"созвон":{"field":"calendar","value":"работа"}}}}`)
	got := answer(t, post(t, token(t, user), map[string]string{"text": "созвон с Петей завтра 11:00"}))

	if got.Title != "с Петей" || got.CalendarID == nil || *got.CalendarID != work {
		t.Errorf("title %q calendar %v, want the word to pick «Работа» (%s)", got.Title, deref(got.CalendarID), work)
	}
}

// Negative control for the test above: a user WITHOUT the word gets the same
// line back with the word in the title and no calendar. Without it the test
// above could pass on a parser that drops the first word of every line.
func TestParseWithoutTheWordKeepsItInTheTitle(t *testing.T) {
	withClock(t)
	user, _ := seedUser(t, `{}`)
	got := answer(t, post(t, token(t, user), map[string]string{"text": "созвон с Петей завтра 11:00"}))

	if got.Title != "созвон с Петей" || got.CalendarID != nil {
		t.Errorf("title %q calendar %v", got.Title, deref(got.CalendarID))
	}
}

func TestParseAsksWhatTheBotWouldAsk(t *testing.T) {
	withClock(t)
	user, _ := seedUser(t, `{}`)
	for line, missing := range map[string]string{
		"стоматолог 15:00":           "date",
		"зарядка завтра 8:00 повтор": "freq",
	} {
		got := answer(t, post(t, token(t, user), map[string]string{"text": line}))
		if got.Kind != KindAsk || got.Missing != missing {
			t.Errorf("%q: kind %q missing %q, want ask/%s", line, got.Kind, got.Missing, missing)
		}
	}
}

func TestParseTimezoneOverrideAndValidation(t *testing.T) {
	withClock(t)
	user, _ := seedUser(t, `{}`)
	got := answer(t, post(t, token(t, user), map[string]string{
		"text": "стоматолог завтра 15:00", "timezone": "America/New_York"}))
	// 15:00 in New York (EDT, UTC−4) on 27.09 is 19:00 UTC.
	if got.StartsAt == nil || *got.StartsAt != "2026-09-27T19:00:00Z" || got.Timezone != "America/New_York" {
		t.Errorf("starts %v zone %q", deref(got.StartsAt), got.Timezone)
	}
	for _, body := range []map[string]string{
		{"text": "молоко", "timezone": "Mars/Olympus"},
		{"text": "   "},
	} {
		if rec := post(t, token(t, user), body); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: %d, want 400", body, rec.Code)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
