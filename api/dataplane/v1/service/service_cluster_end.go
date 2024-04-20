package service

import (
	"context"
	"sync"

	clusterv1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/geminio"
)

type frontierNservice struct {
	frontier *clusterv1.Frontier
	service  Service
}

type serviceClusterEnd struct {
	cc clusterv1.ClusterServiceClient

	bimap     *mapmap.BiMap // bidirectional edgeID and frontierID
	frontiers sync.Map      // key: frontierID; value: frontierNservice

	// options
	*serviceOption
	rpcs   map[string]geminio.RPC
	rpcMtx sync.RWMutex

	// update
	updating sync.RWMutex

	// fan-in channels
	acceptStreamCh chan geminio.Stream
	acceptMsgCh    chan geminio.Message
}

func (service *serviceClusterEnd) update() error {
	rsp, err := service.cc.ListFrontiers(context.TODO(), &clusterv1.ListFrontiersRequest{})
	if err != nil {
		service.logger.Errorf("list frontiers err: %s", err)
		return err
	}

	return nil
}
