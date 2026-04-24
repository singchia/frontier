#!/usr/bin/env bash
# Launch a local WireGuard-over-frontier demo from the repo root:
#
#     ./examples/wireguard/scripts/demo.sh
#
# On macOS, keep the original lightweight demo: frontier + wg-router + two
# wg-edges + udpping send/echo.
#
# On Linux, if run as root with `ip` and `wg` installed, perform a real
# end-to-end WireGuard test in two temporary network namespaces and verify
# connectivity with `ping`.
set -euo pipefail

HOLD_OPEN=0
DETACH=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --hold)
      HOLD_OPEN=1
      shift
      ;;
    --detach)
      DETACH=1
      HOLD_OPEN=1
      shift
      ;;
    -h|--help)
      cat <<'EOF'
usage: ./examples/wireguard/scripts/demo.sh [--hold] [--detach]

  --hold    Keep the demo running after Linux verification succeeds.
            Stop it with Ctrl-C.
  --detach  Run the demo in the background. Implies --hold.
EOF
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 1
      ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
EX="$REPO_ROOT/examples/wireguard"
BIN="$EX/bin"
FRONTIER_BIN="${FRONTIER_BIN:-$REPO_ROOT/bin/frontier}"
# TCP config by default — the UDP path (etc/frontier_udp.yaml) has a known
# handshake-timeout issue with geminio (tracked separately); TCP works today.
FRONTIER_CFG="${FRONTIER_CFG:-$REPO_ROOT/etc/frontier.yaml}"
FRONTIER_NET="${FRONTIER_NET:-tcp}"
FRONTIER_SERVICE_ADDR="${FRONTIER_SERVICE_ADDR:-127.0.0.1:30011}"
FRONTIER_EDGE_ADDR="${FRONTIER_EDGE_ADDR:-127.0.0.1:30012}"
OS="$(uname -s)"

if [[ "$DETACH" -eq 1 && "${DEMO_ALREADY_DETACHED:-0}" -ne 1 ]]; then
  DETACHED_LOG_DIR="$(mktemp -d)"
  DETACHED_CONSOLE_LOG="$DETACHED_LOG_DIR/console.log"
  echo "starting detached demo..."
  echo "logs: $DETACHED_LOG_DIR"
  nohup env DEMO_ALREADY_DETACHED=1 LOG_DIR="$DETACHED_LOG_DIR" \
    FRONTIER_BIN="$FRONTIER_BIN" FRONTIER_CFG="$FRONTIER_CFG" \
    FRONTIER_NET="$FRONTIER_NET" FRONTIER_SERVICE_ADDR="$FRONTIER_SERVICE_ADDR" \
    FRONTIER_EDGE_ADDR="$FRONTIER_EDGE_ADDR" \
    "$0" --hold >"$DETACHED_CONSOLE_LOG" 2>&1 &
  echo "pid: $!"
  exit 0
fi

LOG_DIR="${LOG_DIR:-$(mktemp -d)}"
RUNTIME_DIR="$(mktemp -d)"
echo "logs: $LOG_DIR"

PIDS=()
NETNS_A="frontier-wg-a-$$"
NETNS_B="frontier-wg-b-$$"
ID_SUFFIX="$(( $$ % 10000 ))"
VETH_A_HOST="fwa${ID_SUFFIX}h"
VETH_A_NS="fwa${ID_SUFFIX}n"
VETH_B_HOST="fwb${ID_SUFFIX}h"
VETH_B_NS="fwb${ID_SUFFIX}n"
HOST_A_IP="10.200.1.1/24"
NS_A_IP="10.200.1.2/24"
HOST_B_IP="10.200.2.1/24"
NS_B_IP="10.200.2.2/24"
HOST_A_ADDR="10.200.1.1"
HOST_B_ADDR="10.200.2.1"
WG_A_ADDR="10.44.0.1/24"
WG_B_ADDR="10.44.0.2/24"

require_file() {
  local path="$1"
  if [[ ! -x "$path" ]]; then
    echo "missing binary: $path" >&2
    echo "run 'make' at repo root and 'make all' in examples/wireguard/" >&2
    exit 1
  fi
}

