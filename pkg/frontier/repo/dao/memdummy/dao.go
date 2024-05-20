package memdummy

import (
	"errors"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

type dao struct {
}

func NewDao(config *config.Configuration) (*dao, error) {
	return &dao{}, nil
}
func (dao *dao) Close() error {
	return nil
}

func (dao *dao) CountEdgeRPCs(query *query.EdgeRPCQuery) (int64, error) {
	return 0, nil
}

func (dao *dao) CountEdges(query *query.EdgeQuery) (int64, error) {
	return 0, nil
}

func (dao *dao) CountServiceRPCs(query *query.ServiceRPCQuery) (int64, error) {
	return 0, nil
}

func (dao *dao) CountServiceTopics(query *query.ServiceTopicQuery) (int64, error) {
	return 0, nil
}

func (dao *dao) CountServices(query *query.ServiceQuery) (int64, error) {
	return 0, nil
}

func (dao *dao) CreateEdge(edge *model.Edge) error {
	return nil
}

func (dao *dao) CreateEdgeRPC(rpc *model.EdgeRPC) error {
	return nil
}

func (dao *dao) CreateService(service *model.Service) error {
	return nil
}

func (dao *dao) CreateServiceRPC(rpc *model.ServiceRPC) error {
	return nil
}

func (dao *dao) CreateServiceTopic(topic *model.ServiceTopic) error {
	return nil
}

func (dao *dao) DeleteEdge(delete *query.EdgeDelete) error {
	return nil
}

func (dao *dao) DeleteEdgeRPCs(edgeID uint64) error {
	return nil
}

func (dao *dao) DeleteService(delete *query.ServiceDelete) error {
	return nil
}

func (dao *dao) DeleteServiceRPCs(serviceID uint64) error {
	return nil
}

func (dao *dao) DeleteServiceTopics(serviceID uint64) error {
	return nil
}

func (dao *dao) GetEdge(edgeID uint64) (*model.Edge, error) {
	return nil, errors.New("not found")
}

func (dao *dao) GetService(serviceID uint64) (*model.Service, error) {
	return nil, errors.New("not found")
}

func (dao *dao) GetServiceByName(name string) (*model.Service, error) {
	return nil, errors.New("not found")
}

func (dao *dao) GetServiceRPC(rpc string) (*model.ServiceRPC, error) {
	return nil, errors.New("not found")
}

func (dao *dao) GetServiceTopic(topic string) (*model.ServiceTopic, error) {
	return nil, errors.New("not found")
}

func (dao *dao) ListEdgeRPCs(query *query.EdgeRPCQuery) ([]string, error) {
	return nil, errors.New("not found")
}

func (dao *dao) ListEdges(query *query.EdgeQuery) ([]*model.Edge, error) {
	return nil, errors.New("not found")
}

func (dao *dao) ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error) {
	return nil, errors.New("not found")
}

func (dao *dao) ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error) {
	return nil, errors.New("not found")
}

func (dao *dao) ListServices(query *query.ServiceQuery) ([]*model.Service, error) {
	return nil, errors.New("not found")
}
