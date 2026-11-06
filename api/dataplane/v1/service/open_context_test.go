package service

import (
	"context"
	"errors"
	armlog "github.com/jumboframes/armorigo/log"
	"net"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/options"
)

type contextOpeningEnd struct {
	geminio.End
	peer    string
	closed  bool
	entered chan struct{}
}

func (e *contextOpeningEnd) OpenStreamContext(ctx context.Context, opts ...*options.OpenStreamOptions) (geminio.Stream, error) {
	opt := options.MergeOpenStreamOptions(opts...)
	if opt.Peer != nil {
		e.peer = *opt.Peer
	}
	close(e.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}
func (e *contextOpeningEnd) Close() error { e.closed = true; return nil }

func TestCanceledOpenKeepsSharedFrontier(t *testing.T) {
	underlying := &contextOpeningEnd{entered: make(chan struct{})}
	endpoint := &serviceEnd{End: underlying}
	cluster := &clusterServiceEnd{edgefrontiers: mapmap.NewBiMap()}
	cluster.edgefrontiers.Set(uint64(1001), "frontier-a")
	cluster.frontiers.Store("frontier-a", &frontierNend{end: endpoint})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := cluster.OpenStream(ctx, 1001); done <- err }()
	select {
	case <-underlying.entered:
	case <-time.After(time.Second):
		t.Fatal("context did not reach opener")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel ignored")
	}
	if underlying.peer != "1001" {
		t.Fatal("edge selector lost")
	}
	if underlying.closed {
		t.Fatal("request cancellation closed shared transport")
	}
	if _, ok := cluster.frontiers.Load("frontier-a"); !ok {
		t.Fatal("shared frontier was evicted")
	}
}

func TestInitialFrontierHandshakeRespectsContext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		cn, err := listener.Accept()
		if err == nil {
			accepted <- cn
		}
	}()
	cluster := &clusterServiceEnd{serviceOption: &serviceOption{logger: armlog.DefaultLog, readBufferSize: 32, writeBufferSize: 32}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = cluster.newServiceEndContext(ctx, listener.Addr().String())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("handshake ignored deadline")
	}
	select {
	case cn := <-accepted:
		cn.Close()
	case <-time.After(time.Second):
		t.Fatal("connection was not accepted")
	}
}
