package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/mq"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/server"
	"github.com/singchia/frontier/pkg/utils"
	"k8s.io/klog/v2"
)

func main() {
	conf, err := config.Parse()
	if err != nil {
		klog.Errorf("parse flags err: %s", err)
		return
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
			return
		}
	}

	klog.Infof("frontier starts")
	defer func() {
		klog.Infof("frontier ends")
		klog.Flush()
	}()

	// new repo and mqm
	repo, mqm, err := newMidwares(conf)
	if err != nil {
		klog.Errorf("new midwares err: %s", err)
		return
	}
	defer func() {
		repo.Close()
		mqm.Close()
	}()

	// servers
	srvs, err := server.NewServer(conf, repo, mqm)
	if err != nil {
		klog.Errorf("new server failed")
		return
	}
	klog.V(2).Infof("new servers succeed")
	srvs.Serve()
	defer func() {
		srvs.Close()
	}()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())
}

func newMidwares(conf *config.Configuration) (apis.Repo, apis.MQM, error) {
	// repo
	repo, err := repo.NewRepo(conf)
	if err != nil {
		klog.Errorf("new repo err: %s", err)
		return nil, nil, err
	}
	klog.V(2).Infof("new repo succeed")

	// mqm
	mqm, err := mq.NewMQM(conf)
	if err != nil {
		klog.Errorf("new mq manager err: %s", err)
		return nil, nil, err
	}
	klog.V(2).Infof("new mq manager succeed")

	return repo, mqm, nil
}
