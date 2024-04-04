package edgebound

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/rproxy"
	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"github.com/soheilhy/cmux"
	"k8s.io/klog/v2"
)

func NewEdgebound(conf *config.Configuration, repo apis.Repo, informer apis.EdgeInformer,
	exchange apis.Exchange, tmr timer.Timer) (apis.Edgebound, error) {
	return newEdgeManager(conf, repo, informer, exchange, tmr)
}

type edgeManager struct {
	*delegate.UnimplementedDelegate
	conf *config.Configuration

	informer apis.EdgeInformer
	exchange apis.Exchange

	// edgeID allocator
	idFactory id.IDFactory
	shub      *synchub.SyncHub
	// cache
	// key: edgeID; value: geminio.End
	// edges sync.Map
	edges map[uint64]geminio.End
	mtx   sync.RWMutex
	// key: edgeID; subkey: streamID; value: geminio.Stream
	// we don't store stream info to repo, because they may will be too much.
	streams *mapmap.MapMap

	// repo and repo for edges
	repo apis.Repo
	// listener for edges
	cm        cmux.CMux
	geminioLn net.Listener
	rp        *rproxy.RProxy

	// timer for all edge ends
	tmr timer.Timer
}

// support for tls, mtls and tcp listening
func newEdgeManager(conf *config.Configuration, repo apis.Repo, informer apis.EdgeInformer,
	exchange apis.Exchange, tmr timer.Timer) (*edgeManager, error) {
	listen := &conf.Edgebound.Listen

	em := &edgeManager{
		conf:                  conf,
		tmr:                   tmr,
		streams:               mapmap.NewMapMap(),
		repo:                  repo,
		shub:                  synchub.NewSyncHub(synchub.OptionTimer(tmr)),
		edges:                 make(map[uint64]geminio.End),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
		// a simple unix timestamp incemental id factory
		idFactory: id.DefaultIncIDCounter,
		informer:  informer,
		exchange:  exchange,
	}
	exchange.AddEdgebound(em)

	ln, err := utils.Listen(listen)
	if err != nil {
		klog.Errorf("edge manager listen err: %s", err)
		return nil, err
	}

	geminioLn := ln
	bypass := conf.Edgebound.BypassEnable
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
	return utils.Dial(&em.conf.Edgebound.Bypass)
}

// Serve blocks until the Accept error
func (em *edgeManager) Serve() error {
	if em.conf.Edgebound.BypassEnable {
		go em.cm.Serve()
		go em.rp.Proxy(context.TODO())
	}

	for {
		conn, err := em.geminioLn.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), apis.ErrStrUseOfClosedConnection) {
				klog.V(1).Infof("edge manager listener accept err: %s", err)
				return err
			}
			break
		}
		go em.handleConn(conn)
	}
	return nil
}

func (em *edgeManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(em.tmr)
	opt.SetDelegate(em)
	// stream handler
	opt.SetAcceptStreamFunc(em.acceptStream)
	opt.SetClosedStreamFunc(em.closedStream)
	opt.SetLog(log.NewKLog())
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

// management apis
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
	// TODO test it
	em.mtx.RLock()
	defer em.mtx.RUnlock()

	edge, ok := em.edges[edgeID]
	if !ok {
		return apis.ErrEdgeNotOnline
	}
	return edge.Close()
}

// Close all edges and manager
func (em *edgeManager) Close() error {
	if em.conf.Edgebound.BypassEnable {
		em.cm.Close()
		em.rp.Close()
	}
	if err := em.geminioLn.Close(); err != nil {
		return err
	}
	return nil
}
