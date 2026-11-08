package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/singchia/geminio"
)

type registrationPeer struct {
	geminio.End
	mu    sync.Mutex
	fail  bool
	calls int
}

func (p *registrationPeer) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.fail {
		return errors.New("registration unavailable")
	}
	return nil
}
func TestRegistrationPartialFailureRetriesAndContinues(t *testing.T) {
	cluster := &clusterServiceEnd{rpcs: make(map[string]geminio.RPC)}
	good := &registrationPeer{}
	bad := &registrationPeer{fail: true}
	cluster.frontiers.Store("a", &frontierNend{end: &serviceEnd{End: bad}})
	cluster.frontiers.Store("b", &frontierNend{end: &serviceEnd{End: good}})
	rpc := func(context.Context, geminio.Request, geminio.Response) {}
	if cluster.Register(context.Background(), "callback", rpc) == nil {
		t.Fatal("expected partial failure")
	}
	if good.calls != 1 || bad.calls != 1 {
		t.Fatal("not all frontiers attempted")
	}
	bad.fail = false
	if err := cluster.reconcileRegistrations(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if good.calls != 1 || bad.calls != 2 {
		t.Fatal("retry should only target failed peer")
	}
	added := &registrationPeer{}
	cluster.publishFrontier("c", &frontierNend{end: &serviceEnd{End: added}})
	cluster.retryRegistrations()
	if added.calls != 1 {
		t.Fatal("new frontier did not receive callback")
	}
}
func TestRegistrationConcurrentReplace(t *testing.T) {
	cluster := &clusterServiceEnd{rpcs: make(map[string]geminio.RPC)}
	peer := &registrationPeer{}
	cluster.frontiers.Store("a", &frontierNend{end: &serviceEnd{End: peer}})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cluster.Register(context.Background(), "callback", func(context.Context, geminio.Request, geminio.Response) {}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	cluster.retryRegistrations()
	cluster.registrationMtx.Lock()
	defer cluster.registrationMtx.Unlock()
	for _, methods := range cluster.pending {
		if len(methods) != 0 {
			t.Fatal("pending replacement not reconciled")
		}
	}
}
func TestRegistrationDropsRemovedConnection(t *testing.T) {
	cluster := &clusterServiceEnd{rpcs: make(map[string]geminio.RPC)}
	peer := &registrationPeer{fail: true}
	endpoint := &serviceEnd{End: peer}
	cluster.frontiers.Store("a", &frontierNend{end: endpoint})
	if cluster.Register(context.Background(), "callback", func(context.Context, geminio.Request, geminio.Response) {}) == nil {
		t.Fatal("expected error")
	}
	cluster.frontiers.Delete("a")
	cluster.retryRegistrations()
	if len(cluster.pending) != 0 {
		t.Fatal("removed connection retained")
	}
}
