package clientbound

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"sync"

	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type Clientbound interface {
	ListClients() []geminio.End
	GetClientByID(clientID uint64) geminio.End
	DelClientByID(clientID uint64) error
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

type clientManager struct {
	// key: clientID; value: geminio.End
	clients sync.Map

	// database for clients
	db *gorm.DB

	tmr timer.Timer

	ln net.Listener
}

func newclientManager(conf *config.Clientbound, tmr timer.Timer) (*clientManager, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		klog.Errorf("client manager open sqlite3 err: %s", err)
		return nil, err
	}

	cm := &clientManager{
		tmr: tmr,
		db:  db,
	}

	var (
		ln      net.Listener
		network string = conf.Listen.Network
		addr    string = conf.Listen.Addr
	)

	if !conf.Listen.TLSEnable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}

	} else {
		// load all certs to listen
		certs := []tls.Certificate{}
		for _, certFile := range conf.Listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
				continue
			}
			certs = append(certs, cert)
		}

		if !conf.Listen.TLS.MTLS {
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
			// mtls, require for client cert
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range conf.Listen.TLS.CACerts {
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

	cm.ln = ln
	return cm, nil
}

func (cm *clientManager) Serve() {
	for {
		conn, err := cm.ln.Accept()
		if err != nil {
			klog.Errorf("listener accept err: %s", err)
			return
		}
		go cm.handleConn(conn)
	}
}

func (cm *clientManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(cm.tmr)
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("geminio server new end err: %s", err)
		return err
	}

	// handle online event for end
	if err = cm.online(end); err != nil {
		return err
	}

	return nil
}

func (cm *clientManager) ListClients() []geminio.End {
	ends := []geminio.End{}
	cm.clients.Range(func(key, value any) bool {
		ends = append(ends, value.(geminio.End))
		return true
	})
	return ends
}

// delegations for all ends
func (cm *clientManager) Online(clientID uint64, meta []byte, addr net.Addr) error {
	return nil
}

// Close all clients and manager
func (cm *clientManager) Close() error {
	if err := cm.ln.Close(); err != nil {
		return err
	}
	return nil
}
