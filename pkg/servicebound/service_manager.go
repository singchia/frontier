package servicebound

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"sync"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/security"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

type Servicebound interface {
	ListService() []geminio.End
	// for management
	GetService(service string) geminio.End
	DelSerivces(service string) error
}

type ServiceInformer interface {
	ServiceOnline(serviceID uint64, service string, addr net.Addr)
	ServiceOffline(serviceID uint64, service string, addr net.Addr)
	ServiceHeartbeat(serviceID uint64, service string, addr net.Addr)
}

type Exchange interface {
	// rpc, message and raw io to edge
	ForwardToService(geminio.End)
	// stream to edge
	StreamToEdge(geminio.Stream)
}

type serviceManager struct {
	*delegate.UnimplementedDelegate

	informer ServiceInformer
	exchange Exchange
	conf     *config.Configuration
	// serviceID allocator
	idFactory id.IDFactory
	shub      *synchub.SyncHub
	// cache
	// key: serviceID; value: geminio.End
	services map[uint64]geminio.End
	mtx      sync.RWMutex
	// key: serviceID; subkey: streamID; value: geminio.Stream
	// we don't store stream info to dao, because they may will be too much.
	streams *mapmap.MapMap

	// dao and repo for services
	dao *dao.Dao
	ln  net.Listener

	// timer for all service ends
	tmr timer.Timer
}

func newServiceManager(conf *config.Configuration, dao *dao.Dao, informer ServiceInformer,
	exchange Exchange, tmr timer.Timer) (*serviceManager, error) {
	listen := &conf.Servicebound.Listen
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

	sm := &serviceManager{
		conf:                  conf,
		tmr:                   tmr,
		streams:               mapmap.NewMapMap(),
		dao:                   dao,
		shub:                  synchub.NewSyncHub(synchub.OptionTimer(tmr)),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
		// a simple unix timestamp incremental id factory
		idFactory: id.DefaultIncIDCounter,
		informer:  informer,
	}

	if !listen.TLS.Enable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("service manager net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}
	} else {
		// load all certs to listen
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
	sm.ln = ln
	return sm, nil
}

func (sm *serviceManager) Serve() {
	for {
		conn, err := sm.ln.Accept()
		if err != nil {
			klog.V(4).Infof("service manager listener accept err: %s", err)
			return
		}
		go sm.handleConn(conn)
	}
}

func (sm *serviceManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(sm.tmr)
	opt.SetDelegate(sm)
	// stream handler
	opt.SetAcceptStreamFunc(sm.acceptStream)
	opt.SetClosedStreamFunc(sm.closedStream)
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("service manager geminio server new end err: %s", err)
		return err
	}

	// handle online event for end
	if err = sm.online(end); err != nil {
		return err
	}

	// register methods for service
	if err = end.Register(context.TODO(), "topic_claim", sm.RemoteReceiveClaim); err != nil {
		return err
	}

	// forward and stream up to edge
	sm.forward(end)
	return nil
}
