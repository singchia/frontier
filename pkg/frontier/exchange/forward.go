package exchange

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/misc"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/options"
	"k8s.io/klog/v2"
)

func (ex *exchange) ForwardToEdge(meta *apis.Meta, end geminio.End) {
	// raw, TODO now we didn't supply raw RW to user
	// ex.forwardRawToEdge(end)

	// message
	ex.forwardMessageToEdge(end)
	// rpc
	ex.forwardRPCToEdge(end)
}

func (ex *exchange) forwardRawToEdge(end geminio.End) {
	//drop the io, actually we won't be here
	go func() {
		klog.V(3).Infof("exchange forward raw to edge, discard for now, serviceID: %d", end.ClientID())
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
			klog.V(1).Infof("service forward rpc, serviceID: %d, call edgeID: %d, is not online", serviceID, edgeID)
			r2.SetError(apis.ErrEdgeNotOnline)
			return
		}
		// call edge
		ropt := options.NewRequest()
		ropt.SetCustom(r1.Custom())
		r3 := edge.NewRequest(r1.Data(), ropt)
		// call option
		copt := options.Call()
		copt.SetTimeout(30 * time.Second)
		r4, err := edge.Call(ctx, method, r3, copt)
		if err != nil {
			klog.V(2).Infof("service forward rpc, serviceID: %d, call edgeID: %d, err: %s", serviceID, edgeID, err)
			r2.SetError(err)
			return
		}
		// we record the edgeID back to r2, for service
		tail := make([]byte, 8)
		binary.BigEndian.PutUint64(tail, edgeID)
		custom = r4.Custom()
		if custom == nil {
			custom = tail
		} else {
			custom = append(custom, tail...)
		}
		r2.SetCustom(custom)
		// return
		r2.SetData(r4.Data())
		r2.SetError(r4.Error())
	})
}

func (ex *exchange) forwardMessageToEdge(end geminio.End) {
	serviceID := end.ClientID()
	go func() {
		for {
			msg, err := end.Receive(context.TODO())
			if err != nil {
				if err == io.EOF {
					klog.V(2).Infof("service forward message, serviceID: %d, receive EOF", serviceID)
					return
				}
				klog.Errorf("service forward message, serviceID: %d, receive err: %s", serviceID, err)
				continue
			}
			klog.V(2).Infof("service forward message, receive msg: %s from: %d", string(msg.Data()), end.ClientID())
			// get target edgeID
			custom := msg.Custom()
			edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
			msg.SetCustom(custom[:len(custom)-8])

			// get edge
			edge := ex.Edgebound.GetEdgeByID(edgeID)
			if edge == nil {
				klog.V(1).Infof("service forward message, serviceID: %d, the edge: %d is not online", serviceID, edgeID)
				msg.Error(apis.ErrEdgeNotOnline)
				return
			}
			// publish in sync, TODO publish in async
			mopt := options.NewMessage()
			mopt.SetCustom(msg.Custom())
			mopt.SetTopic(msg.Topic())
			mopt.SetCnss(msg.Cnss())
			newmsg := edge.NewMessage(msg.Data(), mopt)
			// publish option
			popt := options.Publish()
			popt.SetTimeout(30 * time.Second)
			err = edge.Publish(context.TODO(), newmsg, popt)
			if err != nil {
				klog.V(2).Infof("service forward message, serviceID: %d, publish edge: %d err: %s", serviceID, edgeID, err)
				msg.Error(err)
				return
			}
			msg.Done()
		}
	}()
}

func (ex *exchange) ForwardToService(end geminio.End) {
	// raw, TODO now we didn't supply raw RW to user
	// ex.forwardRawToService(end)

	// message
	ex.forwardMessageToService(end)
	// rpc
	ex.forwardRPCToService(end)
}

// raw io from edge, and forward to service
func (ex *exchange) forwardRawToService(end geminio.End) {
	//drop the io, actually we won't be here
	go func() {
		klog.V(3).Infof("exchange forward raw to service, discard for now, edgeID: %d", end.ClientID())
		io.Copy(io.Discard, end)
	}()
}

// rpc from edge, and forward to service
func (ex *exchange) forwardRPCToService(end geminio.End) {
	edgeID := end.ClientID()
	addr := end.RemoteAddr()
	// we hijack all rpcs and forward them to service
	end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
		// get service
		svcs, err := ex.Servicebound.GetServicesByRPC(method)
		if err != nil {
			klog.V(2).Infof("exchange forward rpc to service, get service by rpc err: %s, edgeID: %d", err, edgeID)
			r2.SetError(err)
			return
		}
		index := misc.Hash(ex.conf.Exchange.HashBy, len(svcs), edgeID, addr)
		svc := svcs[index]
		serviceID := svc.ClientID()
		// we record the edgeID to service
		tail := make([]byte, 8)
		binary.BigEndian.PutUint64(tail, edgeID)
		custom := r1.Custom()
		if custom == nil {
			custom = tail
		} else {
			custom = append(custom, tail...)
		}

		// call
		ropt := options.NewRequest()
		ropt.SetCustom(custom)
		r3 := svc.NewRequest(r1.Data(), ropt)
		// call option
		copt := options.Call()
		copt.SetTimeout(30 * time.Second)
		r4, err := svc.Call(ctx, method, r3, copt)
		if err != nil {
			if err != apis.ErrRPCNotOnline {
				klog.Errorf("edge forward rpc to service, call service: %d err: %s, edgeID: %d", serviceID, err, edgeID)
			}
			r2.SetError(err)
			return
		}
		klog.V(3).Infof("edge forward rpc to service, call service: %d rpc: %s to edgeID: %d success", serviceID, method, edgeID)

		r2.SetData(r4.Data())
		r2.SetCustom(r4.Custom())
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
					klog.V(3).Infof("edge forward message, edgeID: %d, receive EOF", edgeID)
					return
				}
				klog.Errorf("edge forward message, receive err: %s, edgeID: %d, ", err, edgeID)
				continue
			}
			topic := msg.Topic()
			// TODO separate async and sync produce
			err = ex.MQM.Produce(topic, msg.Data(),
				apis.WithOrigin(msg),
				apis.WithEdgeID(edgeID),
				apis.WithAddr(end.RemoteAddr()))
			if err != nil {
				if err != apis.ErrTopicNotOnline {
					klog.Errorf("edge forward message, produce err: %s, edgeID: %d", err, edgeID)
				}
				msg.Error(err)
				continue
			}
			msg.Done()
		}
	}()
}
