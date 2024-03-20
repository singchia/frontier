package apis

type Instance struct {
	InstanceID string `yaml:"instance_id"`
	// in k8s, it should be podIP:port
	AdvertisedServiceboundAddr string `yaml:"advertised_servicebound_addr"`
	// in k8s, it should be NodeportIP:port
	AdvertisedEdgeboundAddr string `yaml:"advertised_edgebound_addr"`
}
