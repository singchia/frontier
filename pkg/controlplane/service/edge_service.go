package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/v1"
	"github.com/singchia/frontier/pkg/repo/dao"
	"github.com/singchia/frontier/pkg/repo/model"
)

func (cps *ControlPlaneService) ListEdges(ctx context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	return cps.listEdges(ctx, req)
}

func (cps *ControlPlaneService) GetEdge(ctx context.Context, req *v1.GetEdgeRequest) (*v1.Edge, error) {
	return cps.getEdge(ctx, req)
}

func (cps *ControlPlaneService) KickEdge(ctx context.Context, req *v1.KickEdgeRequest) (*v1.KickEdgeResponse, error) {
	return cps.kickEdge(ctx, req)
}

func (cps *ControlPlaneService) ListEdgeRPCs(ctx context.Context, req *v1.ListEdgeRPCsRequest) (*v1.ListEdgeRPCsResponse, error) {
	return cps.listEdgeRPCs(ctx, req)
}

func (cps *ControlPlaneService) listEdges(_ context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	query := &dao.EdgeQuery{}
	if req.Meta != nil {
		query.Meta = *req.Meta
	}
	if req.Addr != nil {
		query.Addr = *req.Addr
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
	if req.Rpc != nil {
		query.RPC = *req.Rpc
	}
	if req.EdgeId != nil {
		query.EdgeID = *req.EdgeId
	}
	// pagination
	query.Page = int(req.Page)
	query.PageSize = int(req.PageSize)
	query.StartTime = *req.StartTime
	query.EndTime = *req.EndTime

	edges, err := cps.dao.ListEdges(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.dao.CountEdges(query)
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
	edge, err := cps.dao.GetEdge(req.EdgeId)
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
	query := &dao.EdgeRPCQuery{}
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
	query.StartTime = *req.StartTime
	query.EndTime = *req.EndTime

	rpcs, err := cps.dao.ListEdgeRPCs(query)
	if err != nil {
		return nil, err
	}
	count, err := cps.dao.CountEdgeRPCs(query)
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
