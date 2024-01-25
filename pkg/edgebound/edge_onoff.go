package edgebound

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/repo/model"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

func (em *edgeManager) online(end geminio.End) error {
	// TODO transaction
	// cache
	var sync synchub.Sync
	em.mtx.Lock()
	old, ok := em.edges[end.ClientID()]
	if ok {
		// if the old connection exits, offline it
		oldend := old.(geminio.End)
		// we wait the cache and db to clear old end's data
		syncKey := "edge" + "-" + strconv.FormatUint(oldend.ClientID(), 10) + "-" + oldend.RemoteAddr().String()
		sync = em.shub.Add(syncKey)
		if err := oldend.Close(); err != nil {
			klog.Warningf("edge online, kick off old end err: %s, edgeID: %d", err, end.ClientID())
		}
	}
	em.edges[end.ClientID()] = end
	em.mtx.Unlock()

	if sync != nil {
		// unlikely here
		<-sync.C()
	}

	// memdb
	edge := &model.Edge{
		EdgeID:     end.ClientID(),
		Meta:       string(end.Meta()),
		Addr:       end.RemoteAddr().String(),
		CreateTime: time.Now().Unix(),
	}
	if err := em.dao.CreateEdge(edge); err != nil {
		klog.Errorf("edge online, dao create err: %s, edgeID: %d", err, end.ClientID())
		return err
	}
	return nil
}

func (em *edgeManager) offline(edgeID uint64, addr net.Addr) error {
	// TODO transaction
	legacy := false
	// cache
	em.mtx.Lock()
	value, ok := em.edges[edgeID]
	if ok {
		end := value.(geminio.End)
		if end != nil && end.RemoteAddr().String() == addr.String() {
			legacy = true
			delete(em.edges, edgeID)
			klog.V(5).Infof("edge offline, edgeID: %d, remote addr: %s", edgeID, end.RemoteAddr().String())
		}
	} else {
		klog.Warningf("edge offline, edgeID: %d not found in cache", edgeID)
	}
	em.mtx.Unlock()

	defer func() {
		if legacy {
			syncKey := "edge" + "-" + strconv.FormatUint(edgeID, 10) + "-" + addr.String()
			em.shub.Done(syncKey)
		}
	}()

	// memdb
	if err := em.dao.DeleteEdge(&dao.EdgeDelete{
		EdgeID: edgeID,
		Addr:   addr.String(),
	}); err != nil {
		klog.Errorf("edge offline, dao delete edge err: %s, edgeID: %d", err, edgeID)
		return err
	}
	if err := em.dao.DeleteEdgeRPCs(edgeID); err != nil {
		klog.Errorf("edge offline, dao delete edge rpcs err: %s, edgeID: %d", err, edgeID)
		return err
	}
	return nil
}

// delegations for all ends from edgebound, called by geminio
func (em *edgeManager) ConnOnline(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(4).Infof("edge online, edgeID: %d, meta: %s, addr: %s", edgeID, meta, addr)
	// notification for others
	if em.informer != nil {
		em.informer.EdgeOnline(edgeID, d.Meta(), addr)
	}
	return nil
}

func (em *edgeManager) ConnOffline(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(4).Infof("edge offline, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	// offline the cache
	err := em.offline(edgeID, addr)
	if err != nil {
		klog.Errorf("edge offline, cache or db offline err: %s, edgeID: %d, meta: %s, addr: %s",
			err, edgeID, string(meta), addr)
		return err
	}
	if em.informer != nil {
		em.informer.EdgeOffline(edgeID, d.Meta(), addr)
	}
	// notification for others
	return nil
}

func (em *edgeManager) Heartbeat(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(5).Infof("edge heartbeat, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	if em.informer != nil {
		em.informer.EdgeHeartbeat(edgeID, d.Meta(), addr)
	}
	return nil
}

func (em *edgeManager) RemoteRegistration(rpc string, edgeID, streamID uint64) {
	klog.V(5).Infof("edge remote rpc registration, rpc: %s, edgeID: %d, streamID: %d", rpc, edgeID, streamID)

	// memdb
	er := &model.EdgeRPC{
		RPC:        rpc,
		EdgeID:     edgeID,
		CreateTime: time.Now().Unix(),
	}
	err := em.dao.CreateEdgeRPC(er)
	if err != nil {
		klog.Errorf("edge remote registration, create edge rpc err: %s, rpc: %s, edgeID: %d, streamID: %d", err, rpc, edgeID, streamID)
	}
}

func (em *edgeManager) GetClientID(meta []byte) (uint64, error) {
	// TODO
	if em.conf.Edgebound.EdgeIDAllocWhenNoIDServiceOn {
		return em.idFactory.GetID(), nil
	}
	return 0, errors.New("unable to get an edgeID")
}
