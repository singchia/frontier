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
	em.streams.MSet(edgeID, streamID, stream)
	// exchange to service
	if em.exchange != nil {
		// TODO em.exchange.StreamToService(stream)
	}
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

// forward to exchange
func (em *edgeManager) forward(end geminio.End) {
	edgeID := end.ClientID()
	meta := end.Meta()
	klog.V(5).Infof("edge forward stream, edgeID: %d, meta: %s", edgeID, meta)
	if em.exchange != nil {
		em.exchange.ForwardToService(end)
	}
}
