package edge

import (
	"context"

	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"github.com/singchia/geminio/options"
)

type edgeEnd struct {
	geminio.End
}

func newEdgeEnd(dialer client.Dialer, opts ...EdgeOption) (*edgeEnd, error) {
	// options
	eopt := &edgeOption{
		readBufferSize:  -1,
		writeBufferSize: -1,
	}
	for _, opt := range opts {
		opt(eopt)
	}

	// turn to end options
	eopts := client.NewEndOptions()
	if eopt.tmr != nil {
		eopts.SetTimer(eopt.tmr)
	}
	if eopt.logger != nil {
		eopts.SetLog(eopt.logger)
	}
	if eopt.edgeID != nil {
		eopts.SetClientID(*eopt.edgeID)
	}
	if eopt.meta != nil {
		eopts.SetMeta(eopt.meta)
	}
	if eopt.readBufferSize != -1 || eopt.writeBufferSize != -1 {
		eopts.SetBufferSize(eopt.readBufferSize, eopt.writeBufferSize)
	}

	// new geminio end
	end, err := client.NewEndWithDialer(dialer, eopts)
	if err != nil {
		return nil, err
	}
	return &edgeEnd{end}, nil
}

func newRetryEdgeEnd(dialer client.Dialer, opts ...EdgeOption) (*edgeEnd, error) {
	// options
	eopt := &edgeOption{
		readBufferSize:  -1,
		writeBufferSize: -1,
	}
	for _, opt := range opts {
		opt(eopt)
	}

	// turn to end options
	eopts := client.NewEndOptions()
	if eopt.tmr != nil {
		eopts.SetTimer(eopt.tmr)
	}
	if eopt.logger != nil {
		eopts.SetLog(eopt.logger)
	}
	if eopt.edgeID != nil {
		eopts.SetClientID(*eopt.edgeID)
	}
	if eopt.meta != nil {
		eopts.SetMeta(eopt.meta)
	}
	if eopt.readBufferSize != -1 || eopt.writeBufferSize != -1 {
		eopts.SetBufferSize(eopt.readBufferSize, eopt.writeBufferSize)
	}

	// new geminio end
	end, err := client.NewRetryEndWithDialer(dialer, eopts)
	if err != nil {
		return nil, err
	}
	return &edgeEnd{end}, nil
}

// RPCer
func (end *edgeEnd) NewRequest(data []byte) geminio.Request {
	return end.End.NewRequest(data)
}

func (end *edgeEnd) Call(ctx context.Context, method string, req geminio.Request) (geminio.Response, error) {
	return end.End.Call(ctx, method, req)
}

func (end *edgeEnd) CallAsync(ctx context.Context, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error) {
	return end.End.CallAsync(ctx, method, req, ch)
}

func (end *edgeEnd) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	return end.End.Register(ctx, method, rpc)
}

// Messager
func (end *edgeEnd) NewMessage(data []byte) geminio.Message {
	return end.End.NewMessage(data)
}

func (end *edgeEnd) Publish(ctx context.Context, topic string, msg geminio.Message) error {
	msg.SetTopic(topic)
	return end.End.Publish(ctx, msg)
}

func (end *edgeEnd) PublishAsync(ctx context.Context, topic string, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error) {
	msg.SetTopic(topic)
	return end.End.PublishAsync(ctx, msg, ch)
}

func (end *edgeEnd) Receive(ctx context.Context) (geminio.Message, error) {
	return end.End.Receive(ctx)
}

// Multiplexer
func (end *edgeEnd) OpenStream(serviceName string) (geminio.Stream, error) {
	opt := options.OpenStream()
	opt.SetPeer(serviceName)
	return end.End.OpenStream(opt)
}

func (end *edgeEnd) AcceptStream() (geminio.Stream, error) {
	return end.End.AcceptStream()
}

func (end *edgeEnd) ListStreams() []geminio.Stream {
	return end.End.ListStreams()
}

// Meta
func (end *edgeEnd) EdgeID() uint64 {
	return end.End.ClientID()
}
