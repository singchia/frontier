package bench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
)

// BenchmarkFrontierConcurrent uses a fixed worker count, independently of
// GOMAXPROCS. Every operation traverses Service -> Frontier -> Edge and verifies
// the echo. All components run in one process, over real loopback TCP sockets.
func BenchmarkFrontierConcurrent(b *testing.B) {
	concurrencies := []int{1, 16, 64, 256}
	if raw := os.Getenv("FRONTIER_BENCH_CONCURRENCY"); raw != "" {
		concurrencies = nil
		for _, part := range strings.Split(raw, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 || n > 4096 {
				b.Fatalf("invalid concurrency %q (expected 1..4096)", part)
			}
			concurrencies = append(concurrencies, n)
		}
	}
	for _, mode := range []string{"rpc", "reuse", "open"} {
		b.Run(mode, func(b *testing.B) {
			for _, concurrency := range concurrencies {
				b.Run(fmt.Sprintf("c%d", concurrency), func(b *testing.B) { benchConcurrent(b, mode, concurrency) })
			}
		})
	}
}

// Expected reset completion is not a transport failure. Keep all other server
// diagnostics, without writing gigabytes of normal short-stream teardown logs.
type benchmarkLogWriter struct{}

func (benchmarkLogWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("stream forward raw, copy err: stream aborted")) {
		return len(p), nil
	}
	return os.Stderr.Write(p)
}

