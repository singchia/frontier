package edgebound

import (
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (em *edgeManager) acceptStream(stream geminio.Stream) {
	klog.V(5).Infof("edge accept stream, edgeID: %d, streamID: %d, meta: %s", stream.ClientID(), stream.StreamID(), stream.Meta())
	em.exchange.StreamToService(stream)
	// cache
}

func (em *edgeManager) closedStream(stream geminio.Stream) {
	klog.V(5).Infof("edge closed stream, edgeID: %d, streamID: %d, meta: %s", stream.ClientID(), stream.StreamID(), stream.Meta())
	// when the stream ends, the exchange can be noticed by functional error, so we don't update exchange
	// cache
}
