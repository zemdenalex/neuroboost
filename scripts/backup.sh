#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/root/backups}"
COMPOSE_FILE="${COMPOSE_FILE:-/opt/neuroboost/docker-compose.yml}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

echo "Starting backup..."
docker compose -f "$COMPOSE_FILE" exec -T db \
  pg_dump -U neuroboost neuroboost | gzip > "$BACKUP_DIR/neuroboost_${TIMESTAMP}.sql.gz"

# Rotate: delete backups older than 7 days
find "$BACKUP_DIR" -name "neuroboost_*.sql.gz" -mtime +7 -delete

SIZE=$(du -h "$BACKUP_DIR/neuroboost_${TIMESTAMP}.sql.gz" | cut -f1)
echo "Backup complete: neuroboost_${TIMESTAMP}.sql.gz (${SIZE})"
