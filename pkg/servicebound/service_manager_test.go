package servicebound

import (
	"net"
	"testing"

	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/go-timer/v2"
)

func TestServiceManagerStream(t *testing.T) {
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
	// service manager
	sm, err := newServiceManager(conf, dao, nil, nil, timer.NewTimer())
	if err != nil {
		t.Error(err)
		return
	}
	defer sm.Close()
	go sm.Serve()

	// service
	_ = func() (net.Conn, error) {
		return net.Dial(network, addr)
	}

}
