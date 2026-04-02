package e2e

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E-RES-001: After edge closes, frontier side resources are cleaned up (no panic/hang)
func TestResourceCleanupOnEdgeClose(t *testing.T) {
	e, err := edge.NewEdge(edgeDialer())
	require.NoError(t, err)

	// open a few streams to ensure there is something to clean up
	svc := newService(t, service.OptionServiceName("cleanup-sink"))
	go func() {
		for {
			st, err := svc.AcceptStream()
			if err != nil {
				return
			}
			st.Close()
		}
	}()
	time.Sleep(20 * time.Millisecond)

	for i := 0; i < 5; i++ {
		st, err := e.OpenStream("cleanup-sink")
		if err == nil {
			st.Close()
		}
	}
	time.Sleep(20 * time.Millisecond)

	// close the edge — frontier must not panic or deadlock
	e.Close()
	time.Sleep(100 * time.Millisecond)
}

// E2E-RES-002: After service closes, subsequent edge RPC calls return an error
func TestResourceCleanupOnServiceClose(t *testing.T) {
	// start a service, register an RPC, then close it
	svc, err := service.NewService(serviceDialer(), service.OptionServiceName("gone-svc"))
	require.NoError(t, err)
	err = svc.Register(context.TODO(), "probe", func(_ context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData(req.Data())
	})
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)

	svc.Close()
	time.Sleep(50 * time.Millisecond)

	// now an edge should get an error calling the gone service
	e := newEdge(t)
	req := e.NewRequest([]byte("hello"))
	_, err = e.Call(context.TODO(), "probe", req)
	assert.Error(t, err, "expected error after service closed")
}

// E2E-RES-003: goroutine count does not grow unboundedly after repeated edge connect/close
func TestGoroutineNoLeak(t *testing.T) {
	// let the frontier settle, then record baseline
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	const iterations = 30
	for i := 0; i < iterations; i++ {
		e, err := edge.NewEdge(edgeDialer())
		require.NoError(t, err)
		e.Close()
	}

	// allow goroutines to wind down
	time.Sleep(500 * time.Millisecond)
	runtime.GC()

	after := runtime.NumGoroutine()
	// Leak threshold: must not grow by more than iterations goroutines above baseline
	assert.Less(t, after, baseline+iterations,
		"possible goroutine leak: baseline=%d after=%d", baseline, after)
}
