package repo

// key: serviceID; value: Service
type Service struct {
	Service    string `json:"service"`
	FrontierID string `json:"frontier_id"`
	Addr       string `json:"addr"`
	UpdateTime int64  `json:"update_time"`
}
