package service

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	clusterapi "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/exchange"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"github.com/singchia/geminio/options"
	"github.com/singchia/geminio/server"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type routeDirectory struct {
	clusterapi.ClusterServiceClient
	owner string
	calls atomic.Int32
}

func (d *routeDirectory) GetFrontierByEdge(ctx context.Context, req *clusterapi.GetFrontierByEdgeIDRequest, _ ...grpc.CallOption) (*clusterapi.GetFrontierByEdgeIDResponse, error) {
	d.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &clusterapi.GetFrontierByEdgeIDResponse{Fontier: &clusterapi.Frontier{FrontierId: d.owner}}, nil
}

type localEdges struct {
	apis.Edgebound
	mu   sync.RWMutex
	ends map[uint64]geminio.End
}

func (e *localEdges) GetEdgeByID(id uint64) geminio.End {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ends[id]
}
func (e *localEdges) set(id uint64, end geminio.End) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ends[id] = end
}

// Real TCP/Geminio connections: setup replies, RPC forwarding and raw payloads
// traverse the same exchange implementation as deployed Frontier servers.
func wirePair(t *testing.T, accept func(geminio.Stream)) (geminio.End, geminio.End) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	type result struct {
		end geminio.End
		err error
	}
	done := make(chan result, 1)
	go func() {
		cn, err := ln.Accept()
		if err != nil {
			done <- result{err: err}
			return
		}
		opt := server.NewEndOptions()
		opt.SetAcceptStreamFunc(accept)
		end, err := server.NewEndWithConn(cn, opt)
		done <- result{end, err}
	}()
	c, err := client.NewEndWithDialer(func() (net.Conn, error) { return net.Dial("tcp", ln.Addr().String()) })
	require.NoError(t, err)
	var s result
	select {
	case s = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server handshake stalled")
	}
	require.NoError(t, s.err)
	t.Cleanup(func() { c.Close(); s.end.Close() })
	return c, s.end
}
func echoEdge(t *testing.T) geminio.End {
	t.Helper()
	// The server end opens streams towards the simulated Edge client.
	c, s := wirePair(t, nil)
	go func() {
		for {
			st, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer geminio.AbortStream(st)
				io.Copy(st, st)
			}()
		}
	}()
	return s
}
func routedFrontier(t *testing.T, edges *localEdges, legacy bool) *serviceEnd {
	t.Helper()
	ex := exchange.NewExchange(&config.Configuration{}, nil)
	ex.AddEdgebound(edges)
	accept := ex.StreamToEdge
	if legacy {
		accept = func(st geminio.Stream) { ex.StreamToEdge(legacyStream{st}) }
	}
	c, _ := wirePair(t, accept)
	return &serviceEnd{End: c}
}

type legacyStream struct{ geminio.Stream }

func (legacyStream) Meta() []byte { return nil }
func assertEcho(t *testing.T, st geminio.Stream) {
	t.Helper()
	defer geminio.AbortStream(st)
	require.NoError(t, st.SetDeadline(time.Now().Add(2*time.Second)))
	payload := []byte("raw-data-must-not-contain-setup-bytes")
	_, err := st.Write(payload)
	require.NoError(t, err)
	got := make([]byte, len(payload))
	_, err = io.ReadFull(st, got)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}
func TestEdgeMigrationRecoversWithoutRestart(t *testing.T) {
	edge := echoEdge(t)
	a := &localEdges{ends: map[uint64]geminio.End{1001: edge, 1002: edge}}
	b := &localEdges{ends: map[uint64]geminio.End{}}
	fa, fb := routedFrontier(t, a, false), routedFrontier(t, b, false)
	directory := &routeDirectory{owner: "b"}
	cluster := &clusterServiceEnd{cc: directory, edgefrontiers: mapmap.NewBiMap(), serviceOption: &serviceOption{logger: armlog.DefaultLog}}
	cluster.frontiers.Store("a", &frontierNend{end: fa})
	cluster.frontiers.Store("b", &frontierNend{end: fb})
	cluster.edgefrontiers.Set(uint64(1001), "a")
	cluster.edgefrontiers.Set(uint64(1002), "a")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kept, err := cluster.OpenStream(ctx, 1002)
	require.NoError(t, err)
	before, err := cluster.OpenStream(ctx, 1001)
	require.NoError(t, err)
	assertEcho(t, before)
	a.set(1001, nil)
	b.set(1001, edge)
	after, err := cluster.OpenStream(ctx, 1001)
	require.NoError(t, err)
	assertEcho(t, after)
	require.Equal(t, int32(1), directory.calls.Load())
	owner, ok := cluster.edgefrontiers.GetValue(uint64(1001))
	require.True(t, ok)
	require.Equal(t, "b", owner)
	other, ok := cluster.edgefrontiers.GetValue(uint64(1002))
	require.True(t, ok)
	require.Equal(t, "a", other)
	require.True(t, func() bool { _, ok := cluster.frontiers.Load("a"); return ok }())
	assertEcho(t, kept) // A and its unrelated established flow must survive recovery.
}
func TestStreamSetupCompatibility(t *testing.T) {
	for _, legacyServer := range []bool{false, true} {
		for _, legacySDK := range []bool{false, true} {
			name := "new-server"
			if legacyServer {
				name = "legacy-server"
			}
			if legacySDK {
				name += "/legacy-sdk"
			} else {
				name += "/new-sdk"
			}
			t.Run(name, func(t *testing.T) {
				f := routedFrontier(t, &localEdges{ends: map[uint64]geminio.End{1001: echoEdge(t)}}, legacyServer)
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var st geminio.Stream
				var err error
				if legacySDK {
					opt := options.OpenStream()
					opt.SetPeer("1001")
					st, err = geminio.OpenStreamContext(ctx, f.End, opt)
				} else {
					st, err = f.OpenStream(ctx, 1001)
				}
				require.NoError(t, err)
				assertEcho(t, st)
			})
		}
	}
}
func TestMissingEdgeRefreshIsBounded(t *testing.T) {
	for _, owner := range []string{"a", "b"} {
		t.Run(owner, func(t *testing.T) {
			empty := &localEdges{ends: map[uint64]geminio.End{}}
			fa, fb := routedFrontier(t, empty, false), routedFrontier(t, empty, false)
			directory := &routeDirectory{owner: owner}
			cluster := &clusterServiceEnd{cc: directory, edgefrontiers: mapmap.NewBiMap()}
			cluster.frontiers.Store("a", &frontierNend{end: fa})
			cluster.frontiers.Store("b", &frontierNend{end: fb})
			cluster.edgefrontiers.Set(uint64(1001), "a")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			st, err := cluster.OpenStream(ctx, 1001)
			require.Nil(t, st)
			require.ErrorIs(t, err, ErrEdgeNotOnline)
			require.Equal(t, int32(1), directory.calls.Load())
			_, ok := cluster.frontiers.Load("a")
			require.True(t, ok)
			_, ok = cluster.frontiers.Load("b")
			require.True(t, ok)
		})
	}
}

