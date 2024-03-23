package frontierbound

import (
	"context"
	"net"
	"strings"

	"github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

const (
	// edge related
	RPCEdgeOnline    = "edge_online"
	RPCEdgeOffline   = "edge_offline"
	RPCEdgeHeartbeat = "edge_heartbeat"

	// service related
	RPCServiceOnline    = "service_online"
	RPCServiceOffline   = "service_offline"
	RPCServiceHeartbeat = "service_heartbeat"

	// frontier related
	RPCFrontierStats = "frontier_stats"
)

type FrontierManager struct {
	*delegate.UnimplementedDelegate
	conf *config.Configuration

	repo *repo.Dao
	tmr  timer.Timer

	ln net.Listener
}

func NewFrontierManager(conf *config.Configuration, dao *repo.Dao, tmr timer.Timer) (*FrontierManager, error) {
	listen := &conf.FrontierManager.Listen

	fm := &FrontierManager{
		conf:                  conf,
		tmr:                   tmr,
		repo:                  dao,
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
	}
	ln, err := utils.Listen(listen)
	if err != nil {
		klog.Errorf("frontier plane listen err: %s", err)
		return nil, err
	}

	fm.ln = ln
	return fm, nil
}

func (fm *FrontierManager) Serve() error {
	for {
		conn, err := fm.ln.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), apis.ErrStrUseOfClosedConnection) {
				klog.V(1).Infof("frontiper manager listener accept err: %s", err)
				return err
			}
			break
		}
		go fm.handleConn(conn)
	}
	return nil
}

func (fm *FrontierManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(fm.tmr)
	opt.SetDelegate(fm)
	opt.SetLog(log.NewKLog())
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("frontier manager handle conn, geminio server new err: %s", err)
		return err
	}
	err = fm.register(end)
	if err != nil {
		klog.Errorf("frontier manager handle conn, register err: %s", err)
		return err
	}
	return nil
}

func (fm *FrontierManager) register(end geminio.End) error {
	// edge_online, edge_offline, edge_heartbeat
	err := end.Register(context.TODO(), RPCEdgeOnline, fm.EdgeOnline)
	if err != nil {
		klog.Errorf("register edge_online err: %s", err)
		return err
	}
	err = end.Register(context.TODO(), RPCEdgeOffline, fm.EdgeOffline)
	if err != nil {
		klog.Errorf("register edge_offline err: %s", err)
		return err
	}
	err = end.Register(context.TODO(), RPCEdgeHeartbeat, fm.EdgeHeartbeat)
	if err != nil {
		klog.Errorf("register edge_heartbeat err: %s", err)
		return err
	}

	// service_online, service_offline, service_heartbeat
	err = end.Register(context.TODO(), RPCServiceOnline, fm.ServiceOnline)
	if err != nil {
		klog.Errorf("register service_online err: %s", err)
		return err
	}
	err = end.Register(context.TODO(), RPCServiceOffline, fm.ServiceOffline)
	if err != nil {
		klog.Errorf("register service_offline err: %s", err)
		return err
	}
	err = end.Register(context.TODO(), RPCServiceHeartbeat, fm.ServiceHeartbeat)
	if err != nil {
		klog.Errorf("register service_heartbeat err: %s", err)
		return err
	}

	// frontier_stats, frontier_metrics
	return nil
}
