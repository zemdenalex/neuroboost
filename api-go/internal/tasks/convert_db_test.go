package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/events"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/reminders"
)

// Задача ↔ событие через дверь клиента, против настоящей базы.
//
// ⚠ Нужны DATABASE_URL и -count=1: без первого тест скипается и печатает ok,
// без второго Go отдаёт кэш — чужая база не входит в то, что он хэширует.

func callTaskRoute(h http.HandlerFunc, path, taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", taskID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	rec := httptest.NewRecorder()
	h(rec, req.WithContext(ctx))
	return rec
}

func callConvert(taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	return callTaskRoute(ConvertHandler, "/api/tasks/"+taskID+"/convert", taskID, userID, body)
}

func setZone(t *testing.T, userID, tz string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		`UPDATE "user" SET timezone = $2 WHERE id = $1`, userID, tz); err != nil {
		t.Fatalf("set zone: %v", err)
	}
}

type storedEvent struct {
	ID, Timezone    string
	Rrule           *string
	TaskID          *string
	StartsAt, Ends  time.Time
	Description     *string
	Tags            []string
	ReminderOffsets []int
}

func loadEvent(t *testing.T, id string) storedEvent {
	t.Helper()
	var e storedEvent
	if err := db.Pool.QueryRow(context.Background(), `
		SELECT id::text, COALESCE(timezone,''), rrule, task_id::text, starts_at, ends_at,
		       description, COALESCE(tags,'{}'), COALESCE(reminder_offsets,'{}')
		  FROM event WHERE id = $1`, id).Scan(&e.ID, &e.Timezone, &e.Rrule, &e.TaskID,
		&e.StartsAt, &e.Ends, &e.Description, &e.Tags, &e.ReminderOffsets); err != nil {
		t.Fatalf("load event %s: %v", id, err)
	}
	return e
}

func createdEventID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var got struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Data.ID == "" {
		t.Fatalf("no event id in %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	return got.Data.ID
}

func TestConvertStoresTheUsersZone(t *testing.T) {
	d, ctx, user := repeatDB(t)
	setZone(t, user, "America/New_York")

	task, err := insertTask(ctx, user, CreateTaskRequest{Title: "созвон"}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "starts_at": "2026-10-20T09:00:00-04:00", "ends_at": "2026-10-20T10:00:00-04:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, ev.ID) })

	if ev.Timezone != "America/New_York" {
		t.Errorf("event.timezone = %q, want the user's America/New_York", ev.Timezone)
	}
}

// 🔴 Поведенческая половина: что именно ломает чужая зона. Серия в 09:00 по
// Нью-Йорку, развёрнутая после 01.11.2026 (США переходят на зимнее время),
// обязана остаться в 09:00. С «Europe/Moscow» (без перехода) она съезжает на 08:00.
//
// ⚠ Колонку проверяет тест выше; этот нужен, потому что момент starts_at верен
// и с чужой зоной — RFC3339 несёт смещение. Тест на «событие в его час» был бы
// зелёным до починки.
func TestAScheduledSeriesKeepsItsLocalHourAcrossDST(t *testing.T) {
	d, ctx, user := repeatDB(t)
	setZone(t, user, "America/New_York")

	due := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	dueStr := due.Format("2006-01-02")
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "зарядка", Rrule: str("FREQ=DAILY"), DueDate: &dueStr,
	}, &due)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "repeat": "series",
		"starts_at": "2026-10-28T09:00:00-04:00", "ends_at": "2026-10-28T09:30:00-04:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))

	ny, _ := time.LoadLocation("America/New_York")
	from := time.Date(2026, 11, 3, 0, 0, 0, 0, ny)
	occ := events.OccurrencesInRange(events.Event{
		StartsAt: ev.StartsAt, EndsAt: ev.Ends, Rrule: ev.Rrule, Timezone: ev.Timezone,
	}, from, from.AddDate(0, 0, 1), nil)
	if len(occ) != 1 {
		t.Fatalf("expanded %d occurrences on 03.11, want 1 (rrule %v)", len(occ), ev.Rrule)
	}
	if h := occ[0].In(ny).Hour(); h != 9 {
		t.Errorf("03.11 occurrence at %02d:00 New York, want 09:00 — the series drifted with a foreign zone", h)
	}
}

func seriesTask(t *testing.T, ctx context.Context, user, rule string, due time.Time) *Task {
	t.Helper()
	s := due.Format("2006-01-02")
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "таблетки", Rrule: str(rule), DueDate: &s,
		Description: str("после еды"), Tags: []string{"здоровье"},
		ReminderOffsets: &[]int{10},
	}, &due)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })
	return task
}

func taskExists(t *testing.T, id string) (exists bool, status string) {
	t.Helper()
	err := db.Pool.QueryRow(context.Background(),
		`SELECT status FROM task WHERE id = $1`, id).Scan(&status)
	return err == nil, status
}

