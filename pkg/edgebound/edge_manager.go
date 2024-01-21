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
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

type Edgebound interface {
	ListEdges() []geminio.End
	GetEdgeByID(edgeID uint64) geminio.End
	DelEdgeByID(edgeID uint64) error
}

type EdgeInformer interface {
	EdgeOnline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeOffline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr)
}

type Exchange interface {
	// rpc, message and raw io to service
	ForwardToService(geminio.End)
	// stream to service
	StreamToService(geminio.Stream)
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

	informer EdgeInformer
	exchange Exchange
	conf     *config.Configuration
	// edgeID allocator
	idFactory id.IDFactory
	shub      *synchub.SyncHub
	// cache
	// key: edgeID; value: geminio.End
	edges sync.Map
	// key: edgeID-streamID; value: geminio.Stream
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
func newEdgeManager(conf *config.Configuration, dao *dao.Dao, informer EdgeInformer,
	exchange Exchange, tmr timer.Timer) (*edgeManager, error) {
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
	}

	if !listen.TLS.Enable {
		if ln, err = net.Listen(network, addr); err != nil {
			klog.Errorf("net listen err: %s, network: %s, addr: %s", err, network, addr)
			return nil, err
		}

	} else {
		// load all certs to listen
		certs := []tls.Certificate{}
		for _, certFile := range listen.TLS.Certs {
			cert, err := tls.LoadX509KeyPair(certFile.Cert, certFile.Key)
			if err != nil {
				klog.Errorf("tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
				continue
			}
			certs = append(certs, cert)
		}

		if !listen.TLS.MTLS {
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
			for _, caFile := range listen.TLS.CACerts {
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
			klog.Errorf("new rproxy err: %s", err)
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
				klog.Errorf("tls load x509 cert err: %s, cert: %s, key: %s", err, certFile.Cert, certFile.Key)
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
				klog.Errorf("tls dial err: %s, network: %s, addr: %s", err, network, addr)
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
					klog.Errorf("read ca cert err: %s, file: %s", err, caFile)
					return nil, err
				}
				if !caPool.AppendCertsFromPEM(ca) {
					klog.Warningf("append ca cert to ca pool err: %s, file: %s", err, caFile)
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
				klog.Errorf("tls dial err: %s, network: %s, addr: %s", err, network, addr)
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
			klog.V(4).Infof("listener accept err: %s", err)
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
		klog.Errorf("geminio server new end err: %s", err)
		return err
	}

	// handle online event for end
	if err = em.online(end); err != nil {
		return err
	}
	// TODO forward and stream up to service
	return nil
}

func (em *edgeManager) ListEdges() []geminio.End {
	ends := []geminio.End{}
	em.edges.Range(func(key, value any) bool {
		ends = append(ends, value.(geminio.End))
		return true
	})
	return ends
}

func (em *edgeManager) ListStreams(edgeID uint64) []geminio.Stream {
	all := em.streams.MGetAll(edgeID)
	return slice2streams(all)
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
