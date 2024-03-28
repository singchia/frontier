package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontlas/apis"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"k8s.io/klog/v2"
)

func (cs *ClusterService) getEdgeByID(_ context.Context, req *v1.GetEdgeByIDRequest) (*v1.GetEdgeByIDResponse, error) {
	edge, err := cs.repo.GetEdge(req.EdgeId)
	if err != nil {
		klog.Errorf("cluster service get edge by ID err: %s", err)
		return nil, err
	}
	return &v1.GetEdgeByIDResponse{
		Edge: &v1.Edge{
			EdgeId:     req.EdgeId,
			Addr:       edge.Addr,
			FrontierId: edge.FrontierID,
			UpdateTime: uint64(edge.UpdateTime),
		},
	}, nil
}

func (cs *ClusterService) getEdgesCount(_ context.Context, _ *v1.GetEdgesCountRequest) (*v1.GetEdgesCountResponse, error) {
	count, err := cs.repo.CountEdges()
	if err != nil {
		klog.Errorf("cluster service count edges err: %s", err)
		return nil, err
	}
	return &v1.GetEdgesCountResponse{
		Count: uint64(count),
	}, nil
}

func (cs *ClusterService) listEdges(_ context.Context, req *v1.ListEdgesRequest) (*v1.ListEdgesResponse, error) {
	// list edges by count and cursor
	if req.Count != nil && req.Cursor != nil {
		edges, cursor, err := cs.repo.GetEdgesByCursor(&repo.EdgeQuery{
			Cursor: uint64(*req.Cursor),
			Count:  int64(*req.Count),
		})
		if err != nil {
			klog.Errorf("cluster service get edges by cursor err: %s", err)
			return nil, err
		}
		cursorUint32 := uint32(cursor)
		return &v1.ListEdgesResponse{
			Cursor: &cursorUint32,
			Edges:  transferEdges(edges),
		}, nil
	}

	// list edges by ids
	if req.EdgeIds != nil {
		edges, err := cs.repo.GetEdges(req.EdgeIds)
		if err != nil {
			klog.Errorf("cluster service get edges err: %s", err)
			return nil, err
		}
		return &v1.ListEdgesResponse{
			Edges: transferEdges(edges),
		}, nil
	}

	return nil, apis.ErrIllegalRequest
}

func transferEdges(edges []*repo.Edge) []*v1.Edge {
	retEdges := make([]*v1.Edge, len(edges))
	for i, edge := range edges {
		retEdge := transferEdge(edge)
		retEdges[i] = retEdge
	}
	return retEdges
}

func transferEdge(edge *repo.Edge) *v1.Edge {
	if edge == nil {
		return nil
	}
	return &v1.Edge{
		FrontierId: edge.FrontierID,
		Addr:       edge.Addr,
		UpdateTime: uint64(edge.UpdateTime),
	}
}
