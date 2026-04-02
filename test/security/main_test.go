package security

import (
	"flag"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	gconfig "github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/edgebound"
	"github.com/singchia/frontier/pkg/frontier/exchange"
	"github.com/singchia/frontier/pkg/frontier/mq"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/servicebound"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

func init() {
	klog.InitFlags(nil)
	flag.Set("v", "0")
	flag.Set("logtostderr", "false")
	flag.Set("stderrthreshold", "FATAL")

	log.SetLevel(log.LevelFatal)
	log.SetOutput(io.Discard)
}

const (
	edgeboundAddr    = "127.0.0.1:13200"
	serviceboundAddr = "127.0.0.1:13201"
	testNetwork      = "tcp"
)

var (
	testEdgeDial   edge.Dialer
	testSvcDial    service.Dialer
)

func TestMain(m *testing.M) {
	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen: gconfig.Listen{Network: testNetwork, Addr: edgeboundAddr},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
		Servicebound: config.Servicebound{
			Listen: gconfig.Listen{Network: testNetwork, Addr: serviceboundAddr},
		},
	}

	r, err := repo.NewRepo(conf)
	if err != nil {
		panic(err)
	}
	mqm, err := mq.NewMQM(conf)
	if err != nil {
		panic(err)
	}
	tmr := timer.NewTimer()
	ex := exchange.NewExchange(conf, mqm)

	sb, err := servicebound.NewServicebound(conf, r, nil, ex, mqm, tmr)
	if err != nil {
		panic(err)
	}
	eb, err := edgebound.NewEdgebound(conf, r, nil, ex, tmr)
	if err != nil {
		panic(err)
	}
	go sb.Serve()
	go eb.Serve()
	time.Sleep(30 * time.Millisecond)

	testEdgeDial = func() (net.Conn, error) {
		return net.Dial(testNetwork, edgeboundAddr)
	}
	testSvcDial = func() (net.Conn, error) {
		return net.Dial(testNetwork, serviceboundAddr)
	}

	code := m.Run()

	eb.Close()
	sb.Close()
	r.Close()
	mqm.Close()
	tmr.Close()
	os.Exit(code)
}
