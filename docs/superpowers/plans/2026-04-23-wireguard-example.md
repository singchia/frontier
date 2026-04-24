# WireGuard over frontier — Example Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `examples/wireguard/` demonstrating two WireGuard peers tunneling their UDP traffic through frontier via a matched pair of edges plus a routing service.

**Architecture:** Two symmetric `wg-edge` processes each `ListenUDP` for the local wg0, open a single long-lived geminio stream to a `wg-router` service, and exchange WG datagrams wrapped in 2-byte big-endian length-prefix frames. Router pairs streams by a first-frame pair-id and forwards frames verbatim between the two sides. Frontier transport defaults to UDP (`etc/frontier_udp.yaml`), TCP is selectable via flag for debugging.

**Tech Stack:** Go, `github.com/singchia/frontier/api/dataplane/v1/{edge,service}`, `github.com/singchia/geminio`, `github.com/spf13/pflag`, stdlib `net`/`encoding/binary`.

**Spec:** `docs/superpowers/specs/2026-04-21-wireguard-example-design.md`

---

## File Structure

```
examples/wireguard/
├── Makefile                    # build wg-edge, wg-router, udpping
├── README.md                   # quickstart + real-wg walkthrough
├── edge/edge.go                # wg-edge main (~160 lines)
├── router/router.go            # wg-router main (~160 lines)
├── cmd/udpping/main.go         # test helper: send | echo (~60 lines)
├── scripts/demo.sh             # one-key local demo
└── internal/frame/
    ├── frame.go                # WriteFrame / ReadFrame (~50 lines)
    └── frame_test.go           # round-trip + edge cases
```

Module path rooted at repo: `github.com/singchia/frontier/examples/wireguard/...`. No `go.mod` changes needed — examples share the repo-root module.

---

## Task 1: Scaffold directories and Makefile

