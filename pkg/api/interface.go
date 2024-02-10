package api

import (
	"net"

	"github.com/singchia/geminio"
)

type Exchange interface {
	// rpc, message and raw io to edge
	ForwardToEdge(*Meta, geminio.End)
	// stream to edge
	StreamToEdge(geminio.Stream)
	// rpc, message and raw io to service
	ForwardToService(geminio.End)
	// stream to service
	StreamToService(geminio.Stream)
}

// edge related
type Edgebound interface {
	ListEdges() []geminio.End
	// for management
	GetEdgeByID(edgeID uint64) geminio.End
	DelEdgeByID(edgeID uint64) error
}

type EdgeInformer interface {
	EdgeOnline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeOffline(edgeID uint64, meta []byte, addr net.Addr)
	EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr)
}

// service related
type Servicebound interface {
	ListService() []geminio.End
	// for management
	GetService(service string) geminio.End
	DelSerivces(service string) error
}

type ServiceInformer interface {
	ServiceOnline(serviceID uint64, service string, addr net.Addr)
	ServiceOffline(serviceID uint64, service string, addr net.Addr)
	ServiceHeartbeat(serviceID uint64, service string, addr net.Addr)
}
