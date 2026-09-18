#!/usr/bin/env bash
# backup.sh — rsync the data directory (Badger + SQLite + badger snapshots)
# to a secondary storage host. Run nightly via cron after the SAE sync.
# Env: BACKUP_HOST (user@host:/path), DATA_DIR
set -euo pipefail

DATA_DIR="${DATA_DIR:-./data}"
BACKUP_HOST="${BACKUP_HOST:?set BACKUP_HOST=user@host:/backups/menguz}"
STAMP="$(date +%Y%m%d-%H%M%S)"

echo "→ backing up $DATA_DIR to $BACKUP_HOST/$STAMP"
rsync -avz --delete "$DATA_DIR/" "$BACKUP_HOST/$STAMP/"

# keep only the last 14 backups on the remote
ssh "${BACKUP_HOST%%:*}" "ls -1dt ${BACKUP_HOST##*:}/*/ | tail -n +15 | xargs -r rm -rf" || true
echo "✓ backup complete: $STAMP"
