package config

// listen related
type CertKey struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

type TLSCallback struct {
	PrivateKeyFile  string   `yaml:"private_key_file" json:"private_key_file"`
	Service         string   `yaml:"service" json:"service"`
	ProviderKeys    []string `yaml:"provider_keys" json:"provider_keys"` // Base64 Ed25519 public keys, provisioned out of band.
	RefreshInterval string   `yaml:"refresh_interval" json:"refresh_interval"`
	Timeout         string   `yaml:"timeout" json:"timeout"`
}

type TLS struct {
	ClientIdentityURIPrefix string      `yaml:"client_identity_uri_prefix,omitempty" json:"client_identity_uri_prefix,omitempty"`
	Source                  string      `yaml:"source,omitempty" json:"source,omitempty"`
	Callback                TLSCallback `yaml:"callback,omitempty" json:"callback,omitempty"`
	Enable                  bool        `yaml:"enable" json:"enable"`
	MTLS                    bool        `yaml:"mtls" json:"mtls"`
	CACerts                 []string    `yaml:"ca_certs" json:"ca_certs"`                         // ca certs paths
	Certs                   []CertKey   `yaml:"certs" json:"certs"`                               // certs paths
	InsecureSkipVerify      bool        `yaml:"insecure_skip_verify" json:"insecure_skip_verify"` // for client use
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
