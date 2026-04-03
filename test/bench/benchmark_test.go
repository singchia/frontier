package bench

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"sync/atomic"
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
	"github.com/singchia/geminio"
	"github.com/singchia/go-timer/v2"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
)

func init() {
	// Set klog to only show fatal errors
	klog.InitFlags(nil)
	flag.Set("v", "0")
	flag.Set("logtostderr", "false")
	flag.Set("alsologtostderr", "false")
	flag.Set("stderrthreshold", "FATAL")

	// Set armorigo log to only show fatal errors
	log.SetLevel(log.LevelFatal)
	log.SetOutput(io.Discard)
}

var benchPortCounter int32 = 15000

// benchFrontier holds the in-process frontier addresses.
type benchFrontier struct {
	edgeAddr string
	svcAddr  string
}

// allocatePorts allocates two consecutive ports for a benchmark
func allocatePorts() (edgeAddr, svcAddr string) {
	port := atomic.AddInt32(&benchPortCounter, 20) // Use 20-port spacing to avoid conflicts
	edgeAddr = fmt.Sprintf("127.0.0.1:%d", port-19)
	svcAddr = fmt.Sprintf("127.0.0.1:%d", port-18)
	return
}

// startFrontier spins up an in-process frontier and
// registers b.Cleanup to shut it down.
func startFrontier(b *testing.B) *benchFrontier {
	b.Helper()

	edgeAddr, svcAddr := allocatePorts()

	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen:                       gconfig.Listen{Network: "tcp", Addr: edgeAddr},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
		Servicebound: config.Servicebound{
			Listen: gconfig.Listen{Network: "tcp", Addr: svcAddr},
		},
	}

	r, err := repo.NewRepo(conf)
	require.NoError(b, err)
	mqm, err := mq.NewMQM(conf)
	require.NoError(b, err)
	tmr := timer.NewTimer()
	ex := exchange.NewExchange(conf, mqm)

	sb, err := servicebound.NewServicebound(conf, r, nil, ex, mqm, tmr)
	require.NoError(b, err)
	eb, err := edgebound.NewEdgebound(conf, r, nil, ex, tmr)
	require.NoError(b, err)

	go sb.Serve()
	go eb.Serve()
	time.Sleep(30 * time.Millisecond)

	b.Cleanup(func() {
		eb.Close()
		sb.Close()
		r.Close()
		mqm.Close()
		tmr.Close()
	})

	return &benchFrontier{edgeAddr: edgeAddr, svcAddr: svcAddr}
}

// dialEdge opens a new Edge connection and registers cleanup.
func (f *benchFrontier) dialEdge(b *testing.B, opts ...edge.EdgeOption) edge.Edge {
	b.Helper()
	dialer := func() (net.Conn, error) { return net.Dial("tcp", f.edgeAddr) }
	e, err := edge.NewEdge(dialer, opts...)
	require.NoError(b, err)
	b.Cleanup(func() { e.Close() })
	return e
}

// dialService opens a new Service connection and registers cleanup.
func (f *benchFrontier) dialService(b *testing.B, name string, opts ...service.ServiceOption) service.Service {
	b.Helper()
	dialer := func() (net.Conn, error) { return net.Dial("tcp", f.svcAddr) }
	opts = append([]service.ServiceOption{service.OptionServiceName(name)}, opts...)
	svc, err := service.NewService(dialer, opts...)
	require.NoError(b, err)
	b.Cleanup(func() { svc.Close() })
	return svc
}

// BENCH-CALL-001: Edge → Frontier → Service RPC 吞吐 (QPS)
func BenchmarkEdgeCallService(b *testing.B) {
	f := startFrontier(b)

	svc := f.dialService(b, "bench-rpc-svc")
	require.NoError(b, svc.Register(context.TODO(), "echo",
		func(_ context.Context, req geminio.Request, resp geminio.Response) {
			resp.SetData(req.Data())
		},
	))
	time.Sleep(300 * time.Millisecond)

	// verify echo works before benchmark
	e0 := f.dialEdge(b)
	req0 := e0.NewRequest([]byte("test"))
	_, err := e0.Call(context.TODO(), "echo", req0)
	require.NoError(b, err, "pre-bench verification failed")

	payload := []byte("ping")

	// pre-create edges to avoid timing issues with RPC routing
	const numWorkers = 10
	edges := make([]edge.Edge, numWorkers)
	for i := 0; i < numWorkers; i++ {
		edges[i] = f.dialEdge(b)
	}
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			e := edges[i%numWorkers]
			i++
			req := e.NewRequest(payload)
			if _, err := e.Call(context.TODO(), "echo", req); err != nil {
				b.Error(err)
			}
		}
	})
	b.StopTimer()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "qps")
}

