---
name: add-endpoint
description: Add a new API endpoint — handler, types, route registration, frontend API wrapper
---

# Add a New API Endpoint

Follow this workflow when adding a new REST endpoint to NeuroBoost.

## Steps

### 1. Backend Handler

Create or update handler in `api-go/internal/{module}/`:

- **`handlers.go`** — HTTP handler functions
- **`types.go`** — Request/response structs

Follow existing patterns:
- Some packages use `NewHandler(db, cfg)` (events, tasks)
- Others use `InitDB(db)` + exported functions (auth, feedback)
- Check neighboring handlers to match the pattern

Response envelope: `{"data": ..., "error": "...", "meta": {...}}`
Use `util.RespondJSON(w, status, data)` and `util.RespondError(w, status, code, message)` from
`internal/util/response.go`. Not-yet-built feature: `util.Write501(w, feature, endpoint)`.
Authenticated user: `middleware.UserIDFromContext(r.Context())`.

### 2. Register Route

Add the route in `api-go/cmd/api/main.go` (find the router groups by reading the file — line
numbers drift):

- Public routes go in the top-level router group
- Protected routes go inside `r.Group(func(r chi.Router) { r.Use(middleware.JWT(...)) })`
- Follow RESTful conventions: GET list, POST create, GET/:id read, PATCH/:id update, DELETE/:id delete

### 3. Database (if needed)

- Use `pgx` parameterized queries: `db.QueryRow(ctx, "SELECT ... WHERE id = $1", id)`
- NEVER use string concatenation for SQL
- If schema changes needed, create a new migration (use `/add-migration` skill)

### 4. Frontend API Wrapper

Add API functions in `web/src/api/{module}.ts`:

```typescript
import { api } from './client';

export const moduleApi = {
  list: () => api.get<Module[]>('/api/modules'),
  get: (id: string) => api.get<Module>(`/api/modules/${id}`),
  create: (data: CreateModuleInput) => api.post<Module>('/api/modules', data),
  update: (id: string, data: UpdateModuleInput) => api.patch<Module>(`/api/modules/${id}`, data),
  delete: (id: string) => api.delete(`/api/modules/${id}`),
};
```

Export from `web/src/api/index.ts`.

### 5. Frontend Types

Add TypeScript interfaces in `web/src/types/`:
- 🔴 snake_case → camelCase conversion is **NOT** automatic. Write the converter next to the API
  module (`web/src/api/toTask.ts` is the pattern) and cover it with a test. Casting the raw JSON
  to a camelCase type compiles and silently yields `undefined` — that was bug T1.

### 6. Verify

```bash
cd api-go && go build ./cmd/api
cd web && pnpm typecheck && pnpm build
```
