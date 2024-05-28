package edgebound

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/jumboframes/armorigo/synchub"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

func (em *edgeManager) online(end geminio.End) error {
	// TODO transaction
	// cache
	var sync synchub.Sync
	em.mtx.RLock()
	old, ok := em.edges[end.ClientID()]
	if ok {
		klog.Warningf("edge online, old end exists, edgeID: %d", end.ClientID())
		// if the old connection exits, offline it
		oldend := old.(geminio.End)
		// we wait the cache and db to clear old end's data
		syncKey := "edge" + "-" + strconv.FormatUint(oldend.ClientID(), 10) + "-" + oldend.RemoteAddr().String()
		sync = em.shub.Add(syncKey)
		if err := oldend.Close(); err != nil {
			klog.Warningf("edge online, kick off old end err: %s, edgeID: %d", err, end.ClientID())
		}
	}
	em.mtx.RUnlock()

	// we don't want the channel block the mtx
	if sync != nil {
		// unlikely here
		<-sync.C()
	}

	em.mtx.Lock()
	// double check
	old, ok = em.edges[end.ClientID()]
	if ok {
		klog.Warningf("edge online same time, old end exists, edgeID: %d", end.ClientID())
		em.mtx.Unlock()
		return errors.New("please connect later")
	}
	em.edges[end.ClientID()] = end
	if em.informer != nil {
		em.informer.SetEdgeCount(len(em.edges))
	}
	em.mtx.Unlock()

	// memdb
	edge := &model.Edge{
		EdgeID:     end.ClientID(),
		Meta:       string(end.Meta()),
		Addr:       end.RemoteAddr().String(),
		CreateTime: time.Now().Unix(),
	}
	if err := em.repo.CreateEdge(edge); err != nil {
		klog.Errorf("edge online, repo create err: %s, edgeID: %d", err, end.ClientID())
		return err
	}

	// inform others
	if em.informer != nil {
		em.informer.EdgeOnline(end.ClientID(), end.Meta(), end.RemoteAddr())
	}
	// exchange to service
	if em.exchange != nil {
		err := em.exchange.EdgeOnline(end.ClientID(), end.Meta(), end.RemoteAddr())
		if err == apis.ErrServiceNotOnline {
			return nil
		}
	}
	return nil
}

func (em *edgeManager) offline(edgeID uint64, meta []byte, addr net.Addr) error {
	// TODO transaction
	legacy := false
	// cache
	em.mtx.Lock()
	value, ok := em.edges[edgeID]
	if ok {
		end := value.(geminio.End)
		if end.RemoteAddr().String() == addr.String() {
			legacy = true
			delete(em.edges, edgeID)
			klog.V(2).Infof("edge offline, edgeID: %d, remote addr: %s", edgeID, end.RemoteAddr().String())
		} else {
			// same edgeID but different connection addr
			klog.V(1).Infof("edge offline, edgeID: %d, remote addr: %s, offline addr: %s", edgeID, end.RemoteAddr(), addr.String())
			em.mtx.Unlock()
			return nil
		}
	} else {
		klog.Warningf("edge offline, edgeID: %d not found in cache", edgeID)
	}
	if em.informer != nil {
		// TODO merge events
		em.informer.SetEdgeCount(len(em.edges))
	}
	em.mtx.Unlock()

	defer func() {
		if legacy {
			syncKey := "edge" + "-" + strconv.FormatUint(edgeID, 10) + "-" + addr.String()
			em.shub.Done(syncKey)
		}
	}()

	// memdb
	if err := em.repo.DeleteEdge(&query.EdgeDelete{
		EdgeID: edgeID,
		Addr:   addr.String(),
	}); err != nil {
		klog.Errorf("edge offline, repo delete edge err: %s, edgeID: %d", err, edgeID)
		return err
	}
	if err := em.repo.DeleteEdgeRPCs(edgeID); err != nil {
		klog.Errorf("edge offline, repo delete edge rpcs err: %s, edgeID: %d", err, edgeID)
		return err
	}

	// inform others
	if em.informer != nil {
		em.informer.EdgeOffline(edgeID, meta, addr)
	}
	// exchange to service
	if em.exchange != nil {
		em.exchange.EdgeOffline(edgeID, meta, addr)
	}
	return nil
}

// delegations for all ends from edgebound, called by geminio
func (em *edgeManager) ConnOnline(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := d.Meta()
	addr := d.RemoteAddr()

	klog.V(2).Infof("edge online, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	return nil
}

func (em *edgeManager) ConnOffline(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := d.Meta()
	addr := d.RemoteAddr()

	klog.V(2).Infof("edge offline, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	// offline the cache
	err := em.offline(edgeID, meta, addr)
	if err != nil {
		klog.Errorf("edge offline, cache or db offline err: %s, edgeID: %d, meta: %s, addr: %s",
			err, edgeID, string(meta), addr)
		return err
	}
	return nil
}

func (em *edgeManager) Heartbeat(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := string(d.Meta())
	addr := d.RemoteAddr()
	klog.V(3).Infof("edge heartbeat, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	if em.informer != nil {
		em.informer.EdgeHeartbeat(edgeID, d.Meta(), addr)
	}
	return nil
}

func (em *edgeManager) RemoteRegistration(rpc string, edgeID, streamID uint64) {
	klog.V(3).Infof("edge remote rpc registration, rpc: %s, edgeID: %d, streamID: %d", rpc, edgeID, streamID)

	// memdb
	er := &model.EdgeRPC{
		RPC:        rpc,
		EdgeID:     edgeID,
		CreateTime: time.Now().Unix(),
	}
	err := em.repo.CreateEdgeRPC(er)
	if err != nil {
		klog.Errorf("edge remote registration, create edge rpc err: %s, rpc: %s, edgeID: %d, streamID: %d", err, rpc, edgeID, streamID)
	}
}

func (em *edgeManager) GetClientID(_ uint64, meta []byte) (uint64, error) {
	var (
		edgeID uint64
		err    error
	)
	if em.exchange != nil {
		edgeID, err = em.exchange.GetEdgeID(meta)
		if err == nil {
			klog.V(2).Infof("edge get edgeID: %d from exchange, meta: %s", edgeID, string(meta))
			return edgeID, nil
		}
	}

	if (err == apis.ErrServiceNotOnline || err == apis.ErrRPCNotOnline) && em.conf.Edgebound.EdgeIDAllocWhenNoIDServiceOn {
		edgeID = em.idFactory.GetID()
		klog.V(2).Infof("edge get edgeID: %d, meta: %s, after no ID acquired from exchange", edgeID, string(meta))
		return em.idFactory.GetID(), nil
	}
	return 0, err
}
