#!/usr/bin/env bash
set -euo pipefail

SOCK=${KRELLIN_SOCK:-/tmp/krellin.sock}
ROOT=$(pwd)

echo "Building binaries..."
go build -o "$ROOT/krellind" ./cmd/krellind
go build -o "$ROOT/krellin" ./cmd/krellin

echo "Stopping existing daemon (if any)..."
pkill -f "$ROOT/krellind" >/dev/null 2>&1 || true
pkill -f "krellind -sock $SOCK" >/dev/null 2>&1 || true
rm -f "$SOCK"

if [[ ! -S "$SOCK" ]]; then
  echo "Starting daemon..."
  "$ROOT/krellind" -sock "$SOCK" >/tmp/krellind.log 2>&1 &
  sleep 0.5
fi

echo "Starting TUI..."
"$ROOT/krellin" -sock "$SOCK"
