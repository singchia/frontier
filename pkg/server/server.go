package server

import (
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/edgebound"
	"github.com/singchia/frontier/pkg/exchange"
	"github.com/singchia/frontier/pkg/servicebound"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

type Server struct {
	tmr          timer.Timer
	servicebound apis.Servicebound
	edgebound    apis.Edgebound
}

func NewServer(conf *config.Configuration, repo apis.Repo, mqm apis.MQM) (*Server, error) {
	tmr := timer.NewTimer()

	// exchange
	exchange := exchange.NewExchange(conf, mqm)

	// servicebound
	servicebound, err := servicebound.NewServicebound(conf, repo, nil, exchange, mqm, tmr)
	if err != nil {
		klog.Errorf("new servicebound err: %s", err)
		return nil, err
	}
	klog.V(2).Infof("new servicebound succeed")

	// edgebound
	edgebound, err := edgebound.NewEdgebound(conf, repo, nil, exchange, tmr)
	if err != nil {
		klog.Errorf("new edgebound err: %s", err)
		return nil, err
	}
	klog.V(2).Infof("new edgebound succeed")

	return &Server{
		tmr:          tmr,
		servicebound: servicebound,
		edgebound:    edgebound,
	}, nil

}

func (s *Server) Serve() {
	go s.servicebound.Serve()
	go s.edgebound.Serve()
}

func (s *Server) Close() {
	s.servicebound.Close()
	s.edgebound.Close()
	s.tmr.Close()
}
