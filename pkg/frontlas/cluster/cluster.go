package cluster

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/singchia/frontier/pkg/frontlas/cluster/server"
	"github.com/singchia/frontier/pkg/frontlas/cluster/service"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

type Cluster struct {
	cm  cmux.CMux
	app *kratos.App
}

func NewCluster(conf *config.Configuration, dao *repo.Dao) (*Cluster, error) {
	listen := &conf.ControlPlane.Listen
	ln, err := utils.Listen(listen)
	if err != nil {
		klog.Errorf("control plane listen err: %s", err)
		return nil, err
	}

	// service
	svc := service.NewClusterService(dao)

	// http and grpc server
	cm := cmux.New(ln)
	grpcLn := cm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpLn := cm.Match(cmux.Any())

	gs := server.NewGRPCServer(grpcLn, svc)
	hs := server.NewHTTPServer(httpLn, svc)
	app := kratos.New(kratos.Server(gs, hs))

	return &Cluster{
		cm:  cm,
		app: app,
	}, nil
}

func (cluster *Cluster) Serve() error {
	go func() {
		err := cluster.cm.Serve()
		if err != nil {
			klog.Errorf("cluster cmux serve err: %s", err)
		}
	}()
	err := cluster.app.Run()
	if err != nil {
		klog.Errorf("cluster app run err: %s", err)
		return err
	}
	return nil
}

func (cluster *Cluster) Close() error {
	cluster.cm.Close()
	return cluster.app.Stop()
}
