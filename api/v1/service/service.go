package service

import (
	"context"
	"net"

	"github.com/singchia/geminio"
)

// RPCer is edge and it's method oriented
type RPCer interface {
	NewRequest(data []byte) geminio.Request

	Call(ctx context.Context, edgeID uint64, method string, req geminio.Request) (geminio.Response, error)
	CallAsync(ctx context.Context, edgeID uint64, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error)
	Register(ctx context.Context, method string, rpc geminio.RPC) error
}

// Messager is edge oriented
type Messager interface {
	NewMessage(data []byte) geminio.Message

	Publish(ctx context.Context, topic string, msg geminio.Message) error
	PublishAsync(ctx context.Context, topic string, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error)
	Receive(ctx context.Context) (geminio.Message, error)
}

type RPCMessager interface {
	RPCer
	Messager
}

// Stream multiplexer
type Multiplexer interface {
	// Open a stream to specific edgeID
	OpenStream(edgeID uint64) (geminio.Stream, error)
	AcceptStream() (geminio.Stream, error)
	ListStream() []geminio.Stream
}

// controller functions
type GetEdgeID func(meta []byte) (uint64, error)
type EdgeOnline func(edgeID uint64, meta []byte, addr net.Addr) error
type EdgeOffline func(edgeID uint64, meta []byte, addr net.Addr) error

type ControlRegister interface {
	RegisterGetEdgeID(getEdgeID GetEdgeID) error
	RegisterEdgeOnline(edgeOnline EdgeOnline) error
	RegisterEdgeOnffline(edgeOffline EdgeOffline) error
}

type Service interface {
	// Service can direct Message or RPC
	RPCMessager

	// Service can manage streams from or to a Edge
	Multiplexer

	// Service is a net.Listener, actually it's wrapper of Multiplexer
	// The Accept is a wrapper for AcceptStream
	// The Addr is a wrapper for LocalAddr
	net.Listener

	// Service can register some control functions that be called by frontier when edge updated
	ControlRegister

	Close() error
}

type Dialer func() (net.Conn, error)

// the service field specific the role for this Service, and then Edge can OpenStream to this service
func NewService(dialer Dialer, service string, opts ...ServiceOption) (Service, error) {
	return nil, nil
}
