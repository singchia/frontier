# WireGuard over frontier

This example tunnels [WireGuard](https://www.wireguard.com/) UDP traffic
between two peers through a frontier instance. WireGuard is a UDP-only
protocol — when two peers cannot reach each other directly (NAT, separate
networks), this example lets them meet through frontier as a relay.

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

## Build

From the repo root:

```bash
make                                   # build frontier
cd examples/wireguard && make all      # build wg-edge, wg-router, udpping
```

## Quick demo (no real WireGuard needed)

```bash
./examples/wireguard/scripts/demo.sh
```

This starts frontier, wg-router, two wg-edges, and two udpping processes
(one sending, one echoing). You should see lines like:

```
[udpping send] recv 8 bytes from 127.0.0.1:51820: "ping #1"
[udpping send] recv 8 bytes from 127.0.0.1:51820: "ping #2"
```

Ctrl-C tears everything down.

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
./bin/wg-router --frontier-addr <frontier>:30011 --frontier-network udp

# on each host
./bin/wg-edge --name $(hostname) --listen 127.0.0.1:51820 --pair-id mytunnel \
  --frontier-addr <frontier>:30012 --frontier-network udp

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
| `--frontier-network` | `udp` | `tcp` or `udp` |
| `--listen` | `127.0.0.1:51820` | UDP address wg0 sends to |
| `--pair-id` | `hello` | must match on both sides |
| `--service-name` | `wg` | router's service name |
| `--name` | `edge` | log prefix |

### `wg-router`

| flag | default | meaning |
|---|---|---|
| `--frontier-addr` | `127.0.0.1:30011` | frontier servicebound |
| `--frontier-network` | `udp` | `tcp` or `udp` |
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
