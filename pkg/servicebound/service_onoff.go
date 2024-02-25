package servicebound

import (
	"net"
	"strconv"
	"time"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/repo/model"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

func (sm *serviceManager) online(end geminio.End, meta *api.Meta) error {
	// cache
	var sync synchub.Sync
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	old, ok := sm.services[end.ClientID()]
	if ok {
		// it the old connection exits, offline it
		// in service the possibility is narrow, because service generally doesn't store ID
		oldend := old.(geminio.End)
		// we wait the cache and db to clear old end's data
		syncKey := "service" + "-" + strconv.FormatUint(oldend.ClientID(), 10) + "-" + oldend.RemoteAddr().String()
		sync = sm.shub.Add(syncKey)
		if err := oldend.Close(); err != nil {
			klog.Warningf("service online, kick off old end err: %s, serviceID: %d", err, end.ClientID())
		}
	}
	sm.services[end.ClientID()] = end

	if sync != nil {
		// unlikely here
		<-sync.C()
	}

	// memdb
	service := &model.Service{
		ServiceID:  end.ClientID(),
		Service:    meta.Service,
		Addr:       end.RemoteAddr().String(),
		CreateTime: time.Now().Unix(),
	}
	if err := sm.dao.CreateService(service); err != nil {
		klog.Errorf("service online, dao create err: %s, serviceID: %d", err, end.ClientID())
		return err
	}
	return nil
}

func (sm *serviceManager) offline(serviceID uint64, addr net.Addr) error {
	// TODO transaction
	legacy := false
	// clear cache
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	value, ok := sm.services[serviceID]
	if ok {
		end := value.(geminio.End)
		if end != nil && end.RemoteAddr().String() == addr.String() {
			legacy = true
			delete(sm.services, serviceID)
		}
	} else {
		klog.Warningf("service offline, serviceID: %d not found in cache", serviceID)
	}

	defer func() {
		if legacy {
			syncKey := "service" + "-" + strconv.FormatUint(serviceID, 10) + "-" + addr.String()
			sm.shub.Done(syncKey)
		}
	}()

	// clear memdb
	if err := sm.dao.DeleteService(&dao.ServiceDelete{
		ServiceID: serviceID,
		Addr:      addr.String(),
	}); err != nil {
		klog.Errorf("service offline, dao delete service err: %s, serviceID: %d", err, serviceID)
		return err
	}

	if err := sm.dao.DeleteServiceRPCs(serviceID); err != nil {
		klog.Errorf("service offline, dao delete service rpcs err: %s, serviceID: %d", err, serviceID)
		return err
	}
	klog.V(5).Infof("service offline, remote rpc de-register succeed, serviceID: %d", serviceID)

	if err := sm.dao.DeleteServiceTopics(serviceID); err != nil {
		klog.Errorf("service offline, dao delete service topics err: %s, serviceID: %d", err, serviceID)
		return err
	}

	// clear mqm
	if sm.mqm != nil {
		sm.mqm.DelMQByEnd(value)
	}

	klog.V(5).Infof("service offline, remote topics declaim succeed, serviceID: %d", serviceID)
	return nil
}

// delegations for all ends from servicebound, called by geminio
func (sm *serviceManager) ConnOnline(d delegate.ConnDescriber) error {
	serviceID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(4).Infof("service online, serviceID: %d, service: %s, addr: %s", serviceID, meta, addr)
	// notification for others
	if sm.informer != nil {
		sm.informer.ServiceOnline(serviceID, meta, addr)
	}
	return nil
}

func (sm *serviceManager) ConnOffline(d delegate.ConnDescriber) error {
	serviceID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(5).Infof("service offline, serviceID: %d, service: %s, remote addr: %s", serviceID, meta, addr)
	// offline the cache
	err := sm.offline(serviceID, addr)
	if err != nil {
		klog.Errorf("service offline, cache or db offline err: %s, serviceID: %d, meta: %s, addr: %s",
			err, serviceID, meta, addr)
		return err
	}
	if sm.informer != nil {
		sm.informer.ServiceOffline(serviceID, meta, addr)
	}
	// notification for others
	return nil
}

func (sm *serviceManager) Heartbeat(d delegate.ConnDescriber) error {
	serviceID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(6).Infof("service heartbeat, serviceID: %d, meta: %s, addr: %s", serviceID, string(meta), addr)
	if sm.informer != nil {
		sm.informer.ServiceHeartbeat(serviceID, meta, addr)
	}
	return nil
}

// actually the meta is service
func (sm *serviceManager) GetClientID(meta []byte) (uint64, error) {
	// TODO
	return sm.idFactory.GetID(), nil
}
