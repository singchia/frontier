package dao

import (
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

type Dao interface {
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
	GetServiceRPC(rpc string) (*model.ServiceRPC, error)
	GetServiceTopic(topic string) (*model.ServiceTopic, error)
	ListEdgeRPCs(query *query.EdgeRPCQuery) ([]string, error)
	ListEdges(query *query.EdgeQuery) ([]*model.Edge, error)
	ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error)
	ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error)
	ListServices(query *query.ServiceQuery) ([]*model.Service, error)
}

type dao struct {
	config *config.Configuration
}

func NewDao(config *config.Configuration) (Dao, error) {
	return nil, nil
}

func (dao *dao) Close() error {
	return nil
}
