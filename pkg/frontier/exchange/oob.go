package exchange

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"time"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/geminio/options"

	"k8s.io/klog/v2"
)

func (ex *exchange) GetEdgeID(meta []byte) (uint64, error) {
	svc, err := ex.Servicebound.GetServiceByRPC(apis.RPCGetEdgeID)
	if err != nil {
		klog.V(2).Infof("exchange get edgeID, get service err: %s, meta: %s", err, string(meta))
		if err == apis.ErrRecordNotFound {
			return 0, apis.ErrServiceNotOnline
		}
		return 0, err
	}
	if svc == nil {
		return 0, apis.ErrRecordNotFound
	}
	// call service
	req := svc.NewRequest(meta)
	opt := options.Call()
	opt.SetTimeout(30 * time.Second)
	rsp, err := svc.Call(context.TODO(), apis.RPCGetEdgeID, req, opt)
	if err != nil {
		klog.V(2).Infof("exchange call service: %d, get edgeID err: %s, meta: %s", svc.ClientID(), err, meta)
		return 0, err
	}
	data := rsp.Data()
	if data == nil || len(data) != 8 {
		return 0, apis.ErrIllegalEdgeID
	}
	return binary.BigEndian.Uint64(data), nil
}

func (ex *exchange) EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) error {
	svc, err := ex.Servicebound.GetServiceByRPC(apis.RPCEdgeOnline)
	if err != nil {
		klog.V(2).Infof("exchange edge online, get service err: %s, edgeID: %d, meta: %s, addr: %s", err, edgeID, string(meta), addr)
		if err == apis.ErrRecordNotFound {
			return apis.ErrServiceNotOnline
		}
		return err
	}
	// call service the edge online event
	event := &apis.OnEdgeOnline{
		EdgeID: edgeID,
		Meta:   meta,
		Net:    addr.Network(),
		Str:    addr.String(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		klog.Errorf("exchange edge online, json marshal err: %s, edgeID: %d, meta: %s, addr: %s", err, edgeID, string(meta), addr)
		return err
	}
	// call service
	req := svc.NewRequest(data)
	opt := options.Call()
	opt.SetTimeout(30 * time.Second)
	_, err = svc.Call(context.TODO(), apis.RPCEdgeOnline, req, opt)
	if err != nil {
		klog.V(2).Infof("exchange call service: %d, edge online err: %s, meta: %s, addr: %s", svc.ClientID(), err, meta, addr)
		return err
	}
	return nil
}

func (ex *exchange) EdgeOffline(edgeID uint64, meta []byte, addr net.Addr) error {
	svc, err := ex.Servicebound.GetServiceByRPC(apis.RPCEdgeOffline)
	if err != nil {
		klog.V(2).Infof("exchange edge offline, get service err: %s, edgeID: %d, meta: %s, addr: %s", err, edgeID, string(meta), addr)
		if err == apis.ErrRecordNotFound {
			return apis.ErrServiceNotOnline
		}
		return err
	}
	// call service the edge offline event
	event := &apis.OnEdgeOffline{
		EdgeID: edgeID,
		Meta:   meta,
		Net:    addr.Network(),
		Str:    addr.String(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		klog.Errorf("exchange edge offline, json marshal err: %s, edgeID: %d, meta: %s, addr: %s", err, edgeID, string(meta), addr)
		return err
	}
	// call service
	req := svc.NewRequest(data)
	opt := options.Call()
	opt.SetTimeout(30 * time.Second)
	_, err = svc.Call(context.TODO(), apis.RPCEdgeOffline, req, opt)
	if err != nil {
		klog.V(2).Infof("exchange call service: %d, edge offline err: %s, meta: %s, addr: %s", svc.ClientID(), err, meta, addr)
		return err
	}
	return nil
}
