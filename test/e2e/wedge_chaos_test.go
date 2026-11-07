// Wedge / head-of-line reproductions for the failure family seen in
// liaison INC-20260527 (servicebound long connection went "write-alive,
// read-dead" after ~5 days; docker restart was the only recovery) and
// INC-20260605 (same wedge, but with a known trigger — the operator
// forcibly closed a public SSH client whose rproxy copy goroutine was
// then stuck writing to the dead socket and stopped Read'ing the geminio
// stream).
//
// Mechanism, in one paragraph:
//
//   geminio's per-conn read path is a single-goroutine pipeline:
//     conn_base.readPkt → conn.readInCh (a bare blocking send)
//       → dialogue_mgr dispatch goroutine
//       → dialogue.readInCh
//       → dialogue.handlePkt → app readOutCh
//   If ONE stream's application stops calling Read, its readOutCh fills.
//   handlePkt's select on (readOutCh<-pkt | ctx.Done()) blocks; the
//   dialogue's readInCh fills; the dispatch's select on
//   (readInCh<-pkt | ctx.Done()) blocks (the stuck dialogue's ctx is not
//   Done — it is alive but undrained); conn.readInCh fills;
//   conn_base.go:271's bare `readInCh <- pkt` blocks. The entire conn's
//   read goroutine is frozen. New OpenStream calls — which need a
//   SessionAck back over the same read path — time out at the synchub's
//   30s ceiling. Meanwhile the write goroutine and the heartbeat timer
//   are independent and keep running, so the conn looks alive from
//   liveness's POV. geminio's client-side has no inbound-deadline check
//   either; the only recovery is process restart.
//
// This file reproduces that wedge deterministically through the same
// stack liaison uses in production (frontier.api/dataplane/v1/service).

package e2e

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/require"
)

// floodUntilBackpressured spawns a goroutine that writes payload to s in a
// tight loop and returns once writes have stalled (the send path is fully
// back-pressured) or maxWait elapses. It reports how many bytes were
// accepted before the stall. A stall means every channel between this
// writer and the peer's application Read is full — exactly the condition
// that, for a stream the peer never Read's, freezes the peer conn's read
// goroutine.
func floodUntilBackpressured(s geminio.Stream, payload []byte, maxWait time.Duration) (written int64, stalled bool) {
	var w int64
	go func() {
		for {
			n, err := s.Write(payload)
			if err != nil {
				return
			}
			atomic.AddInt64(&w, int64(n))
		}
	}()

	deadline := time.Now().Add(maxWait)
	var last int64
	stable := 0
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		cur := atomic.LoadInt64(&w)
		if cur == last && cur > 0 {
			stable++
			if stable >= 3 { // ~600ms with no progress → writer is blocked
				return cur, true
			}
		} else {
			stable = 0
		}
		last = cur
	}
	return atomic.LoadInt64(&w), false
}

