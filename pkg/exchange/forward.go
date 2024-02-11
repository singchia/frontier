package exchange

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

func (ex *exchange) ForwardToEdge(meta *api.Meta, end geminio.End) {
	// raw
	ex.forwardRawToEdge(end)
	// message
	ex.forwardMessageToEdge(end)
	// rpc
	ex.forwardRPCToEdge(end)
}

func (ex *exchange) forwardRawToEdge(end geminio.End) {
	//drop the io, actually we won't be here
	go func() {
		klog.V(6).Infof("exchange forward raw to edge, discard for now, serviceID: %d", end.ClientID())
		io.Copy(io.Discard, end)
	}()
}

func (ex *exchange) forwardRPCToEdge(end geminio.End) {
	// we hijack all rpcs and forward them to edge
	end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
		serviceID := end.ClientID()
		// get target edgeID
		custom := r1.Custom()
		edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
		r1.SetCustom(custom[:len(custom)-8])

		// get edge
		edge := ex.Edgebound.GetEdgeByID(edgeID)
		if edge == nil {
			klog.V(4).Infof("forward rpc, serviceID: %d, call edgeID: %d, is not online", serviceID, edgeID)
			r2.SetError(api.ErrEdgeNotOnline)
			return
		}
		// call edge
		r3, err := edge.Call(ctx, method, r1)
		if err != nil {
			klog.V(5).Infof("forward rpc, serviceID: %d, call edgeID: %d, err: %s", serviceID, edgeID, err)
			r2.SetError(err)
			return
		}
		// we record the edgeID back to r2, for service
		tail := make([]byte, 8)
		binary.BigEndian.PutUint64(tail, edgeID)
		custom = r3.Custom()
		if custom == nil {
			custom = tail
		} else {
			custom = append(custom, tail...)
		}
		r2.SetCustom(custom)
		// return
		r2.SetData(r3.Data())
		r2.SetError(r3.Error())
	})
}

func (ex *exchange) forwardMessageToEdge(end geminio.End) {
	serviceID := end.ClientID()
	go func() {
		for {
			msg, err := end.Receive(context.TODO())
			if err != nil {
				if err == io.EOF {
					klog.V(5).Infof("forward message, serviceID: %d, receive EOF", serviceID)
					return
				}
				klog.Errorf("forward message, serviceID: %d, receive err: %s", serviceID, err)
				continue
			}
			// get target edgeID
			custom := msg.Custom()
			edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
			msg.SetCustom(custom[:len(custom)-8])

			// get edge
			edge := ex.Edgebound.GetEdgeByID(edgeID)
			if edge == nil {
				klog.V(4).Infof("forward message, serviceID: %d, the edge: %d is not online", serviceID, edgeID)
				msg.Error(api.ErrEdgeNotOnline)
				return
			}
			// publish in sync, TODO publish in async
			err = edge.Publish(context.TODO(), msg)
			if err != nil {
				klog.V(5).Infof("forward message, serviceID: %d, publish edge: %d err: %s", serviceID, edgeID, err)
				msg.Error(err)
				return
			}
			msg.Done()
		}
	}()
}

// raw io from edge, and forward to service
func (ex *exchange) forwardRawToService(end geminio.End) {
	//drop the io, actually we won't be here
	go func() {
		klog.V(6).Infof("exchange forward raw to service, discard for now, edgeID: %d", end.ClientID())
		io.Copy(io.Discard, end)
	}()
}

// rpc from edge, and forward to service
func (ex *exchange) forwardRPCToService(end geminio.End) {
	edgeID := end.ClientID()
	// we hijack all rpcs and forward them to service
	end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
		// get service
		edge, err := ex.Servicebound.GetServiceByRPC(method)
		if err != nil {
			klog.Errorf("exchange forward rpc to service, get service by rpc err: %s, edgeID: %d", err, edgeID)
			r2.SetError(err)
			return
		}
		// we record the edgeID to service
		tail := make([]byte, 8)
		binary.BigEndian.PutUint64(tail, edgeID)
		custom := r1.Custom()
		if custom == nil {
			custom = tail
		} else {
			custom = append(custom, tail...)
		}
		r1.SetCustom(custom)
		// call
		r3, err := edge.Call(ctx, method, r1)
		if err != nil {
			klog.Errorf("exchange forward rpc to service, call service err: %s, edgeID: %d", err, edgeID)
			r2.SetError(err)
			return
		}
		klog.V(6).Infof("exchange forward rpc to service, call service rpc: %s success, edgeID: %d", method, edgeID)
		r2.SetData(r3.Data())
	})
}

// message from edge, and forward to topic owner
func (ex *exchange) forwardMessageToService(end geminio.End) {
	edgeID := end.ClientID()
	go func() {
		for {
			msg, err := end.Receive(context.TODO())
			if err != nil {
				if err == io.EOF {
					klog.V(5).Infof("forward message, edgeID: %d, receive EOF", edgeID)
					return
				}
				klog.Errorf("forward message, receive err: %s, edgeID: %d, ", err, edgeID)
				continue
			}
			topic := msg.Topic()
			// TODO seperate async and sync produce
			err = ex.MQ.Produce(topic, msg.Data(), api.WithOrigin(msg), api.WithEdgeID(edgeID))
			if err != nil {
				klog.Errorf("forward message, produce err: %s, edgeID: %d", err, edgeID)
				msg.Error(err)
				continue
			}
			msg.Done()
		}
	}()
}
