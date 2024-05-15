package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontlas/repo"
)

type ClusterService struct {
	v1.UnimplementedClusterServiceServer
	v1.UnimplementedHealthServer

	// repo
	repo      *repo.Dao
	readiness Readiness
}

func NewClusterService(repo *repo.Dao, readiness Readiness) *ClusterService {
	cs := &ClusterService{
		repo:      repo,
		readiness: readiness,
	}
	return cs
}

func (cs *ClusterService) GetEdgeByID(ctx context.Context, req *v1.GetEdgeByIDRequest) (*v1.GetEdgeByIDResponse, error) {
	return cs.getEdgeByID(ctx, req)
}

func (cs *ClusterService) GetEdgesCount(ctx context.Context, req *v1.GetEdgesCountRequest) (*v1.GetEdgesCountResponse, error) {
	return cs.getEdgesCount(ctx, req)
}

func (cs *ClusterService) GetFrontierByEdge(ctx context.Context, req *v1.GetFrontierByEdgeIDRequest) (*v1.GetFrontierByEdgeIDResponse, error) {
	return cs.getFrontierByEdge(ctx, req)
}

func (cs *ClusterService) GetServiceByID(ctx context.Context, req *v1.GetServiceByIDRequest) (*v1.GetServiceByIDResponse, error) {
	return cs.getServiceByID(ctx, req)
}

func (cs *ClusterService) GetServicesCount(ctx context.Context, req *v1.GetServicesCountRequest) (*v1.GetServicesCountResponse, error) {
	return cs.getServicesCount(ctx, req)
}
func (cs *ClusterService) ListEdges(ctx context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	return cs.listEdges(ctx, req)
}

func (cs *ClusterService) ListFrontiers(ctx context.Context, req *v1.ListFrontiersRequest) (*v1.ListFrontiersResponse, error) {
	return cs.listFrontiers(ctx, req)
}

func (cs *ClusterService) ListServices(ctx context.Context, req *v1.ListServicesRequest) (*v1.ListServicesResponse, error) {
	return cs.listServices(ctx, req)
}
