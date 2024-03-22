package repo

// key: frontierID; value: Frontier
type Frontier struct {
	FrontierID                 string `yaml:"frontier_id"`
	AdvertisedServiceboundAddr string `yaml:"advertised_sb_addr"`
	AdvertisedEdgeboundAddr    string `yaml:"advertised_eb_addr"`
	EdgeCount                  int    `yaml:"edge_count"`
	ServiceCount               int    `yaml:"service_count"`
}
