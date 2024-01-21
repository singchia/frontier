package edgebound

import (
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (em *edgeManager) acceptStream(stream geminio.Stream) {
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(5).Infof("edge accept stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)

	// cache
	em.streams.MSet(edgeID, streamID, meta)
	// exchange to service
	em.exchange.StreamToService(stream)
}

func (em *edgeManager) closedStream(stream geminio.Stream) {
	edgeID := stream.ClientID()
	streamID := stream.StreamID()
	meta := stream.Meta()
	klog.V(5).Infof("edge closed stream, edgeID: %d, streamID: %d, meta: %s", edgeID, streamID, meta)
	// cache
	em.streams.MDel(edgeID, streamID)
	// when the stream ends, the exchange can be noticed by functional error, so we don't update exchange
}
