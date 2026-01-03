# Foundation Fixes - Deployment Guide

## What's Included

This package contains foundation fixes for NeuroBoost v0.4.0:

1. **Fixed Healthcheck** - API now properly reports health status
2. **golang-migrate** - Database migrations work on existing databases  
3. **Auth Schema** - User table extended with authentication fields
4. **Authentication System** - Telegram Login + Email/Password + JWT
5. **Updated CI** - GitHub Actions workflow with proper testing

## Files Changed

```
docker-compose.yml                           # Updated healthcheck
neuroboost-api-go/
├── Dockerfile                               # Alpine image + migrate
├── go.mod                                   # Updated dependencies
├── cmd/
│   ├── api/main.go                         # Updated with config & DB
│   └── healthcheck/main.go                 # NEW: healthcheck binary
├── internal/
│   ├── config/config.go                    # NEW: env config
│   ├── database/db.go                      # NEW: pgx connection pool
│   ├── middleware/jwt.go                   # Updated JWT validation
│   ├── util/response.go                    # Updated JSON responses
│   ├── status/handlers.go                  # Updated with DB check
│   ├── auth/handlers.go                    # NEW: full auth system
│   ├── events/handlers.go                  # Updated imports
│   ├── tasks/handlers.go                   # Updated imports
│   ├── opportunities/handlers.go           # Updated imports
│   ├── needs/handlers.go                   # Updated imports
│   ├── reflections/handlers.go             # Updated imports
│   ├── patterns/handlers.go                # Updated imports
│   └── planning/handlers.go                # Updated imports
└── migrations/
    ├── 000001_baseline.up.sql              # Base schema (IF NOT EXISTS)
    ├── 000001_baseline.down.sql            # Rollback
    ├── 000002_add_auth_fields.up.sql       # Auth fields for user table
    └── 000002_add_auth_fields.down.sql     # Rollback
.github/workflows/ci.yml                     # Updated CI with migrations
```

## Deployment Steps

### 1. Backup Current Data (Safety First)

```bash
ssh deploy@62.76.228.106
cd /opt/neuroboost

# Create backup
docker compose exec db pg_dump -U neuroboost neuroboost > backup_$(date +%Y%m%d_%H%M%S).sql
```

### 2. Download and Extract Fixes

```bash
# On server
cd /opt/neuroboost

# Pull latest from git (if committed) OR
# Upload the zip file and extract

# Replace files (careful!)
cp -r ~/neuroboost-fixes/* .
```

### 3. Remove Old Migrations (One-time)

The old migrations in `docker-entrypoint-initdb.d` format need to be removed:

```bash
cd /opt/neuroboost/neuroboost-api-go

# Remove old numbered migration files
rm -f migrations/001_*.sql migrations/002_*.sql ... migrations/018_*.sql
```

### 4. Rebuild and Deploy

```bash
cd /opt/neuroboost

# Rebuild images
docker compose down
docker compose build --no-cache

# Start database first
docker compose up -d db
sleep 10

# Start API (will run migrations automatically)
docker compose up -d api

# Check logs
docker compose logs -f api
```

### 5. Verify

```bash
# Check health
curl -s https://neuroboost.website/api/health | jq

# Expected output:
# {
#   "status": "ok",
#   "service": "neuroboost-api",
#   "version": "0.4.0",
#   "database": "connected"
# }

# Check containers
docker compose ps
# All should show "healthy"

# Test auth endpoint
curl -s https://neuroboost.website/api/auth/me
# Should return 401 (no token)
```

### 6. Set Up Telegram Bot Domain

In Telegram, message @BotFather:
```
/setdomain
Select @NeuroBoost_assistant_bot
Enter: neuroboost.website
```

## Auth Endpoints

After deployment, these endpoints are available:

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/auth/telegram` | POST | No | Telegram Login Widget |
| `/api/auth/register` | POST | No | Email/password registration |
| `/api/auth/login` | POST | No | Email/password login |
| `/api/auth/me` | GET | JWT | Get current user |
| `/api/auth/logout` | POST | No | Logout (client-side) |

### Request Examples

**Register:**
```bash
curl -X POST https://neuroboost.website/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'
```

**Login:**
```bash
curl -X POST https://neuroboost.website/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

**Get Me (with token):**
```bash
curl https://neuroboost.website/api/auth/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Troubleshooting

### Migrations fail
```bash
# Check migration status
docker compose exec api migrate -path /srv/migrations -database "$DATABASE_URL" version

# Force to specific version if needed
docker compose exec api migrate -path /srv/migrations -database "$DATABASE_URL" force 2
```

### API won't start
```bash
# Check logs
docker compose logs api --tail=100

# Common issues:
# - DATABASE_URL wrong
# - JWT_SECRET not set
# - Migrations failed
```

### Auth not working
```bash
# Test directly
curl -v -X POST http://127.0.0.1:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123"}'
```

## Next Steps

After foundation is working:
1. Build frontend Login page
2. Test Telegram Login Widget integration
3. Implement Events/Tasks CRUD (replace stubs)
