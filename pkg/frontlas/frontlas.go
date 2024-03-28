package frontlas

import (
	"net/http"
	"runtime"

	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/frontier/pkg/frontlas/server"
	"github.com/singchia/frontier/pkg/utils"
	"k8s.io/klog/v2"
)

type Frontlas struct {
	repo   *repo.Dao
	server *server.Server
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

	return &Frontlas{
		repo:   repo,
		server: server,
	}, nil
}

func (frontlas *Frontlas) Run() {
	frontlas.server.Serve()
}

func (frontlas *Frontlas) Close() {
	frontlas.repo.Close()
	frontlas.server.Close()
	klog.Infof("frontlas ends")
	klog.Flush()
}
