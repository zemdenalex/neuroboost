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

ARCHIVE="$BACKUP_DIR/neuroboost_${TIMESTAMP}.sql.gz"

# 🔴 Verify the artefact, do not just report that the command ran.
#
# Between 2026-04-23 and 2026-08-14 this script produced nothing at all: it was
# committed without an execute bit, so `./scripts/backup.sh` in the deploy job
# failed instantly — and that job swallowed the failure with `|| echo`. Four
# months of releases each logged "Backup failed, continuing deploy..." and
# nobody read it. Production went from one backup to zero without a single
# alarm.
#
# `set -o pipefail` above catches a pg_dump that exits non-zero. It does NOT
# catch a dump that succeeds and writes nothing useful — an empty database, a
# truncated pipe, a wrong -U. These three checks do.

if ! gzip -t "$ARCHIVE" 2>/dev/null; then
  echo "FATAL: $ARCHIVE is not a readable gzip archive" >&2
  exit 1
fi

# A schema-only dump of this database is already several KB; anything smaller
# means the data did not make it.
BYTES=$(stat -c%s "$ARCHIVE")
if [ "$BYTES" -lt 2000 ]; then
  echo "FATAL: $ARCHIVE is only ${BYTES} bytes — too small to contain the database" >&2
  exit 1
fi

# The table that must always have rows. A dump with no users is a dump of the
# wrong database, or of an empty one.
if ! zcat "$ARCHIVE" | grep -q 'COPY public."user"'; then
  echo "FATAL: $ARCHIVE contains no \"user\" table data" >&2
  exit 1
fi

SIZE=$(du -h "$ARCHIVE" | cut -f1)
echo "Backup complete and verified: neuroboost_${TIMESTAMP}.sql.gz (${SIZE}, ${BYTES} bytes)"
