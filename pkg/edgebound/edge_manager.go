package edgebound

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"sync"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

type Clientbound interface {
	ListClients() []geminio.End
	GetClientByID(edgeID uint64) geminio.End
	DelClientByID(edgeID uint64) error
}

var (
	// safe ciperSuites with DH exchange algorithms.
	ciperSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_FALLBACK_SCSV,
	}
)

type edgeManager struct {
	*delegate.UnimplementedDelegate
	// cache
	// key: edgeID; value: geminio.End
	edges sync.Map

	shub *synchub.SyncHub

	// dao and repo for edges
	dao *dao.Dao

	// timer for all edge ends
	tmr timer.Timer

	// listener for edges
	ln net.Listener
}

// support for tls, mtls and tcp listening
func newedgeManager(conf *config.Configuration, dao *dao.Dao, tmr timer.Timer) (*edgeManager, error) {
	var (
		ln      net.Listener
		network string = conf.Edgebound.Listen.Network
		addr    string = conf.Edgebound.Listen.Addr
		err     error
	)

	em := &edgeManager{
		tmr:                   tmr,
		dao:                   dao,
		shub:                  synchub.NewSyncHub(synchub.OptionTimer(tmr)),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
	}

	if !conf.Edgebound.Listen.TLSEnable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}

	} else {
		// load all certs to listen
		certs := []tls.Certificate{}
		for _, certFile := range conf.Edgebound.Listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
				continue
			}
			certs = append(certs, cert)
		}

		if !conf.Edgebound.Listen.TLS.MTLS {
			// tls
			if ln, err = tls.Listen(network, addr, &tls.Config{
				MinVersion:   tls.VersionTLS12,
				CipherSuites: ciperSuites,
				Certificates: certs,
			}); err != nil {
				klog.Errorf("tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}

		} else {
			// mtls, require for edge cert
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range conf.Edgebound.Listen.TLS.CACerts {
				ca, err := os.ReadFile(caFile)
				if err != nil {
					klog.Errorf("read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("append ca cert to ca pool err: %s, file: %s", err, caFile)
					continue
				}
			}
			if ln, err = tls.Listen(network, addr, &tls.Config{
				MinVersion:   tls.VersionTLS12,
				CipherSuites: ciperSuites,
				ClientCAs:    caPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				Certificates: certs,
			}); err != nil {
				klog.Errorf("tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
		}
	}

	em.ln = ln
	return em, nil
}

func (em *edgeManager) Serve() {
	for {
		conn, err := em.ln.Accept()
		if err != nil {
			klog.Errorf("listener accept err: %s", err)
			return
		}
		go em.handleConn(conn)
	}
}

func (em *edgeManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(em.tmr)
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("geminio server new end err: %s", err)
		return err
	}

	// handle online event for end
	if err = em.online(end); err != nil {
		return err
	}
	return nil
}

func (em *edgeManager) ListClients() []geminio.End {
	ends := []geminio.End{}
	em.edges.Range(func(key, value any) bool {
		ends = append(ends, value.(geminio.End))
		return true
	})
	return ends
}

// Close all edges and manager
func (em *edgeManager) Close() error {
	if err := em.ln.Close(); err != nil {
		return err
	}
	return nil
}
