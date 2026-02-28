#!/usr/bin/env bash
set -euo pipefail

say() { printf "%s\n" "$*"; }
fail() { printf "error: %s\n" "$*" >&2; exit 1; }

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing $1. Please install it and try again."
  fi
}

require git
require go
require docker

if ! docker info >/dev/null 2>&1; then
  fail "Docker is not running or not accessible. Start Docker and try again."
fi

ROOT="${KRELLIN_ROOT:-$HOME/.krellin}"
if [[ ! -f "$ROOT/go.mod" ]]; then
  say "Cloning Krellin into $ROOT..."
  mkdir -p "$ROOT"
  git clone https://github.com/cbdeane/krellin "$ROOT"
fi

say "Building Krellin..."
(
  cd "$ROOT"
  go build -o "$ROOT/krellind" ./cmd/krellind
  go build -o "$ROOT/krellin" ./cmd/krellin
)

SOCK="${KRELLIN_SOCK:-/tmp/krellin.sock}"

say "Starting daemon..."
pkill -f "$ROOT/krellind" >/dev/null 2>&1 || true
pkill -f "krellind -sock $SOCK" >/dev/null 2>&1 || true
rm -f "$SOCK"

if [[ ! -S "$SOCK" ]]; then
  "$ROOT/krellind" -sock "$SOCK" >/tmp/krellind.log 2>&1 &
  sleep 0.5
fi

say "Launching TUI..."
"$ROOT/krellin" -sock "$SOCK"
