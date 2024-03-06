package edgebound

import (
	"net"
	"sync"
	"testing"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
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

	inf := &informer{
		wg: new(sync.WaitGroup),
	}
	inf.wg.Add(2)
	// edge manager
	em, err := newEdgeManager(conf, dao, inf, nil, timer.NewTimer())
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
	inf.wg.Wait()
	// if the test failed, it will timeout
}

type informer struct {
	wg *sync.WaitGroup
}

func (inf *informer) EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) {
	inf.wg.Done()
}

func (inf *informer) EdgeOffline(edgeID uint64, meta []byte, addr net.Addr) {
	inf.wg.Done()
}

func (inf *informer) EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr) {}
