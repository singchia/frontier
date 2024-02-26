package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"strconv"

	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"github.com/singchia/geminio/options"
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
	// meta
	meta := &api.Meta{}
	if sopt.topics != nil {
		// we deliver topics in meta
		meta.Topics = sopt.topics
	}
	if sopt.service != "" {
		meta.Service = sopt.service
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	sopts.SetMeta(data)
	// delegate
	if sopt.delegate != nil {
		sopts.SetDelegate(sopt.delegate)
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
	return service.End.Register(ctx, api.RPCGetEdgeID, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
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
	return service.End.Register(ctx, api.RPCEdgeOnline, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		on := &api.OnEdgeOnline{}
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
	return service.End.Register(ctx, api.RPCEdgeOffline, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		off := &api.OnEdgeOffline{}
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
		rpc(ctx, req, rsp)
		return
	}
	return service.End.Register(ctx, method, wrap)
}

// Messager
func (service *serviceEnd) NewMessage(data []byte) geminio.Message {
	return service.End.NewMessage(data)
}

func (service *serviceEnd) Publish(ctx context.Context, edgeID uint64, msg geminio.Message) error {
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)
	custom := msg.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	msg.SetCustom(custom)

	// publish real end
	err := service.End.Publish(ctx, msg)
	msg.SetClientID(edgeID)
	// TODO we need to set EdgeID to let user know
	return err
}

func (service *serviceEnd) PublishAsync(ctx context.Context, edgeID uint64, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error) {
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)
	custom := msg.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	msg.SetCustom(custom)

	// publish async
	pub, err := service.End.PublishAsync(ctx, msg, ch)
	// TODO we need to set EdgeID to let user know
	return pub, err
}

func (service *serviceEnd) Receive(ctx context.Context) (geminio.Message, error) {
	msg, err := service.End.Receive(ctx)
	if err != nil {
		return nil, err
	}
	custom := msg.Custom()
	if custom == nil || len(custom) < 8 {
		// shoudn't be here
		return msg, nil
	}
	edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
	custom = custom[:len(custom)-8]
	msg.SetClientID(edgeID)
	msg.SetCustom(custom)
	return msg, nil
}

// Multiplexer
func (service *serviceEnd) OpenStream(ctx context.Context, edgeID uint64) (geminio.Stream, error) {
	id := strconv.FormatUint(edgeID, 10)
	opt := options.OpenStream()
	opt.SetPeer(id)
	return service.End.OpenStream(opt)
}

func (service *serviceEnd) AcceptStream() (geminio.Stream, error) {
	return service.End.AcceptStream()
}

func (service *serviceEnd) ListStreams() []geminio.Stream {
	return service.End.ListStreams()
}

func (service *serviceEnd) Close() error {
	return service.End.Close()
}
