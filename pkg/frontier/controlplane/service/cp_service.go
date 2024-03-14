package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/v1"
	"github.com/singchia/frontier/pkg/frontier/apis"
)

// @title Frontier Swagger API
// @version 1.0
// @contact.name Austin Zhai
// @contact.email singchia@163.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

type ControlPlaneService struct {
	v1.UnimplementedControlPlaneServer

	// dao and repo
	repo         apis.Repo
	servicebound apis.Servicebound
	edgebound    apis.Edgebound
}

func NewControlPlaneService(repo apis.Repo, servicebound apis.Servicebound, edgebound apis.Edgebound) *ControlPlaneService {
	cp := &ControlPlaneService{
		repo:         repo,
		servicebound: servicebound,
		edgebound:    edgebound,
	}
	return cp
}

// @Summary ListEdges
// @Tags 1.0
// @Param params query v1.ListEdgesRequest true "queries"
// @Success 200 {object} v1.ListEdgesResponse "result"
// @Router /v1/edges [get]
func (cps *ControlPlaneService) ListEdges(ctx context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	return cps.listEdges(ctx, req)
}

// @Summary Get Edge
// @Tags 1.0
// @Param params query v1.GetEdgeRequest true "queries"
// @Success 200 {object} v1.Edge "result"
// @Router /v1/edges/{edge_id} [get]
func (cps *ControlPlaneService) GetEdge(ctx context.Context, req *v1.GetEdgeRequest) (*v1.Edge, error) {
	return cps.getEdge(ctx, req)
}

// @Summary Kick Edge
// @Tags 1.0
// @Param params query v1.KickEdgeRequest true "queries"
// @Success 200 {object} v1.KickEdgeResponse "result"
// @Router /v1/edges/{edge_id} [delete]
func (cps *ControlPlaneService) KickEdge(ctx context.Context, req *v1.KickEdgeRequest) (*v1.KickEdgeResponse, error) {
	return cps.kickEdge(ctx, req)
}

// @Summary List Edges RPCs
// @Tags 1.0
// @Param params query v1.ListEdgeRPCsRequest true "queries"
// @Success 200 {object} v1.ListEdgeRPCsResponse "result"
// @Router /v1/edges/rpcs [get]
func (cps *ControlPlaneService) ListEdgeRPCs(ctx context.Context, req *v1.ListEdgeRPCsRequest) (*v1.ListEdgeRPCsResponse, error) {
	return cps.listEdgeRPCs(ctx, req)
}

// @Summary List Services
// @Tags 1.0
// @Param params query v1.ListServicesRequest true "queries"
// @Success 200 {object} v1.ListServicesResponse "result"
// @Router /v1/services [get]
func (cps *ControlPlaneService) ListServices(ctx context.Context, req *v1.ListServicesRequest) (*v1.ListServicesResponse, error) {
	return cps.listServices(ctx, req)
}

// @Summary Get Service
// @Tags 1.0
// @Param params query v1.GetServiceRequest true "queries"
// @Success 200 {object} v1.Service "result"
// @Router /v1/services/{service_id} [get]
func (cps *ControlPlaneService) GetService(ctx context.Context, req *v1.GetServiceRequest) (*v1.Service, error) {
	return cps.getService(ctx, req)
}

// @Summary Kick Service
// @Tags 1.0
// @Param params query v1.KickServiceRequest true "queries"
// @Success 200 {object} v1.KickServiceResponse "result"
// @Router /v1/services/{service_id} [delete]
func (cps *ControlPlaneService) KickService(ctx context.Context, req *v1.KickServiceRequest) (*v1.KickServiceResponse, error) {
	return cps.kickService(ctx, req)
}

// @Summary List Services RPCs
// @Tags 1.0
// @Param params query v1.ListServiceRPCsRequest true "queries"
// @Success 200 {object} v1.ListServiceRPCsResponse "result"
// @Router /v1/services/rpcs [get]
func (cps *ControlPlaneService) ListServiceRPCs(ctx context.Context, req *v1.ListServiceRPCsRequest) (*v1.ListServiceRPCsResponse, error) {
	return cps.listServiceRPCs(ctx, req)
}

// @Summary List Services Topics
// @Tags 1.0
// @Param params query v1.ListServiceTopicsRequest true "queries"
// @Success 200 {object} v1.ListServiceTopicsResponse "result"
// @Router /v1/services/topics [get]
func (cps *ControlPlaneService) ListServiceTopics(ctx context.Context, req *v1.ListServiceTopicsRequest) (*v1.ListServiceTopicsResponse, error) {
	return cps.listServiceTopics(ctx, req)
}
