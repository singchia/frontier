package apis

import (
	"net"

	"github.com/singchia/geminio"
)

type Exchange interface {
	// For Service
	// rpc, message and raw io to edge
	ForwardToEdge(*Meta, geminio.End)
	// stream to edge
	StreamToEdge(geminio.Stream)

	// For Edge
	GetEdgeID(meta []byte) (uint64, error) // get EdgeID for edge
	EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) error
	EdgeOffline(edgeID uint64, meta []byte, addr net.Addr) error
	// rpc, message and raw io to service
	ForwardToService(geminio.End)
	// stream to service
	StreamToService(geminio.Stream)

	// for exchange
	AddEdgebound(Edgebound)
	AddServicebound(Servicebound)
}

// edge related
type Edgebound interface {
	ListEdges() []geminio.End
	// for management
	GetEdgeByID(edgeID uint64) geminio.End
	DelEdgeByID(edgeID uint64) error

	Serve()
	Close() error
}

// service related
type Servicebound interface {
	ListService() []geminio.End
	// for management
	GetServiceByName(service string) (geminio.End, error)
	GetServiceByRPC(rpc string) (geminio.End, error)
	GetServiceByTopic(topic string) (geminio.End, error)
	DelSerivces(service string) error

	Serve()
	Close() error
}

// informer
type EdgeInformer interface {
	EdgeOnline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeOffline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr)
}
type ServiceInformer interface {
	ServiceOnline(serviceID uint64, service string, addr net.Addr)
	ServiceOffline(serviceID uint64, service string, addr net.Addr)
	ServiceHeartbeat(serviceID uint64, service string, addr net.Addr)
}

// mq manager and mq related
type MQM interface {
	// MQM is a MQ wrapper
	MQ
	AddMQ(topics []string, mq MQ)
	AddMQByEnd(topics []string, end geminio.End)
	DelMQ(mq MQ)
	DelMQByEnd(end geminio.End)
	GetMQ(topic string) MQ
	GetMQs(topic string) []MQ
}

type MQ interface {
	Produce(topic string, data []byte, opts ...OptionProduce) error
	Close() error
}

type ProduceOption struct {
	Origin interface{}
	EdgeID uint64
}

type OptionProduce func(*ProduceOption)

func WithEdgeID(edgeID uint64) OptionProduce {
	return func(po *ProduceOption) {
		po.EdgeID = edgeID
	}
}

func WithOrigin(origin interface{}) OptionProduce {
	return func(po *ProduceOption) {
		po.Origin = origin
	}
}
