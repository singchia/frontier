package repo

// key: frontierID; value: Frontier
type Frontier struct {
	AdvertisedServiceboundAddr string `yaml:"advertised_servicebound_addr"`
	AdvertisedEdgeboundAddr    string `yaml:"advertised_edgebound_addr"`
	EdgeCount                  int    `yaml:"edge_count"`
}
