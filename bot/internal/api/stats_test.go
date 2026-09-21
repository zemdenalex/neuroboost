package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Both answers are {"data": …}; a method that decodes the bare body reports
// success with nothing in it — the OccurrenceResult lesson of 21.09.
func TestStatsReadsTheEnvelopes(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tasks/occurrences":
			query = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"task_id":"t1","occurrence":"2026-09-21","state":"done"}]}`))
		case "/api/reflections":
			_, _ = w.Write([]byte(`{"data":[{"id":"r1","created_at":"2026-09-21T18:00:00Z"}]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	occ, err := c.TaskOccurrences("tok", "2026-09-21", "2026-09-27")
	if err != nil || len(occ) != 1 || occ[0].State != "done" {
		t.Errorf("occurrences = %+v, %v", occ, err)
	}
	if query != "from=2026-09-21&to=2026-09-27" {
		t.Errorf("query = %q", query)
	}
	refl, err := c.Reflections("tok")
	if err != nil || len(refl) != 1 || refl[0].CreatedAt == "" {
		t.Errorf("reflections = %+v, %v", refl, err)
	}
}
