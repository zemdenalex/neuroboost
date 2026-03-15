#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKUP_SCRIPT="$SCRIPT_DIR/backup.sh"

# Make backup script executable
chmod +x "$BACKUP_SCRIPT"

# Add cron job (daily at 03:00) if not already present
if crontab -l 2>/dev/null | grep -q "backup.sh"; then
  echo "Backup cron already installed"
else
  (crontab -l 2>/dev/null; echo "0 3 * * * $BACKUP_SCRIPT >> /root/backups/backup.log 2>&1") | crontab -
  echo "Backup cron installed: daily at 03:00"
fi

# Create backup directory
mkdir -p /root/backups
echo "Backup directory: /root/backups"
