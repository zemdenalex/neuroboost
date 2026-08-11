package calendars

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
)

// TestValidColor pins the hex-colour rule the handler enforces: a nil pointer
// (field omitted) is always valid — it means "not supplied" — while a
// non-nil pointer must be a strict 6-digit hex colour. An empty string is
// deliberately rejected rather than accepted as "clear the colour".
func TestValidColor(t *testing.T) {
	str := func(s string) *string { return &s }

	cases := []struct {
		name  string
		color *string
		want  bool
	}{
		{"nil is valid (omitted)", nil, true},
		{"lowercase hex", str("#7c3aed"), true},
		{"uppercase hex", str("#7C3AED"), true},
		{"empty string is invalid", str(""), false},
		{"missing hash", str("7c3aed"), false},
		{"too short", str("#7c3ae"), false},
		{"too long", str("#7c3aed1"), false},
		{"non-hex characters", str("red"), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validColor(c.color); got != c.want {
				t.Errorf("validColor(%v) = %v, want %v", c.color, got, c.want)
			}
		})
	}
}

// withDummyDB points the package-level db at a non-nil, never-connected
// *database.DB for the duration of a test. That makes caller() take its
// "authenticated?" branch instead of its "DB not initialized" branch,
// without ever dialing a real connection — every path exercised below
// returns before touching db.Pool. Restores the previous value on cleanup.
func withDummyDB(t *testing.T) {
	t.Helper()
	prev := db
	db = &database.DB{}
	t.Cleanup(func() { db = prev })
}

// assertErrorResponse decodes the standard util.RespondError envelope and
// checks both the HTTP status and the error.code string. Callers assert one
// error at a time (not as a table walked in aggregate) because the defect
// this guards against — two arms of respondCalendarError swapped — produces
// a response that is well-formed and merely wrong; only a per-case exact
// match catches that.
func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d (body: %s)", wantStatus, rec.Code, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, rec.Body.String())
	}
	if body.Error.Code != wantCode {
		t.Fatalf("expected error.code %q, got %q", wantCode, body.Error.Code)
	}
}

// --- respondCalendarError: the store-error -> HTTP mapping, pinned one arm
// at a time. This is the mapping that the 404-vs-403 security property (a
// non-member gets 404, never a 403 that would confirm the calendar exists)
// depends on, and nothing else in the suite exercises the handler's side of
// that translation — the store layer's own distinction between the two
// errors is tested in crud_test.go, not this one.

func TestRespondCalendarError_NotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, ErrCalendarNotFound)
	assertErrorResponse(t, rec, http.StatusNotFound, "CALENDAR_NOT_FOUND")
}

func TestRespondCalendarError_NotOwner(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, ErrNotCalendarOwner)
	assertErrorResponse(t, rec, http.StatusForbidden, "NOT_CALENDAR_OWNER")
}

func TestRespondCalendarError_IsPersonal(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, ErrCalendarIsPersonal)
	assertErrorResponse(t, rec, http.StatusConflict, "CALENDAR_IS_PERSONAL")
}

func TestRespondCalendarError_InvalidName(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, ErrInvalidName)
	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_NAME")
}

// TestRespondCalendarError_Unknown checks the fall-through default: an error
// respondCalendarError does not recognise still gets a well-formed 500 with
// a generic message, never the error's own (possibly internal) text.
func TestRespondCalendarError_Unknown(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, errors.New("pgx: connection to host db-primary.internal refused"))
	assertErrorResponse(t, rec, http.StatusInternalServerError, "CALENDAR_ERROR")

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Message != "Failed to process calendar request" {
		t.Fatalf("expected a generic message leaking no internal detail, got %q", body.Error.Message)
	}
}

// TestRespondCalendarError_NotEmpty asserts the one response body that does
// not go through util.RespondError at all: respondNotEmpty hand-builds the
// envelope, so its shape (code and message sitting beside events/tasks in
// the same "error" object) is only true by hand and only proven by decoding
// the actual bytes.
func TestRespondCalendarError_NotEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	respondCalendarError(rec, &NotEmptyError{Events: 12, Tasks: 3})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Events  int    `json:"events"`
			Tasks   int    `json:"tasks"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, rec.Body.String())
	}
	if body.Error.Code != "CALENDAR_NOT_EMPTY" {
		t.Fatalf("expected error.code CALENDAR_NOT_EMPTY, got %q", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Fatalf("expected a non-empty message beside the counts")
	}
	if body.Error.Events != 12 || body.Error.Tasks != 3 {
		t.Fatalf("expected events=12 tasks=3 inside the same error object, got events=%d tasks=%d", body.Error.Events, body.Error.Tasks)
	}
}

// --- 401: no authenticated user in the request context ---

func TestListHandler_Unauthenticated(t *testing.T) {
	withDummyDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/calendars", nil)
	rec := httptest.NewRecorder()
	ListHandler(rec, req)
	assertErrorResponse(t, rec, http.StatusUnauthorized, "NOT_AUTHENTICATED")
}

// --- Colour validation through the real handlers ---
//
// Only the rejection paths are exercised here through CreateHandler and
// UpdateHandler themselves: an invalid colour is rejected before either
// handler ever calls into the store, so these run with no database. The
// acceptance paths (a valid hex colour, an omitted colour) necessarily
// proceed past validation into Create/Update, which dereference the
// connection pool — asserting those through the real handler would need a
// live database, so that direction is covered only at the unit level by
// TestValidColor above. See the report for this limitation stated plainly.

func authedCreateRequest(body createRequest) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/calendars", bytes.NewReader(b))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "test-user")
	return req.WithContext(ctx)
}

func authedUpdateRequest(id string, body updateRequest) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/calendars/"+id, bytes.NewReader(b))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, "test-user")
	return req.WithContext(ctx)
}

func TestCreateHandler_RejectsMalformedColor(t *testing.T) {
	withDummyDB(t)
	color := "not-a-color"
	rec := httptest.NewRecorder()
	CreateHandler(rec, authedCreateRequest(createRequest{Name: "Team", Color: &color}))
	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_COLOR")
}

func TestCreateHandler_RejectsEmptyStringColor(t *testing.T) {
	withDummyDB(t)
	color := ""
	rec := httptest.NewRecorder()
	CreateHandler(rec, authedCreateRequest(createRequest{Name: "Team", Color: &color}))
	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_COLOR")
}

func TestUpdateHandler_RejectsMalformedColor(t *testing.T) {
	withDummyDB(t)
	color := "#zzzzzz"
	rec := httptest.NewRecorder()
	UpdateHandler(rec, authedUpdateRequest("some-id", updateRequest{Color: &color}))
	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_COLOR")
}

func TestUpdateHandler_RejectsEmptyStringColor(t *testing.T) {
	withDummyDB(t)
	color := ""
	rec := httptest.NewRecorder()
	UpdateHandler(rec, authedUpdateRequest("some-id", updateRequest{Color: &color}))
	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_COLOR")
}
