package edgebound

import (
	"net"
	"time"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

func (em *edgeManager) online(end geminio.End) error {
	edge := &model.Edge{
		EdgeID:     end.ClientID(),
		Meta:       string(end.Meta()),
		Addr:       end.RemoteAddr().String(),
		CreateTime: time.Now().Unix(),
	}
	// Keep the in-memory repository and active session in the same critical section.
	// A late offline callback must never delete a replacement session's data.
	em.mtx.Lock()
	if err := em.repo.CreateEdge(edge); err != nil {
		em.mtx.Unlock()
		klog.Errorf("edge online, repo create err: %s, edgeID: %d", err, end.ClientID())
		return err
	}
	old := em.edges[end.ClientID()]
	em.edges[end.ClientID()] = end
	count := len(em.edges)
	em.mtx.Unlock()

	if old != nil && old != end {
		klog.Warningf("edge online, replacing old end, edgeID: %d", end.ClientID())
		go func() {
			// Close may wait for old streams; forwarding the new session must not wait.
			if err := old.Close(); err != nil {
				klog.Warningf("edge online, kick off old end err: %s, edgeID: %d", err, end.ClientID())
			}
		}()
	}

	if em.informer != nil {
		em.informer.SetEdgeCount(count)
	}

	return nil
}

func (em *edgeManager) offline(edgeID uint64, meta []byte, addr net.Addr) error {
	em.mtx.Lock()
	end, ok := em.edges[edgeID]
	if !ok || end.RemoteAddr().String() != addr.String() {
		em.mtx.Unlock()
		return nil
	}

	if err := em.repo.DeleteEdge(&query.EdgeDelete{
		EdgeID: edgeID,
		Addr:   addr.String(),
	}); err != nil {
		em.mtx.Unlock()
		klog.Errorf("edge offline, repo delete edge err: %s, edgeID: %d", err, edgeID)
		return err
	}
	if err := em.repo.DeleteEdgeRPCs(edgeID); err != nil {
		em.mtx.Unlock()
		klog.Errorf("edge offline, repo delete edge rpcs err: %s, edgeID: %d", err, edgeID)
		return err
	}
	delete(em.edges, edgeID)
	count := len(em.edges)
	em.mtx.Unlock()
	klog.V(2).Infof("edge offline, edgeID: %d, remote addr: %s", edgeID, addr)

	if em.informer != nil {
		em.informer.SetEdgeCount(count)
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

	// exchange to service
	if em.exchange != nil {
		err := em.exchange.EdgeOnline(edgeID, meta, addr)
		if err != nil && err != apis.ErrServiceNotOnline {
			return err
		}
	}
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
