#!/usr/bin/env bash
# restore.sh — restore the latest backup into ./data. The server must be STOPPED.
# Usage: systemctl stop menguz && ./scripts/restore.sh [backup-path]
set -euo pipefail

DATA_DIR="${DATA_DIR:-./data}"
SRC="${1:?usage: restore.sh <backup-path> (e.g. user@host:/backups/menguz/20260918-030000)}"

echo "⚠ this will replace $DATA_DIR"
read -r -p "continue? [y/N] " ans
[[ "$ans" == "y" ]] || exit 1

mkdir -p "$DATA_DIR"
rsync -avz --delete "$SRC/" "$DATA_DIR/"
echo "✓ restored from $SRC — start the server: systemctl start menguz"
