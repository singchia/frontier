package edgebound

import (
	"net"
	"sync"
	"testing"

	"github.com/singchia/frontier/api/v1/edge"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/go-timer/v2"
)

func TestEdgeManager(t *testing.T) {
	network := "tcp"
	addr := "0.0.0.0:1202"

	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen: config.Listen{
				Network: network,
				Addr:    addr,
			},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
	}
	dao, err := dao.NewDao(conf)
	if err != nil {
		t.Error(err)
		return
	}

	h := &handler{
		wg: new(sync.WaitGroup),
	}
	h.wg.Add(2)
	// edge manager
	em, err := newEdgeManager(conf, dao, h, timer.NewTimer())
	if err != nil {
		t.Error(err)
		return
	}
	defer em.Close()
	go em.Serve()

	// edge
	dialer := func() (net.Conn, error) {
		return net.Dial(network, addr)
	}
	edge, err := edge.NewEdge(dialer)
	if err != nil {
		t.Error(err)
		return
	}
	edge.Close()
	h.wg.Wait()
	// if the test failed, it will timeout
}

type handler struct {
	wg *sync.WaitGroup
}

func (h *handler) EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) {
	h.wg.Done()
}

func (h *handler) EdgeOffline(edgeID uint64, meta []byte, addr net.Addr) {
	h.wg.Done()
}

func (h *handler) EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr) {}