type blockedEdge struct {
	geminio.End
	entered chan struct{}
	exited  chan struct{}
}

func (e *blockedEdge) OpenStreamContext(ctx context.Context, _ ...*options.OpenStreamOptions) (geminio.Stream, error) {
	close(e.entered)
	<-ctx.Done()
	close(e.exited)
	return nil, ctx.Err()
}
func TestSetupCancellationDoesNotEvictFrontier(t *testing.T) {
	edge := &blockedEdge{entered: make(chan struct{}), exited: make(chan struct{})}
	f := routedFrontier(t, &localEdges{ends: map[uint64]geminio.End{1001: edge}}, false)
	cluster := &clusterServiceEnd{edgefrontiers: mapmap.NewBiMap()}
	cluster.edgefrontiers.Set(uint64(1001), "a")
	cluster.frontiers.Store("a", &frontierNend{end: f})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := cluster.OpenStream(ctx, 1001); done <- err }()
	select {
	case <-edge.entered:
	case <-time.After(time.Second):
		t.Fatal("setup not reached")
	}
	cancel()
	select {
	case err := <-done:
		require.True(t, errors.Is(err, context.Canceled))
	case <-time.After(time.Second):
		t.Fatal("cancel ignored")
	}
	select {
	case <-edge.exited:
	case <-time.After(time.Second):
		t.Fatal("next-hop open was not canceled")
	}
	_, ok := cluster.frontiers.Load("a")
	require.True(t, ok)
}

type failedEdge struct{ geminio.End }

func (*failedEdge) OpenStreamContext(context.Context, ...*options.OpenStreamOptions) (geminio.Stream, error) {
	return nil, errors.New("next hop unavailable")
}
func TestEdgeOpenFailureDoesNotGuessMigration(t *testing.T) {
	f := routedFrontier(t, &localEdges{ends: map[uint64]geminio.End{1001: &failedEdge{}}}, false)
	directory := &routeDirectory{owner: "b"}
	cluster := &clusterServiceEnd{cc: directory, edgefrontiers: mapmap.NewBiMap()}
	cluster.frontiers.Store("a", &frontierNend{end: f})
	cluster.edgefrontiers.Set(uint64(1001), "a")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	st, err := cluster.OpenStream(ctx, 1001)
	require.Nil(t, st)
	require.ErrorIs(t, err, ErrEdgeStreamOpen)
	require.Zero(t, directory.calls.Load())
	_, ok := cluster.frontiers.Load("a")
	require.True(t, ok)
}

func TestConcurrentMigrationRecovery(t *testing.T) {
	edge := echoEdge(t)
	a := routedFrontier(t, &localEdges{ends: map[uint64]geminio.End{}}, false)
	b := routedFrontier(t, &localEdges{ends: map[uint64]geminio.End{1001: edge}}, false)
	directory := &routeDirectory{owner: "b"}
	cluster := &clusterServiceEnd{cc: directory, edgefrontiers: mapmap.NewBiMap()}
	cluster.frontiers.Store("a", &frontierNend{end: a})
	cluster.frontiers.Store("b", &frontierNend{end: b})
	cluster.edgefrontiers.Set(uint64(1001), "a")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const count = 32
	results := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() {
			st, err := cluster.OpenStream(ctx, 1001)
			if err == nil {
				geminio.AbortStream(st)
			}
			results <- err
		}()
	}
	for i := 0; i < count; i++ {
		require.NoError(t, <-results)
	}
	owner, ok := cluster.edgefrontiers.GetValue(uint64(1001))
	require.True(t, ok)
	require.Equal(t, "b", owner)
	_, ok = cluster.frontiers.Load("a")
	require.True(t, ok)
}
