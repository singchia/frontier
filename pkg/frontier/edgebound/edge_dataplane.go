package edgebound

import (
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (em *edgeManager) acceptStream(stream geminio.Stream) {
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(2).Infof("edge accept stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)

	// A stream can arrive before handleConn installs its end. Do not cache
	// streams from a different connection under the current session.
	em.mtx.RLock()
	end := em.edges[edgeID]
	if end != nil && end.RemoteAddr().String() == stream.RemoteAddr().String() {
		em.streams.MSet(edgeID, streamID, stream)
	}
	em.mtx.RUnlock()
	// exchange to service
	if em.exchange != nil {
		em.exchange.StreamToService(stream)
	}
}

func (em *edgeManager) closedStream(stream geminio.Stream) {
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(2).Infof("edge closed stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)
	// A late close must not remove a replacement stream with the same ID.
	em.mtx.RLock()
	if em.streams.MGet(edgeID, streamID) == stream {
		em.streams.MDel(edgeID, streamID)
	}
	em.mtx.RUnlock()
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