cleanup() {
  echo "shutting down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true

  if [[ "$OS" == "Linux" ]]; then
    ip netns del "$NETNS_A" 2>/dev/null || true
    ip netns del "$NETNS_B" 2>/dev/null || true
    ip link del "$VETH_A_HOST" 2>/dev/null || true
    ip link del "$VETH_B_HOST" 2>/dev/null || true
  fi

  rm -rf "$RUNTIME_DIR"
}
trap cleanup EXIT INT TERM

start_frontier_stack() {
  "$FRONTIER_BIN" --config "$FRONTIER_CFG" >"$LOG_DIR/frontier.log" 2>&1 &
  PIDS+=($!)

  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if grep -q "servicebound server listening" "$LOG_DIR/frontier.log" 2>/dev/null \
       && grep -q "edgebound server listening" "$LOG_DIR/frontier.log" 2>/dev/null; then
      break
    fi
    sleep 0.5
  done

  "$BIN/wg-router" --frontier-addr "$FRONTIER_SERVICE_ADDR" --frontier-network "$FRONTIER_NET" \
    >"$LOG_DIR/router.log" 2>&1 &
  PIDS+=($!)
  sleep 0.5
}

run_macos_demo() {
  require_file "$BIN/udpping"
  start_frontier_stack

  "$BIN/wg-edge" --name edge-a --listen 127.0.0.1:51820 --pair-id demo \
    --frontier-addr "$FRONTIER_EDGE_ADDR" --frontier-network "$FRONTIER_NET" \
    >"$LOG_DIR/edge-a.log" 2>&1 &
  PIDS+=($!)

  "$BIN/wg-edge" --name edge-b --listen 127.0.0.1:51821 --pair-id demo \
    --frontier-addr "$FRONTIER_EDGE_ADDR" --frontier-network "$FRONTIER_NET" \
    >"$LOG_DIR/edge-b.log" 2>&1 &
  PIDS+=($!)
  sleep 1

  "$BIN/udpping" --mode echo --listen 127.0.0.1:7001 --target 127.0.0.1:51821 \
    >"$LOG_DIR/udpping-echo.log" 2>&1 &
  PIDS+=($!)
  sleep 0.5

  "$BIN/udpping" --mode send --listen 127.0.0.1:7000 --target 127.0.0.1:51820 \
    --interval 1s 2>&1 | tee "$LOG_DIR/udpping-send.log"
}

linux_requirements() {
  if [[ "$EUID" -ne 0 ]]; then
    echo "linux real-wireguard mode requires root; rerun with sudo" >&2
    exit 1
  fi
  for cmd in ip wg ping; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "missing command: $cmd" >&2
      exit 1
    fi
  done
}

setup_linux_netns() {
  ip netns add "$NETNS_A"
  ip netns add "$NETNS_B"

  ip link add "$VETH_A_HOST" type veth peer name "$VETH_A_NS"
  ip link add "$VETH_B_HOST" type veth peer name "$VETH_B_NS"

  ip link set "$VETH_A_NS" netns "$NETNS_A"
  ip link set "$VETH_B_NS" netns "$NETNS_B"

  ip addr add "$HOST_A_IP" dev "$VETH_A_HOST"
  ip addr add "$HOST_B_IP" dev "$VETH_B_HOST"
  ip link set "$VETH_A_HOST" up
  ip link set "$VETH_B_HOST" up

  ip netns exec "$NETNS_A" ip link set lo up
  ip netns exec "$NETNS_A" ip addr add "$NS_A_IP" dev "$VETH_A_NS"
  ip netns exec "$NETNS_A" ip link set "$VETH_A_NS" up
  ip netns exec "$NETNS_A" ip route add default via "$HOST_A_ADDR"

  ip netns exec "$NETNS_B" ip link set lo up
  ip netns exec "$NETNS_B" ip addr add "$NS_B_IP" dev "$VETH_B_NS"
  ip netns exec "$NETNS_B" ip link set "$VETH_B_NS" up
  ip netns exec "$NETNS_B" ip route add default via "$HOST_B_ADDR"
}

generate_keypair() {
  local prefix="$1"
  local priv pub
  priv="$(wg genkey)"
  pub="$(printf '%s' "$priv" | wg pubkey)"
  printf '%s\n' "$priv" >"$RUNTIME_DIR/$prefix.key"
  printf '%s\n' "$pub" >"$RUNTIME_DIR/$prefix.pub"
}

