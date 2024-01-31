package service

import (
	"context"
	"encoding/binary"

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

// RPCer
func (service *serviceEnd) Call(ctx context.Context, edgeID uint64, method string, req geminio.Request) (geminio.Response, error) {
	// we append the likely short one to slice
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)
	custom := req.Custom()
	if custom == nil {
		custom = []byte{}
	}
	custom = append(custom, tail...)
	req.SetCustom(custom)

	// call real end
	rsp, err := service.End.Call(ctx, method, req)
	if err != nil {
		return nil, err
	}
	rsp.SetClientID(edgeID)
	return rsp, nil
}
