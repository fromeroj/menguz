#!/usr/bin/env bash
# deploy.sh — scp the binary to the VPS and restart the service.
# Env: VPS_USER, VPS_HOST (or edit defaults). Usage: make deploy
set -euo pipefail

VPS_USER="${VPS_USER:-root}"
VPS_HOST="${VPS_HOST:-menguz.konen.guru}"
BIN="bin/menguz-server-linux-amd64"

echo "→ building linux binary"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$BIN" .
echo "→ uploading to $VPS_USER@$VPS_HOST:/opt/menguz/"
scp "$BIN" "$VPS_USER@$VPS_HOST:/opt/menguz/menguz-server"
echo "→ restarting service"
ssh "$VPS_USER@$VPS_HOST" "systemctl restart menguz && systemctl status menguz --no-pager -l | head -5"
echo "✓ deployed"
