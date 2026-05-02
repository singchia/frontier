// Package observability 给 frontier 和 frontlas 提供统一的可观测性 HTTP 端点：
//
//   /healthz - liveness：进程存活即 200
//   /readyz  - readiness：业务侧准备就绪后 200，否则 503
//   /metrics - Prometheus exporter，使用默认 registry（含 Go runtime + process collector）
//
// gospec 红线："所有对外服务必须暴露 /healthz、/readyz、/metrics"。
package observability

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

// Config 控制 observability HTTP server 行为。
type Config struct {
	// Enable 关闭时整个 server 不启动；默认 true（zero value 反向初始化在 New 中处理）。
	Enable bool
	// Addr 是 HTTP 监听地址，例如 "0.0.0.0:9091"。
	Addr string
}

// ReadinessFn 在每次 /readyz 被请求时调用，返回 nil 表示 ready。
// 业务侧通过它注入"必要依赖是否就绪"的判断（例如 Frontlas 已注册、Redis 可达）。
// 默认为永远 ready。
type ReadinessFn func(ctx context.Context) error

// Server 提供生命周期受控的 HTTP server。
type Server struct {
	cfg       Config
	srv       *http.Server
	readiness atomic.Pointer[ReadinessFn]
}

// New 构造一个 Server。注册 Prometheus 默认 registry 上 promauto 增量定义的所有指标。
// 如果想注册自定义 collector，业务包直接 prometheus.MustRegister 即可，这里不需要传引用。
func New(cfg Config) *Server {
	s := &Server{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.Handle("/metrics", promhttp.Handler())
	s.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	defaultReady := ReadinessFn(func(context.Context) error { return nil })
	s.readiness.Store(&defaultReady)
	return s
}

// SetReadiness 替换当前的就绪检查函数。可以在进程生命周期里多次调用，
// 比如启动初期返回 NotReady、注册到 Frontlas 后切到 Ready、SIGTERM 后切回 NotReady。
func (s *Server) SetReadiness(fn ReadinessFn) {
	if fn == nil {
		fn = func(context.Context) error { return nil }
	}
	s.readiness.Store(&fn)
}

// Run 在独立 goroutine 启动监听。Run 返回前同步检查 cfg.Enable。
func (s *Server) Run() {
	if !s.cfg.Enable {
		klog.Infof("observability server disabled")
		return
	}
	go func() {
		klog.Infof("observability server listening on %s", s.cfg.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			klog.Errorf("observability server stopped with error: %s", err)
		}
	}()
}

// Shutdown 触发 graceful shutdown，最多等 timeout。
func (s *Server) Shutdown(timeout time.Duration) {
	if !s.cfg.Enable {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		klog.Warningf("observability server shutdown: %s", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	// liveness 仅证明进程仍在响应请求；不调用 readinessFn。
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	fn := *s.readiness.Load()
	if err := fn(r.Context()); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready: " + err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
