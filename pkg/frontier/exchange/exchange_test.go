package exchange

import (
	"context"
	"flag"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	gconfig "github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/edgebound"
	"github.com/singchia/frontier/pkg/frontier/mq"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/servicebound"
	"github.com/singchia/geminio"
	"github.com/singchia/go-timer/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	testNetwork      = "tcp"
	edgeboundAddr    = "127.0.0.1:13300"
	serviceboundAddr = "127.0.0.1:13301"
)

// exchangeHarness starts an in-process exchange + edgebound + servicebound.
type exchangeHarness struct {
	eb  interface{ Close() error }
	sb  interface{ Close() error }
	r   interface{ Close() error }
	mqm interface{ Close() error }
	tmr timer.Timer
}

func newHarness(t *testing.T) *exchangeHarness {
	t.Helper()
	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen:                       gconfig.Listen{Network: testNetwork, Addr: edgeboundAddr},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
		Servicebound: config.Servicebound{
			Listen: gconfig.Listen{Network: testNetwork, Addr: serviceboundAddr},
		},
	}
	r, err := repo.NewRepo(conf)
	require.NoError(t, err)

	mqm, err := mq.NewMQM(conf)
	require.NoError(t, err)

	tmr := timer.NewTimer()
	ex := NewExchange(conf, mqm)

	sb, err := servicebound.NewServicebound(conf, r, nil, ex, mqm, tmr)
	require.NoError(t, err)

	eb, err := edgebound.NewEdgebound(conf, r, nil, ex, tmr)
	require.NoError(t, err)

	go sb.Serve()
	go eb.Serve()
	time.Sleep(30 * time.Millisecond)

	h := &exchangeHarness{eb: eb, sb: sb, r: r, mqm: mqm, tmr: tmr}
	t.Cleanup(func() {
		eb.Close()
		sb.Close()
		r.Close()
		mqm.Close()
		tmr.Close()
	})
	return h
}

func edgeDial() edge.Dialer {
	return func() (net.Conn, error) { return net.Dial(testNetwork, edgeboundAddr) }
}
func svcDial() service.Dialer {
	return func() (net.Conn, error) { return net.Dial(testNetwork, serviceboundAddr) }
}

// UNIT-EXCH-001: RPC from Edge forwarded to Service
func TestExchangeForwardRPCToService(t *testing.T) {
	newHarness(t)

	svc, err := service.NewService(svcDial(), service.OptionServiceName("rpc-svc"))
	require.NoError(t, err)
	defer svc.Close()
	require.NoError(t, svc.Register(context.TODO(), "echo", func(_ context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData(req.Data())
	}))
	time.Sleep(20 * time.Millisecond)

	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()

	req := e.NewRequest([]byte("ping"))
	resp, err := e.Call(context.TODO(), "echo", req)
	require.NoError(t, err)
	assert.Equal(t, []byte("ping"), resp.Data())
}

