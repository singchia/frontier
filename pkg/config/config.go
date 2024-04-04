package config

// listen related
type CertKey struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type TLS struct {
	Enable             bool      `yaml:"enable"`
	MTLS               bool      `yaml:"mtls"`
	CACerts            []string  `yaml:"ca_certs"`             // ca certs paths
	Certs              []CertKey `yaml:"certs"`                // certs paths
	InsecureSkipVerify bool      `yaml:"insecure_skip_verify"` // for client use
}

type Listen struct {
	Network string `yaml:"network"`
	Addr    string `yaml:"addr"`
	TLS     TLS    `yaml:"tls"`
}

type Dial struct {
	Network string `yaml:"network"`
	Addr    string `yaml:"addr"`
	TLS     TLS    `yaml:"tls"`
}
