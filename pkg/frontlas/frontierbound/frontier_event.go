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

const (
	edgeHeartbeatInterval     = 30 * time.Second
	serviceHeartbeatInterval  = 30 * time.Second
	frontierHeartbeatInterval = 30 * time.Second
)

// delegates for frontier itself
func (fm *FrontierManager) GetClientID(meta []byte) (uint64, error) {
	return fm.idFactory.GetID(), nil
}

func (fm *FrontierManager) ConnOnline(d delegate.ConnDescriber) error {
	instance := &gapis.FrontierInstance{}
	err := json.Unmarshal(d.Meta(), instance)
	if err != nil {
		klog.Errorf("frontier manager conn online, json unmarshal err: %s", err)
		return err
	}
	klog.V(1).Infof("frontier online, frontierID: %s, advertised_servicebound_addr: %s, advertised_edgebound_addr: %s",
		instance.FrontierID, instance.AdvertisedServiceboundAddr, instance.AdvertisedEdgeboundAddr)

	set, err := fm.repo.SetFrontierAndAlive(instance.FrontierID, &repo.Frontier{
		FrontierID:                 instance.FrontierID,
		AdvertisedServiceboundAddr: instance.AdvertisedServiceboundAddr,
		AdvertisedEdgeboundAddr:    instance.AdvertisedEdgeboundAddr,
		EdgeCount:                  0,
		ServiceCount:               0,
	}, frontierHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager conn online, set frontier and alive err: %s", err)
		return err
	}
	if !set {
		klog.V(1).Infof("frontier manager conn online, frontier: %s already set", instance.FrontierID)
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
	klog.V(1).Infof("frontier offline, frontierID: %s", instance.FrontierID)

	err = fm.repo.DeleteFrontier(instance.FrontierID)
	if err != nil {
		klog.Errorf("frontier manager conn offline, delete frontier: %s err: %s", instance.FrontierID, err)
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
	err = fm.repo.ExpireFrontier(instance.FrontierID, frontierHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager heartbeat, expire frontier err: %s", err)
		return err
	}
	return nil
}

// rpcs of edges events
func (fm *FrontierManager) EdgeOnline(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	edgeOnline := &gapis.EdgeOnline{}
	err := json.Unmarshal(req.Data(), edgeOnline)
	if err != nil {
		klog.Errorf("frontier manager edge online, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.SetEdgeAndAlive(edgeOnline.EdgeID, &repo.Edge{
		FrontierID: edgeOnline.FrontierID,
		Addr:       edgeOnline.Addr,
		UpdateTime: time.Now().Unix(),
	}, edgeHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager edge online, set edge and alive err: %s", err)
		rsp.SetError(err)
		return
	}
}

func (fm *FrontierManager) EdgeOffline(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	edgeOffline := &gapis.EdgeOffline{}
	err := json.Unmarshal(req.Data(), edgeOffline)
	if err != nil {
		klog.Errorf("frontier manager edge offline, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.DeleteEdge(edgeOffline.EdgeID)
	if err != nil {
		klog.Errorf("frontier manager edge offline, delete edge err: %s", err)
		rsp.SetError(err)
		return
	}
}

func (fm *FrontierManager) EdgeHeartbeat(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	edgeHB := &gapis.EdgeHeartbeat{}
	err := json.Unmarshal(req.Data(), edgeHB)
	if err != nil {
		klog.Errorf("frontier manager edge heartbeat, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.ExpireEdge(edgeHB.EdgeID, edgeHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager edge heartbeat, expire edge err: %s", err)
		rsp.SetError(err)
		return
	}
}

// rpcs of services events
func (fm *FrontierManager) ServiceOnline(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	serviceOnline := &gapis.ServiceOnline{}
	err := json.Unmarshal(req.Data(), serviceOnline)
	if err != nil {
		klog.Errorf("frontier manager service online, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.SetServiceAndAlive(serviceOnline.ServiceID, &repo.Service{
		FrontierID: serviceOnline.FrontierID,
		Service:    serviceOnline.Service,
		Addr:       serviceOnline.Addr,
		UpdateTime: time.Now().Unix(),
	}, serviceHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager service online, set service and alive err: %s", err)
		rsp.SetError(err)
		return
	}
}

func (fm *FrontierManager) ServiceOffline(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	serviceOffline := &gapis.ServiceOffline{}
	err := json.Unmarshal(req.Data(), serviceOffline)
	if err != nil {
		klog.Errorf("frontier manager service offline, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.DeleteService(serviceOffline.ServiceID)
	if err != nil {
		klog.Errorf("frontier manager service offline, delete service err: %s", err)
		rsp.SetError(err)
		return
	}
}

func (fm *FrontierManager) ServiceHeartbeat(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	serviceHB := &gapis.ServiceHeartbeat{}
	err := json.Unmarshal(req.Data(), serviceHB)
	if err != nil {
		klog.Errorf("frontier manager service heartbeat, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.ExpireService(serviceHB.ServiceID, serviceHeartbeatInterval)
	if err != nil {
		klog.Errorf("frontier manager service heartbeat, expire service err: %s", err)
		rsp.SetError(err)
		return
	}
}

// rpcs of frontiers events
func (fm *FrontierManager) FrontierStats(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	stats := &gapis.FrontierStats{}
	err := json.Unmarshal(req.Data(), stats)
	if err != nil {
		klog.Errorf("frontier manager frontier stats, json unmarshal err: %s", err)
		rsp.SetError(err)
		return
	}
	err = fm.repo.SetFrontierCount(stats.FrontierID, stats.EdgeCount, stats.ServiceCount)
	if err != nil {
		klog.Errorf("frontier manager frontier stats, set frontier count err: %s", err)
		rsp.SetError(err)
		return
	}
}
