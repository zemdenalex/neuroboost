package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

// This package had zero tests until 2026-08-13. Both handlers expose operator
// data — database timings and application logs — and the admin check is the
// entirety of what stands between that data and any authenticated user.
//
// The cases below stop before the handlers reach the pool, so a Handler with a
// nil DB is the right instrument: what is under test is that these requests
// never get that far.

func handlerCases(h *Handler) map[string]func(http.ResponseWriter, *http.Request) {
	return map[string]func(http.ResponseWriter, *http.Request){
		"Health": h.Health,
		"Logs":   h.Logs,
	}
}

// lazyBrokenDB builds a pool that CONNECTS ON FIRST USE and always fails.
//
// 🔴 Do not reach for database.New here: it pings during construction, so an
// unreachable DSN fails immediately and the t.Skip that used to guard it turned
// this whole test into a silent no-op. It skipped in CI and locally, and it was
// reported as covering the refusal path when it covered nothing.
// pgxpool.NewWithConfig does not dial until the first query, which is exactly
// the failure being tested: a pool that looks fine and fails when used.
func lazyBrokenDB(t *testing.T) *database.DB {
	t.Helper()
	// Port 1 is reserved and never listening.
	cfg, err := pgxpool.ParseConfig("postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("could not parse the test DSN: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("could not build a lazy pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &database.DB{Pool: pool}
}

func TestRefusesAnonymousCaller(t *testing.T) {
	for name, call := range handlerCases(&Handler{}) {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			call(rec, httptest.NewRequest(http.MethodGet, "/api/admin/health", nil))

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			var body util.Response
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("response is not the standard envelope: %v", err)
			}
			if body.Error == nil || body.Error.Code != "NOT_AUTHENTICATED" {
				t.Fatalf("error = %+v, want NOT_AUTHENTICATED", body.Error)
			}
		})
	}
}

// A user id in the context is authentication, not authorisation. When the admin
// lookup cannot be answered — the database is down, the query fails — the
// handler must refuse, not serve.
//
// 🔴 What this test does NOT prove, stated so nobody reads more into it than it
// carries: it cannot tell the `err != nil` clause apart from `!isAdmin`.
// pgx leaves the destination at its zero value when Scan fails, so isAdmin is
// false on error either way, and deleting the error check keeps this test
// green. Verified by doing exactly that on 13.08.
//
// What it DOES prove is still worth having: on a failing lookup these handlers
// answer 403 rather than panicking, hanging, or falling through to the operator
// data below. Distinguishing the two clauses needs a live database where the
// query can succeed with is_admin = false.
func TestRefusesWhenTheAdminLookupCannotBeAnswered(t *testing.T) {
	h := &Handler{db: lazyBrokenDB(t)}
	for name, call := range handlerCases(h) {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "some-user"))

			rec := httptest.NewRecorder()
			defer func() {
				// A panic here would mean the handler reached the pool before
				// deciding — which is itself the defect this test guards.
				if p := recover(); p != nil {
					t.Fatalf("handler touched the database before refusing: %v", p)
				}
			}()
			call(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 — an unresolvable admin check must refuse", rec.Code)
			}
			var body util.Response
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("response is not the standard envelope: %v", err)
			}
			if body.Error == nil || body.Error.Code != "NOT_ADMIN" {
				t.Fatalf("error = %+v, want NOT_ADMIN", body.Error)
			}
		})
	}
}
