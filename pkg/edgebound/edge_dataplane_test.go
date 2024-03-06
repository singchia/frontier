package edgebound

import (
	"net"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/go-timer/v2"
)

func TestEdgeManagerStream(t *testing.T) {
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
	// edge manager
	em, err := newEdgeManager(conf, dao, nil, nil, timer.NewTimer())
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
	count := 1000
	for i := 0; i < count; i++ {
		_, err = edge.OpenStream("test")
		if err != nil {
			t.Error(err)
		}
	}

	// compare
	edges := em.ListEdges()
	if len(edges) != 1 {
		t.Error("unmatch count of edges")
	}
	streams := em.ListStreams(edge.EdgeID())
	if count != len(streams) {
		t.Error("unmatch count of streams")
	}

	// close and compare
	edge.Close()
	time.Sleep(5 * time.Second)
	streams = em.ListStreams(edge.EdgeID())
	if len(streams) != 0 {
		t.Error("should have no streams", len(streams))
	}
}
