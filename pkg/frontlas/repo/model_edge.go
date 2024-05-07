package repo

// key: edgeID; value: Edge
type Edge struct {
	FrontierID string `json:"frontier_id"`
	Addr       string `json:"addr"`
	UpdateTime int64  `json:"update_time"`
}
