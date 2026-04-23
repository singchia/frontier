# WireGuard over frontier

This example tunnels [WireGuard](https://www.wireguard.com/) UDP traffic
between two peers through a frontier instance. WireGuard is a UDP-only
protocol — when two peers cannot reach each other directly (NAT, separate
networks), this example lets them meet through frontier as a relay.

## Status

| piece | status |
|---|---|
| frame codec (2B length-prefix) | ✅ unit-tested under `-race` |
| wg-edge binary | ✅ works |
| wg-router binary | ✅ works |
| udpping test helper | ✅ works |
| demo script (TCP frontier transport) | ✅ verified end-to-end |
| UDP frontier transport | ⚠️ experimental — geminio handshake currently times out over the pion-wrapped UDP listener; flag is plumbed but the path doesn't complete handshake. Tracked as a frontier-framework issue, not an example bug. |
| real WireGuard walkthrough | 📝 documented below, not automated |

Default `--frontier-network` on all three binaries is `tcp`. Set to `udp`
once the framework-side handshake issue is resolved.

## Architecture

```
 host-A: wg0  ──UDP──►  wg-edge-A  ──stream──►  frontier  ──stream──►  wg-router  ──stream──►  frontier  ──stream──►  wg-edge-B  ──UDP──►  host-B: wg0
                                                                      (pair-id match)
```

- `wg-edge` listens UDP locally for the host's WireGuard peer, opens one
  geminio stream to frontier, writes the pair-id first, then shuttles
  datagrams as 2-byte length-prefixed frames.
- `wg-router` runs as a frontier service, reads the pair-id from each new
  stream, and once two streams share an id it forwards frames verbatim.

See the design doc for details: `docs/superpowers/specs/2026-04-21-wireguard-example-design.md`.

## One-command demo

From the repo root:

```bash
make && make -C examples/wireguard all && ./examples/wireguard/scripts/demo.sh
```

Expected output within a few seconds:

```
[udpping send] recv 7 bytes from 127.0.0.1:51820: "ping #1"
[udpping send] recv 7 bytes from 127.0.0.1:51820: "ping #2"
[udpping send] recv 7 bytes from 127.0.0.1:51820: "ping #3"
...
```

Ctrl-C tears everything down. Per-process logs land under the `logs:` path
printed on startup (a mktemp dir under `$TMPDIR`).

The script launches: `frontier` (TCP config `etc/frontier.yaml`),
`wg-router`, two `wg-edge` instances (ports 51820 / 51821, same
`--pair-id demo`), and two `udpping` processes (one `send`, one `echo`).
The path exercises:

```
udpping(send) → wg-edge-A → frontier → wg-router → frontier → wg-edge-B → udpping(echo)
                                                                              │
udpping(send) ← wg-edge-A ← frontier ← wg-router ← frontier ← wg-edge-B ←─────┘
```

### Piecewise build

If you prefer running build steps individually:

```bash
make                                  # build bin/frontier
make -C examples/wireguard all        # build wg-edge, wg-router, udpping
./examples/wireguard/scripts/demo.sh  # run the demo
```

### Running tests

```bash
go test ./examples/wireguard/internal/frame/... -race
```

## Real WireGuard (Linux)

Generate keys on both hosts:

```bash
wg genkey | tee privkey | wg pubkey > pubkey
```

On host-A, `/etc/wireguard/wg0.conf`:

```ini
[Interface]
PrivateKey = <A-priv>
Address    = 10.0.0.1/24
ListenPort = 51821

[Peer]
PublicKey           = <B-pub>
AllowedIPs          = 10.0.0.2/32
Endpoint            = 127.0.0.1:51820
PersistentKeepalive = 25
```

On host-B, `/etc/wireguard/wg0.conf`:

```ini
[Interface]
PrivateKey = <B-priv>
Address    = 10.0.0.2/24
ListenPort = 51821

[Peer]
PublicKey           = <A-pub>
AllowedIPs          = 10.0.0.1/32
Endpoint            = 127.0.0.1:51820
PersistentKeepalive = 25
```

On each host, start `frontier` (or point the edge at a shared one), then:

```bash
# wg-router (anywhere reachable by both hosts' frontier)
./bin/wg-router --frontier-addr <frontier>:30011 --frontier-network tcp

# on each host
./bin/wg-edge --name $(hostname) --listen 127.0.0.1:51820 --pair-id mytunnel \
  --frontier-addr <frontier>:30012 --frontier-network tcp

# bring up wg
sudo wg-quick up wg0

# verify
ping 10.0.0.2   # from host-A; reaches 10.0.0.1 from host-B
```

## Flags

### `wg-edge`

| flag | default | meaning |
|---|---|---|
| `--frontier-addr` | `127.0.0.1:30012` | frontier edgebound address |
| `--frontier-network` | `tcp` | `tcp` (works) or `udp` (experimental) |
| `--listen` | `127.0.0.1:51820` | UDP address wg0 sends to |
| `--pair-id` | `hello` | must match on both sides |
| `--service-name` | `wg` | router's service name |
| `--name` | `edge` | log prefix |

### `wg-router`

| flag | default | meaning |
|---|---|---|
| `--frontier-addr` | `127.0.0.1:30011` | frontier servicebound |
| `--frontier-network` | `tcp` | `tcp` (works) or `udp` (experimental) |
| `--service-name` | `wg` | registered service name |
| `--pair-timeout` | `60s` | max wait for a stream's partner |
| `--max-pair-id-len` | `256` | sanity cap on first-frame length |

### `udpping`

| flag | default | meaning |
|---|---|---|
| `--mode` | `send` | `send` or `echo` |
| `--listen` | `127.0.0.1:7000` | local UDP addr |
| `--target` | `127.0.0.1:51820` | dest (send) / seed target (echo) |
| `--interval` | `1s` | send period |
| `--payload` | `ping` | bytes to send |

## Caveats (read before using in production)

- **Not authenticated.** Any edge that knows the pair-id can join. The
  example deliberately stays minimal — rely on WG's own end-to-end crypto
  for confidentiality, and wrap with network-level ACLs or a HMAC
  pair-id layer if you need access control.
- **Stream over reliable transport adds head-of-line blocking.** A lost
  packet stalls subsequent WG datagrams until recovery. On lossy links
  expect worse behaviour than raw WG. This is inherent to tunnelling UDP
  over any reliable substrate.
- **`B` must occasionally send first.** The edge learns the local reply
  address from the first datagram it receives. Configure
  `PersistentKeepalive` on both peers so both sides always produce
  traffic.