// TestServiceboundHeadOfLineWedge: one stream stops being read; new
// OpenStream on the same service conn must NOT block.
//
// Expected outcome:
//   - geminio v1.3.0-rc.2 (current upstream): test FAILS with
//     "HOL wedge confirmed" — the new OpenStream times out because the
//     shared conn read goroutine is frozen.
//   - A geminio build that makes the dispatch send non-blocking (drop
//     packet OR cancel the stuck dialogue on full readInCh): test PASSES
//     because the dispatch never blocks; other dialogues' SessionAcks
//     flow.
func TestServiceboundHeadOfLineWedge(t *testing.T) {
	const (
		// Generous flood: 64 KiB × 256 = 16 MiB. Vastly exceeds any
		// reasonable conn/dialogue buffer size, so all three channels
		// in the pipeline (stream streamCh, dialogue readInCh, conn
		// readInCh) fill regardless of default sizing.
		floodPayloadSize = 64 * 1024

		// Time budgets.
		acceptTimeout      = 3 * time.Second
		wedgeSettleTime    = 3 * time.Second
		openStreamDeadline = 10 * time.Second // < geminio's 30s synchub cap
	)

	// One service, two edges. Both edges reach the same in-process
	// frontier started by TestMain; the service's underlying geminio
	// conn is shared across OpenStream calls — that is the wedge target.
	svc := newService(t, service.OptionServiceName("wedge-test-svc"))
	edgeA := newEdge(t)
	edgeB := newEdge(t)

	// Stage edgeA's AcceptStream before the OpenStream so the open
	// negotiation completes promptly.
	edgeAStreamCh := make(chan geminio.Stream, 1)
	edgeAAcceptErr := make(chan error, 1)
	go func() {
		s, err := edgeA.AcceptStream()
		if err != nil {
			edgeAAcceptErr <- err
			return
		}
		edgeAStreamCh <- s
	}()
	time.Sleep(50 * time.Millisecond)

	// Service opens a stream to edgeA. The test holds stuckStream and
	// **never calls Read** — this is the rproxy-copy-goroutine-stuck
	// behaviour that triggered INC-20260605.
	stuckStream, err := svc.OpenStream(context.TODO(), edgeA.EdgeID())
	require.NoError(t, err, "initial OpenStream to edgeA")
	t.Cleanup(func() { _ = geminio.AbortStream(stuckStream) })

	var edgeAStream geminio.Stream
	select {
	case edgeAStream = <-edgeAStreamCh:
	case e := <-edgeAAcceptErr:
		t.Fatalf("edgeA.AcceptStream: %v", e)
	case <-time.After(acceptTimeout):
		t.Fatal("edgeA.AcceptStream did not return within acceptTimeout")
	}
	t.Cleanup(func() { _ = geminio.AbortStream(edgeAStream) })

	// Edge A floods the stream from its side until writes back-pressure.
	// Bytes travel:
	//   edgeAStream.Write → edge conn writeOutCh → wire → service conn
	//   readInCh → (dialogue_mgr dispatch) → service-side dialogue
	//   readInCh → handlePkt → stuckStream.streamCh.
	// With the test never Read-ing stuckStream, the chain fills; once
	// edgeAStream.Write itself stalls, every channel up to and including
	// the service conn's read path is full, and the shared dispatch
	// goroutine is blocked at dialogue_mgr.go's `case readInCh <- pkt:`,
	// which freezes conn.readPkt at conn_base.go:271.
	payload := make([]byte, floodPayloadSize)
	written, stalled := floodUntilBackpressured(edgeAStream, payload, 15*time.Second)
	t.Logf("flood accepted %d bytes before stall=%v", written, stalled)
	if !stalled {
		t.Logf("WARNING: flood never stalled — back-pressure did not reach the edge writer; "+
			"wedge may not have formed (buffers larger than %d bytes flooded)", written)
	}
	// Small extra settle so the frozen state is fully established.
	time.Sleep(wedgeSettleTime)

	// The decisive assertion: open a NEW stream from the SAME service
	// to a DIFFERENT edge. If geminio is wedged at the conn read path,
	// the SessionAck for this open never arrives on the service side,
	// and the call times out (synchub 30s upstream; we cap at 10s via
	// context to keep the test bounded).
	edgeBAcceptCh := make(chan geminio.Stream, 1)
	go func() {
		s, err := edgeB.AcceptStream()
		if err == nil {
			edgeBAcceptCh <- s
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), openStreamDeadline)
	defer cancel()

	start := time.Now()
	edgeBStream, err := svc.OpenStream(ctx, edgeB.EdgeID())
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf(
			"HOL wedge reproduced: OpenStream to a fresh edge blocked %v on the same\n"+
				"  service conn whose read path is jammed by one stuck stream.\n"+
				"  geminio v1.3.0-rc.2 cascades back-pressure into the shared read goroutine\n"+
				"  (conn_base.go:271 bare `readInCh <- pkt`); dispatch goroutine cannot escape\n"+
				"  the stuck dialogue's full readInCh because the only select case is ctx.Done()\n"+
				"  and the stuck dialogue is still alive.\n"+
				"  Expected: FAIL on rc.2 (bug confirmed); PASS on a fix that makes the\n"+
				"  dispatch send non-blocking (drop or cancel-stuck-dialogue).\n"+
				"  err=%v",
			elapsed, err,
		)
	} else {
		t.Logf("✓ OpenStream to edgeB succeeded in %v despite stuck stream on edgeA — wedge mitigated", elapsed)
		_ = edgeBStream.Close()
		select {
		case s := <-edgeBAcceptCh:
			_ = s.Close()
		default:
		}
	}

	// Release the wedge before TestMain's shared frontier sees other
	// tests: closing stuckStream cancels its dialogue ctx, which
	// unblocks the dispatch's select escape, which unblocks conn.readPkt.
	// The flood goroutine inside floodUntilBackpressured exits on its
	// next Write error once the dialogue closes.
	_ = geminio.AbortStream(stuckStream)
	_ = geminio.AbortStream(edgeAStream)
}

