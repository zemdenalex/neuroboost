---
description: Go backend conventions for NeuroBoost API
paths:
  - "api-go/**"
---

# Go Backend Rules

- Use `pgx` parameterized queries (`$1`, `$2`) — NEVER string concatenation for SQL
- Follow existing handler pattern in the module: either `NewHandler(db, cfg)` or `InitDB(db)` + exported functions
- Response envelope: `util.RespondJSON(w, status, data)` and `util.RespondError(w, status, code, message)`
  (`internal/util/response.go`). Unimplemented feature: `util.Write501(w, feature, endpoint)`
- Get authenticated user via `middleware.UserIDFromContext(r.Context())` — always check it exists
- Register routes in `cmd/api/main.go` — public routes in top group, protected inside JWT middleware group
- Run `go build ./cmd/api` after any changes
- Run `go test -v ./...` if tests exist for the modified package
- 🔴 Every background goroutine needs `recover()` at its top level. A panic in any goroutine
  terminates the whole process — the reminder ticker took the entire API down every 60s
- 🔴 `bot/` is a SEPARATE Go module. `go build ./...` from `api-go` never enters it, so a broken
  bot passes a green CI. Verify both: `cd api-go && go build ./... && go test ./...` **and**
  `cd bot && go build ./... && go test ./...`
- Use `context.Context` from request: `r.Context()`
- Use `json:"field_name"` tags (snake_case) for API responses
- Log errors with enough context but never expose internal details to clients
