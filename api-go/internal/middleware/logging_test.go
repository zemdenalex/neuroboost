package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The Admin Logs tab polls GET /api/admin/logs every 10 s, and each poll used
// to log itself: on a quiet server the tab showed mostly its own requests
// (audit 24.09, A7). That one path is not logged; everything else still is.
func TestRequestLoggerSkipsTheLogsPoll(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewTextHandler(&out, nil))
	h := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/admin/logs?limit=100", nil))
	if out.Len() != 0 {
		t.Fatalf("the logs poll was logged: %s", out.String())
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/tasks", nil))
	if !strings.Contains(out.String(), "path=/api/tasks") {
		t.Fatalf("an ordinary request was not logged: %q", out.String())
	}
}
