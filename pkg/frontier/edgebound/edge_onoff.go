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

// A separate delegate binds callbacks to a connection even if its address is reused.
// All session fields are protected by edgeManager.mtx.
type edgeSession struct {
	*edgeManager
	end     geminio.End
	retired bool
	streams map[uint64]geminio.Stream
	rpcs    map[string]*model.EdgeRPC
}

func (em *edgeManager) online(session *edgeSession, end geminio.End) error {
	edge := &model.Edge{
		EdgeID:     end.ClientID(),
		Meta:       string(end.Meta()),
		Addr:       end.RemoteAddr().String(),
		CreateTime: time.Now().Unix(),
	}
	// Keep the in-memory repository and active session in the same critical section.
	// A late offline callback must never delete a replacement session's data.
	em.mtx.Lock()
	if session.retired {
		em.mtx.Unlock()
		return net.ErrClosed
	}
	if err := em.repo.CreateEdge(edge); err != nil {
		em.mtx.Unlock()
		klog.Errorf("edge online, repo create err: %s, edgeID: %d", err, end.ClientID())
		return err
	}
	old := em.edges[end.ClientID()]
	if old != nil && old != session {
		old.retired = true
		old.streams = nil
		old.rpcs = nil
	}
	session.end = end
	em.edges[end.ClientID()] = session
	// Registrations can arrive before online installs the connection. Replace the
	// old session's RPC inventory with only this session's pending registrations.
	if err := em.repo.DeleteEdgeRPCs(end.ClientID()); err != nil {
		klog.Errorf("edge online, repo delete edge rpcs err: %s, edgeID: %d", err, end.ClientID())
	}
	for _, rpc := range session.rpcs {
		if err := em.repo.CreateEdgeRPC(rpc); err != nil {
			klog.Errorf("edge online, repo create rpc err: %s, edgeID: %d, rpc: %s", err, end.ClientID(), rpc.RPC)
		}
	}
	// Publish the count in cache-update order; a delayed snapshot can go stale.
	if em.informer != nil {
		em.informer.SetEdgeCount(len(em.edges))
	}
	em.mtx.Unlock()

	if old != nil && old != session {
		klog.Warningf("edge online, replacing old end, edgeID: %d", end.ClientID())
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					klog.Errorf("edge online, kick off old end panicked: %v, edgeID: %d", recovered, end.ClientID())
				}
			}()
			// Close may wait for old streams; forwarding the new session must not wait.
			if err := old.end.Close(); err != nil {
				klog.Warningf("edge online, kick off old end err: %s, edgeID: %d", err, end.ClientID())
			}
		}()
	}

	return nil
}

func (em *edgeManager) offline(session *edgeSession, edgeID uint64, meta []byte, addr net.Addr) error {
	em.mtx.Lock()
	session.retired = true
	session.streams = nil
	session.rpcs = nil
	if em.edges[edgeID] != session {
		em.mtx.Unlock()
		return nil
	}
	// A failed repository cleanup must not leave a closed connection routable.
	delete(em.edges, edgeID)
	if em.informer != nil {
		em.informer.SetEdgeCount(len(em.edges))
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
	em.mtx.Unlock()
	klog.V(2).Infof("edge offline, edgeID: %d, remote addr: %s", edgeID, addr)

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

func (session *edgeSession) ConnOffline(d delegate.ConnDescriber) error {
	edgeID := d.ClientID()
	meta := d.Meta()
	addr := d.RemoteAddr()

	klog.V(2).Infof("edge offline, edgeID: %d, meta: %s, addr: %s", edgeID, string(meta), addr)
	// offline the cache
	err := session.edgeManager.offline(session, edgeID, meta, addr)
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

func (session *edgeSession) RemoteRegistration(rpc string, edgeID, streamID uint64) {
	em := session.edgeManager
	klog.V(3).Infof("edge remote rpc registration, rpc: %s, edgeID: %d, streamID: %d", rpc, edgeID, streamID)

	em.mtx.Lock()
	defer em.mtx.Unlock()
	if session.retired {
		return
	}
	if session.rpcs == nil {
		session.rpcs = make(map[string]*model.EdgeRPC)
	}
	if _, ok := session.rpcs[rpc]; ok {
		return
	}
	er := &model.EdgeRPC{
		RPC:        rpc,
		EdgeID:     edgeID,
		CreateTime: time.Now().Unix(),
	}
	session.rpcs[rpc] = er
	if session.end == nil {
		return
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
