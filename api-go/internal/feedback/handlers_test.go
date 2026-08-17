package feedback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// This package had zero tests until 2026-08-13, while POST /api/feedback is one
// of the four routes that decode a request body with NO authentication at all.
//
// Every case below stops before the handler reaches the database, which is why
// a Handler with a nil DB is enough: the point is precisely that these requests
// must never get that far. A case that needed a live pool would prove something
// else, and would skip silently without DATABASE_URL.
func nilDBHandler() *Handler { return &Handler{} }

func postFeedback(t *testing.T, body string, userID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(body))
	if userID != "" {
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
	}
	rec := httptest.NewRecorder()
	nilDBHandler().Create(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body util.Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not the standard envelope: %v", err)
	}
	if body.Error == nil {
		t.Fatalf("expected an error envelope, got %+v", body)
	}
	return body.Error.Code
}

func TestCreateRejectsUnknownType(t *testing.T) {
	rec := postFeedback(t, `{"type":"exploit","title":"t","description":"d"}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if code := errorCode(t, rec); code != "INVALID_TYPE" {
		t.Errorf("code = %q, want INVALID_TYPE", code)
	}
}

func TestCreateRejectsMissingFields(t *testing.T) {
	cases := map[string]string{
		"no title":       `{"type":"bug","title":"","description":"d"}`,
		"no description": `{"type":"bug","title":"t","description":""}`,
		"neither":        `{"type":"bug"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := postFeedback(t, body, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			if code := errorCode(t, rec); code != "MISSING_FIELDS" {
				t.Errorf("code = %q, want MISSING_FIELDS", code)
			}
		})
	}
}

func TestCreateRejectsMalformedBody(t *testing.T) {
	rec := postFeedback(t, `{"type":`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// The negative control. Without it, a Create that rejected everything — or one
// whose validation ran in the wrong order — would satisfy every test above.
// A valid body must get PAST validation, which with a nil pool means it panics
// on the query rather than answering 400.
func TestCreateAcceptsAValidBodyAndReachesTheDatabase(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a valid feedback body must reach the query; it was rejected before that")
		}
	}()
	postFeedback(t, `{"type":"bug","title":"t","description":"d"}`, "")
}

// List, Update and Import are admin-only, and the check is the whole of their
// security. Each must refuse an anonymous caller before touching anything.
func TestAdminOnlyHandlersRefuseAnonymousCallers(t *testing.T) {
	h := nilDBHandler()
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"List":   h.List,
		"Update": h.Update,
		"Import": h.Import,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/feedback", strings.NewReader(`{}`))
			call(rec, req)

			// Update answers 400 first when the id is missing from the URL; what
			// matters is that no path reaches the database for an anonymous
			// caller, i.e. nothing here is a 2xx.
			if rec.Code < 400 {
				t.Fatalf("anonymous caller got %d — must not succeed", rec.Code)
			}
		})
	}
}

// ORDER BY cannot take a placeholder, so the sort column is interpolated into
// the SQL string (handlers.go:242). That is safe ONLY because the value comes
// out of an allowlist map rather than from the query string. This test guards
// the property, not the implementation: if someone later widens the map or
// passes user input through, the injection is immediate and silent.
func TestSortColumnsAreAllowlistedAndInert(t *testing.T) {
	if len(validSortColumns) == 0 {
		t.Fatal("the allowlist is empty — every request would fall back silently")
	}
	safe := regexp.MustCompile(`^[a-z_]+$`)
	for key, col := range validSortColumns {
		if !safe.MatchString(col) {
			t.Errorf("sort column %q (from key %q) is not a bare identifier — it is interpolated into SQL", col, key)
		}
	}
	// The lookup must MISS for anything not listed; a map returning a zero
	// value that then got interpolated would produce broken SQL, and one
	// returning the input unchanged would be an injection.
	if _, ok := validSortColumns["created_at; DROP TABLE feedback--"]; ok {
		t.Error("an arbitrary string resolved to a sort column")
	}
}

// The other three allowlists gate values that DO travel as placeholders, so
// they are not an injection risk — but an empty one would reject every request,
// and that failure looks like a broken client rather than a broken server.
func TestValueAllowlistsAreNotEmpty(t *testing.T) {
	for name, m := range map[string]map[string]bool{
		"validTypes":      validTypes,
		"validStatuses":   validStatuses,
		"validPriorities": validPriorities,
	} {
		if len(m) == 0 {
			t.Errorf("%s is empty — every request carrying that field would be rejected", name)
		}
	}
}
