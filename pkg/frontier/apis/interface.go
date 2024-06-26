package apis

import (
	"net"

	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
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

	Serve() error
	Close() error
}

// service related
type Servicebound interface {
	ListService() []geminio.End
	// for management
	GetServiceByName(service string) (geminio.End, error)
	GetServicesByName(service string) ([]geminio.End, error)
	GetServiceByRPC(rpc string) (geminio.End, error)
	GetServicesByRPC(rpc string) ([]geminio.End, error)
	GetServiceByTopic(topic string) (geminio.End, error)
	GetServicesByTopic(topic string) ([]geminio.End, error)
	DelServiceByID(serviceID uint64) error
	DelSerivces(service string) error

	Serve() error
	Close() error
}

// informer
type EdgeInformer interface {
	EdgeOnline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeOffline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr)
	SetEdgeCount(count int)
}
type ServiceInformer interface {
	ServiceOnline(serviceID uint64, service string, addr net.Addr)
	ServiceOffline(serviceID uint64, service string, addr net.Addr)
	ServiceHeartbeat(serviceID uint64, service string, addr net.Addr)
	SetServiceCount(count int)
}

// repo
type Repo interface {
	Close() error
	CountEdgeRPCs(query *query.EdgeRPCQuery) (int64, error)
	CountEdges(query *query.EdgeQuery) (int64, error)
	CountServiceRPCs(query *query.ServiceRPCQuery) (int64, error)
	CountServiceTopics(query *query.ServiceTopicQuery) (int64, error)
	CountServices(query *query.ServiceQuery) (int64, error)
	CreateEdge(edge *model.Edge) error
	CreateEdgeRPC(rpc *model.EdgeRPC) error
	CreateService(service *model.Service) error
	CreateServiceRPC(rpc *model.ServiceRPC) error
	CreateServiceTopic(topic *model.ServiceTopic) error
	DeleteEdge(delete *query.EdgeDelete) error
	DeleteEdgeRPCs(edgeID uint64) error
	DeleteService(delete *query.ServiceDelete) error
	DeleteServiceRPCs(serviceID uint64) error
	DeleteServiceTopics(serviceID uint64) error
	GetEdge(edgeID uint64) (*model.Edge, error)
	GetService(serviceID uint64) (*model.Service, error)
	GetServiceByName(name string) (*model.Service, error)
	GetServicesByName(name string) ([]*model.Service, error)
	GetServiceRPC(rpc string) (*model.ServiceRPC, error)
	GetServiceRPCs(rpc string) ([]*model.ServiceRPC, error)
	GetServiceTopic(topic string) (*model.ServiceTopic, error)
	GetServiceTopics(topic string) ([]*model.ServiceTopic, error)
	ListEdgeRPCs(query *query.EdgeRPCQuery) ([]string, error)
	ListEdges(query *query.EdgeQuery) ([]*model.Edge, error)
	ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error)
	ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error)
	ListServices(query *query.ServiceQuery) ([]*model.Service, error)
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
	Addr   net.Addr
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

func WithAddr(addr net.Addr) OptionProduce {
	return func(po *ProduceOption) {
		po.Addr = addr
	}
}
