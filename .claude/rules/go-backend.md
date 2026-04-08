---
description: Go backend conventions for NeuroBoost API
paths:
  - "api-go/**"
---

# Go Backend Rules

- Use `pgx` parameterized queries (`$1`, `$2`) — NEVER string concatenation for SQL
- Follow existing handler pattern in the module: either `NewHandler(db, cfg)` or `InitDB(db)` + exported functions
- Response envelope: `util.JSON(w, status, data)` and `util.Error(w, status, msg)`
- Get authenticated user via `middleware.GetUserID(r)` — always check it exists
- Register routes in `cmd/api/main.go` — public routes in top group, protected inside JWT middleware group
- Run `go build ./cmd/api` after any changes
- Run `go test -v ./...` if tests exist for the modified package
- Use `context.Context` from request: `r.Context()`
- Use `json:"field_name"` tags (snake_case) for API responses
- Log errors with enough context but never expose internal details to clients
