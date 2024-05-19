package frontier

import (
	"net/http"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/mq"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/server"
	"github.com/singchia/frontier/pkg/utils"
	"k8s.io/klog/v2"
)

type Frontier struct {
	repo   apis.Repo
	mqm    apis.MQM
	server *server.Server
}

func NewFrontier() (*Frontier, error) {
	conf, err := config.Parse()
	if err != nil {
		klog.Errorf("parse flags err: %s", err)
		return nil, err
	}
	// pprof
	if conf.Daemon.PProf.Enable {
		//runtime.SetCPUProfileRate(conf.Daemon.PProf.CPUProfileRate)
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
	klog.Infof("frontier starts")

	// new repo and mqm
	repo, mqm, err := newMidwares(conf)
	if err != nil {
		klog.Errorf("new midwares err: %s", err)
		return nil, err
	}

	// servers
	server, err := server.NewServer(conf, repo, mqm)
	if err != nil {
		klog.Errorf("new server err: %s", err)
		return nil, err
	}

	return &Frontier{
		repo:   repo,
		mqm:    mqm,
		server: server,
	}, nil
}

func (frontier *Frontier) Run() {
	frontier.server.Serve()
}

func (frontier *Frontier) Close() {
	frontier.repo.Close()
	frontier.mqm.Close()
	frontier.server.Close()
	klog.Infof("frontier ends")
	klog.Flush()
}

func newMidwares(conf *config.Configuration) (apis.Repo, apis.MQM, error) {
	// repo
	repo, err := repo.NewRepo(conf)
	if err != nil {
		klog.Errorf("new repo err: %s", err)
		return nil, nil, err
	}

	// mqm
	mqm, err := mq.NewMQM(conf)
	if err != nil {
		klog.Errorf("new mq manager err: %s", err)
		return nil, nil, err
	}
	return repo, mqm, nil
}
