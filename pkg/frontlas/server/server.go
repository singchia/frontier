package server

import (
	"github.com/singchia/frontier/pkg/frontlas/cluster"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/singchia/frontier/pkg/frontlas/frontierbound"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

type Server struct {
	tmr     timer.Timer
	cluster *cluster.Cluster
	fm      *frontierbound.FrontierManager
}

func NewServer(conf *config.Configuration, repo *repo.Dao) (*Server, error) {
	tmr := timer.NewTimer()

	// frontierbound
	fm, err := frontierbound.NewFrontierManager(conf, repo, tmr)
	if err != nil {
		klog.Errorf("new frontier manager err: %s", err)
		return nil, err
	}

	// cluster
	cluster, err := cluster.NewCluster(conf, repo)
	if err != nil {
		klog.Errorf("new cluster err: %s", err)
		return nil, err
	}

	return &Server{
		tmr:     tmr,
		cluster: cluster,
		fm:      fm,
	}, nil
}

func (s *Server) Serve() {
	go s.cluster.Serve()
	go s.fm.Serve()
}

func (s *Server) Close() {
	s.cluster.Close()
	s.fm.Close()
}