// TestServiceboundClientSendOnlyLiveness: continuation of the same
// scenario; after the wedge has formed, geminio's client retry layer
// must NOT keep the conn forever — a missed-ack / read-deadline
// detector should declare the conn dead within a bounded window so
// the retry layer can reconnect.
//
// Expected outcome:
//   - geminio v1.3.0-rc.2: test FAILS — conn never recovers because
//     ClientConn has no read-side liveness; only the cyclic
//     sendHeartbeat timer runs, and heartbeats keep going out even
//     while the read goroutine is frozen, so the conn appears alive.
//   - A geminio build that adds a read-ack timeout on ClientConn
//     (mirror of conn_server.go's resetHeartbeatTimeout): test
//     PASSES — within ~hb*2 the conn tears down and a fresh
//     OpenStream succeeds via the retry layer.
func TestServiceboundClientSendOnlyLiveness(t *testing.T) {
	const (
		floodPayloadSize = 64 * 1024
		acceptTimeout    = 3 * time.Second
		wedgeSettleTime  = 3 * time.Second
		// Liveness window: geminio default heartbeat is 20s; a sane
		// missed-ack detector should fire within hb*2 ≈ 40s, plus a
		// little for the retry layer to reconnect and a fresh
		// OpenStream to succeed.
		recoveryDeadline = 60 * time.Second
	)

	svc := newService(t, service.OptionServiceName("liveness-test-svc"))
	edgeA := newEdge(t)
	edgeB := newEdge(t)

	edgeAStreamCh := make(chan geminio.Stream, 1)
	go func() {
		s, err := edgeA.AcceptStream()
		if err == nil {
			edgeAStreamCh <- s
		}
	}()
	time.Sleep(50 * time.Millisecond)

	stuckStream, err := svc.OpenStream(context.TODO(), edgeA.EdgeID())
	require.NoError(t, err)
	t.Cleanup(func() { _ = geminio.AbortStream(stuckStream) })

	var edgeAStream geminio.Stream
	select {
	case edgeAStream = <-edgeAStreamCh:
	case <-time.After(acceptTimeout):
		t.Fatal("edgeA.AcceptStream timeout")
	}
	t.Cleanup(func() { _ = geminio.AbortStream(edgeAStream) })

	payload := make([]byte, floodPayloadSize)
	written, stalled := floodUntilBackpressured(edgeAStream, payload, 20*time.Second)
	t.Logf("flood accepted %d bytes before stall=%v", written, stalled)
	time.Sleep(wedgeSettleTime)

	// Poll svc.OpenStream until either it succeeds (geminio recovered)
	// or recoveryDeadline elapses (geminio did not recover within
	// liveness budget).
	go func() {
		// edge B keeps accepting so any successful open completes.
		for {
			s, err := edgeB.AcceptStream()
			if err != nil {
				return
			}
			_ = s.Close()
		}
	}()

	deadline := time.Now().Add(recoveryDeadline)
	var lastErr error
	tries := 0
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s, err := svc.OpenStream(ctx, edgeB.EdgeID())
		cancel()
		tries++
		if err == nil {
			_ = s.Close()
			t.Logf("✓ recovered after %d tries (%v elapsed) — liveness detector tore down the wedged conn", tries, time.Since(deadline.Add(-recoveryDeadline)))
			return
		}
		lastErr = err
		// Avoid hammering; sleep a beat before retrying.
		time.Sleep(1 * time.Second)
	}

	t.Errorf(
		"Client-side liveness gap reproduced: %d OpenStream attempts over %v all failed\n"+
			"  (last err=%v). geminio v1.3.0-rc.2 ClientConn only sends heartbeats (cyclic\n"+
			"  timer + independent write goroutine); there is no inbound deadline / missed-\n"+
			"  ack detector. While conn.readPkt is frozen at conn_base.go:271, heartbeats\n"+
			"  keep going out, the peer keeps seeing the conn alive, and geminio's retry\n"+
			"  layer never reconnects. Expected: FAIL on rc.2; PASS when conn_client.go\n"+
			"  gains a resetHeartbeatTimeout symmetric to conn_server.go:540.",
		tries, recoveryDeadline, lastErr,
	)
}
