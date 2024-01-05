package client

import (
	"context"
	"net"

	"github.com/singchia/geminio"
)

// RPCer is method oriented
type RPCer interface {
	NewRequest(data []byte) geminio.Request

	Call(ctx context.Context, method string, req geminio.Request) (geminio.Response, error)
	CallAsync(ctx context.Context, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error)
	Register(ctx context.Context, method string, rpc geminio.RPC) error
}

// Messager is topic oriented
type Messager interface {
	NewMessage(data []byte) geminio.Message

	// Publish a message to specific topic
	Publish(ctx context.Context, topic string, msg geminio.Message) error
	// Publish async a message to specific topic
	PublishAsync(ctx context.Context, topic string, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error)
	Receive(ctx context.Context) (geminio.Message, error)
}

type RPCMessager interface {
	RPCer
	Messager
}

// Stream multiplexer
type Multiplexer interface {
	// Open a stream to specific service
	OpenStream(service string) (geminio.Stream, error)
	AcceptStream() (geminio.Stream, error)
	ListStreams() []geminio.Stream
}

type Client interface {
	// Client can direct Message or RPC
	RPCMessager

	// Client can manage streams from or to a Service
	Multiplexer

	// Client is a net.Listener
	// The Accept is a wrapper for AccetpStream
	// The Addr is a wrapper for LocalAddr
	net.Listener
}

type Dialer func() (net.Conn, error)

func NewClient(dialer Dialer, opts ...ClientOption) (Client, error) {
	return nil, nil
}
