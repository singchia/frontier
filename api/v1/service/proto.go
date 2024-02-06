package service

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
