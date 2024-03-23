package frontierbound

import (
	"context"
	"encoding/json"
	"time"

	gapis "github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/frontlas/apis"
	"github.com/singchia/frontier/pkg/frontlas/repo"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"k8s.io/klog/v2"
)

// delegates for frontier itself
func (fm *FrontierManager) ConnOnline(d delegate.ConnDescriber) error {
	instance := &gapis.FrontierInstance{}
	err := json.Unmarshal(d.Meta(), instance)
	if err != nil {
		klog.Errorf("frontier manager conn online, json unmarshal err: %s", err)
		return err
	}
	set, err := fm.repo.SetFrontierAndAlive(instance.InstanceID, &repo.Frontier{
		FrontierID:                 instance.InstanceID,
		AdvertisedServiceboundAddr: instance.AdvertisedServiceboundAddr,
		AdvertisedEdgeboundAddr:    instance.AdvertisedEdgeboundAddr,
		EdgeCount:                  0,
		ServiceCount:               0,
	}, 30*time.Second)
	if err != nil {
		klog.Errorf("frontier manager conn online, set frontier and alive err: %s", err)
		return err
	}
	if !set {
		klog.V(1).Infof("frontier manager conn online, frontier: %s already set", instance.InstanceID)
		return apis.ErrFrontierAlreadySet
	}
	return nil
}

func (fm *FrontierManager) ConnOffline(d delegate.ConnDescriber) error {
	instance := &gapis.FrontierInstance{}
	err := json.Unmarshal(d.Meta(), instance)
	if err != nil {
		klog.Errorf("frontier manager conn offline, json unmarshal err: %s", err)
		return err
	}
	err = fm.repo.DeleteFrontier(instance.InstanceID)
	if err != nil {
		klog.Errorf("frontier manager conn offline, delete frontier: %s err: %s", instance.InstanceID, err)
		return err
	}
	return nil
}

func (fm *FrontierManager) Heartbeat(d delegate.ConnDescriber) error {
	instance := &gapis.FrontierInstance{}
	err := json.Unmarshal(d.Meta(), instance)
	if err != nil {
		klog.Errorf("frontier manager heartbeat, json unmarshal err: %s", err)
		return err
	}
	// the heartbeat comes every 20s, but we allow 10 seconds deviations.
	err = fm.repo.ExpireFrontier(instance.InstanceID, 30*time.Second)
	if err != nil {
		klog.Errorf("frontier manager heartbeat, expire frontier err: %s", err)
		return err
	}
	return nil
}

// rpcs
// rpcs of frontier stats
func (fm *FrontierManager) SyncStats(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	stats := &gapis.FrontierStats{}
	err := json.Unmarshal(req.Data(), stats)
	if err != nil {
		klog.Errorf("frontier manager sync stats, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
}

// rpcs of edges events
func (fm *FrontierManager) EdgeOnline(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}

func (fm *FrontierManager) EdgeOffline(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}

func (fm *FrontierManager) EdgeHeartbeat(ctx context.Context, req geminio.Request, rsp geminio.Response) {

}

// rpcs of services events
