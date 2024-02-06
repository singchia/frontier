package service

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
)

type serviceEnd struct {
	geminio.End
}

func newServiceEnd(dialer client.Dialer, opts ...ServiceOption) (*serviceEnd, error) {
	// options
	sopt := &serviceOption{}
	for _, opt := range opts {
		opt(sopt)
	}
	sopts := &client.RetryEndOptions{
		EndOptions: &client.EndOptions{},
	}
	if sopt.tmr != nil {
		sopts.SetTimer(sopt.tmr)
	}
	if sopt.logger != nil {
		sopts.SetLog(sopt.logger)
	}
	// new geminio end
	end, err := client.NewRetryEndWithDialer(dialer, sopts)
	if err != nil {
		return nil, err
	}
	return &serviceEnd{end}, nil
}

// Control Register
func (service *serviceEnd) RegisterGetEdgeID(ctx context.Context, getEdgeID GetEdgeID) error {
	return service.End.Register(ctx, "get_edge_id", func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		id, err := getEdgeID(req.Data())
		if err != nil {
			// we just deliver the err back
			rsp.SetError(err)
			return
		}
		hex := make([]byte, 8)
		binary.BigEndian.PutUint64(hex, id)
		rsp.SetData(hex)
	})
}

func (service *serviceEnd) RegisterEdgeOnline(ctx context.Context, edgeOnline EdgeOnline) error {
	return service.End.Register(ctx, "edge_online", func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		on := &OnEdgeOnline{}
		err := json.Unmarshal(req.Data(), on)
		if err != nil {
			// shouldn't be here
			rsp.SetError(err)
			return
		}
		err = edgeOnline(on.EdgeID, on.Meta, on)
		if err != nil {
			rsp.SetError(err)
			return
		}
		// if allowed, the edge will continue the connection
	})
}

func (service *serviceEnd) RegisterEdgeOffline(ctx context.Context, edgeOffline EdgeOffline) error {
	return service.End.Register(ctx, "edge_offline", func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		off := &OnEdgeOffline{}
		err := json.Unmarshal(req.Data(), off)
		if err != nil {
			// shouldn't be here
			rsp.SetError(err)
			return
		}
		err = edgeOffline(off.EdgeID, off.Meta, off)
		if err != nil {
			rsp.SetError(err)
			return
		}
	})
}

// RPCer
func (service *serviceEnd) NewRequest(data []byte) geminio.Request {
	return service.End.NewRequest(data)
}

func (service *serviceEnd) Call(ctx context.Context, edgeID uint64, method string, req geminio.Request) (geminio.Response, error) {
	// we append the likely short one to slice
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)
	custom := req.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	req.SetCustom(custom)

	// call real end
	rsp, err := service.End.Call(ctx, method, req)
	if err != nil {
		return nil, err
	}
	rsp.SetClientID(edgeID)
	return rsp, nil
}

// It's just like the go rpc way
func (service *serviceEnd) CallAsync(ctx context.Context, edgeID uint64, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error) {
	// we append the likely short one to slice
	// the last 8 bytes is for frontier
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)
	custom := req.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	req.SetCustom(custom)

	// call real end
	call, err := service.End.CallAsync(ctx, method, req, ch)
	if err != nil {
		return nil, err
	}
	// TODO we need to set EdgeID back
	return call, nil
}

func (service *serviceEnd) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	wrap := func(_ context.Context, req geminio.Request, rsp geminio.Response) {
		custom := req.Custom()
		if len(custom) < 8 {
			rpc(ctx, req, rsp)
			return
		}
		edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
		req.SetClientID(edgeID)
		rsp.SetClientID(edgeID)
		return
	}
	return service.End.Register(ctx, method, wrap)
}