// UNIT-EXCH-002: Message from Edge forwarded to Service via topic
func TestExchangeForwardMessageToService(t *testing.T) {
	newHarness(t)

	const topic = "news"
	svc, err := service.NewService(svcDial(),
		service.OptionServiceName("msg-svc"),
		service.OptionServiceReceiveTopics([]string{topic}),
	)
	require.NoError(t, err)
	defer svc.Close()

	received := make(chan []byte, 1)
	go func() {
		msg, err := svc.Receive(context.TODO())
		if err == nil {
			received <- msg.Data()
			msg.Done()
		}
	}()
	time.Sleep(20 * time.Millisecond)

	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()

	msg := e.NewMessage([]byte("headline"))
	require.NoError(t, e.Publish(context.TODO(), topic, msg))

	select {
	case data := <-received:
		assert.Equal(t, []byte("headline"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}

// UNIT-EXCH-003: RPC from Service forwarded to specific Edge
func TestExchangeForwardRPCToEdge(t *testing.T) {
	newHarness(t)

	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()
	require.NoError(t, e.Register(context.TODO(), "greet", func(_ context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData([]byte("hello-from-edge"))
	}))
	time.Sleep(20 * time.Millisecond)

	svc, err := service.NewService(svcDial(), service.OptionServiceName("rpc-caller"))
	require.NoError(t, err)
	defer svc.Close()

	req := svc.NewRequest([]byte(""))
	resp, err := svc.Call(context.TODO(), e.EdgeID(), "greet", req)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello-from-edge"), resp.Data())
}

// UNIT-EXCH-004: Message from Service delivered to specific Edge
func TestExchangeForwardMessageToEdge(t *testing.T) {
	newHarness(t)

	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()

	received := make(chan []byte, 1)
	go func() {
		msg, err := e.Receive(context.TODO())
		if err == nil {
			received <- msg.Data()
			msg.Done()
		}
	}()
	time.Sleep(20 * time.Millisecond)

	svc, err := service.NewService(svcDial(), service.OptionServiceName("msg-pub"))
	require.NoError(t, err)
	defer svc.Close()

	msg := svc.NewMessage([]byte("push-to-edge"))
	require.NoError(t, svc.Publish(context.TODO(), e.EdgeID(), msg))

	select {
	case data := <-received:
		assert.Equal(t, []byte("push-to-edge"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}

// UNIT-EXCH-005: Stream opened from Edge transparently forwarded to Service
func TestExchangeStreamToService(t *testing.T) {
	newHarness(t)

	accepted := make(chan geminio.Stream, 1)
	svc, err := service.NewService(svcDial(), service.OptionServiceName("stream-svc"))
	require.NoError(t, err)
	defer svc.Close()
	go func() {
		if st, err := svc.AcceptStream(); err == nil {
			accepted <- st
		}
	}()
	time.Sleep(20 * time.Millisecond)

	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()

	st, err := e.OpenStream("stream-svc")
	require.NoError(t, err)
	defer st.Close()

	select {
	case serverSt := <-accepted:
		assert.NotNil(t, serverSt)
		serverSt.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for stream on service side")
	}
}

// UNIT-EXCH-006: Stream opened from Service transparently forwarded to Edge
func TestExchangeStreamToEdge(t *testing.T) {
	newHarness(t)

	accepted := make(chan geminio.Stream, 1)
	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	defer e.Close()
	go func() {
		if st, err := e.AcceptStream(); err == nil {
			accepted <- st
		}
	}()
	time.Sleep(20 * time.Millisecond)

	svc, err := service.NewService(svcDial(), service.OptionServiceName("stream-opener"))
	require.NoError(t, err)
	defer svc.Close()

	st, err := svc.OpenStream(context.TODO(), e.EdgeID())
	require.NoError(t, err)
	defer st.Close()

	select {
	case edgeSt := <-accepted:
		assert.NotNil(t, edgeSt)
		edgeSt.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for stream on edge side")
	}
}

// UNIT-EXCH-007: Edge online/offline events are forwarded to Service via control RPCs
func TestExchangeEdgeOnlineOffline(t *testing.T) {
	newHarness(t)

	onlineCh := make(chan uint64, 1)
	offlineCh := make(chan uint64, 1)

	svc, err := service.NewService(svcDial(), service.OptionServiceName("event-watcher"))
	require.NoError(t, err)
	defer svc.Close()

	require.NoError(t, svc.RegisterEdgeOnline(context.TODO(), func(edgeID uint64, meta []byte, addr net.Addr) error {
		onlineCh <- edgeID
		return nil
	}))
	require.NoError(t, svc.RegisterEdgeOffline(context.TODO(), func(edgeID uint64, meta []byte, addr net.Addr) error {
		offlineCh <- edgeID
		return nil
	}))

	time.Sleep(20 * time.Millisecond)

	// connect then disconnect an edge
	e, err := edge.NewEdge(edgeDial())
	require.NoError(t, err)
	edgeID := e.EdgeID()

	select {
	case id := <-onlineCh:
		assert.Equal(t, edgeID, id)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for EdgeOnline event")
	}

	e.Close()

	select {
	case id := <-offlineCh:
		assert.Equal(t, edgeID, id)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for EdgeOffline event")
	}
}