func TestASeriesMustSayWhichPart(t *testing.T) {
	_, ctx, user := repeatDB(t)
	task := seriesTask(t, ctx, user, "FREQ=DAILY", time.Now())
	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "starts_at": "2026-10-20T09:00:00Z", "ends_at": "2026-10-20T10:00:00Z",
	})
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("REPEAT_CHOICE_REQUIRED")) {
		t.Errorf("got %d %s, want 400 REPEAT_CHOICE_REQUIRED", rec.Code, rec.Body.String())
	}
}

func TestSeriesLinkCopiesTheRuleAndLeavesTheStatus(t *testing.T) {
	_, ctx, user := repeatDB(t)
	day := time.Now().AddDate(0, 0, 2)
	task := seriesTask(t, ctx, user, "FREQ=WEEKLY", day)
	start := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "repeat": "series",
		"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))
	if ev.Rrule == nil || *ev.Rrule != "FREQ=WEEKLY" {
		t.Errorf("event rrule = %v, want FREQ=WEEKLY", ev.Rrule)
	}
	if ev.Description == nil || *ev.Description != "после еды" || len(ev.Tags) != 1 {
		t.Errorf("description/tags not copied: %v %v", ev.Description, ev.Tags)
	}
	if len(ev.ReminderOffsets) != 1 || ev.ReminderOffsets[0] != 10 {
		t.Errorf("reminder_offsets = %v, want [10] — an event with {} never reminds", ev.ReminderOffsets)
	}
	if ok, st := taskExists(t, task.ID); !ok || st != string(StatusTodo) {
		t.Errorf("task after link: exists=%v status=%q, want TODO untouched", ok, st)
	}
}

func TestOnceMoveTakesOneDayAndTheSeriesLives(t *testing.T) {
	_, ctx, user := repeatDB(t)
	day := time.Now().AddDate(0, 0, 1)
	task := seriesTask(t, ctx, user, "FREQ=DAILY", day)
	// 09:00 UTC is the same calendar day in Europe/Moscow, the seeded user's zone.
	start := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.UTC)

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "move", "repeat": "once",
		"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM event WHERE id = $1`, ev.ID) })

	if ev.Rrule != nil {
		t.Errorf("once produced a repeating event: %v", *ev.Rrule)
	}
	if ev.TaskID != nil {
		t.Errorf("a move is not a link: event.task_id = %v", *ev.TaskID)
	}
	if ok, _ := taskExists(t, task.ID); !ok {
		t.Fatal("the series was deleted — Denis 21.09: one day moves, the series stays")
	}
	var state string
	if err := db.Pool.QueryRow(ctx,
		`SELECT state FROM task_occurrence WHERE task_id = $1 AND occurrence = $2`,
		task.ID, day.Format("2006-01-02")).Scan(&state); err != nil || state != StateSkipped {
		t.Errorf("the moved day: state=%q err=%v, want skipped — else the series still shows it", state, err)
	}
}

func TestOnceOnADayOutsideTheSeriesIsRefused(t *testing.T) {
	_, ctx, user := repeatDB(t)
	day := time.Now().AddDate(0, 0, 1)
	task := seriesTask(t, ctx, user, "FREQ=WEEKLY", day)
	other := day.AddDate(0, 0, 1) // weekly from `day`: the next day is not in it
	start := time.Date(other.Year(), other.Month(), other.Day(), 9, 0, 0, 0, time.UTC)

	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "repeat": "once",
		"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
	})
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("NOT_AN_OCCURRENCE")) {
		t.Errorf("got %d %s, want 400 NOT_AN_OCCURRENCE", rec.Code, rec.Body.String())
	}
}

