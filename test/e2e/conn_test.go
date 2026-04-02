package e2e

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E-CONN-001
func TestEdgeConnect(t *testing.T) {
	e := newEdge(t)
	assert.NotZero(t, e.EdgeID())
}

// E2E-CONN-002
func TestEdgeConnectAndClose(t *testing.T) {
	done := make(chan struct{})
	e, err := edge.NewEdge(edgeDialer())
	require.NoError(t, err)
	go func() {
		defer close(done)
		e.Close()
	}()
	waitTimeout(t, done, 3*time.Second)
}

// E2E-CONN-003: Edge carries meta, Service receives it via EdgeOnline callback
func TestEdgeConnectWithMeta(t *testing.T) {
	meta := []byte("hello-frontier")
	gotMeta := make(chan []byte, 1)

	svc := newService(t,
		service.OptionServiceName("meta-checker"),
		service.OptionServiceReceiveTopics([]string{}),
	)
	err := svc.RegisterEdgeOnline(context.Background(), func(edgeID uint64, m []byte, addr net.Addr) error {
		gotMeta <- m
		return nil
	})
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)
	_ = newEdge(t, edge.OptionEdgeMeta(meta))

	select {
	case m := <-gotMeta:
		assert.Equal(t, meta, m)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for EdgeOnline callback")
	}
}

// E2E-CONN-004: 100 edges connect concurrently, all succeed with unique IDs
func TestMultiEdgeConnect(t *testing.T) {
	const n = 100
	ids := make(chan uint64, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			e := newEdge(t)
			ids <- e.EdgeID()
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[uint64]struct{}, n)
	for id := range ids {
		assert.NotZero(t, id)
		_, dup := seen[id]
		assert.False(t, dup, "duplicate edgeID: %d", id)
		seen[id] = struct{}{}
	}
	assert.Len(t, seen, n)
}

// E2E-CONN-005: Service connects and registers successfully
func TestServiceConnect(t *testing.T) {
	svc := newService(t, service.OptionServiceName("my-service"))
	assert.NotNil(t, svc)
}

// E2E-CONN-006: After Service disconnects, Edge RPC call returns an error
func TestServiceConnectAndClose(t *testing.T) {
	svc, err := service.NewService(serviceDialer(),
		service.OptionServiceName("gone-service"),
	)
	require.NoError(t, err)
	svc.Close()

	time.Sleep(50 * time.Millisecond)

	e := newEdge(t)
	req := e.NewRequest([]byte("ping"))
	_, err = e.Call(context.Background(), "anything", req)
	assert.Error(t, err)
}
