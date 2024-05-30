package config

// listen related
type CertKey struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type TLS struct {
	Enable             bool      `yaml:"enable" json:"enable"`
	MTLS               bool      `yaml:"mtls" json:"mtls"`
	CACerts            []string  `yaml:"ca_certs" json:"ca_certs"`                         // ca certs paths
	Certs              []CertKey `yaml:"certs" json:"certs"`                               // certs paths
	InsecureSkipVerify bool      `yaml:"insecure_skip_verify" json:"insecure_skip_verify"` // for client use
}

type Listen struct {
	Network        string `yaml:"network" json:"network"`
	Addr           string `yaml:"addr" json:"addr"`
	AdvertisedAddr string `yaml:"advertised_addr,omitempty" json:"advertised_addr"`
	TLS            TLS    `yaml:"tls,omitempty" json:"tls"`
}

type Dial struct {
	Network        string   `yaml:"network" json:"network"`
	Addrs          []string `yaml:"addrs" json:"addrs"`
	AdvertisedAddr string   `yaml:"advertised_addr,omitempty" json:"advertised_addr"`
	TLS            TLS      `yaml:"tls,omitempty" json:"tls"`
}