// 🔴 «Только этот раз» не имеет права оставить остальную серию без журнала
// напоминаний. Строка вставляется в журнал напрямую: это не колонка задачи, а
// запись об отправке, и у клиента нет двери, через которую её создают.
func TestOnceLeavesTheSeriesItsReminderLog(t *testing.T) {
	_, ctx, user := repeatDB(t)
	day := time.Now().AddDate(0, 0, 1)
	task := seriesTask(t, ctx, user, "FREQ=DAILY", day)
	var remID string
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO reminder (user_id, task_id, source_kind, minutes_before, occurrence_start, remind_at)
		VALUES ($1, $2, 'TASK', 10, now() + interval '3 days', now() + interval '3 days' - interval '10 minutes')
		RETURNING id::text`,
		user, task.ID).Scan(&remID); err != nil {
		t.Fatalf("seed reminder log: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM reminder WHERE id = $1`, remID) })

	start := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.UTC)
	rec := callConvert(task.ID, user, map[string]any{
		"mode": "link", "repeat": "once",
		"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var owner *string
	_ = db.Pool.QueryRow(ctx, `SELECT task_id::text FROM reminder WHERE id = $1`, remID).Scan(&owner)
	if owner == nil || *owner != task.ID {
		t.Errorf("the series' reminder row moved to the one-off event (task_id now %v)", owner)
	}
}

func callSchedule(taskID, userID string, body map[string]any) *httptest.ResponseRecorder {
	return callTaskRoute(ScheduleHandler, "/api/tasks/"+taskID+"/schedule", taskID, userID, body)
}

// Denis 21.09: the quick button and the full path are one link with two doors.
// Until now the quick one dropped the description and tags, and the event it
// made had reminder_offsets {} — silent for ever.
func TestQuickScheduleCopiesWhatConvertCopies(t *testing.T) {
	_, ctx, user := repeatDB(t)
	setZone(t, user, "Asia/Tokyo")
	task, err := insertTask(ctx, user, CreateTaskRequest{
		Title: "банк", Description: str("паспорт"), Tags: []string{"дела"}, ReminderOffsets: &[]int{15},
	}, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

	rec := callSchedule(task.ID, user, map[string]any{
		"starts_at": "2026-10-20T09:00:00+09:00", "ends_at": "2026-10-20T10:00:00+09:00",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	ev := loadEvent(t, createdEventID(t, rec))

	if ev.Timezone != "Asia/Tokyo" {
		t.Errorf("timezone = %q, want Asia/Tokyo", ev.Timezone)
	}
	if ev.Description == nil || *ev.Description != "паспорт" || len(ev.Tags) != 1 {
		t.Errorf("description/tags not copied: %v %v", ev.Description, ev.Tags)
	}
	if len(ev.ReminderOffsets) != 1 || ev.ReminderOffsets[0] != 15 {
		t.Errorf("reminder_offsets = %v, want [15]", ev.ReminderOffsets)
	}
	// The web relies on SCHEDULED; the bot lists it next to TODO (A2). Denis 21.09.
	if _, st := taskExists(t, task.ID); st != "SCHEDULED" {
		t.Errorf("status = %q, want SCHEDULED — the web contract is unchanged", st)
	}
}

// 🔴 «Событие = время на задачу. Напоминания у события» (Denis 21.09) — and
// only there. Copying reminder_offsets onto the event made it remind for the
// first time; this asks the real scanner, before and after, whether the task
// then reminds as well. Two rows for one offset is two messages for one thing.
func TestALinkedTaskRemindsOnceThroughItsEvent(t *testing.T) {
	for _, door := range []string{"schedule", "convert"} {
		t.Run(door, func(t *testing.T) {
			d, ctx, user := repeatDB(t)
			reminders.InitDB(d)
			events.InitDB(d)
			if _, err := d.Pool.Exec(ctx, `UPDATE "user" SET tg_id = $2, timezone = 'UTC' WHERE id = $1`,
				user, time.Now().UnixNano()%1_000_000_000); err != nil {
				t.Fatalf("tg_id: %v", err)
			}

			tomorrow := time.Now().UTC().AddDate(0, 0, 1)
			due := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
			dueStr := due.Format(time.RFC3339)
			task, err := insertTask(ctx, user, CreateTaskRequest{
				Title: "банк", DueDate: &dueStr, ReminderOffsets: &[]int{10},
			}, &due)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM event WHERE task_id = $1`, task.ID) })
			t.Cleanup(func() { _, _ = d.Pool.Exec(ctx, `DELETE FROM task WHERE id = $1`, task.ID) })

			quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
			from, to := time.Now(), time.Now().Add(72*time.Hour)
			// First pass: the task's own reminder is already queued, as it would
			// be on a real server before anyone presses anything.
			if _, err := reminders.Scan(ctx, from, to, quiet); err != nil {
				t.Fatalf("scan: %v", err)
			}

			start := due.Add(9 * time.Hour)
			body := map[string]any{
				"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
			}
			var rec *httptest.ResponseRecorder
			if door == "schedule" {
				rec = callSchedule(task.ID, user, body)
			} else {
				body["mode"] = "link"
				rec = callConvert(task.ID, user, body)
			}
			if rec.Code != http.StatusCreated {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			evID := createdEventID(t, rec)

			if _, err := reminders.Scan(ctx, from, to, quiet); err != nil {
				t.Fatalf("scan: %v", err)
			}

			var total, onEvent int
			if err := d.Pool.QueryRow(ctx, `
				SELECT count(*), count(*) FILTER (WHERE event_id = $2)
				  FROM reminder WHERE user_id = $1 AND minutes_before = 10 AND status = 'PENDING'`,
				user, evID).Scan(&total, &onEvent); err != nil {
				t.Fatalf("count: %v", err)
			}
			if total != 1 || onEvent != 1 {
				t.Errorf("pending 10-minute reminders: %d, on the event: %d — want exactly one, the event's", total, onEvent)
			}
		})
	}
}
