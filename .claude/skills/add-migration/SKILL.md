---
name: add-migration
description: Create a new database migration — SQL files, naming, testing
---

# Add a New Database Migration

Follow this workflow when modifying the PostgreSQL schema.

## Steps

### 1. Determine Next Migration Number

Check existing migrations in `api-go/migrations/`:
- Current latest: `000006_add_missing_columns`
- Next number: `000007`

### 2. Create Migration Files

Create both up and down files:

```
api-go/migrations/000007_description.up.sql
api-go/migrations/000007_description.down.sql
```

**Naming:** Use snake_case, descriptive: `add_`, `create_`, `alter_`, `drop_`

### 3. Write SQL

**Up migration** — apply the change:
```sql
ALTER TABLE users ADD COLUMN preferences JSONB DEFAULT '{}';
CREATE INDEX idx_users_preferences ON users USING GIN (preferences);
```

**Down migration** — reverse the change:
```sql
DROP INDEX IF EXISTS idx_users_preferences;
ALTER TABLE users DROP COLUMN IF EXISTS preferences;
```

### 4. Rules

- NEVER modify existing migration files — always create new ones
- Always include both up and down files
- Use `IF EXISTS` / `IF NOT EXISTS` for safety
- Test down migration reverses up migration cleanly
- Migrations auto-run on API container startup

### 5. Update Related Code

- Update Go structs in `internal/{module}/types.go` to match new schema
- Update frontend types in `web/src/types/` if exposed via API
- Update `docs/CODEBASE_MAP.md` migration table

### 6. Test

```bash
# Apply migration manually
migrate -path api-go/migrations -database "$DATABASE_URL" up

# Verify rollback works
migrate -path api-go/migrations -database "$DATABASE_URL" down 1
migrate -path api-go/migrations -database "$DATABASE_URL" up

# Rebuild to verify Go compiles with new types
cd api-go && go build ./cmd/api
```