func benchConcurrent(b *testing.B, mode string, concurrency int) {
	if os.Getenv("FRONTIER_BENCH_MONITOR") == "1" {
		klog.SetOutput(benchmarkLogWriter{})
	}
	if os.Getenv("FRONTIER_BENCH_DIAGNOSTICS") == "1" {
		log.SetLevel(log.LevelError)
		log.SetOutput(os.Stderr)
	}
	f := startFrontier(b)
	edge := f.dialEdge(b)
	payload := bytes.Repeat([]byte("frontier"), 32) // 256 bytes each way.
	require.NoError(b, edge.Register(context.Background(), "echo", func(_ context.Context, req geminio.Request, rsp geminio.Response) { rsp.SetData(req.Data()) }))
	var echoes sync.WaitGroup
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			st, err := edge.AcceptStream()
			if err != nil {
				return
			}
			echoes.Add(1)
			go func() { defer echoes.Done(); defer geminio.AbortStream(st); io.Copy(st, st) }()
		}
	}()
	b.Cleanup(func() { edge.Close(); <-acceptDone; echoes.Wait() })
	svc := f.dialService(b, "fixed-concurrency")
	// Readiness and payload correctness are checked outside timed measurement.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	st, err := svc.OpenStream(ctx, edge.EdgeID())
	cancel()
	require.NoError(b, err)
	require.NoError(b, benchEcho(st, payload, make([]byte, len(payload))))
	geminio.AbortStream(st)
	streams := make([]geminio.Stream, concurrency)
	if mode == "reuse" {
		for i := range streams {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			streams[i], err = svc.OpenStream(ctx, edge.EdgeID())
			cancel()
			require.NoError(b, err)
		}
	}
	b.Cleanup(func() {
		for _, st := range streams {
			if st != nil {
				geminio.AbortStream(st)
			}
		}
	})
	// Each index has one writer. Sorting and percentile computation are not timed.
	latencies := make([]time.Duration, b.N)
	failures := make([]int, concurrency)
	samples := make([]error, concurrency)
	firstFailure := make([]time.Duration, concurrency)
	classes := make([]map[string]int, concurrency)
	initialEdgeID := edge.EdgeID()
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.SetBytes(int64(len(payload) * 2))
	var completed, failedLive atomic.Int64
	monitorStop, monitorDone := make(chan struct{}), make(chan struct{})
	monitor := os.Getenv("FRONTIER_BENCH_MONITOR") == "1"
	started := time.Now()
	if monitor {
		go func() {
			defer close(monitorDone)
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-monitorStop:
					return
				case <-ticker.C:
					var ms runtime.MemStats
					runtime.ReadMemStats(&ms)
					fmt.Fprintf(os.Stderr, "PROGRESS mode=%s concurrency=%d elapsed=%.1fs completed=%d failed=%d edge_dials=%d service_dials=%d goroutines=%d heap_mib=%.1f heap_objects=%d num_gc=%d\n", mode, concurrency, time.Since(started).Seconds(), completed.Load(), failedLive.Load(), atomic.LoadInt64(&f.edgeDials), atomic.LoadInt64(&f.svcDials), runtime.NumGoroutine(), float64(ms.HeapAlloc)/(1024*1024), ms.HeapObjects, ms.NumGC)
				}
			}
		}()
	} else {
		close(monitorDone)
	}
	b.ResetTimer()
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			buf := make([]byte, len(payload))
			for i := worker; i < b.N; i += concurrency {
				begin := time.Now()
				var err error
				switch mode {
				case "rpc":
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					rsp, e := svc.Call(ctx, edge.EdgeID(), "echo", svc.NewRequest(payload))
					cancel()
					err = e
					if err == nil && !bytes.Equal(rsp.Data(), payload) {
						err = fmt.Errorf("RPC echo mismatch")
					}
				case "reuse":
					err = benchEcho(streams[worker], payload, buf)
				case "open":
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					st, e := svc.OpenStream(ctx, edge.EdgeID())
					cancel()
					err = e
					if err == nil {
						err = benchEcho(st, payload, buf)
						geminio.AbortStream(st)
					}
				}
				latencies[i] = time.Since(begin)
				if monitor {
					completed.Add(1)
					if err != nil {
						failedLive.Add(1)
					}
				}
				if err != nil {
					failures[worker]++
					if samples[worker] == nil {
						samples[worker] = err
						firstFailure[worker] = time.Since(started)
					}
					class := "other"
					switch {
					case errors.Is(err, service.ErrEdgeNotOnline):
						class = "edge_not_online"
					case errors.Is(err, service.ErrEdgeStreamOpen):
						class = "edge_stream_open"
					case errors.Is(err, context.DeadlineExceeded):
						class = "deadline"
					case errors.Is(err, io.EOF):
						class = "eof"
					}
					if classes[worker] == nil {
						classes[worker] = make(map[string]int)
					}
					classes[worker][class]++
				}
			}
		}(worker)
	}
	wg.Wait()
	b.StopTimer()
	close(monitorStop)
	<-monitorDone
	if monitor && b.N >= concurrency {
		fmt.Fprintf(os.Stderr, "FINAL mode=%s concurrency=%d elapsed=%.3fs completed=%d failed=%d edge_dials=%d service_dials=%d edge_id_initial=%d edge_id_final=%d\n", mode, concurrency, b.Elapsed().Seconds(), completed.Load(), failedLive.Load(), atomic.LoadInt64(&f.edgeDials), atomic.LoadInt64(&f.svcDials), initialEdgeID, edge.EdgeID())
	}
	failed := 0
	var first error
	for i, n := range failures {
		failed += n
		if first == nil {
			first = samples[i]
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	percentile := func(q float64) float64 {
		i := int(float64(len(latencies)-1) * q)
		return float64(latencies[i]) / float64(time.Millisecond)
	}
	b.ReportMetric(float64(b.N-failed)/b.Elapsed().Seconds(), "success/s")
	b.ReportMetric(float64(failed), "failures")
	b.ReportMetric(percentile(.50), "p50-ms")
	b.ReportMetric(percentile(.95), "p95-ms")
	b.ReportMetric(percentile(.99), "p99-ms")
	b.ReportMetric(float64(runtime.NumGoroutine()), "goroutines-end")
	if failed != 0 {
		firstAt := time.Duration(1<<63 - 1)
		totals := make(map[string]int)
		for i, counts := range classes {
			if failures[i] > 0 && firstFailure[i] < firstAt {
				firstAt = firstFailure[i]
				first = samples[i]
			}
			for class, count := range counts {
				totals[class] += count
			}
		}
		b.Logf("attempts=%d success=%d elapsed=%.3fs success/s=%.0f failures=%v first_failure=%.3fs edge_id_initial=%d edge_id_final=%d p99_ms=%.3f", b.N, b.N-failed, b.Elapsed().Seconds(), float64(b.N-failed)/b.Elapsed().Seconds(), totals, firstAt.Seconds(), initialEdgeID, edge.EdgeID(), percentile(.99))
		b.Errorf("%d/%d failed; first: %v", failed, b.N, first)
	}
}
func benchEcho(st geminio.Stream, payload, buf []byte) error {
	if err := st.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	if _, err := st.Write(payload); err != nil {
		return err
	}
	if _, err := io.ReadFull(st, buf); err != nil {
		return err
	}
	if !bytes.Equal(payload, buf) {
		return fmt.Errorf("stream echo mismatch")
	}
	return nil
}
