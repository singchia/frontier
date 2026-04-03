package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E-RPC-001: Edge calls a method registered by Service via frontier
func TestEdgeCallService(t *testing.T) {
	svc := newService(t, service.OptionServiceName("echo-service"))
	err := svc.Register(context.TODO(), "echo", func(ctx context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData(req.Data())
	})
	require.NoError(t, err)

	// give servicebound time to index the RPC
	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	payload := []byte("hello")
	req := e.NewRequest(payload)
	resp, err := e.Call(context.TODO(), "echo", req)
	require.NoError(t, err)
	assert.Equal(t, payload, resp.Data())
}

// E2E-RPC-002: Service calls a method registered by Edge via frontier (specifying edgeID)
func TestServiceCallEdge(t *testing.T) {
	e := newEdge(t)
	err := e.Register(context.TODO(), "ping", func(ctx context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData([]byte("pong"))
	})
	require.NoError(t, err)

	time.Sleep(30 * time.Millisecond)

	svc := newService(t, service.OptionServiceName("caller"))
	req := svc.NewRequest([]byte(""))
	resp, err := svc.Call(context.TODO(), e.EdgeID(), "ping", req)
	require.NoError(t, err)
	assert.Equal(t, []byte("pong"), resp.Data())
}

// E2E-RPC-003: RPC not found on edge returns an error (no matching RPC registered)
func TestRPCTargetRPCNotFound(t *testing.T) {
	svc := newService(t, service.OptionServiceName("noop-service"))
	// register a method so the service itself is reachable
	_ = svc.Register(context.TODO(), "placeholder", func(_ context.Context, req geminio.Request, resp geminio.Response) {})

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	// call a method the edge never registered
	req := e.NewRequest([]byte("x"))
	_, err := e.Call(context.TODO(), "nonexistent-method", req)
	assert.Error(t, err)
}

// E2E-RPC-004: Service calls edge that is already offline => ErrEdgeNotOnline
func TestRPCTargetEdgeOffline(t *testing.T) {
	// create an edge then close it immediately (without t.Cleanup so we control timing)
	offlineEdge, err := edge.NewEdge(edgeDialer())
	require.NoError(t, err)
	offlineID := offlineEdge.EdgeID()
	offlineEdge.Close()

	time.Sleep(50 * time.Millisecond)

	svc := newService(t, service.OptionServiceName("caller2"))
	req := svc.NewRequest([]byte("data"))
	_, err = svc.Call(context.TODO(), offlineID, "any-method", req)
	assert.Error(t, err)
}

// E2E-RPC-005: 10 edges concurrently call the same Service RPC, all succeed
func TestRPCConcurrent(t *testing.T) {
	svc := newService(t, service.OptionServiceName("concurrent-echo"))
	err := svc.Register(context.TODO(), "echo", func(ctx context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData(req.Data())
	})
	require.NoError(t, err)

	// create all edges first and wait for them to be indexed before calling
	const n = 10
	edges := make([]edge.Edge, n)
	for i := 0; i < n; i++ {
		edges[i] = newEdge(t)
	}
	// give frontier time to propagate the RPC registration to all edges
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		e := edges[i]
		go func() {
			defer wg.Done()
			payload := []byte("concurrent")
			req := e.NewRequest(payload)
			resp, err := e.Call(context.TODO(), "echo", req)
			if err != nil {
				errs <- err
				return
			}
			if string(resp.Data()) != string(payload) {
				errs <- assert.AnError
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}
}
