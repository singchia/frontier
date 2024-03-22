package frontierbound

import (
	"context"
	"encoding/json"

	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

// delegates for frontier itself
func (fm *FrontierManager) ConnOnline(d delegate.ConnDescriber) error {
	instance := &apis.FrontierInstance{}
	err := json.Unmarshal(d.Meta(), instance)
	if err != nil {
		klog.Errorf("frontier manager conn online, json unmarshal err: %s", err)
		return err
	}
	err = fm.repo.SetFrontier(instance.InstanceID, &repo.Frontier{
		FrontierID:                 instance.InstanceID,
		AdvertisedServiceboundAddr: instance.AdvertisedServiceboundAddr,
		AdvertisedEdgeboundAddr:    instance.AdvertisedEdgeboundAddr,
		EdgeCount:                  0,
		ServiceCount:               0,
	})
	return nil
}

func (fm *FrontierManager) ConnOffline(d delegate.ConnDescriber) error {
	return nil
}

func (fm *FrontierManager) Heartbeat(delegate.ConnDescriber) error {
	return nil
}

// rpcs
func (fm *FrontierManager) EdgeOnline(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}

func (fm *FrontierManager) EdgeOffline(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}

func (fm *FrontierManager) EdgeHeartbeat(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}
