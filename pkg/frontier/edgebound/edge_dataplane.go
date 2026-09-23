package edgebound

import (
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (session *edgeSession) acceptStream(stream geminio.Stream) {
	em := session.edgeManager
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(2).Infof("edge accept stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)

	em.mtx.Lock()
	if session.retired {
		em.mtx.Unlock()
		return
	}
	if session.streams == nil {
		session.streams = make(map[uint64]geminio.Stream)
	}
	session.streams[streamID] = stream
	em.mtx.Unlock()
	// exchange to service
	if em.exchange != nil {
		em.exchange.StreamToService(stream)
	}
}

func (session *edgeSession) closedStream(stream geminio.Stream) {
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(2).Infof("edge closed stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)
	session.edgeManager.mtx.Lock()
	delete(session.streams, streamID)
	session.edgeManager.mtx.Unlock()
	// when the stream ends, the exchange can be noticed by functional error, so we don't update exchange
}

// forward to exchange
func (em *edgeManager) forward(end geminio.End) {
	edgeID := end.ClientID()
	meta := end.Meta()
	klog.V(2).Infof("edge forward raw message and rpc, edgeID: %d, meta: %s", edgeID, meta)
	if em.exchange != nil {
		em.exchange.ForwardToService(end)
	}
}
