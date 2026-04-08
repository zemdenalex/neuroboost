---
name: deploy
description: Deploy NeuroBoost to production server — backup DB, pull, rebuild, verify
disable-model-invocation: true
---

# Deploy NeuroBoost to Production

Deploy the current branch to production at neuroboost.website.

## Pre-flight Checks

1. Ensure all changes are committed and pushed
2. Verify `pnpm typecheck && pnpm build` passes locally
3. Verify `go build ./cmd/api` passes locally
4. Check CI status on GitHub — must be green

## Deployment Steps

1. **SSH to server:**
   ```bash
   ssh root@62.76.228.106
   ```

2. **Backup database:**
   ```bash
   cd /root/neuroboost
   docker compose exec db pg_dump -U neuroboost neuroboost > backups/backup_$(date +%Y%m%d_%H%M%S).sql
   ```

3. **Pull latest code:**
   ```bash
   git pull origin main
   ```

4. **Rebuild and restart:**
   ```bash
   docker compose down
   docker compose up -d --build
   ```

5. **Verify deployment:**
   ```bash
   # Wait for containers to be healthy
   sleep 10

   # Check health endpoint
   curl -s https://neuroboost.website/api/health | jq .

   # Check all containers are running
   docker compose ps

   # Check logs for errors
   docker compose logs --tail=20 api
   docker compose logs --tail=20 web
   ```

6. **Verify in browser:** Open https://neuroboost.website and check:
   - Page loads without errors
   - Login works
   - Console has no errors

## Rollback

If something goes wrong:
```bash
# Restore database
docker compose exec -T db psql -U neuroboost neuroboost < backups/backup_YYYYMMDD_HHMMSS.sql

# Revert code
git revert HEAD
docker compose up -d --build
```

## Reference
See @DEPLOY.md for full deployment documentation.
See @scripts/redeploy.sh for automated deployment script.
