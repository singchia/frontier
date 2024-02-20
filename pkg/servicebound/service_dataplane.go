package servicebound

import (
	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (sm *serviceManager) acceptStream(stream geminio.Stream) {
	serviceID := stream.ClientID()
	streamID := stream.StreamID()
	service := stream.Meta()
	klog.V(5).Infof("service accept stream, serviceID: %d, streamID: %d, service: %s", serviceID, streamID, service)

	// cache
	sm.streams.MSet(serviceID, streamID, stream)
	// exchange to edge
	if sm.exchange != nil {
		// TODO sm.exchange.StreamToEdge(stream)
	}
}

func (sm *serviceManager) closedStream(stream geminio.Stream) {
	serviceID := stream.ClientID()
	streamID := stream.StreamID()
	service := stream.Meta()
	klog.V(5).Infof("service closed stream, serviceID: %d, streamID: %d, service: %s", serviceID, streamID, service)
	// cache
	sm.streams.MDel(serviceID, streamID)
	// when the stream ends, the exchange can be noticed by functional error, so we don't update exchange
}

// forward to exchange
func (sm *serviceManager) forward(meta *api.Meta, end geminio.End) {
	serviceID := end.ClientID()
	service := meta.Service
	klog.V(5).Infof("service forward raw message and rpc, serviceID: %d, service: %s", serviceID, service)
	if sm.exchange != nil {
		sm.exchange.ForwardToEdge(meta, end)
	}
}
