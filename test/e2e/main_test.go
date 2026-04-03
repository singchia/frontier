package e2e

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
	edgeboundAddr    = "127.0.0.1:13100"
	serviceboundAddr = "127.0.0.1:13101"
	network          = "tcp"
)

// TestMain starts one shared frontier instance for the whole test binary.
func TestMain(m *testing.M) {
	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen:                       gconfig.Listen{Network: network, Addr: edgeboundAddr},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
		Servicebound: config.Servicebound{
			Listen: gconfig.Listen{Network: network, Addr: serviceboundAddr},
		},
	}

	r, err := repo.NewRepo(conf)
	if err != nil {
		panic("new repo: " + err.Error())
	}
	mqm, err := mq.NewMQM(conf)
	if err != nil {
		panic("new mqm: " + err.Error())
	}
	tmr := timer.NewTimer()
	ex := exchange.NewExchange(conf, mqm)

	sb, err := servicebound.NewServicebound(conf, r, nil, ex, mqm, tmr)
	if err != nil {
		panic("new servicebound: " + err.Error())
	}
	eb, err := edgebound.NewEdgebound(conf, r, nil, ex, tmr)
	if err != nil {
		panic("new edgebound: " + err.Error())
	}

	go sb.Serve()
	go eb.Serve()
	time.Sleep(30 * time.Millisecond)

	code := m.Run()

	eb.Close()
	sb.Close()
	r.Close()
	mqm.Close()
	tmr.Close()
	os.Exit(code)
}

// edgeDialer returns a Dialer that connects to the edgebound.
func edgeDialer() edge.Dialer {
	return func() (net.Conn, error) {
		return net.Dial(network, edgeboundAddr)
	}
}

// serviceDialer returns a Dialer that connects to the servicebound.
func serviceDialer() service.Dialer {
	return func() (net.Conn, error) {
		return net.Dial(network, serviceboundAddr)
	}
}

// newEdge creates an Edge connected to the shared test frontier.
func newEdge(t *testing.T, opts ...edge.EdgeOption) edge.Edge {
	t.Helper()
	e, err := edge.NewEdge(edgeDialer(), opts...)
	if err != nil {
		t.Fatalf("new edge: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

// newService creates a Service connected to the shared test frontier.
func newService(t *testing.T, opts ...service.ServiceOption) service.Service {
	t.Helper()
	svc, err := service.NewService(serviceDialer(), opts...)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

// waitTimeout waits for done to be closed, failing the test if deadline is exceeded.
func waitTimeout(t *testing.T, done <-chan struct{}, d time.Duration) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatal("timed out waiting")
	}
}
