package apis

// frontier -> service
// global rpcs
var (
	RPCGetEdgeID   = "get_edge_id"
	RPCEdgeOnline  = "edge_online"
	RPCEdgeOffline = "edge_offline"
)

type OnEdgeOnline struct {
	EdgeID uint64
	Meta   []byte
	Net    string
	Str    string
}

func (online *OnEdgeOnline) Network() string {
	return online.Net
}

func (online *OnEdgeOnline) String() string {
	return online.Str
}

// frontier -> service
type OnEdgeOffline struct {
	EdgeID uint64
	Meta   []byte
	Net    string
	Str    string
}

func (offline *OnEdgeOffline) Network() string {
	return offline.Net
}

func (offline *OnEdgeOffline) String() string {
	return offline.Str
}

// service -> frontier
// meta carried when service inited
type Meta struct {
	Service string
	Topics  []string
}