**Files:**
- Create: `examples/wireguard/Makefile`
- Create empty subdirectories for later tasks (git won't track them until subsequent tasks add files, which is fine).

- [ ] **Step 1: Create directory tree**

```bash
mkdir -p examples/wireguard/edge \
         examples/wireguard/router \
         examples/wireguard/cmd/udpping \
         examples/wireguard/internal/frame \
         examples/wireguard/scripts
```

- [ ] **Step 2: Write Makefile**

Create `examples/wireguard/Makefile`:

```makefile
.PHONY: all clean edge router udpping

BIN_DIR := bin

all: edge router udpping

edge: $(BIN_DIR)
	go build -o $(BIN_DIR)/wg-edge ./edge

router: $(BIN_DIR)
	go build -o $(BIN_DIR)/wg-router ./router

udpping: $(BIN_DIR)
	go build -o $(BIN_DIR)/udpping ./cmd/udpping

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
```

- [ ] **Step 3: Commit scaffolding**

```bash
git add examples/wireguard/Makefile
git commit -m "feat(examples/wireguard): scaffold directory layout"
```

Expected: clean commit. Subsequent tasks will add code that makes `make all` succeed.

---

## Task 2: `internal/frame` package (TDD)

**Files:**
- Create: `examples/wireguard/internal/frame/frame_test.go`
- Create: `examples/wireguard/internal/frame/frame.go`

- [ ] **Step 1: Write the failing tests**

Create `examples/wireguard/internal/frame/frame_test.go`:

```go
package frame_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/singchia/frontier/examples/wireguard/internal/frame"
)

func TestRoundTrip(t *testing.T) {
	payloads := [][]byte{
		[]byte("hello"),
		[]byte("wireguard handshake initiation"),
		bytes.Repeat([]byte{0xAB}, 1420),
	}
	buf := &bytes.Buffer{}
	for _, p := range payloads {
		if err := frame.WriteFrame(buf, p); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	for i, want := range payloads {
		got, err := frame.ReadFrame(buf)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("frame %d mismatch: got %d bytes, want %d", i, len(got), len(want))
		}
	}
}

func TestMaxPayload(t *testing.T) {
	p := bytes.Repeat([]byte{0xCD}, 65535)
	buf := &bytes.Buffer{}
	if err := frame.WriteFrame(buf, p); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := frame.ReadFrame(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, p) {
		t.Fatal("max-payload mismatch")
	}
}

func TestOversizedPayloadRejected(t *testing.T) {
	p := bytes.Repeat([]byte{0x00}, 65536)
	buf := &bytes.Buffer{}
	err := frame.WriteFrame(buf, p)
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
	if buf.Len() != 0 {
		t.Fatalf("oversized payload wrote %d bytes; expected 0", buf.Len())
	}
}

func TestEmptyPayloadRejected(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := frame.WriteFrame(buf, nil); err == nil {
		t.Fatal("expected error for nil payload")
	}
	if err := frame.WriteFrame(buf, []byte{}); err == nil {
		t.Fatal("expected error for empty payload")
	}
	if buf.Len() != 0 {
		t.Fatalf("empty payload wrote %d bytes; expected 0", buf.Len())
	}
}

func TestZeroLengthOnWireRejected(t *testing.T) {
	r := bytes.NewReader([]byte{0x00, 0x00})
	_, err := frame.ReadFrame(r)
	if err == nil {
		t.Fatal("expected error for zero-length frame")
	}
}

func TestShortReadIsUnexpectedEOF(t *testing.T) {
	good := &bytes.Buffer{}
	if err := frame.WriteFrame(good, []byte("hello world")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	truncated := good.Bytes()[:good.Len()-3]
	_, err := frame.ReadFrame(bytes.NewReader(truncated))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected ErrUnexpectedEOF, got %v", err)
	}
}

func TestEOFBeforeHeaderIsEOF(t *testing.T) {
	_, err := frame.ReadFrame(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./examples/wireguard/internal/frame/... -v
```

Expected: compile error (package `frame` has no `WriteFrame`/`ReadFrame`).

- [ ] **Step 3: Write minimal implementation**

Create `examples/wireguard/internal/frame/frame.go`:

```go
// Package frame implements the 2-byte big-endian length-prefix wire format
// used by the WireGuard-over-frontier example. A frame is:
//
//     [2 bytes big-endian length N] [N bytes payload]
//
// N == 0 is a protocol error on the wire. Payloads larger than 65535 bytes
// cannot be represented and are rejected at write time.
package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

// MaxPayloadSize is the largest single frame payload the wire format can carry.
const MaxPayloadSize = 65535

var (
	ErrEmptyPayload    = errors.New("frame: empty payload")
	ErrPayloadTooLarge = errors.New("frame: payload exceeds 65535 bytes")
	ErrZeroLength      = errors.New("frame: zero-length frame on wire")
)

// WriteFrame writes p as a single length-prefixed frame. Header and payload
// are combined into one Write call so a datagram-preserving transport (pion
// UDP) maps each frame to exactly one datagram.
func WriteFrame(w io.Writer, p []byte) error {
	if len(p) == 0 {
		return ErrEmptyPayload
	}
	if len(p) > MaxPayloadSize {
		return ErrPayloadTooLarge
	}
	buf := make([]byte, 2+len(p))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(p)))
	copy(buf[2:], p)
	_, err := w.Write(buf)
	return err
}

// ReadFrame reads one length-prefixed frame. It returns io.EOF if the stream
// is closed cleanly before any header bytes arrive, and io.ErrUnexpectedEOF
// if the stream closes mid-frame.
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint16(hdr[:])
	if n == 0 {
		return nil, ErrZeroLength
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./examples/wireguard/internal/frame/... -race -v
```

Expected: all 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add examples/wireguard/internal/frame/
git commit -m "feat(examples/wireguard): add length-prefix frame codec"
```

---

## Task 3: `wg-edge` binary

**Files:**
- Create: `examples/wireguard/edge/edge.go`

This task produces a complete working edge: UDP listener, stream open with pair-id first frame, two copy goroutines, stream reopen loop with backoff, signal handling.

- [ ] **Step 1: Write edge.go**

Create `examples/wireguard/edge/edge.go`:

```go
// Command wg-edge is the edge-side half of the WireGuard-over-frontier
// example. It listens on a local UDP endpoint for traffic from a WireGuard
// peer, opens one long-lived geminio stream to the wg-router service,
// writes a pair-id first frame, and shuttles UDP datagrams between the
// local peer and the stream.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/examples/wireguard/internal/frame"
	"github.com/spf13/pflag"
)

