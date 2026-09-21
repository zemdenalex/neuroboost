package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuroboost/api-go/internal/middleware"
)

func callListOccurrences(userID, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/occurrences?"+query, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
	rec := httptest.NewRecorder()
	ListOccurrencesHandler(rec, req)
	return rec
}

// Statistics counts the days a series was actually done (spec 22.09 §5). The
// days come through the same door the bot will use, and only the caller's.
func TestOccurrencesAreListedForARangeAndOnlyTheCallers(t *testing.T) {
	d, ctx, user := repeatDB(t)
	stranger := seedTaskUser(t, ctx, d, "stranger")

	day0 := userToday()
	task := seriesTask(t, ctx, user, "FREQ=DAILY", day0)
	for _, off := range []int{0, 1, 3} {
		if err := MarkOccurrence(ctx, user, task.ID, day0.AddDate(0, 0, off), StateDone); err != nil {
			t.Fatalf("mark +%d: %v", off, err)
		}
	}
	if err := MarkOccurrence(ctx, user, task.ID, day0.AddDate(0, 0, 2), StateSkipped); err != nil {
		t.Fatalf("skip: %v", err)
	}

	q := "from=" + day0.Format("2006-01-02") + "&to=" + day0.AddDate(0, 0, 2).Format("2006-01-02")
	rec := callListOccurrences(user, q)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Data []OccurrenceRow `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 3 {
		t.Fatalf("rows = %+v, want 3 (days 0,1 done; 2 skipped; 3 is outside the range)", got.Data)
	}
	states := map[string]string{}
	for _, r := range got.Data {
		states[r.Occurrence] = r.State
	}
	if states[day0.AddDate(0, 0, 2).Format("2006-01-02")] != StateSkipped {
		t.Errorf("states = %v", states)
	}

	other := callListOccurrences(stranger, q)
	var none struct {
		Data []OccurrenceRow `json:"data"`
	}
	_ = json.Unmarshal(other.Body.Bytes(), &none)
	if other.Code != http.StatusOK || len(none.Data) != 0 {
		t.Errorf("a stranger sees %d rows (status %d)", len(none.Data), other.Code)
	}
}

func TestOccurrencesRefuseABadOrHugeRange(t *testing.T) {
	_, _, user := repeatDB(t)
	for q, code := range map[string]string{
		"from=2026-01-01&to=2027-06-01": "RANGE_TOO_LARGE",
		"from=2026-02-01&to=2026-01-01": "INVALID_RANGE",
		"from=yesterday&to=2026-01-01":  "INVALID_RANGE",
		"":                              "INVALID_RANGE",
	} {
		rec := callListOccurrences(user, q)
		if rec.Code != http.StatusBadRequest || !json.Valid(rec.Body.Bytes()) ||
			!containsCode(rec.Body.Bytes(), code) {
			t.Errorf("%q → %d %s, want 400 %s", q, rec.Code, rec.Body.String(), code)
		}
	}
}

func containsCode(body []byte, code string) bool {
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(body, &e) == nil && e.Error.Code == code
}
