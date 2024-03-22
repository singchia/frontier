package controlplane

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/controlplane/server"
	"github.com/singchia/frontier/pkg/frontier/controlplane/service"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

type ControlPlane struct {
	cm  cmux.CMux
	app *kratos.App
}

func NewControlPlane(conf *config.Configuration, repo apis.Repo, servicebound apis.Servicebound, edgebound apis.Edgebound) (*ControlPlane, error) {
	listen := &conf.ControlPlane.Listen
	ln, err := utils.Listen(listen)
	if err != nil {
		klog.Errorf("control plane listen err: %s", err)
		return nil, err
	}

	// service
	svc := service.NewControlPlaneService(repo, servicebound, edgebound)

	// http and grpc server
	cm := cmux.New(ln)
	grpcLn := cm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpLn := cm.Match(cmux.Any())

	gs := server.NewGRPCServer(grpcLn, svc)
	hs := server.NewHTTPServer(httpLn, svc)
	app := kratos.New(kratos.Server(gs, hs))

	return &ControlPlane{
		cm:  cm,
		app: app,
	}, nil
}

func (cp *ControlPlane) Serve() error {
	go func() {
		err := cp.cm.Serve()
		if err != nil {
			klog.Errorf("control plane cmux serve err: %s", err)
		}
	}()
	err := cp.app.Run()
	if err != nil {
		klog.Errorf("control plane app run err: %s", err)
		return err
	}
	return nil
}

func (cp *ControlPlane) Close() error {
	cp.cm.Close()
	return cp.app.Stop()
}
