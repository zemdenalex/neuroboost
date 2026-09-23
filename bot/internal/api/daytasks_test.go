package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// «Задачи дня» D2: the bot's client for the D1 endpoints. Each test pins the
// path, the query and the body — the only contract between the two modules.

type dayCall struct {
	method, path, query, body string
}

func dayServer(t *testing.T, status int, reply string) (*Client, *[]dayCall) {
	t.Helper()
	var calls []dayCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, dayCall{r.Method, r.URL.Path, r.URL.RawQuery, string(b)})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL), &calls
}

const oneDay = `{"day":"2026-09-23","target":5,"confirmed":true,"items":[{"task_id":"a","title":"отчёт","done":true}],"done":1,"level":1}`

func TestDayTasksReadsARange(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":[`+oneDay+`]}`)
	days, err := c.DayTasks("jwt", "2026-09-23", "2026-09-23")
	if err != nil || len(days) != 1 {
		t.Fatalf("days %v, err %v", days, err)
	}
	d := days[0]
	if d.Day != "2026-09-23" || d.Target != 5 || !d.Confirmed || d.Done != 1 || d.Level != 1 ||
		len(d.Items) != 1 || d.Items[0].TaskID != "a" || d.Items[0].Title != "отчёт" || !d.Items[0].Done {
		t.Errorf("decoded %+v", d)
	}
	if got := (*calls)[0]; got.method != "GET" || got.path != "/api/day-tasks" || got.query != "from=2026-09-23&to=2026-09-23" {
		t.Errorf("call %+v", got)
	}
}

func TestDayProposalAsksForOneDay(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":[{"task_id":"a","title":"отчёт","done":false}]}`)
	items, err := c.DayProposal("jwt", "2026-09-23")
	if err != nil || len(items) != 1 || items[0].TaskID != "a" {
		t.Fatalf("items %v, err %v", items, err)
	}
	if got := (*calls)[0]; got.path != "/api/day-tasks/proposal" || got.query != "day=2026-09-23" {
		t.Errorf("call %+v", got)
	}
}

func TestConfirmDaySendsTheIDs(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":`+oneDay+`}`)
	d, err := c.ConfirmDay("jwt", "2026-09-23", []string{"a", "b"})
	if err != nil || !d.Confirmed {
		t.Fatalf("day %+v, err %v", d, err)
	}
	got := (*calls)[0]
	var body map[string]any
	_ = json.Unmarshal([]byte(got.body), &body)
	if got.method != "POST" || got.path != "/api/day-tasks/confirm" || body["day"] != "2026-09-23" ||
		len(body["task_ids"].([]any)) != 2 {
		t.Errorf("call %+v", got)
	}
}

// An empty confirm still sends a list, not null: «take the day with nothing
// pinned» is a real press.
func TestConfirmDayWithNoTasksSendsAnEmptyList(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":`+oneDay+`}`)
	if _, err := c.ConfirmDay("jwt", "2026-09-23", nil); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte((*calls)[0].body), &body)
	if ids, ok := body["task_ids"].([]any); !ok || len(ids) != 0 {
		t.Errorf("task_ids = %v, want []", body["task_ids"])
	}
}

func TestAddDayTaskPostsOneTask(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":`+oneDay+`}`)
	if _, err := c.AddDayTask("jwt", "2026-09-24", "a"); err != nil {
		t.Fatal(err)
	}
	got := (*calls)[0]
	var body map[string]any
	_ = json.Unmarshal([]byte(got.body), &body)
	if got.method != "POST" || got.path != "/api/day-tasks" || body["day"] != "2026-09-24" || body["task_id"] != "a" {
		t.Errorf("call %+v", got)
	}
}

func TestRemoveDayTaskDeletesByPath(t *testing.T) {
	c, calls := dayServer(t, 200, `{"data":`+oneDay+`}`)
	if err := c.RemoveDayTask("jwt", "2026-09-23", "a"); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[0]; got.method != "DELETE" || got.path != "/api/day-tasks/2026-09-23/a" {
		t.Errorf("call %+v", got)
	}
}

// 🔴 The code of a refused DELETE is what lets the bot say «убрать можно только
// до 12:00». del() used to throw the body away and return a code-less error.
func TestARefusedRemoveCarriesItsCode(t *testing.T) {
	c, _ := dayServer(t, 409, `{"error":{"code":"TOO_LATE","message":"too late"}}`)
	if err := c.RemoveDayTask("jwt", "2026-09-23", "a"); CodeOf(err) != "TOO_LATE" {
		t.Errorf("err = %v (code %q), want TOO_LATE", err, CodeOf(err))
	}
}
