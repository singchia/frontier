package servicebound

import (
	"net"
	"sync"
	"testing"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/go-timer/v2"
)

func TestServiceManager(t *testing.T) {
	network := "tcp"
	addr := "0.0.0.0:1202"

	conf := &config.Configuration{
		Servicebound: config.Servicebound{
			Listen: config.Listen{
				Network: network,
				Addr:    addr,
			},
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
	// service manager
	sm, err := newServiceManager(conf, dao, inf, nil, nil, timer.NewTimer())
	if err != nil {
		t.Error(err)
		return
	}
	defer sm.Close()
	go sm.Serve()

	// service
	dialer := func() (net.Conn, error) {
		return net.Dial(network, addr)
	}
	service, err := service.NewService(dialer)
	if err != nil {
		t.Error(err)
		return
	}
	service.Close()
	inf.wg.Wait()
	// if the test failed, it will timeout
}

type informer struct {
	wg *sync.WaitGroup
}

func (inf *informer) ServiceOnline(serviceID uint64, service string, addr net.Addr) {
	inf.wg.Done()
}

func (inf *informer) ServiceOffline(serviceID uint64, service string, addr net.Addr) {
	inf.wg.Done()
}

func (inf *informer) ServiceHeartbeat(serviceID uint64, service string, addr net.Addr) {}
