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
	go func() {
		klog.V(6).Infof("exchange forward raw, discard for now")
		//drop the io, actually we won't be here
		io.Copy(io.Discard, end)
	}()
}

func (ex *exchange) forwardRPCToEdge(end geminio.End) {
	// we hijack all rpc and forward them to edge
	end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
		serviceID := end.ClientID()
		// get target edgeID
		custom := r1.Custom()
		edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
		r1.SetCustom(custom[:len(custom)-8])

		// get edge
		edge := ex.Edgebound.GetEdgeByID(edgeID)
		if edge == nil {
			klog.V(4).Infof("forward rpc, service: %d, call edge: %d is not online", serviceID, edgeID)
			r2.SetError(api.ErrEdgeNotOnline)
			return
		}
		// call edge
		r3, err := edge.Call(ctx, method, r1)
		if err != nil {
			klog.V(5).Infof("forward rpc, service: %d, call edge: %d err: %s", serviceID, edgeID, err)
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
		r2.SetData(r3.Data())
		r2.SetError(r3.Error())
		r2.SetCustom(custom)
	})
}

func (ex *exchange) forwardMessageToEdge(end geminio.End) {
	serviceID := end.ClientID()
	go func() {
		for {
			msg, err := end.Receive(context.TODO())
			if err != nil {
				if err == io.EOF {
					klog.V(5).Infof("forward message, service: %d receive EOF", serviceID)
					return
				}
				klog.Errorf("forward message, service: %d receive err: %s", serviceID, err)
				continue
			}
			// get target edgeID
			custom := msg.Custom()
			edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
			msg.SetCustom(custom[:len(custom)-8])

			// get edge
			edge := ex.Edgebound.GetEdgeByID(edgeID)
			if edge == nil {
				klog.V(4).Infof("forward message, service: %d, the edge: %d is not online", serviceID, edgeID)
				msg.Error(api.ErrEdgeNotOnline)
				return
			}
			// publish in sync, TODO publish in async
			err = edge.Publish(context.TODO(), msg)
			if err != nil {
				klog.V(5).Infof("forward message, service: %d, publish edge: %d err: %s", serviceID, edgeID, err)
				msg.Error(err)
				return
			}
			msg.Done()
		}
	}()
}
