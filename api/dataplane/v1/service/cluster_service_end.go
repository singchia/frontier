package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"

	mapset "github.com/deckarep/golang-set/v2"
	clusterv1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/mapmap"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/delegate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type frontierNend struct {
	frontier *clusterv1.Frontier
	end      *serviceEnd
}

type clusterServiceEnd struct {
	*delegate.UnimplementedDelegate
	cc clusterv1.ClusterServiceClient

	edgefrontiers *mapmap.BiMap // bidirectional edgeID and frontierID
	frontiers     sync.Map      // key: frontierID; value: frontierNservice

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

func newclusterServiceEnd(addr string, opts ...ServiceOption) (*clusterServiceEnd, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	cc := clusterv1.NewClusterServiceClient(conn)

	end := &clusterServiceEnd{
		cc:             cc,
		serviceOption:  &serviceOption{},
		rpcs:           map[string]geminio.RPC{},
		topics:         mapset.NewSet[string](),
		edgefrontiers:  mapmap.NewBiMap(),
		acceptStreamCh: make(chan geminio.Stream, 128),
		acceptMsgCh:    make(chan geminio.Message, 128),
		closed:         make(chan struct{}),
	}
	end.serviceOption.delegate = end

	for _, opt := range opts {
		opt(end.serviceOption)
	}
	if end.serviceOption.logger == nil {
		end.serviceOption.logger = armlog.DefaultLog
	}
	err = end.update()
	if err != nil {
		return nil, err
	}
	go end.start()
	return end, nil
}

func (end *clusterServiceEnd) start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := end.update()
			if err != nil {
				end.logger.Warnf("cluster update err: %s", err)
				continue
			}
		case <-end.closed:
			return
		}
	}
}

func (end *clusterServiceEnd) clear(frontierID string) {
	end.updating.Lock()
	defer end.updating.Unlock()

	frontier, ok := end.frontiers.LoadAndDelete(frontierID)
	if ok {
		frontier.(*frontierNend).end.Close()
	}
	// clear map for edgeID and frontierID
	end.edgefrontiers.DelValue(frontierID)
}

func (end *clusterServiceEnd) update() error {
	rsp, err := end.cc.ListFrontiers(context.TODO(), &clusterv1.ListFrontiersRequest{})
	if err != nil {
		end.logger.Errorf("list frontiers err: %s", err)
		return err
	}

	keeps := []string{}
	removes := []*frontierNend{}

	end.frontiers.Range(func(key, value interface{}) bool {
		frontierID := key.(string)
		frontierNend := value.(*frontierNend)
		for _, frontier := range rsp.Frontiers {
			if frontierEqual(frontierNend.frontier, frontier) {
				keeps = append(keeps, frontierID)
				return true
			}
		}
		// out of date frontier
		end.logger.Debugf("frontier: %v needs to be removed", key)
		end.frontiers.Delete(key)
		removes = append(removes, frontierNend)
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
			remove.end.Close()
			// clear unavaiable frontier and it's edges
			end.edgefrontiers.DelValue(remove.frontier.FrontierId)
		}
		for _, new := range news {
			serviceEnd, err := end.newServiceEnd(new.AdvertisedSbAddr)
			if err != nil {
				end.logger.Errorf("new service end err: %s", err)
				continue
			}
			end.logger.Debugf("new service end succeed, frontierID: %s, addr: %s", new.FrontierId, new.AdvertisedSbAddr)
			// new frontier
			prev, ok := end.frontiers.Swap(new.FrontierId, &frontierNend{
				frontier: new,
				end:      serviceEnd,
			})
			if ok {
				prev.(*frontierNend).end.Close()
			}
		}
	}()
	return nil
}

func (end *clusterServiceEnd) lookup(edgeID uint64) (string, *serviceEnd, error) {
	var (
		frontier   *clusterv1.Frontier
		serviceEnd *serviceEnd
		err        error
	)
	frontierID, ok := end.edgefrontiers.GetValue(edgeID)
	// get or set edgeID to frontierID map
	if !ok {
		rsp, err := end.cc.GetFrontierByEdge(context.TODO(), &clusterv1.GetFrontierByEdgeIDRequest{
			EdgeId: edgeID,
		})
		if err != nil {
			end.logger.Errorf("get frontier by edge: %d err: %s", edgeID, err)
			return "", nil, err
		}
		frontier = rsp.Fontier
		frontierID = frontier.FrontierId
		end.edgefrontiers.Set(edgeID, frontierID)
	}

	fe, ok := end.frontiers.Load(frontierID)
	if !ok {
		serviceEnd, err = end.newServiceEnd(frontier.AdvertisedSbAddr)
		if err != nil {
			end.logger.Errorf("new service end err: %s while lookup", err)
			return "", nil, err
		}
		found, ok := end.frontiers.Swap(frontierID, &frontierNend{
			frontier: frontier,
			end:      serviceEnd,
		})
		if ok {
			found.(*frontierNend).end.Close()
		}
		end.logger.Debugf("new service end succeed, addr: %s", frontier.AdvertisedSbAddr)
	} else {
		serviceEnd = fe.(*frontierNend).end
	}
	return frontierID.(string), serviceEnd, nil
}

