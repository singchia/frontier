package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"github.com/pion/transport/v2/udp"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/security"
	"k8s.io/klog/v2"
)

func Listen(listen *config.Listen) (net.Listener, error) {
	switch listen.Network {
	case "tcp":
		return listenTCP(listen)
	case "udp":
		return listenUDP(listen)
	}
	return nil, fmt.Errorf("unsupported network: %s", listen.Network)
}

func listenTCP(listen *config.Listen) (net.Listener, error) {
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

	if !listen.TLS.Enable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}

	} else {
		// load all certs to listen
		certs := []tls.Certificate{}
		for _, certFile := range listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("listen tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
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
				klog.Errorf("listen tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}

		} else {
			// mtls, require for edge cert
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range listen.TLS.CACerts {
				ca, err := os.ReadFile(caFile)
				if err != nil {
					klog.Errorf("listen read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("listen append ca cert to ca pool err: %s, file: %s", err, caFile)
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
				klog.Errorf("listen tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
		}
	}
	return ln, nil
}

func listenUDP(listen *config.Listen) (net.Listener, error) {
	addr, err := net.ResolveUDPAddr(listen.Network, listen.Addr)
	if err != nil {
		klog.Errorf("listen resolve udp addr err: %s, network: %s, addr: %s", err, listen.Network, listen.Addr)
		return nil, err
	}
	return udp.Listen("udp", addr)
}
