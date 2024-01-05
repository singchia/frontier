package client

import (
	"context"

	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"github.com/singchia/geminio/options"
)

type clientEnd struct {
	geminio.End
}

func newclientEnd(dialer client.Dialer, opts ...ClientOption) (*clientEnd, error) {
	// options
	copt := &clientOption{}
	for _, opt := range opts {
		opt(copt)
	}
	eopts := &client.RetryEndOptions{}
	if copt.tmr != nil {
		eopts.SetTimer(copt.tmr)
	}
	if copt.logger != nil {
		eopts.SetLog(copt.logger)
	}
	if copt.clientID != nil {
		eopts.SetClientID(*copt.clientID)
	}
	if copt.meta != nil {
		eopts.SetMeta(copt.meta)
	}

	// new geminio end
	end, err := client.NewRetryEndWithDialer(dialer, eopts)
	if err != nil {
		return nil, err
	}
	return &clientEnd{end}, nil
}

// RPCer
func (end *clientEnd) NewRequest(data []byte) geminio.Request {
	return end.End.NewRequest(data)
}

func (end *clientEnd) Call(ctx context.Context, method string, req geminio.Request) (geminio.Response, error) {
	return end.End.Call(ctx, method, req)
}

func (end *clientEnd) CallAsync(ctx context.Context, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error) {
	return end.End.CallAsync(ctx, method, req, ch)
}

func (end *clientEnd) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	return end.End.Register(ctx, method, rpc)
}

// Messager
func (end *clientEnd) NewMessage(data []byte) geminio.Message {
	return end.End.NewMessage(data)
}

func (end *clientEnd) Publish(ctx context.Context, topic string, msg geminio.Message) error {
	msg.SetTopic(topic)
	return end.End.Publish(ctx, msg)
}

func (end *clientEnd) PublishAsync(ctx context.Context, topic string, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error) {
	msg.SetTopic(topic)
	return end.End.PublishAsync(ctx, msg, ch)
}

func (end *clientEnd) Receive(ctx context.Context) (geminio.Message, error) {
	return end.End.Receive(ctx)
}

// Multiplexer
func (end *clientEnd) OpenStream(service string) (geminio.Stream, error) {
	opt := options.OpenStream()
	opt.SetPeer(service)
	return end.End.OpenStream(opt)
}

func (end *clientEnd) AcceptStream() (geminio.Stream, error) {
	return end.End.AcceptStream()
}

func (end *clientEnd) ListStreams() []geminio.Stream {
	return end.End.ListStreams()
}