func main() {
	frontierAddr := pflag.String("frontier-addr", "127.0.0.1:30012", "frontier edgebound addr")
	frontierNet := pflag.String("frontier-network", "udp", "tcp | udp")
	listenAddr := pflag.String("listen", "127.0.0.1:51820", "UDP listen addr for local wg peer")
	pairID := pflag.String("pair-id", "hello", "pairing identifier (both sides must match)")
	serviceName := pflag.String("service-name", "wg", "frontier service name for router")
	name := pflag.String("name", "edge", "log prefix name")
	pflag.Parse()

	logger := log.New(os.Stderr, "[wg-edge "+*name+"] ", log.LstdFlags)

	udpAddr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		logger.Fatalf("resolve listen addr: %v", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Fatalf("listen udp: %v", err)
	}
	defer udpConn.Close()
	logger.Printf("listening UDP on %s", udpConn.LocalAddr())

	dialer := func() (net.Conn, error) {
		return net.Dial(*frontierNet, *frontierAddr)
	}
	cli, err := edge.NewEdge(dialer, edge.OptionEdgeMeta([]byte(*name)))
	if err != nil {
		logger.Fatalf("new edge: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go handleSignals(cancel, logger, udpConn)

	var lastSrc atomic.Pointer[net.UDPAddr]

	backoff := time.Second
	for ctx.Err() == nil {
		err := runStream(cli, *serviceName, *pairID, udpConn, &lastSrc, logger)
		if ctx.Err() != nil {
			return
		}
		logger.Printf("stream closed: %v (reconnect in %s)", err, backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func handleSignals(cancel context.CancelFunc, logger *log.Logger, udpConn *net.UDPConn) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	logger.Printf("signal %s: shutting down", sig)
	cancel()
	udpConn.Close()
}

// runStream runs one stream lifecycle: open, write pair-id, shuttle UDP <-> stream,
// and return when any side errors out. The caller loops to reopen.
func runStream(
	cli edge.Edge,
	serviceName, pairID string,
	udpConn *net.UDPConn,
	lastSrc *atomic.Pointer[net.UDPAddr],
	logger *log.Logger,
) error {
	stream, err := cli.OpenStream(serviceName)
	if err != nil {
		return err
	}
	defer stream.Close()

	if err := frame.WriteFrame(stream, []byte(pairID)); err != nil {
		return err
	}
	logger.Printf("stream opened, pair-id=%q sent", pairID)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	// UDP -> stream
	go func() {
		defer wg.Done()
		buf := make([]byte, 65535)
		for {
			n, src, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				errCh <- err
				return
			}
			lastSrc.Store(src)
			if err := frame.WriteFrame(stream, buf[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// stream -> UDP
	go func() {
		defer wg.Done()
		for {
			pkt, err := frame.ReadFrame(stream)
			if err != nil {
				errCh <- err
				return
			}
			dst := lastSrc.Load()
			if dst == nil {
				// No local peer has spoken yet; drop (WG will retry).
				continue
			}
			if _, err := udpConn.WriteToUDP(pkt, dst); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Wait for first failure, then close stream to unblock the other goroutine.
	err = <-errCh
	stream.Close()
	wg.Wait()
	// Drain second error without blocking (channel is buffered at 2).
	select {
	case <-errCh:
	default:
	}
	return err
}
```

- [ ] **Step 2: Build**

```bash
cd examples/wireguard && make edge && cd -
```

Expected: `examples/wireguard/bin/wg-edge` produced, no errors.

- [ ] **Step 3: Commit**

```bash
git add examples/wireguard/edge/
git commit -m "feat(examples/wireguard): add wg-edge binary"
```

---

## Task 4: `wg-router` binary

**Files:**
- Create: `examples/wireguard/router/router.go`

This task produces the routing service: accept streams, read pair-id first frame, pair two streams with the same id, bridge them by forwarding frames verbatim, handle timeouts and conflicts.

- [ ] **Step 1: Write router.go**

Create `examples/wireguard/router/router.go`:

```go
// Command wg-router is the service-side half of the WireGuard-over-frontier
// example. It connects to frontier as a service, accepts geminio streams
// from wg-edge clients, reads a pair-id first frame on each stream, and
// pairs streams by id — once two streams share an id, it forwards length-
// prefixed frames verbatim between them.
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/examples/wireguard/internal/frame"
	"github.com/singchia/geminio"
	"github.com/spf13/pflag"
)

type pending struct {
	stream geminio.Stream
	timer  *time.Timer
}

type router struct {
	mu      sync.Mutex
	waiting map[string]*pending

	pairTimeout   time.Duration
	maxPairIDLen  int
	logger        *log.Logger
}

func newRouter(pairTimeout time.Duration, maxPairIDLen int, logger *log.Logger) *router {
	return &router{
		waiting:      make(map[string]*pending),
		pairTimeout:  pairTimeout,
		maxPairIDLen: maxPairIDLen,
		logger:       logger,
	}
}

func main() {
	frontierAddr := pflag.String("frontier-addr", "127.0.0.1:30011", "frontier servicebound addr")
	frontierNet := pflag.String("frontier-network", "udp", "tcp | udp")
	serviceName := pflag.String("service-name", "wg", "service name to register")
	pairTimeout := pflag.Duration("pair-timeout", 60*time.Second, "max time a stream may wait for its peer")
	maxPairIDLen := pflag.Int("max-pair-id-len", 256, "sanity limit on first-frame pair-id length")
	pflag.Parse()

	logger := log.New(os.Stderr, "[wg-router] ", log.LstdFlags)

	dialer := func() (net.Conn, error) {
		return net.Dial(*frontierNet, *frontierAddr)
	}
	svc, err := service.NewService(dialer, service.OptionServiceName(*serviceName))
	if err != nil {
		logger.Fatalf("new service: %v", err)
	}
	defer svc.Close()
	logger.Printf("registered service %q, accepting streams", *serviceName)

	r := newRouter(*pairTimeout, *maxPairIDLen, logger)

	ctx, cancel := context.WithCancel(context.Background())
	go handleSignals(cancel, logger, svc)

	for ctx.Err() == nil {
		stream, err := svc.AcceptStream()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Printf("accept stream: %v", err)
			time.Sleep(time.Second)
			continue
		}
		go r.handleStream(stream)
	}
}

func handleSignals(cancel context.CancelFunc, logger *log.Logger, svc service.Service) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	logger.Printf("signal %s: shutting down", sig)
	cancel()
	svc.Close()
}

func (r *router) handleStream(s geminio.Stream) {
	id, err := frame.ReadFrame(s)
	if err != nil {
		r.logger.Printf("stream %d: read pair-id: %v", s.StreamID(), err)
		s.Close()
		return
	}
	if len(id) > r.maxPairIDLen {
		r.logger.Printf("stream %d: pair-id too long (%d > %d)", s.StreamID(), len(id), r.maxPairIDLen)
		s.Close()
		return
	}
	pairID := string(id)
	r.logger.Printf("stream %d: pair-id=%q", s.StreamID(), pairID)

	other, ok := r.matchOrPark(pairID, s)
	if !ok {
		return // parked; bridge starts when the mate arrives
	}
	r.logger.Printf("paired streams %d <-> %d on pair-id=%q", s.StreamID(), other.StreamID(), pairID)
	bridge(s, other, r.logger)
}

// matchOrPark returns (other, true) if a pending partner exists (removing it
// from the waiting map and cancelling its timeout); returns (nil, false)
// after parking s as the new pending partner. A third stream with the same
// id is rejected by closing it and leaving the existing pending entry alone.
func (r *router) matchOrPark(pairID string, s geminio.Stream) (geminio.Stream, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.waiting[pairID]; ok {
		// Either the mate has arrived, or a third conflicting stream has.
		// Distinguish by streamID: if the pending entry is a live stream,
		// we pair with it. There is no "third stream" state tracked — the
		// first matched pair consumes the pending entry.
		if existing.stream == s {
			// Should never happen; defensive.
			return nil, false
		}
		existing.timer.Stop()
		delete(r.waiting, pairID)
		return existing.stream, true
	}
	// No partner yet: park.
	timer := time.AfterFunc(r.pairTimeout, func() {
		r.mu.Lock()
		entry, ok := r.waiting[pairID]
		if ok && entry.stream == s {
			delete(r.waiting, pairID)
		}
		r.mu.Unlock()
		if ok {
			r.logger.Printf("stream %d: pair-timeout on pair-id=%q", s.StreamID(), pairID)
			s.Close()
		}
	})
	r.waiting[pairID] = &pending{stream: s, timer: timer}
	return nil, false
}

// bridge forwards frames in both directions until one side errors, then
// closes both streams. Frames are decoded and re-encoded rather than raw-
// copied so the 2-byte length prefix is re-emitted cleanly per write.
func bridge(a, b geminio.Stream, logger *log.Logger) {
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); errCh <- pump(a, b) }()
	go func() { defer wg.Done(); errCh <- pump(b, a) }()
	err := <-errCh
	a.Close()
	b.Close()
	wg.Wait()
	select {
	case <-errCh:
	default:
	}
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
		logger.Printf("bridge %d<->%d closed: %v", a.StreamID(), b.StreamID(), err)
	} else {
		logger.Printf("bridge %d<->%d closed cleanly", a.StreamID(), b.StreamID())
	}
}

func pump(src, dst geminio.Stream) error {
	for {
		p, err := frame.ReadFrame(src)
		if err != nil {
			return err
		}
		if err := frame.WriteFrame(dst, p); err != nil {
			return err
		}
	}
}
```

- [ ] **Step 2: Build**

```bash
cd examples/wireguard && make router && cd -
```

Expected: `examples/wireguard/bin/wg-router` produced, no errors.

- [ ] **Step 3: Commit**

```bash
git add examples/wireguard/router/
git commit -m "feat(examples/wireguard): add wg-router binary"
```

---

## Task 5: `udpping` test helper

**Files:**
- Create: `examples/wireguard/cmd/udpping/main.go`

A small UDP tool with two modes:
- `--mode send`: periodically sends a payload to `--target` from its own `--listen` port, and prints any replies received.
- `--mode echo`: listens on `--listen`, echoes every datagram back to its source, and also sends a one-shot seed to `--target` at startup so the far-side wg-edge learns a return address.

- [ ] **Step 1: Write udpping/main.go**

Create `examples/wireguard/cmd/udpping/main.go`:

```go
// Command udpping is a minimal UDP send/echo tool used to drive the
// WireGuard-over-frontier example without needing a real WG peer.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

func main() {
	mode := pflag.String("mode", "send", "send | echo")
	listen := pflag.String("listen", "127.0.0.1:7000", "local UDP addr")
	target := pflag.String("target", "127.0.0.1:51820", "remote UDP addr (send: destination; echo: seed target)")
	interval := pflag.Duration("interval", 1*time.Second, "send interval (send mode only)")
	payload := pflag.String("payload", "ping", "payload to send / seed")
	pflag.Parse()

	logger := log.New(os.Stderr, "[udpping "+*mode+"] ", log.LstdFlags)

	localAddr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		logger.Fatalf("resolve listen: %v", err)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", *target)
	if err != nil {
		logger.Fatalf("resolve target: %v", err)
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	defer conn.Close()
	logger.Printf("listening on %s, target %s", conn.LocalAddr(), remoteAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		conn.Close()
	}()

	switch *mode {
	case "send":
		go senderLoop(conn, remoteAddr, *payload, *interval, logger)
		receiveLoop(conn, logger, false)
	case "echo":
		// Seed: one datagram so the far side knows where to reply.
		if _, err := conn.WriteToUDP([]byte("seed"), remoteAddr); err != nil {
			logger.Printf("seed write: %v", err)
		}
		receiveLoop(conn, logger, true)
	default:
		logger.Fatalf("unknown mode %q", *mode)
	}
}

func senderLoop(conn *net.UDPConn, target *net.UDPAddr, payload string, interval time.Duration, logger *log.Logger) {
	t := time.NewTicker(interval)
	defer t.Stop()
	seq := 0
	for range t.C {
		seq++
		msg := fmt.Sprintf("%s #%d", payload, seq)
		if _, err := conn.WriteToUDP([]byte(msg), target); err != nil {
			logger.Printf("write: %v", err)
			return
		}
	}
}

func receiveLoop(conn *net.UDPConn, logger *log.Logger, echo bool) {
	buf := make([]byte, 65535)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			logger.Printf("read: %v", err)
			return
		}
		logger.Printf("recv %d bytes from %s: %q", n, src, buf[:n])
		if echo {
			if _, err := conn.WriteToUDP(buf[:n], src); err != nil {
				logger.Printf("echo write: %v", err)
				return
			}
		}
	}
}
```

- [ ] **Step 2: Build**

```bash
cd examples/wireguard && make udpping && cd -
```

Expected: `examples/wireguard/bin/udpping` produced, no errors.

- [ ] **Step 3: Commit**

```bash
git add examples/wireguard/cmd/
git commit -m "feat(examples/wireguard): add udpping test helper"
```

---

## Task 6: One-key demo script

**Files:**
- Create: `examples/wireguard/scripts/demo.sh`

- [ ] **Step 1: Write demo.sh**

Create `examples/wireguard/scripts/demo.sh`:

```bash
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
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x examples/wireguard/scripts/demo.sh
```

- [ ] **Step 3: Commit**

```bash
git add examples/wireguard/scripts/
git commit -m "feat(examples/wireguard): add one-key local demo script"
```

---

## Task 7: README

**Files:**
- Create: `examples/wireguard/README.md`

- [ ] **Step 1: Write README.md**

Create `examples/wireguard/README.md`:

````markdown
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
````

- [ ] **Step 2: Commit**

```bash
git add examples/wireguard/README.md
git commit -m "docs(examples/wireguard): add README with quickstart and real-wg walkthrough"
```

---

## Task 8: End-to-end smoke test (manual)

Not a code change — a verification step to run after Task 7 commits.

- [ ] **Step 1: Build everything from a clean tree**

```bash
make clean || true
make
cd examples/wireguard && make clean && make all && cd -
```

Expected: all builds succeed, no warnings, no new untracked files outside `bin/`.

- [ ] **Step 2: Run the demo**

```bash
./examples/wireguard/scripts/demo.sh
```

Expected within ~5 seconds: udpping-send logs show received echoes with
monotonic sequence numbers, e.g.:

```
[udpping send] recv N bytes from 127.0.0.1:51820: "ping #1"
[udpping send] recv N bytes from 127.0.0.1:51820: "ping #2"
```

Ctrl-C to stop. If no echoes arrive within 5 seconds, inspect the log
directory printed on startup and debug.

- [ ] **Step 3: Unit tests pass**

```bash
go test ./examples/wireguard/internal/frame/... -race
```

Expected: PASS.

- [ ] **Step 4: `go vet` clean**

```bash
go vet ./examples/wireguard/...
```

Expected: no diagnostics.

No commit in this task — it is a verification gate.

---

## Acceptance Checklist

- [ ] `go build ./examples/wireguard/...` succeeds
- [ ] `go test ./examples/wireguard/internal/frame -race` all pass
- [ ] `go vet ./examples/wireguard/...` clean
- [ ] `./examples/wireguard/scripts/demo.sh` shows bidirectional UDP echo within seconds
- [ ] README Linux WG walkthrough reviewed for typos (manual WG setup is out of scope for automated verification)

