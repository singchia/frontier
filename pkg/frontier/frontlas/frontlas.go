package frontlas

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/singchia/frontier/pkg/apis"
	gconfig "github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"github.com/singchia/go-timer/v2"
	"k8s.io/klog/v2"
)

type Informer struct {
	end  geminio.End
	conf *config.Configuration

	// stats
	edgeCount, serviceCount int32
}

func NewInformer(conf *config.Configuration, tmr timer.Timer) (*Informer, error) {
	dial := conf.Frontlas.Dial

	sbAddr, ebAddr, err := getAdvertisedAddrs(conf.Servicebound.Listen, conf.Edgebound.Listen, dial)
	// meta
	ins := apis.FrontierInstance{
		FrontierID:                 conf.Daemon.FrontierID,
		AdvertisedServiceboundAddr: sbAddr,
		AdvertisedEdgeboundAddr:    ebAddr,
	}
	data, err := json.Marshal(ins)
	if err != nil {
		return nil, err
	}
	dialer := func() (net.Conn, error) {
		conn, err := utils.Dial(&dial)
		if err != nil {
			klog.Errorf("frontlas new informer, dial err: %s", err)
			return nil, err
		}
		return conn, nil
	}
	opt := client.NewRetryEndOptions()
	opt.SetMeta(data)
	end, err := client.NewRetryEndWithDialer(dialer, opt)
	if err != nil {
		klog.Errorf("frontlas new retry end err: %s", err)
		return nil, err
	}
	informer := &Informer{
		end:  end,
		conf: conf,
	}
	// metrics
	if conf.Frontlas.Metrics.Enable {
		go func() {
			ticker := tmr.Add(time.Duration(conf.Frontlas.Metrics.Interval)*time.Second, timer.WithCyclically())
			for {
				_, ok := <-ticker.C()
				if !ok {
					return
				}
				stats := &apis.FrontierStats{
					FrontierID:   conf.Daemon.FrontierID,
					EdgeCount:    int(atomic.LoadInt32(&informer.edgeCount)),
					ServiceCount: int(atomic.LoadInt32(&informer.serviceCount)),
				}
				data, err := json.Marshal(stats)
				if err != nil {
					klog.Errorf("frontier upload frontlas stats, json marshal err: %s", err)
					continue
				}
				req := informer.end.NewRequest(data)
				_, err = informer.end.Call(context.TODO(), apis.RPCFrontierStats, req)
				if err != nil {
					klog.Errorf("fronter upload frontlas stats, call err: %s", err)
					continue
				}
			}
		}()
	}
	return &Informer{
		end:  end,
		conf: conf,
	}, nil
}

// the advertised addrs should be specified
func getAdvertisedAddrs(sblisten, eblisten gconfig.Listen, dial gconfig.Dial) (string, string, error) {
	var (
		once = &sync.Once{}
		host string
		err  error
	)
	getDefaultRouteHost := func() (string, error) {
		once.Do(func() {
			ip, rerr := utils.GetDefaultRouteIP(dial.Network, dial.Addr)
			if err != nil {
				err = rerr
				return
			}
			host = ip.String()
		})
		return host, err
	}
	// advertised ip address
	sbAddr := sblisten.AdvertisedAddr
	ebAddr := eblisten.AdvertisedAddr
	// TODO if advertised addr empty and in k8s, get addr from PodIP
	// if PodIP is empty, then use conf.Servicebound.Listen.Addr instead
	if sbAddr == "" {
		sbAddr = sblisten.Addr
	}
	if ebAddr == "" {
		ebAddr = eblisten.Addr
	}
	sbhost, sbport, err := net.SplitHostPort(sbAddr)
	if err != nil {
		return "", "", err
	}
	if net.ParseIP(sbhost).IsUnspecified() {
		sbhost, err = getDefaultRouteHost()
		if err != nil {
			return "", "", err
		}
	}
	sbAddr = net.JoinHostPort(sbhost, sbport)

	ebhost, ebport, err := net.SplitHostPort(ebAddr)
	if err != nil {
		return "", "", err
	}
	if net.ParseIP(ebhost).IsUnspecified() {
		ebhost, err = getDefaultRouteHost()
		if err != nil {
			return "", "", err
		}
	}
	ebAddr = net.JoinHostPort(ebhost, ebport)
	return sbAddr, ebAddr, nil
}

// edge events
func (informer *Informer) EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) {
	msg := apis.EdgeOnline{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		EdgeID:     edgeID,
		Addr:       addr.String(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform edge online, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCEdgeOnline, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform edge online, call rpc err: %s", err)
	}
}

func (informer *Informer) EdgeOffline(edgeID uint64, meta []byte, addr net.Addr) {
	msg := apis.EdgeOffline{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		EdgeID:     edgeID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform edge offline, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCEdgeOffline, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform edge offline, call rpc err: %s", err)
	}
}

func (informer *Informer) EdgeHeartbeat(edgeID uint64, meta []byte, addr net.Addr) {
	msg := apis.EdgeHeartbeat{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		EdgeID:     edgeID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform edge heartbeat, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCEdgeHeartbeat, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform edge heartbeat, call rpc err: %s", err)
	}
}

// service events
func (informer *Informer) ServiceOnline(serviceID uint64, meta string, addr net.Addr) {
	msg := apis.ServiceOnline{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		ServiceID:  serviceID,
		Service:    string(meta),
		Addr:       addr.String(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform service online, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCServiceOnline, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform service online, call rpc err: %s", err)
	}
}

func (informer *Informer) ServiceOffline(serviceID uint64, meta string, addr net.Addr) {
	msg := apis.ServiceOffline{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		ServiceID:  serviceID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform service offline, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCServiceOffline, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform service offline, call rpc err: %s", err)
	}
}

func (informer *Informer) ServiceHeartbeat(serviceID uint64, meta string, addr net.Addr) {
	msg := apis.ServiceHeartbeat{
		FrontierID: informer.conf.Daemon.FrontierID, // TODO emtpy then takes k8s env
		ServiceID:  serviceID,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("frontlas inform service heartbeat, json marshal err: %s", err)
		return
	}
	_, err = informer.end.Call(context.TODO(), apis.RPCServiceHeartbeat, informer.end.NewRequest(data))
	if err != nil {
		klog.Errorf("frontlas inform service heartbeat, call rpc err: %s", err)
	}
}

func (informer *Informer) SetEdgeCount(count int) {
	count32 := int32(count)
	atomic.StoreInt32(&informer.edgeCount, count32)
}

func (informer *Informer) SetServiceCount(count int) {
	count32 := int32(count)
	atomic.StoreInt32(&informer.serviceCount, count32)
}