configure_linux_wireguard() {
  generate_keypair a
  generate_keypair b

  ip netns exec "$NETNS_A" ip link add wg0 type wireguard
  ip netns exec "$NETNS_A" ip addr add "$WG_A_ADDR" dev wg0
  ip netns exec "$NETNS_A" wg set wg0 \
    private-key "$RUNTIME_DIR/a.key" \
    listen-port 51821 \
    peer "$(cat "$RUNTIME_DIR/b.pub")" \
    allowed-ips 10.44.0.2/32 \
    endpoint 127.0.0.1:51820 \
    persistent-keepalive 5
  ip netns exec "$NETNS_A" ip link set wg0 up

  ip netns exec "$NETNS_B" ip link add wg0 type wireguard
  ip netns exec "$NETNS_B" ip addr add "$WG_B_ADDR" dev wg0
  ip netns exec "$NETNS_B" wg set wg0 \
    private-key "$RUNTIME_DIR/b.key" \
    listen-port 51821 \
    peer "$(cat "$RUNTIME_DIR/a.pub")" \
    allowed-ips 10.44.0.1/32 \
    endpoint 127.0.0.1:51820 \
    persistent-keepalive 5
  ip netns exec "$NETNS_B" ip link set wg0 up
}

run_linux_real_wireguard() {
  linux_requirements
  setup_linux_netns
  start_frontier_stack

  ip netns exec "$NETNS_A" "$BIN/wg-edge" --name edge-a --listen 127.0.0.1:51820 --pair-id demo \
    --frontier-addr "${FRONTIER_EDGE_ADDR/127.0.0.1/$HOST_A_ADDR}" --frontier-network "$FRONTIER_NET" \
    >"$LOG_DIR/edge-a.log" 2>&1 &
  PIDS+=($!)

  ip netns exec "$NETNS_B" "$BIN/wg-edge" --name edge-b --listen 127.0.0.1:51820 --pair-id demo \
    --frontier-addr "${FRONTIER_EDGE_ADDR/127.0.0.1/$HOST_B_ADDR}" --frontier-network "$FRONTIER_NET" \
    >"$LOG_DIR/edge-b.log" 2>&1 &
  PIDS+=($!)
  sleep 1

  configure_linux_wireguard

  local ok=0
  for _ in $(seq 1 20); do
    if ip netns exec "$NETNS_A" ping -n -c 1 -W 1 10.44.0.2 >"$LOG_DIR/ping-a-to-b.log" 2>&1; then
      ok=1
      break
    fi
    sleep 1
  done

  if [[ "$ok" -ne 1 ]]; then
    echo "real WireGuard verification failed; inspect $LOG_DIR" >&2
    ip netns exec "$NETNS_A" wg show >"$LOG_DIR/wg-a-show.log" 2>&1 || true
    ip netns exec "$NETNS_B" wg show >"$LOG_DIR/wg-b-show.log" 2>&1 || true
    exit 1
  fi

  ip netns exec "$NETNS_B" ping -n -c 1 -W 1 10.44.0.1 >"$LOG_DIR/ping-b-to-a.log" 2>&1
  ip netns exec "$NETNS_A" wg show >"$LOG_DIR/wg-a-show.log" 2>&1 || true
  ip netns exec "$NETNS_B" wg show >"$LOG_DIR/wg-b-show.log" 2>&1 || true
  echo "real WireGuard verification succeeded"
  echo "  $NETNS_A: 10.44.0.1/24"
  echo "  $NETNS_B: 10.44.0.2/24"

  if [[ "$HOLD_OPEN" -eq 1 ]]; then
    echo "holding demo open; press Ctrl-C to stop"
    while true; do
      sleep 3600
    done
  fi
}

require_file "$FRONTIER_BIN"
require_file "$BIN/wg-router"
require_file "$BIN/wg-edge"

if [[ ! -f "$FRONTIER_CFG" ]]; then
  echo "missing frontier config: $FRONTIER_CFG" >&2
  exit 1
fi

case "$OS" in
  Darwin)
    echo "mode: macOS demo (udpping)"
    run_macos_demo
    ;;
  Linux)
    echo "mode: linux real WireGuard (netns)"
    run_linux_real_wireguard
    ;;
  *)
    echo "unsupported OS: $OS" >&2
    exit 1
    ;;
esac
