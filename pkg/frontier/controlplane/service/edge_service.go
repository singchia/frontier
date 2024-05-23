package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontier/v1"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

func (cps *ControlPlaneService) listEdges(_ context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	query := &query.EdgeQuery{}
	// conditions
	if req.Meta != nil {
		query.Meta = *req.Meta
	}
	if req.Addr != nil {
		query.Addr = *req.Addr
	}
	if req.Rpc != nil {
		query.RPC = *req.Rpc
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
			query.Desc = true
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

	edges, err := cps.repo.ListEdges(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.repo.CountEdges(query)
	if err != nil {
		return nil, err
	}
	retEdges := transferEdges(edges)
	return &v1.ListEdgesResponse{
		Edges: retEdges,
		Count: uint32(count),
	}, nil
}

func (cps *ControlPlaneService) getEdge(_ context.Context, req *v1.GetEdgeRequest) (*v1.Edge, error) {
	edge, err := cps.repo.GetEdge(req.EdgeId)
	if err != nil {
		return nil, err
	}
	return transferEdge(edge), nil
}

func (cps *ControlPlaneService) kickEdge(_ context.Context, req *v1.KickEdgeRequest) (*v1.KickEdgeResponse, error) {
	err := cps.edgebound.DelEdgeByID(req.EdgeId)
	if err != nil {
		return nil, err
	}
	return &v1.KickEdgeResponse{}, nil
}

func (cps *ControlPlaneService) listEdgeRPCs(_ context.Context, req *v1.ListEdgeRPCsRequest) (*v1.ListEdgeRPCsResponse, error) {
	query := &query.EdgeRPCQuery{}
	// conditions
	if req.EdgeId != nil {
		query.EdgeID = *req.EdgeId
	}
	if req.Meta != nil {
		query.Meta = *req.Meta
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
	query.StartTime = *req.StartTime
	query.EndTime = *req.EndTime

	rpcs, err := cps.repo.ListEdgeRPCs(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.repo.CountEdgeRPCs(query)
	if err != nil {
		return nil, err
	}
	return &v1.ListEdgeRPCsResponse{
		Rpcs:  rpcs,
		Count: uint32(count),
	}, nil
}

func transferEdges(edges []*model.Edge) []*v1.Edge {
	retEdges := make([]*v1.Edge, len(edges))
	for i, edge := range edges {
		retEdge := transferEdge(edge)
		retEdges[i] = retEdge
	}
	return retEdges
}

func transferEdge(edge *model.Edge) *v1.Edge {
	retEdge := &v1.Edge{
		EdgeId:     edge.EdgeID,
		Meta:       edge.Meta,
		Addr:       edge.Addr,
		CreateTime: edge.CreateTime,
	}
	return retEdge
}
