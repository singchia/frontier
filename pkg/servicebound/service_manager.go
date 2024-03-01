package servicebound

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/repo/model"
	"github.com/singchia/frontier/pkg/security"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

func NewServicebound(conf *config.Configuration, dao *dao.Dao, informer apis.ServiceInformer,
	exchange apis.Exchange, mqm apis.MQM, tmr timer.Timer) (apis.Servicebound, error) {
	return newServiceManager(conf, dao, informer, exchange, mqm, tmr)
}

type end struct {
	End  geminio.End
	Meta *apis.Meta
}

type serviceManager struct {
	*delegate.UnimplementedDelegate
	conf *config.Configuration

	informer apis.ServiceInformer
	exchange apis.Exchange
	mqm      apis.MQM

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

func newServiceManager(conf *config.Configuration, dao *dao.Dao, informer apis.ServiceInformer,
	exchange apis.Exchange, mqm apis.MQM, tmr timer.Timer) (*serviceManager, error) {
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
		services:              make(map[uint64]geminio.End),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
		// a simple unix timestamp incremental id factory
		idFactory: id.DefaultIncIDCounter,
		informer:  informer,
		exchange:  exchange,
		mqm:       mqm,
	}
	exchange.AddServicebound(sm)

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
			if !strings.Contains(err.Error(), apis.ErrStrUseOfClosedConnection) {
				klog.V(1).Infof("service manager listener accept err: %s", err)
			}
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
	opt.SetLog(log.NewKLog())
	end, err := server.NewEndWithConn(conn, opt)
	if err != nil {
		klog.Errorf("service manager geminio server new end err: %s", err)
		return err
	}
	meta := &apis.Meta{}
	err = json.Unmarshal(end.Meta(), meta)
	if err != nil {
		klog.Errorf("handle conn, json unmarshal err: %s", err)
		return err
	}
	// register topics claim of end
	sm.remoteReceiveClaim(end.ClientID(), meta.Topics)
	// add the end to MQM
	if sm.mqm != nil {
		sm.mqm.AddMQByEnd(meta.Topics, end)
	}

	// handle online event for end
	if err = sm.online(end, meta); err != nil {
		return err
	}

	// forward and stream up to edge
	sm.forward(meta, end)
	return nil
}

// topics
func (sm *serviceManager) remoteReceiveClaim(serviceID uint64, topics []string) error {
	klog.V(2).Infof("service remote receive claim, topics: %v, serviceID: %d", topics, serviceID)
	var err error
	// memdb
	for _, topic := range topics {
		st := &model.ServiceTopic{
			Topic:     topic,
			ServiceID: serviceID,
		}
		err = sm.dao.CreateServiceTopic(st)
		if err != nil {
			klog.Errorf("service remote receive claim, create service topic: %s, err: %s", topic, err)
			return err
		}
	}
	return nil
}

// rpc, RemoteRegistration is called from underlayer
func (sm *serviceManager) RemoteRegistration(rpc string, serviceID, streamID uint64) {
	// TODO return error
	klog.V(2).Infof("service remote rpc registration, rpc: %s, serviceID: %d, streamID: %d", rpc, serviceID, streamID)

	// memdb
	sr := &model.ServiceRPC{
		RPC:        rpc,
		ServiceID:  serviceID,
		CreateTime: time.Now().Unix(),
	}
	err := sm.dao.CreateServiceRPC(sr)
	if err != nil {
		klog.Errorf("service remote registration, create service rpc: %s, err: %s, serviceID: %d, streamID: %d", err, rpc, serviceID, streamID)
	}
}

func (sm *serviceManager) GetServiceByID(serviceID uint64) geminio.End {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	return sm.services[serviceID]
}

func (sm *serviceManager) GetServiceByName(name string) (geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mservice, err := sm.dao.GetServiceByName(name)
	if err != nil {
		klog.V(2).Infof("get service by name: %s, err: %s", name, err)
		return nil, err
	}

	return sm.services[mservice.ServiceID], nil
}

func (sm *serviceManager) GetServiceByRPC(rpc string) (geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mrpc, err := sm.dao.GetServiceRPC(rpc)
	if err != nil {
		klog.V(2).Infof("get service by rpc: %s, err: %s", rpc, err)
		return nil, err
	}

	return sm.services[mrpc.ServiceID], nil
}

func (sm *serviceManager) GetServiceByTopic(topic string) (geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mtopic, err := sm.dao.GetServiceTopic(topic)
	if err != nil {
		klog.V(2).Infof("get service by topic: %s, err: %s", topic, err)
		return nil, err
	}

	return sm.services[mtopic.ServiceID], nil
}

func (sm *serviceManager) ListService() []geminio.End {
	ends := []geminio.End{}
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	for _, value := range sm.services {
		ends = append(ends, value)
	}
	return ends
}

func (sm *serviceManager) CountServices() int {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()
	return len(sm.services)
}

func (sm *serviceManager) DelSerivces(service string) error {
	panic("TODO")
}

func (sm *serviceManager) ListStreams(serviceID uint64) []geminio.Stream {
	all := sm.streams.MGetAll(serviceID)
	return utils.Slice2streams(all)
}

// close all services
func (sm *serviceManager) Close() error {
	if err := sm.ln.Close(); err != nil {
		return err
	}
	return nil
}
