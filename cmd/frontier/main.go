package main

import (
	"context"

	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/edgebound"
	"github.com/singchia/frontier/pkg/exchange"
	"github.com/singchia/frontier/pkg/mq"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/servicebound"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

func main() {
	conf, err := config.Parse()
	if err != nil {
		klog.Errorf("parse flags err: %s", err)
		return
	}
	// dao
	dao, err := dao.NewDao(conf)
	if err != nil {
		klog.Errorf("new dao err: %s", err)
		return
	}
	// mqm
	mqm, err := mq.NewMQM(conf)
	if err != nil {
		klog.Errorf("new mq manager err: %s", err)
		return
	}
	// exchange
	exchange, err := exchange.NewExchange(conf)
	if err != nil {
		klog.Errorf("new exchange err: %s", err)
		return
	}

	tmr := timer.NewTimer()
	// servicebound
	servicebound, err := servicebound.NewServicebound(conf, dao, nil, exchange, mqm, tmr)
	if err != nil {
		klog.Errorf("new servicebound err: %s", err)
		return
	}
	servicebound.Serve()

	// edgebound
	edgebound, err := edgebound.NewEdgebound(conf, dao, nil, exchange, tmr)
	if err != nil {
		klog.Errorf("new edgebound err: %s", err)
		return
	}
	edgebound.Serve()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())
	edgebound.Close()
	servicebound.Close()
}
