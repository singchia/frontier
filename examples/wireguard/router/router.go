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
				break
			}
			logger.Printf("accept stream: %v", err)
			time.Sleep(time.Second)
			continue
		}
		go r.handleStream(stream)
	}
	r.closeAll()
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

// closeAll drains the waiting map on shutdown: stop each pending stream's
// timeout timer and close the stream. Active bridges are torn down
// indirectly by closing the underlying service (svc.Close in handleSignals),
// which propagates I/O errors to the pump goroutines.
func (r *router) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, p := range r.waiting {
		p.timer.Stop()
		p.stream.Close()
		delete(r.waiting, id)
	}
}

// matchOrPark returns (other, true) if a pending partner exists (removing it
// from the waiting map and cancelling its timeout); returns (nil, false)
// after parking s as the new pending partner.
//
// Pairing is arrival-order: the first two streams sharing a pair-id form a
// bridge; once bridged, the map entry is consumed, so a later stream with
// the same pair-id parks afresh and awaits its own partner. This is
// deliberately NOT "reject third stream" — edges reconnecting after a
// transient fault need to re-park and re-pair, and the reject semantics
// would prevent recovery.
func (r *router) matchOrPark(pairID string, s geminio.Stream) (geminio.Stream, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.waiting[pairID]; ok {
		if existing.stream == s {
			// Defensive: a stream cannot pair with itself.
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
