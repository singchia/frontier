#!/usr/bin/env bash
# Launch a local WireGuard-over-frontier demo: frontier + wg-router + two
# wg-edges + one udpping sender + one udpping echo. Ctrl-C tears everything
# down. Run from repo root:
#
#     ./examples/wireguard/scripts/demo.sh
#
# Prereqs: `make build-frontier` (repo root) and `make all` in
# examples/wireguard/ have both been run.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
EX="$REPO_ROOT/examples/wireguard"
BIN="$EX/bin"
FRONTIER_BIN="${FRONTIER_BIN:-$REPO_ROOT/bin/frontier}"
FRONTIER_CFG="${FRONTIER_CFG:-$REPO_ROOT/etc/frontier_udp.yaml}"

for f in "$FRONTIER_BIN" "$BIN/wg-router" "$BIN/wg-edge" "$BIN/udpping"; do
  if [[ ! -x "$f" ]]; then
    echo "missing binary: $f" >&2
    echo "run 'make' at repo root and 'make all' in examples/wireguard/" >&2
    exit 1
  fi
done
if [[ ! -f "$FRONTIER_CFG" ]]; then
  echo "missing frontier config: $FRONTIER_CFG" >&2
  exit 1
fi

LOG_DIR="$(mktemp -d)"
echo "logs: $LOG_DIR"

PIDS=()
cleanup() {
  echo "shutting down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

"$FRONTIER_BIN" -c "$FRONTIER_CFG" >"$LOG_DIR/frontier.log" 2>&1 &
PIDS+=($!)
sleep 0.5

"$BIN/wg-router" --frontier-addr 127.0.0.1:30011 --frontier-network udp \
  >"$LOG_DIR/router.log" 2>&1 &
PIDS+=($!)
sleep 0.3

"$BIN/wg-edge" --name edge-a --listen 127.0.0.1:51820 --pair-id demo \
  --frontier-addr 127.0.0.1:30012 --frontier-network udp \
  >"$LOG_DIR/edge-a.log" 2>&1 &
PIDS+=($!)

"$BIN/wg-edge" --name edge-b --listen 127.0.0.1:51821 --pair-id demo \
  --frontier-addr 127.0.0.1:30012 --frontier-network udp \
  >"$LOG_DIR/edge-b.log" 2>&1 &
PIDS+=($!)
sleep 0.5

"$BIN/udpping" --mode echo --listen 127.0.0.1:7001 --target 127.0.0.1:51821 \
  >"$LOG_DIR/udpping-echo.log" 2>&1 &
PIDS+=($!)
sleep 0.2

"$BIN/udpping" --mode send --listen 127.0.0.1:7000 --target 127.0.0.1:51820 \
  --interval 1s 2>&1 | tee "$LOG_DIR/udpping-send.log"
