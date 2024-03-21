package repo

// key: serviceID; value: Service
type Service struct {
	Service    string `yaml:"service"`
	FrontierID string `yaml:"frontier_id"`
	Addr       string `yaml:"addr"`
	UpdateTime int64  `yaml:"update_time"`
}