// BENCH-CALL-002: Service → Frontier → Edge RPC 吞吐 (QPS)
func BenchmarkServiceCallEdge(b *testing.B) {
	f := startFrontier(b)

	e := f.dialEdge(b)
	require.NoError(b, e.Register(context.TODO(), "echo",
		func(_ context.Context, req geminio.Request, resp geminio.Response) {
			resp.SetData(req.Data())
		},
	))
	edgeID := e.EdgeID()
	time.Sleep(300 * time.Millisecond)

	s0 := f.dialService(b, "bench-verify")
	req0 := s0.NewRequest([]byte("test"))
	_, err := s0.Call(context.TODO(), edgeID, "echo", req0)
	require.NoError(b, err, "pre-bench verification failed")

	payload := []byte("pong")

	const numWorkers = 10
	svcs := make([]service.Service, numWorkers)
	for i := 0; i < numWorkers; i++ {
		svcs[i] = f.dialService(b, fmt.Sprintf("bench-caller-%d", i))
	}
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			svc := svcs[i%numWorkers]
			i++
			req := svc.NewRequest(payload)
			if _, err := svc.Call(context.TODO(), edgeID, "echo", req); err != nil {
				b.Error(err)
			}
		}
	})
	b.StopTimer()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "qps")
}

// BENCH-MSG-001: Edge → Frontier → Service 消息吞吐 (QPS)
func BenchmarkEdgePublishMessage(b *testing.B) {
	f := startFrontier(b)

	svc := f.dialService(b, "bench-msg-svc", service.OptionServiceReceiveTopics([]string{"bench-topic"}))
	go func() {
		for {
			msg, err := svc.Receive(context.TODO())
			if err != nil {
				return
			}
			msg.Done()
		}
	}()
	time.Sleep(300 * time.Millisecond)

	payload := []byte("message")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		e := f.dialEdge(b)
		for pb.Next() {
			msg := e.NewMessage(payload)
			e.Publish(context.TODO(), "bench-topic", msg)
		}
	})
	b.StopTimer()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "qps")
}

// BENCH-STRM-001: Edge → Frontier → Service 流建立吞吐 (QPS)
// Note: This benchmark may occasionally panic in geminio when run repeatedly
// due to a race condition in stream cleanup. Run with -count=1 if issues occur.
func BenchmarkEdgeOpenStream(b *testing.B) {
	f := startFrontier(b)

	svc := f.dialService(b, "bench-stream-svc")
	go func() {
		for {
			st, err := svc.AcceptStream()
			if err != nil {
				return
			}
			go st.Close()
		}
	}()
	time.Sleep(300 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		e := f.dialEdge(b)
		for pb.Next() {
			st, err := e.OpenStream("bench-stream-svc")
			if err != nil {
				continue
			}
			st.Close()
		}
	})
	b.StopTimer()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "qps")
}

// BENCH-CONN-001: Edge 连接建立与断开吞吐 (QPS / TPS)
func BenchmarkEdgeConnectDisconnect(b *testing.B) {
	// Skip this in parallel runs because it exhausts ports
	if !testing.Short() {
		b.Skip("Skipping connect/disconnect benchmark in non-short mode to avoid port exhaustion")
	}

	edgeAddr, svcAddr := allocatePorts()

	conf := &config.Configuration{
		Edgebound: config.Edgebound{
			Listen:                       gconfig.Listen{Network: "tcp", Addr: edgeAddr},
			EdgeIDAllocWhenNoIDServiceOn: true,
		},
		Servicebound: config.Servicebound{
			Listen: gconfig.Listen{Network: "tcp", Addr: svcAddr},
		},
	}

	r, err := repo.NewRepo(conf)
	require.NoError(b, err)
	mqm, err := mq.NewMQM(conf)
	require.NoError(b, err)
	tmr := timer.NewTimer()
	ex := exchange.NewExchange(conf, mqm)

	sb, err := servicebound.NewServicebound(conf, r, nil, ex, mqm, tmr)
	require.NoError(b, err)
	eb, err := edgebound.NewEdgebound(conf, r, nil, ex, tmr)
	require.NoError(b, err)

	go sb.Serve()
	go eb.Serve()
	time.Sleep(30 * time.Millisecond)

	b.Cleanup(func() {
		eb.Close()
		sb.Close()
		r.Close()
		mqm.Close()
		tmr.Close()
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		dialer := func() (net.Conn, error) { return net.Dial("tcp", edgeAddr) }
		for pb.Next() {
			e, err := edge.NewEdge(dialer)
			if err != nil {
				continue
			}
			e.Close()
		}
	})
	b.StopTimer()

	qps := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(qps, "qps")
}
