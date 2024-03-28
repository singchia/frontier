package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontlas/apis"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"k8s.io/klog/v2"
)

func (cs *ClusterService) getServiceByID(_ context.Context, req *v1.GetServiceByIDRequest) (*v1.GetServiceByIDResponse, error) {
	svc, err := cs.repo.GetService(req.ServiceId)
	if err != nil {
		klog.Errorf("cluster service get service err: %s", err)
		return nil, err
	}
	return &v1.GetServiceByIDResponse{
		Service: &v1.Service{
			Service:    svc.Service,
			Addr:       svc.Addr,
			FrontierId: svc.FrontierID,
			UpdateTime: uint64(svc.UpdateTime),
		},
	}, nil
}

func (cs *ClusterService) getServicesCount(_ context.Context, _ *v1.GetServicesCountRequest) (*v1.GetServicesCountResponse, error) {
	count, err := cs.repo.CountServices()
	if err != nil {
		klog.Errorf("cluster service get services count err: %s", err)
		return nil, err
	}
	return &v1.GetServicesCountResponse{
		Count: uint64(count),
	}, nil
}

func (cs *ClusterService) listServices(_ context.Context, req *v1.ListServicesRequest) (*v1.ListServicesResponse, error) {
	// list services by count and cursor
	if req.Count != nil && req.Cursor != nil {
		services, cursor, err := cs.repo.GetServicesByCursor(&repo.ServiceQuery{
			Cursor: uint64(*req.Cursor),
			Count:  int64(*req.Count),
		})
		if err != nil {
			klog.Errorf("cluster service get services by cursor err: %s", err)
			return nil, err
		}
		cursorUint32 := uint32(cursor)
		return &v1.ListServicesResponse{
			Cursor:   &cursorUint32,
			Services: transferServices(services),
		}, nil
	}

	// list services by IDs
	if req.ServiceIds != nil {
		services, err := cs.repo.GetServices(req.ServiceIds)
		if err != nil {
			klog.Errorf("cluster service get services err: %s", err)
			return nil, err
		}
		return &v1.ListServicesResponse{
			Services: transferServices(services),
		}, nil
	}

	return nil, apis.ErrIllegalRequest
}

func transferServices(services []*repo.Service) []*v1.Service {
	retServices := make([]*v1.Service, len(services))
	for i, service := range services {
		retService := transferService(service)
		retServices[i] = retService
	}
	return retServices
}

func transferService(service *repo.Service) *v1.Service {
	return &v1.Service{
		Service:    service.Service,
		Addr:       service.Addr,
		FrontierId: service.FrontierID,
		UpdateTime: uint64(service.UpdateTime),
	}
}
