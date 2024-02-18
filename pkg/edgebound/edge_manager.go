package edgebound

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"sync"

	"github.com/jumboframes/armorigo/rproxy"
	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/security"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

func NewEdgebound(conf *config.Configuration, dao *dao.Dao, informer api.EdgeInformer,
	exchange api.Exchange, tmr timer.Timer) (api.Edgebound, error) {
	return newEdgeManager(conf, dao, informer, exchange, tmr)
}

type edgeManager struct {
	*delegate.UnimplementedDelegate
	conf *config.Configuration

	informer api.EdgeInformer
	exchange api.Exchange

	// edgeID allocator
	idFactory id.IDFactory
	shub      *synchub.SyncHub
	// cache
	// key: edgeID; value: geminio.End
	// edges sync.Map
	edges map[uint64]geminio.End
	mtx   sync.RWMutex
	// key: edgeID; subkey: streamID; value: geminio.Stream
	// we don't store stream info to dao, because they may will be too much.
	streams *mapmap.MapMap

	// dao and repo for edges
	dao *dao.Dao
	// listener for edges
	cm        cmux.CMux
	geminioLn net.Listener
	rp        *rproxy.RProxy

	// timer for all edge ends
	tmr timer.Timer
}

// support for tls, mtls and tcp listening
func newEdgeManager(conf *config.Configuration, dao *dao.Dao, informer api.EdgeInformer,
	exchange api.Exchange, tmr timer.Timer) (*edgeManager, error) {
	listen := &conf.Edgebound.Listen
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

	em := &edgeManager{
		conf:                  conf,
		tmr:                   tmr,
		streams:               mapmap.NewMapMap(),
		dao:                   dao,
		shub:                  synchub.NewSyncHub(synchub.OptionTimer(tmr)),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
		// a simple unix timestamp incemental id factory
		idFactory: id.DefaultIncIDCounter,
		informer:  informer,
		exchange:  exchange,
	}

	if !listen.TLS.Enable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("edge manager net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}

	} else {
		// load all certs to listen
		certs := []tls.Certificate{}
		for _, certFile := range listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("edge manager tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
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
				klog.Errorf("edge manager tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}

		} else {
			// mtls, require for edge cert
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range listen.TLS.CACerts {
				ca, err := os.ReadFile(caFile)
				if err != nil {
					klog.Errorf("edge manager read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("edge manager append ca cert to ca pool err: %s, file: %s", err, caFile)
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
				klog.Errorf("edge manager tls listen err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
		}
	}

	geminioLn := ln
	bypass := conf.Edgebound.Bypass.Enable
	if bypass {
		// multiplexer
		cm := cmux.New(ln)
		// the first byte is geminio Version, the second byte is geminio ConnPacket
		// TODO we should have a magic number here
		geminioLn = cm.Match(cmux.PrefixMatcher(string([]byte{0x01, 0x01})))
		anyLn := cm.Match(cmux.Any())
		rp, err := rproxy.NewRProxy(anyLn, rproxy.OptionRProxyDial(em.bypassDial))
		if err != nil {
			klog.Errorf("edge manager new rproxy err: %s", err)
			return nil, err
		}
		em.cm = cm
		em.rp = rp
	}
	em.geminioLn = geminioLn
	return em, nil
}

func (em *edgeManager) bypassDial(_ net.Addr, _ interface{}) (net.Conn, error) {
	bypass := &em.conf.Edgebound.Bypass
	var (
		network string = bypass.Network
		addr    string = bypass.Addr
	)

	if !bypass.TLS.Enable {
		conn, err := net.Dial(network, addr)
		if err != nil {
			return nil, err
		}
		return conn, err
	} else {
		// load all certs to dial
		certs := []tls.Certificate{}
		for _, certFile := range bypass.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("edge manager bypass tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
				continue
			}
			certs = append(certs, cert)
		}

		if !bypass.TLS.MTLS {
			// tls
			conn, err := tls.Dial(network, addr, &tls.Config{
				Certificates: certs,
				// it's user's call to verify the server certs or not.
				InsecureSkipVerify: bypass.TLS.InsecureSkipVerify,
			})
			if err != nil {
				klog.Errorf("edge manager bypass tls dial err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
			return conn, nil
		} else {
			// mtls, dial with our certs
			// load all ca certs to pool
			caPool := x509.NewCertPool()
			for _, caFile := range bypass.TLS.CACerts {
				ca, err := os.ReadFile(caFile)
				if err != nil {
					klog.Errorf("edge manager bypass read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("edge manager bypass append ca cert to ca pool err: %s, file: %s", err, caFile)
					continue
				}
			}
			conn, err := tls.Dial(network, addr, &tls.Config{
				Certificates: certs,
				// we should not skip the verify.
				InsecureSkipVerify: bypass.TLS.InsecureSkipVerify,
				RootCAs:            caPool,
			})
			if err != nil {
				klog.Errorf("edge manager bypass tls dial err: %s, network: %s, addr: %s", err, network, addr)
				return nil, err
			}
			return conn, nil
		}
	}
}

// Serve blocks until the Accept error
func (em *edgeManager) Serve() {
	bypass := &em.conf.Edgebound.Bypass
	if bypass.Enable {
		go em.cm.Serve()
		go em.rp.Proxy(context.TODO())
	}

	for {
		conn, err := em.geminioLn.Accept()
		if err != nil {
			klog.V(4).Infof("edge manager listener accept err: %s", err)
			return
		}
		go em.handleConn(conn)
	}
}

func (em *edgeManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(em.tmr)
	opt.SetDelegate(em)
	// stream handler
	opt.SetAcceptStreamFunc(em.acceptStream)
	opt.SetClosedStreamFunc(em.closedStream)
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("edge manager geminio server new end err: %s", err)
		return err
	}

	// handle online event for end
	if err = em.online(end); err != nil {
		return err
	}
	// forward and stream up to service
	em.forward(end)
	return nil
}

func (em *edgeManager) GetEdgeByID(edgeID uint64) geminio.End {
	em.mtx.RLock()
	defer em.mtx.RUnlock()

	return em.edges[edgeID]
}

func (em *edgeManager) ListEdges() []geminio.End {
	ends := []geminio.End{}
	em.mtx.RLock()
	defer em.mtx.RUnlock()

	for _, value := range em.edges {
		ends = append(ends, value)
	}
	return ends
}

func (em *edgeManager) CountEdges() int {
	em.mtx.RLock()
	defer em.mtx.RUnlock()
	return len(em.edges)
}

func (em *edgeManager) ListStreams(edgeID uint64) []geminio.Stream {
	all := em.streams.MGetAll(edgeID)
	return utils.Slice2streams(all)
}

func (em *edgeManager) DelEdgeByID(edgeID uint64) error {
	panic("TODO")
}

// Close all edges and manager
func (em *edgeManager) Close() error {
	bypass := &em.conf.Edgebound.Bypass
	if bypass.Enable {
		em.cm.Close()
		em.rp.Close()
	}
	if err := em.geminioLn.Close(); err != nil {
		return err
	}
	return nil
}
