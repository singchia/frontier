package controlplane

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/controlplane/server"
	"github.com/singchia/frontier/pkg/controlplane/service"
	"github.com/singchia/frontier/pkg/security"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

type ControlPlane struct {
	cm  cmux.CMux
	app *kratos.App
}

func NewControlPlane(conf *config.Configuration, repo apis.Repo, servicebound apis.Servicebound, edgebound apis.Edgebound) (*ControlPlane, error) {
	listen := &conf.ControlPlane.Listen
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

	if !listen.TLS.Enable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("rest net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}
	} else {
		certs := []tls.Certificate{}
		for _, certFile := range listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("service manager tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
				continue
			}
			certs = append(certs, cert)
		}

		if !listen.TLS.MTLS {
			// tls
			if ln, err = tls.Listen(network, addr, &tls.Config{
				MinVersion:   tls.VersionTLS12,
				CipherSuites: security.CiperSuites,
				Certificates: certs,
			}); err != nil {
				klog.Errorf("service manager tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}

		} else {
			// mtls, require for edge cert
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range listen.TLS.CACerts {
				ca, err := os.ReadFile(caFile)
				if err != nil {
					klog.Errorf("service manager read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("service manager append ca cert to ca pool err: %s, file: %s", err, caFile)
					continue
				}
			}
			if ln, err = tls.Listen(network, addr, &tls.Config{
				MinVersion:   tls.VersionTLS12,
				CipherSuites: security.CiperSuites,
				ClientCAs:    caPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				Certificates: certs,
			}); err != nil {
				klog.Errorf("service manager tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
		}
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
