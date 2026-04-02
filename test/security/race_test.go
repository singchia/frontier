package security

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/require"
)

// SEC-RACE-001: Concurrent Connect and Close on many edges — run with -race
func TestRaceEdgeConnectClose(t *testing.T) {
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			e, err := edge.NewEdge(testEdgeDial)
			if err != nil {
				return
			}
			e.Close()
		}()
	}
	wg.Wait()
}

// SEC-RACE-002: Same edge closed concurrently from multiple goroutines — must not panic
func TestRaceMultipleEdgeClose(t *testing.T) {
	e, err := edge.NewEdge(testEdgeDial)
	require.NoError(t, err)

	var wg sync.WaitGroup
	const closers = 10
	wg.Add(closers)
	for i := 0; i < closers; i++ {
		go func() {
			defer wg.Done()
			e.Close()
		}()
	}
	wg.Wait()
}

// SEC-RACE-003: Service concurrently registers and the edge concurrently calls — no data race
func TestRaceServiceRegisterAndCall(t *testing.T) {
	svc, err := service.NewService(testSvcDial, service.OptionServiceName("race-svc"))
	require.NoError(t, err)
	defer svc.Close()

	e, err := edge.NewEdge(testEdgeDial)
	require.NoError(t, err)
	defer e.Close()

	time.Sleep(20 * time.Millisecond)

	var wg sync.WaitGroup
	const workers = 10

	// goroutines registering RPCs
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		method := "method"
		go func() {
			defer wg.Done()
			_ = svc.Register(context.TODO(), method, func(_ context.Context, req geminio.Request, resp geminio.Response) {
				resp.SetData(req.Data())
			})
		}()
	}

	// goroutines calling RPC from edge simultaneously
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			req := e.NewRequest([]byte("race"))
			_, _ = e.Call(context.TODO(), "method", req)
		}()
	}

	wg.Wait()
}

// SEC-RACE-004: Edge closes while its RPC is being forwarded — must not panic
func TestRaceForwardAndClose(t *testing.T) {
	svc, err := service.NewService(testSvcDial, service.OptionServiceName("slow-svc"))
	require.NoError(t, err)
	defer svc.Close()

	// slow handler to ensure forwarding is in-flight when edge closes
	err = svc.Register(context.TODO(), "slow", func(_ context.Context, req geminio.Request, resp geminio.Response) {
		time.Sleep(50 * time.Millisecond)
		resp.SetData(req.Data())
	})
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	e, err := edge.NewEdge(testEdgeDial)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		req := e.NewRequest([]byte("x"))
		_, _ = e.Call(context.TODO(), "slow", req)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		e.Close()
	}()

	wg.Wait()
}
