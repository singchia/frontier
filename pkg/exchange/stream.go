package exchange

import (
	"context"
	"encoding/binary"
	"strconv"

	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (ex *exchange) StreamToEdge(serviceStream geminio.Stream) {
	serviceID := serviceStream.ClientID()
	streamID := serviceStream.StreamID()
	// get edgeID
	peer := serviceStream.Peer()
	edgeID, err := strconv.ParseUint(peer, 10, 64)
	if err != nil {
		klog.Errorf("stream to edge err: %s, serviceID: %d, streamID: %d", err, serviceID, streamID)
		// TODO return err to the stream
		serviceStream.Close()
		return
	}

	// get edge
	edge := ex.Edgebound.GetEdgeByID(edgeID)
	if edge == nil {
		klog.V(1).Infof("stream to edge, serviceID: %d, edgeID: %d, is not online", serviceID, streamID)
		serviceStream.Close()
		return
	}

	// open stream from edge
	edgeStream, err := edge.OpenStream()
	if err != nil {
		klog.Errorf("stream to edge, open stream err: %s, serviceID: %d, edge:ID: %d", err, serviceID, streamID)
		serviceStream.Close()
		return
	}

	// do stream forward
}

func (ex *exchange) streamForward(left, right geminio.Stream) {}

func (ex *exchange) streamForwardMessage(left, right geminio.Stream) {
	RecvPub := func(from, to geminio.Stream) {
		fromID := from.ClientID()
		toID := to.ClientID()
		for {
			msg, err := from.Receive(context.TODO())
			if err != nil {
				klog.Errorf("stream forward message, receive err: %s, fromID: %d, toID: %d", err, fromID, toID)
				return
			}

			// we record the ID to peer
			tail := make([]byte, 8)
			binary.BigEndian.PutUint64(tail, msg.ClientID())
		}
	}
}
