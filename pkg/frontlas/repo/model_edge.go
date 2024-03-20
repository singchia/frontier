package repo

// key: edgeID; value: Edge
type Edge struct {
	FrontierID string `yaml:"frontier_id"`
	Addr       string `yaml:"addr"`
	UpdateTime int64  `yaml:"update_time"`
}