func (end *clusterServiceEnd) pickone() *serviceEnd {
	var serviceEnd *serviceEnd
	end.frontiers.Range(func(_, value interface{}) bool {
		// return first one
		serviceEnd = value.(*frontierNend).end
		return false
	})
	return serviceEnd
}

func frontierEqual(a, b *clusterv1.Frontier) bool {
	return a.AdvertisedSbAddr == b.AdvertisedSbAddr &&
		a.FrontierId == b.FrontierId
}

func (end *clusterServiceEnd) newServiceEnd(addr string) (*serviceEnd, error) {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	}
	serviceEnd, err := newServiceEnd(dialer,
		OptionServiceLog(end.serviceOption.logger),
		OptionServiceDelegate(end.serviceOption.delegate),
		OptionServiceName(end.serviceOption.service),
		OptionServiceReceiveTopics(end.serviceOption.topics),
		OptionServiceTimer(end.serviceOption.tmr),
		OptionServiceID(end.serviceOption.serviceID))
	if err != nil {
		return nil, err
	}
	if end.serviceOption.serviceID == 0 {
		// record serviceID for later using
		end.serviceOption.serviceID = serviceEnd.ClientID()
	}
	go func() {
		for {
			st, err := serviceEnd.AcceptStream()
			if err != nil {
				return
			}
			end.acceptStreamCh <- st
		}
	}()
	go func() {
		for {
			msg, err := serviceEnd.Receive(context.TODO())
			if err != nil {
				return
			}
			end.acceptMsgCh <- msg
		}
	}()

	end.appMtx.RLock()
	defer end.appMtx.RUnlock()

	// rpcs
	for method, rpc := range end.rpcs {
		err = serviceEnd.Register(context.TODO(), method, rpc)
		if err != nil {
			goto ERR
		}
	}
	return serviceEnd, nil

ERR:
	serviceEnd.Close()
	return nil, err
}

// multiplexer
func (end *clusterServiceEnd) AcceptStream() (geminio.Stream, error) {
	st, ok := <-end.acceptStreamCh
	if !ok {
		return nil, io.EOF
	}
	return st, nil
}

func (end *clusterServiceEnd) OpenStream(ctx context.Context, edgeID uint64) (geminio.Stream, error) {
	frontierID, serviceEnd, err := end.lookup(edgeID)
	if err != nil {
		return nil, err
	}
	stream, err := serviceEnd.OpenStream(ctx, edgeID)
	if err != nil {
		end.clear(frontierID)
		return stream, err
	}
	return stream, nil
}

func (end *clusterServiceEnd) ListStreams() []geminio.Stream {
	streams := []geminio.Stream{}
	end.frontiers.Range(func(_, value interface{}) bool {
		sts := value.(*frontierNend).end.ListStreams()
		if sts != nil {
			streams = append(streams, sts...)
		}
		return true
	})
	return streams
}

// Messager
func (end *clusterServiceEnd) NewMessage(data []byte) geminio.Message {
	serviceEnd := end.pickone()
	if serviceEnd == nil {
		return nil
	}
	return serviceEnd.NewMessage(data)
}

func (end *clusterServiceEnd) Publish(ctx context.Context, edgeID uint64, msg geminio.Message) error {
	fronterID, serviceEnd, err := end.lookup(edgeID)
	if err != nil {
		return err
	}
	err = serviceEnd.Publish(ctx, edgeID, msg)
	if err != nil {
		end.clear(fronterID)
		return err
	}
	return nil
}

func (end *clusterServiceEnd) PublishAsync(ctx context.Context, edgeID uint64, msg geminio.Message, ch chan *geminio.Publish) (*geminio.Publish, error) {
	fronterID, serviceEnd, err := end.lookup(edgeID)
	if err != nil {
		return nil, err
	}
	pub, err := serviceEnd.PublishAsync(ctx, edgeID, msg, ch)
	if err != nil {
		end.clear(fronterID)
		return nil, err
	}
	return pub, err
}

