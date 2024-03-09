package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/v1"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/repo/model"
)

func (cps *ControlPlaneService) listServices(_ context.Context, req *v1.ListServicesRequest) (*v1.ListServicesResponse, error) {
	query := &dao.ServiceQuery{}
	// conditions
	if req.Service != nil {
		query.Service = *req.Service
	}
	if req.Addr != nil {
		query.Addr = *req.Addr
	}
	if req.Rpc != nil {
		query.RPC = *req.Rpc
	}
	if req.Topic != nil {
		query.Topic = *req.Topic
	}
	if req.ServiceId != nil {
		query.ServiceID = *req.ServiceId
	}
	// order
	if req.Order != nil && len(*req.Order) != 0 {
		order := *req.Order
		switch order[0] {
		case '-':
			query.Order = order[1:]
			query.Desc = true
		case '+':
			query.Order = order[1:]
			query.Desc = false
		default:
			query.Order = order
			query.Desc = false
		}
	}
	// pagination
	query.Page = int(req.Page)
	query.PageSize = int(req.PageSize)
	// time range
	if req.StartTime != nil && req.EndTime != nil {
		query.StartTime = *req.StartTime
		query.EndTime = *req.EndTime
	}

	services, err := cps.repo.ListServices(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.repo.CountServices(query)
	if err != nil {
		return nil, err
	}
	retServices := transferServices(services)
	return &v1.ListServicesResponse{
		Services: retServices,
		Count:    uint32(count),
	}, nil
}

func (cps *ControlPlaneService) getService(_ context.Context, req *v1.GetServiceRequest) (*v1.Service, error) {
	service, err := cps.repo.GetService(req.ServiceId)
	if err != nil {
		return nil, err
	}
	return transferService(service), nil
}

func (cps *ControlPlaneService) kickService(_ context.Context, req *v1.KickServiceRequest) (*v1.KickServiceResponse, error) {
	err := cps.servicebound.DelServiceByID(req.ServiceId)
	if err != nil {
		return nil, err
	}
	return &v1.KickServiceResponse{}, nil
}

func (cps *ControlPlaneService) listServiceRPCs(_ context.Context, req *v1.ListServiceRPCsRequest) (*v1.ListServiceRPCsResponse, error) {
	query := &dao.ServiceRPCQuery{}
	// conditions
	if req.ServiceId != nil {
		query.ServiceID = *req.ServiceId
	}
	if req.Service != nil {
		query.Service = *req.Service
	}
	// order
	if req.Order != nil && len(*req.Order) != 0 {
		order := *req.Order
		switch order[0] {
		case '-':
			query.Order = order[1:]
			query.Desc = true
		case '+':
			query.Order = order[1:]
			query.Desc = false
		default:
			query.Order = order
			query.Desc = false
		}
	}
	// pagination
	query.Page = int(req.Page)
	query.PageSize = int(req.PageSize)
	// time range
	if req.StartTime != nil && req.EndTime != nil {
		query.StartTime = *req.StartTime
		query.EndTime = *req.EndTime
	}

	rpcs, err := cps.repo.ListServiceRPCs(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.repo.CountServiceRPCs(query)
	if err != nil {
		return nil, err
	}
	return &v1.ListServiceRPCsResponse{
		Rpcs:  rpcs,
		Count: uint32(count),
	}, nil
}

func (cps *ControlPlaneService) listServiceTopics(_ context.Context, req *v1.ListServiceTopicsRequest) (*v1.ListServiceTopicsResponse, error) {
	query := &dao.ServiceTopicQuery{}
	// conditions
	if req.ServiceId != nil {
		query.ServiceID = *req.ServiceId
	}
	if req.Service != nil {
		query.Service = *req.Service
	}
	// order
	if req.Order != nil && len(*req.Order) != 0 {
		order := *req.Order
		switch order[0] {
		case '-':
			query.Order = order[1:]
			query.Desc = true
		case '+':
			query.Order = order[1:]
			query.Desc = false
		default:
			query.Order = order
			query.Desc = false
		}
	}
	// pagination
	query.Page = int(req.Page)
	query.PageSize = int(req.PageSize)
	// time range
	if req.StartTime != nil && req.EndTime != nil {
		query.StartTime = *req.StartTime
		query.EndTime = *req.EndTime
	}

	topics, err := cps.repo.ListServiceTopics(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.repo.CountServiceTopics(query)
	if err != nil {
		return nil, err
	}
	return &v1.ListServiceTopicsResponse{
		Topics: topics,
		Count:  uint32(count),
	}, nil
}

func transferServices(services []*model.Service) []*v1.Service {
	retServices := make([]*v1.Service, len(services))
	for i, service := range services {
		retService := transferService(service)
		retServices[i] = retService
	}
	return retServices
}

func transferService(service *model.Service) *v1.Service {
	retService := &v1.Service{
		ServiceId:  service.ServiceID,
		Service:    service.Service,
		Addr:       service.Addr,
		CreateTime: service.CreateTime,
	}
	return retService
}
