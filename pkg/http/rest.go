package http

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/security"
	"k8s.io/klog/v2"
)

type Rest struct {
	conf   *config.Configuration
	router *mux.Router

	// listener for http
	ln net.Listener
}

func NewRest(conf *config.Configuration) (*Rest, error) {
	listen := &conf.Http.Listen
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

	rest := &Rest{
		conf: conf,
	}

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
	rest.ln = ln

	// router

	return rest, nil
}

func (rest *Rest) Serve() {
	err := http.Serve(rest.ln, rest.router)
	if err != nil {
		if !strings.Contains(err.Error(), apis.ErrStrUseOfClosedConnection) {
			klog.V(1).Infof("rest listener serve err: %s", err)
		}
	}
}
