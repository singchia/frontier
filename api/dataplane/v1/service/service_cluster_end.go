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

	service.updating.Lock()
	defer service.updating.Unlock()

	alive := []string{}
	service.frontiers.Range(func(key, value interface{}) bool {
		frontierID := key.(string)
		frontierNservice := value.(*frontierNservice)
		for _, frontier := range rsp.Frontiers {
			if frontierID == frontier.FrontierId {
				if !frontierEqual(frontierNservice.frontier, frontier) {
					frontierNservice.frontier = frontier
					// servicebound addr changed, but we will reconnect later
				}
				alive = append(alive, frontierID)
				return true
			}
		}

		// out of date frontier
		service.logger.Debugf("frontier: %v needs to be removed", key)
		return true
	})

	return nil
}

func frontierEqual(a, b *clusterv1.Frontier) bool {
	return a.AdvertisedSbAddr == b.AdvertisedEbAddr &&
		a.FrontierId == b.FrontierId
}
