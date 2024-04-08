package frontlas

import (
	"context"
	"encoding/json"
	"net"

	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/utils"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/client"
	"k8s.io/klog/v2"
)

type Informer struct {
	end  geminio.End
	conf *config.Configuration
}

func NewInformer(conf *config.Configuration) (*Informer, error) {
	dial := conf.Frontlas.Dial

	dialer := func() (net.Conn, error) {
		conn, err := utils.Dial(&dial)
		if err != nil {
			klog.Errorf("frontlas new informer, dial err: %s", err)
			return nil, err
		}
		return conn, nil
	}

	// meta
	ins := apis.FrontierInstance{
		FrontierID: conf.Daemon.FrontiesID,
	}
	data, err := json.Marshal(ins)
	if err != nil {
		return nil, err
	}
	opt := client.NewEndOptions()
	opt.SetMeta(data)
	end, err := client.NewRetryEndWithDialer(dialer)
	if err != nil {
		klog.Errorf("frontlas new retry end err: %s", err)
	}
	return &Informer{
		end:  end,
		conf: conf,
	}, nil
}

// edge events
func (informer *Informer) EdgeOnline(edgeID uint64, meta []byte, addr net.Addr) {
	msg := apis.EdgeOnline{
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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
func (informer *Informer) ServiceOnline(serviceID uint64, meta []byte, addr net.Addr) {
	msg := apis.ServiceOnline{
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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

func (informer *Informer) ServiceOffline(serviceID uint64, meta []byte, addr net.Addr) {
	msg := apis.ServiceOffline{
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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

func (informer *Informer) ServiceHeartbeat(serviceID uint64, meta []byte, addr net.Addr) {
	msg := apis.ServiceHeartbeat{
		FrontierID: informer.conf.Daemon.FrontiesID, // TODO emtpy then takes k8s env
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
