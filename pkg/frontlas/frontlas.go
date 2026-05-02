package frontlas

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/frontier/pkg/frontlas/server"
	"github.com/singchia/frontier/pkg/observability"
	"github.com/singchia/frontier/pkg/utils"
	"k8s.io/klog/v2"
)

type Frontlas struct {
	repo   *repo.Dao
	server *server.Server
	obs    *observability.Server
}

func NewFrontlas() (*Frontlas, error) {
	conf, err := config.Parse()
	if err != nil {
		klog.Errorf("parse flags err: %s", err)
		return nil, err
	}
	// pprof
	if conf.Daemon.PProf.Enable {
		runtime.SetCPUProfileRate(conf.Daemon.PProf.CPUProfileRate)
		go func() {
			http.ListenAndServe(conf.Daemon.PProf.Addr, nil)
		}()
	}
	// rlimit
	if conf.Daemon.RLimit.Enable {
		err = utils.SetRLimit(uint64(conf.Daemon.RLimit.NumFile))
		if err != nil {
			klog.Errorf("set rlimit err: %s", err)
			return nil, err
		}
	}
	klog.Infof("frontlas starts")

	// new repo
	repo, err := repo.NewDao(conf)
	if err != nil {
		klog.Errorf("new dao err: %s", err)
		return nil, err
	}

	// servers
	server, err := server.NewServer(conf, repo)
	if err != nil {
		klog.Errorf("new server err: %s", err)
		return nil, err
	}

	// observability：默认开启；地址不填走 0.0.0.0:9092（与 frontier 9091 错开）。
	obsCfg := conf.Observability
	if obsCfg.Addr == "" {
		obsCfg.Addr = "0.0.0.0:9092"
	}
	if !obsCfg.Enable {
		obsCfg.Enable = true
	}
	obs := observability.New(observability.Config{Enable: obsCfg.Enable, Addr: obsCfg.Addr})
	// readiness 反映 Redis 是否可达。
	obs.SetReadiness(func(ctx context.Context) error {
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return repo.Ping(c)
	})

	return &Frontlas{
		repo:   repo,
		server: server,
		obs:    obs,
	}, nil
}

func (frontlas *Frontlas) Run() {
	frontlas.obs.Run()
	frontlas.server.Serve()
}

func (frontlas *Frontlas) Close() {
	frontlas.obs.Shutdown(5 * time.Second)
	frontlas.repo.Close()
	frontlas.server.Close()
	klog.Infof("frontlas ends")
	klog.Flush()
}
