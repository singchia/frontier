package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontlas/apis"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"k8s.io/klog/v2"
)

func (cs *ClusterService) getFrontierByEdge(_ context.Context, req *v1.GetFrontierByEdgeIDRequest) (*v1.GetFrontierByEdgeIDResponse, error) {
	edge, err := cs.repo.GetEdge(req.EdgeId)
	if err != nil {
		klog.Errorf("cluster service edge err: %s", err)
		return nil, err
	}
	frontier, err := cs.repo.GetFrontier(edge.FrontierID)
	if err != nil {
		klog.Errorf("cluster service get frontier err: %s", err)
		return nil, err
	}
	return &v1.GetFrontierByEdgeIDResponse{
		Fontier: &v1.Frontier{
			FrontierId:       frontier.FrontierID,
			AdvertisedSbAddr: frontier.AdvertisedServiceboundAddr,
			AdvertisedEbAddr: frontier.AdvertisedEdgeboundAddr,
		},
	}, nil
}

func (cs *ClusterService) listFrontiers(_ context.Context, req *v1.ListFrontiersRequest) (*v1.ListFrontiersResponse, error) {
	// list frontiers by count and cursor
	if req.Count != nil && req.Cursor != nil {
		frontiers, cursor, err := cs.repo.GetFrontiersByCursor(&repo.FrontierQuery{
			Cursor: uint64(*req.Cursor),
			Count:  int64(*req.Count),
		})
		if err != nil {
			klog.Errorf("cluster service list frontier err: %s", err)
			return nil, err
		}
		cursorUint32 := uint32(cursor)
		return &v1.ListFrontiersResponse{
			Cursor:    &cursorUint32,
			Frontiers: transferFrontiers(frontiers),
		}, nil
	}

	// list frontiers by EdgeIDs
	if req.EdgeIds != nil {
		// TODO optimize by redis lua
		edges, err := cs.repo.GetEdges(req.EdgeIds)
		if err != nil {
			klog.Errorf("cluster service get edges err: %s", err)
			return nil, err
		}
		effectiveFrontierIDs := []string{}
		indexes := []int{}
		for i, edge := range edges {
			if edge == nil {
				continue
			}
			effectiveFrontierIDs = append(effectiveFrontierIDs, edge.FrontierID)
			indexes = append(indexes, i)
		}

		effectiveFrontiers, err := cs.repo.GetFrontiers(effectiveFrontierIDs)
		if err != nil {
			klog.Errorf("cluster service get frontiers err: %s", err)
			return nil, err
		}
		v1frontiers := make([]*v1.Frontier, len(req.EdgeIds))
		for i, frontier := range effectiveFrontiers {
			v1frontiers[indexes[i]] = transferFrontier(frontier)
		}
		return &v1.ListFrontiersResponse{
			Frontiers: v1frontiers,
		}, nil
	}

	// list frontiers by ID
	if req.FrontierIds != nil {
		frontiers, err := cs.repo.GetFrontiers(req.FrontierIds)
		if err != nil {
			klog.Errorf("cluster service get frontiers err: %s", err)
			return nil, err
		}
		v1frontiers := transferFrontiers(frontiers)
		return &v1.ListFrontiersResponse{
			Frontiers: v1frontiers,
		}, nil
	}

	return nil, apis.ErrIllegalRequest
}

func transferFrontiers(frontiers []*repo.Frontier) []*v1.Frontier {
	retFrontiers := make([]*v1.Frontier, len(frontiers))
	for i, frontier := range frontiers {
		retFrontier := transferFrontier(frontier)
		retFrontiers[i] = retFrontier
	}
	return retFrontiers
}

func transferFrontier(frontier *repo.Frontier) *v1.Frontier {
	if frontier == nil {
		return nil
	}
	return &v1.Frontier{
		FrontierId:       frontier.FrontierID,
		AdvertisedSbAddr: frontier.AdvertisedServiceboundAddr,
		AdvertisedEbAddr: frontier.AdvertisedEdgeboundAddr,
	}
}
