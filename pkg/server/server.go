package server

import (
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/controlplane"
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
	controlplane *controlplane.ControlPlane
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

	// controlplane
	controlplane, err := controlplane.NewControlPlane(conf, repo, servicebound, edgebound)
	if err != nil {
		klog.Errorf("new controlplane err: %s", err)
		return nil, err
	}
	klog.V(2).Infof("new controlplane succeed")

	return &Server{
		tmr:          tmr,
		servicebound: servicebound,
		edgebound:    edgebound,
		controlplane: controlplane,
	}, nil

}

func (s *Server) Serve() {
	go s.servicebound.Serve()
	go s.edgebound.Serve()
	go s.controlplane.Serve()
}

func (s *Server) Close() {
	s.servicebound.Close()
	s.edgebound.Close()
	s.controlplane.Close()
	s.tmr.Close()
}
