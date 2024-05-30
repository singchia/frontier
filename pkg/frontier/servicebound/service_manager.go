package servicebound

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/misc"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"github.com/singchia/geminio/pkg/id"
	"github.com/singchia/geminio/server"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

func NewServicebound(conf *config.Configuration, repo apis.Repo, informer apis.ServiceInformer,
	exchange apis.Exchange, mqm apis.MQM, tmr timer.Timer) (apis.Servicebound, error) {
	return newServiceManager(conf, repo, informer, exchange, mqm, tmr)
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
	// we don't store stream info to repo, because they may will be too much.
	streams *mapmap.MapMap

	// repo and repo for services
	repo apis.Repo
	// listener for geminio
	ln net.Listener

	// timer for all service ends
	tmr timer.Timer
}

func newServiceManager(conf *config.Configuration, repo apis.Repo, informer apis.ServiceInformer,
	exchange apis.Exchange, mqm apis.MQM, tmr timer.Timer) (*serviceManager, error) {
	listen := &conf.Servicebound.Listen

	sm := &serviceManager{
		conf:                  conf,
		tmr:                   tmr,
		streams:               mapmap.NewMapMap(),
		repo:                  repo,
		shub:                  synchub.NewSyncHub(synchub.OptionTimer(tmr)),
		services:              make(map[uint64]geminio.End),
		UnimplementedDelegate: &delegate.UnimplementedDelegate{},
		// a simple unix timestamp incremental id factory
		idFactory: id.DefaultIncIDCounter,
		informer:  informer,
		exchange:  exchange,
		mqm:       mqm,
	}
	if misc.IsNil(informer) {
		sm.informer = nil
	}
	exchange.AddServicebound(sm)
	ln, err := utils.Listen(listen)
	if err != nil {
		klog.Errorf("service manager listen err: %s", err)
		return nil, err
	}
	sm.ln = ln
	return sm, nil
}

func (sm *serviceManager) Serve() error {
	for {
		conn, err := sm.ln.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), apis.ErrStrUseOfClosedConnection) {
				klog.V(1).Infof("service manager listener accept err: %s", err)
				return err
			}
			break
		}
		go sm.handleConn(conn)
	}
	return nil
}

func (sm *serviceManager) handleConn(conn net.Conn) error {
	// options for geminio End
	opt := server.NewEndOptions()
	opt.SetTimer(sm.tmr)
	opt.SetDelegate(sm)
	// stream handler
	opt.SetAcceptStreamFunc(sm.acceptStream)
	opt.SetClosedStreamFunc(sm.closedStream)
	opt.SetBufferSize(8192, 8192)
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
		err = sm.repo.CreateServiceTopic(st)
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
	err := sm.repo.CreateServiceRPC(sr)
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

	mservice, err := sm.repo.GetServiceByName(name)
	if err != nil {
		klog.V(2).Infof("get service by name: %s, err: %s", name, err)
		return nil, err
	}

	return sm.services[mservice.ServiceID], nil
}

func (sm *serviceManager) GetServicesByName(name string) ([]geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mservices, err := sm.repo.GetServicesByName(name)
	if err != nil {
		klog.V(2).Infof("get services by name: %s, err: %s", name, err)
		return nil, err
	}
	ends := []geminio.End{}
	for _, mservice := range mservices {
		end, ok := sm.services[mservice.ServiceID]
		if ok {
			ends = append(ends, end)
			continue
		}
		// TOTO warning the consistency
	}
	return ends, nil
}

func (sm *serviceManager) GetServiceByRPC(rpc string) (geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mrpc, err := sm.repo.GetServiceRPC(rpc)
	if err != nil {
		klog.V(2).Infof("get service by rpc: %s, err: %s", rpc, err)
		return nil, err
	}
	return sm.services[mrpc.ServiceID], nil
}

func (sm *serviceManager) GetServicesByRPC(rpc string) ([]geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mrpcs, err := sm.repo.GetServiceRPCs(rpc)
	if err != nil {
		klog.V(2).Infof("get service by rpc: %s, err: %s", rpc, err)
		return nil, err
	}
	ends := []geminio.End{}
	for _, mrpc := range mrpcs {
		end, ok := sm.services[mrpc.ServiceID]
		if ok {
			ends = append(ends, end)
			continue
		}
		// TODO warning the consistency
	}
	return ends, nil
}

func (sm *serviceManager) GetServiceByTopic(topic string) (geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mtopic, err := sm.repo.GetServiceTopic(topic)
	if err != nil {
		klog.V(2).Infof("get service by topic: %s, err: %s", topic, err)
		return nil, err
	}
	return sm.services[mtopic.ServiceID], nil
}

func (sm *serviceManager) GetServicesByTopic(topic string) ([]geminio.End, error) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	mtopics, err := sm.repo.GetServiceTopics(topic)
	if err != nil {
		klog.V(2).Infof("get service by topic: %s, err: %s", topic, err)
		return nil, err
	}
	ends := []geminio.End{}
	for _, mtopic := range mtopics {
		end, ok := sm.services[mtopic.ServiceID]
		if ok {
			ends = append(ends, end)
			continue
		}
		// TODO warning the consistency
	}
	return ends, nil
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

func (sm *serviceManager) DelServiceByID(serviceID uint64) error {
	// TODO test it
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	service, ok := sm.services[serviceID]
	if !ok {
		return apis.ErrEdgeNotOnline
	}
	return service.Close()
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
