package exchange

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/singchia/geminio"
	"github.com/singchia/geminio/options"
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
		klog.Errorf("stream to edge, open stream err: %s, serviceID: %d, edgeID: %d", err, serviceID, streamID)
		serviceStream.Close()
		return
	}

	// do stream forward
	ex.streamForward(serviceStream, edgeStream)
}

func (ex *exchange) StreamToService(edgeStream geminio.Stream) {
	edgeID := edgeStream.ClientID()
	streamID := edgeStream.StreamID()

	// get service
	peer := edgeStream.Peer()
	svc, err := ex.Servicebound.GetServiceByName(peer)
	if err != nil {
		klog.V(1).Infof("stream to service, get service: %s err: %s, edgeID: %d, streamID: %d", peer, err, edgeID, streamID)
		// TODO return err to the stream
		edgeStream.ClientID()
		return
	}

	serviceStream, err := svc.OpenStream()
	if err != nil {
		klog.Errorf("stream to service, open stream err: %s, serviceID: %d, edgeID: %d", err, svc.ClientID(), edgeID)
		edgeStream.Close()
		return
	}

	// do stream forward
	ex.streamForward(edgeStream, serviceStream)
}

func (ex *exchange) streamForward(left, right geminio.Stream) {
	// raw
	ex.streamForwardRaw(left, right)
	// message
	ex.streamForwardMessage(left, right)
	// rpc
	ex.streamForwardRPC(left, right)
}

func (ex *exchange) streamForwardRaw(left, right geminio.Stream) {
	copy := func(from, to geminio.Stream) {
		fromID := from.ClientID()
		toID := to.ClientID()

		n, err := io.Copy(to, from)
		if err != nil {
			klog.Errorf("stream forward raw, copy err: %s, fromID: %d, toID: %d, written: %d", err, fromID, toID, n)
		} else {
			klog.V(4).Infof("stream forward raw done, fromID: %d, toID: %d, written: %d", fromID, toID, n)
		}

		from.Close()
		to.Close()
	}

	go copy(left, right)
	go copy(right, left)
}

func (ex *exchange) streamForwardMessage(left, right geminio.Stream) {
	recvPub := func(from, to geminio.Stream) {
		fromID := from.ClientID()
		toID := to.ClientID()

		for {
			msg, err := from.Receive(context.TODO())
			if err != nil {
				if err != io.EOF {
					klog.Errorf("stream forward message, receive err: %s, fromID: %d, toID: %d", err, fromID, toID)
				}
				return
			}

			// message and options
			mopt := options.NewMessage()
			mopt.SetCustom(msg.Custom())
			mopt.SetTopic(msg.Topic())
			mopt.SetCnss(msg.Cnss())
			newmsg := to.NewMessage(msg.Data(), mopt)
			// publish options
			popt := options.Publish()
			popt.SetTimeout(30 * time.Second)
			err = to.Publish(context.TODO(), newmsg, popt)
			if err != nil {
				klog.Errorf("stream forward message, publish err: %s, fromID: %d, toID: %d", err, fromID, toID)
				msg.Error(err)
				return
			}
			msg.Done()
		}
	}

	go recvPub(left, right)
	go recvPub(right, left)
}

func (ex *exchange) streamForwardRPC(left, right geminio.Stream) {
	forwardRPC := func(from, to geminio.Stream) {
		fromID := from.ClientID()
		toID := to.ClientID()

		from.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
			// TODO to carry the fromID to next Call
			ropt := options.NewRequest()
			ropt.SetCustom(r1.Custom())
			r3 := to.NewRequest(r1.Data(), ropt)
			// call option
			copt := options.Call()
			copt.SetTimeout(30 * time.Second)
			r4, err := to.Call(ctx, method, r3, copt)
			if err != nil {
				klog.Errorf("stream forward rpc, call err: %s, fromID: %d, toID: %d", err, fromID, toID)
				r2.SetError(err)
				return
			}
			klog.V(3).Infof("stream froward rpc, call fromID: %d rpc: %s to toID: %d success", fromID, method, toID)

			r2.SetData(r4.Data())
			r2.SetCustom(r4.Custom())
		})
	}

	forwardRPC(left, right)
	forwardRPC(right, left)
}
