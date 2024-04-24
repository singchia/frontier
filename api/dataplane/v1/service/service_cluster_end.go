package service

import (
	"context"
	"net"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"

	mapset "github.com/deckarep/golang-set/v2"
	clusterv1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type frontierNservice struct {
	frontier *clusterv1.Frontier
	service  Service
}

type serviceClusterEnd struct {
	*delegate.UnimplementedDelegate
	cc clusterv1.ClusterServiceClient

	bimap     *mapmap.BiMap // bidirectional edgeID and frontierID
	frontiers sync.Map      // key: frontierID; value: frontierNservice

	// options
	*serviceOption
	rpcs   map[string]geminio.RPC
	topics mapset.Set[string]
	appMtx sync.RWMutex

	// update
	updating sync.RWMutex

	// fan-in channels
	acceptStreamCh chan geminio.Stream
	acceptMsgCh    chan geminio.Message

	closed chan struct{}
}

func newServiceClusterEnd(addr string, opts ...ServiceOption) (*serviceClusterEnd, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	cc := clusterv1.NewClusterServiceClient(conn)

	serviceClusterEnd := &serviceClusterEnd{
		cc:             cc,
		serviceOption:  &serviceOption{},
		rpcs:           map[string]geminio.RPC{},
		topics:         mapset.NewSet[string](),
		acceptStreamCh: make(chan geminio.Stream, 128),
		acceptMsgCh:    make(chan geminio.Message, 128),
		closed:         make(chan struct{}),
	}
	serviceClusterEnd.serviceOption.delegate = serviceClusterEnd

	for _, opt := range opts {
		opt(serviceClusterEnd.serviceOption)
	}
	if serviceClusterEnd.serviceOption.logger == nil {
		serviceClusterEnd.serviceOption.logger = armlog.DefaultLog
	}
	return serviceClusterEnd, nil
}

func (service *serviceClusterEnd) start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := service.update()
			if err != nil {
				service.logger.Warnf("cluster update err: %s", err)
				continue
			}
		case <-service.closed:
			return
		}
	}
}

func (service *serviceClusterEnd) clear(frontierID string) {
	service.updating.Lock()
	defer service.updating.Unlock()

	frontier, ok := service.frontiers.LoadAndDelete(frontierID)
	if ok {
		frontier.(*frontierNservice).service.Close()
	}
	// clear map for edgeID and frontierID
}

func (service *serviceClusterEnd) update() error {
	rsp, err := service.cc.ListFrontiers(context.TODO(), &clusterv1.ListFrontiersRequest{})
	if err != nil {
		service.logger.Errorf("list frontiers err: %s", err)
		return err
	}

	service.updating.Lock()
	defer service.updating.Unlock()

	keeps := []string{}
	removes := []Service{}

	service.frontiers.Range(func(key, value interface{}) bool {
		frontierID := key.(string)
		frontierNservice := value.(*frontierNservice)
		for _, frontier := range rsp.Frontiers {
			if frontierEqual(frontierNservice.frontier, frontier) {
				keeps = append(keeps, frontierID)
				return true
			}
		}
		// out of date frontier
		service.logger.Debugf("frontier: %v needs to be removed", key)
		service.frontiers.Delete(key)
		removes = append(removes, frontierNservice.service)
		return true
	})

	news := []*clusterv1.Frontier{}
FOUND:
	for _, frontier := range rsp.Frontiers {
		for _, keep := range keeps {
			if frontier.FrontierId == keep {
				continue FOUND
			}
		}
		// new frontier
		news = append(news, frontier)
	}

	// aysnc connect and close
	go func() {
		for _, remove := range removes {
			remove.Close()
		}
		for _, new := range news {
			serviceEnd, err := service.newServiceEnd(new.AdvertisedSbAddr)
			if err != nil {
				service.logger.Errorf("new service end err: %s", err)
				continue
			}
			// new frontier
			prev, ok := service.frontiers.Swap(new.FrontierId, &frontierNservice{
				frontier: new,
				service:  serviceEnd,
			})
			if ok {
				prev.(*frontierNservice).service.Close()
			}
		}
	}()
	return nil
}

func frontierEqual(a, b *clusterv1.Frontier) bool {
	return a.AdvertisedSbAddr == b.AdvertisedEbAddr &&
		a.FrontierId == b.FrontierId
}

func (service *serviceClusterEnd) newServiceEnd(addr string) (*serviceEnd, error) {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	}
	serviceEnd, err := newServiceEnd(dialer,
		OptionServiceLog(service.serviceOption.logger),
		OptionServiceDelegate(service.serviceOption.delegate),
		OptionServiceName(service.serviceOption.service),
		OptionServiceReceiveTopics(service.serviceOption.topics),
		OptionServiceTimer(service.serviceOption.tmr))
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			st, err := serviceEnd.AcceptStream()
			if err != nil {
				return
			}
			service.acceptStreamCh <- st
		}
	}()
	go func() {
		for {
			msg, err := serviceEnd.Receive(context.TODO())
			if err != nil {
				return
			}
			service.acceptMsgCh <- msg
		}
	}()

	service.appMtx.RLock()
	defer service.appMtx.RUnlock()

	// rpcs
	for method, rpc := range service.rpcs {
		err = serviceEnd.Register(context.TODO(), method, rpc)
		if err != nil {
			goto ERR
		}
	}

ERR:
	serviceEnd.Close()
	return nil, err
}