func (end *clusterServiceEnd) Receive(ctx context.Context) (geminio.Message, error) {
	msg, ok := <-end.acceptMsgCh
	if !ok {
		return nil, io.EOF
	}
	return msg, nil
}

// RPCer
func (end *clusterServiceEnd) NewRequest(data []byte) geminio.Request {
	serviceEnd := end.pickone()
	if serviceEnd == nil {
		return nil
	}
	return serviceEnd.NewRequest(data)
}

func (end *clusterServiceEnd) Call(ctx context.Context, edgeID uint64, method string, req geminio.Request) (geminio.Response, error) {
	fronterID, serviceEnd, err := end.lookup(edgeID)
	if err != nil {
		return nil, err
	}
	rsp, err := serviceEnd.Call(ctx, edgeID, method, req)
	if err != nil {
		end.clear(fronterID)
		return nil, err
	}
	return rsp, nil
}

func (end *clusterServiceEnd) CallAsync(ctx context.Context, edgeID uint64, method string, req geminio.Request, ch chan *geminio.Call) (*geminio.Call, error) {
	fronterID, serviceEnd, err := end.lookup(edgeID)
	if err != nil {
		return nil, err
	}
	call, err := serviceEnd.CallAsync(ctx, edgeID, method, req, ch)
	if err != nil {
		end.clear(fronterID)
		return nil, err
	}
	return call, nil
}

func (end *clusterServiceEnd) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	end.appMtx.Lock()
	end.rpcs[method] = rpc
	end.appMtx.Unlock()

	var (
		err error
	)
	// TODO optimize it
	end.frontiers.Range(func(key, value interface{}) bool {
		err = value.(*frontierNend).end.Register(ctx, method, rpc)
		if err != nil {
			return false
		}
		return true
	})
	return err
}

// net.Listener
func (end *clusterServiceEnd) Accept() (net.Conn, error) {
	st, ok := <-end.acceptStreamCh
	if !ok {
		return nil, io.EOF
	}
	return st, nil
}

func (end *clusterServiceEnd) Network() string {
	return "tcp"
}

func (end *clusterServiceEnd) String() string {
	addrs := []string{}
	end.frontiers.Range(func(key, value interface{}) bool {
		addr := value.(*frontierNend).end.Addr().String()
		addrs = append(addrs, addr)
		return true
	})
	return strings.Join(addrs, ";")
}

func (end *clusterServiceEnd) Addr() net.Addr {
	return end
}

// close
func (end *clusterServiceEnd) Close() error {
	close(end.closed)
	close(end.acceptMsgCh)
	close(end.acceptStreamCh)

	var (
		err error
	)
	end.frontiers.Range(func(key, value interface{}) bool {
		closeerr := value.(*frontierNend).end.Close()
		if closeerr != nil {
			err = closeerr
		}
		return true
	})
	return err
}

// Control Register
func (end *clusterServiceEnd) RegisterGetEdgeID(ctx context.Context, getEdgeID GetEdgeID) error {
	// we call Register of clusterServiceEnd because we need rpcs record getEdgeID
	// and also Register getEdgeID to all frontier
	return end.Register(ctx, apis.RPCGetEdgeID, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		id, err := getEdgeID(req.Data())
		if err != nil {
			// we just deliver the err back
			// get ID err will force close the edge unless EdgeIDAllocWhenNoIDServiceOn is configured
			rsp.SetError(err)
			return
		}
		hex := make([]byte, 8)
		binary.BigEndian.PutUint64(hex, id)
		rsp.SetData(hex)
	})
}

func (end *clusterServiceEnd) RegisterEdgeOnline(ctx context.Context, edgeOnline EdgeOnline) error {
	return end.Register(
		ctx, apis.RPCEdgeOnline, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
			on := &apis.OnEdgeOnline{}
			err := json.Unmarshal(req.Data(), on)
			if err != nil {
				// shouldn't be here
				rsp.SetError(err)
				return
			}
			err = edgeOnline(on.EdgeID, on.Meta, on)
			if err != nil {
				// online err will force close the edge
				rsp.SetError(err)
				return
			}
			// if allowed, the edge will continue the connection
		})
}

func (end *clusterServiceEnd) RegisterEdgeOffline(ctx context.Context, edgeOffline EdgeOffline) error {
	return end.Register(ctx, apis.RPCEdgeOffline, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		off := &apis.OnEdgeOffline{}
		err := json.Unmarshal(req.Data(), off)
		if err != nil {
			// shouldn't be here
			rsp.SetError(err)
			return
		}
		err = edgeOffline(off.EdgeID, off.Meta, off)
		if err != nil {
			rsp.SetError(err)
			return
		}
	})
}
