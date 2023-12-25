package config

type Configuration struct {
	Daemon struct {
		RLimit struct {
			Enable  bool   `yaml:"enable"`
			NumFile uint64 `yaml:"nofile"`
		} `yaml:"rlimit"`
		PProf struct {
			Enable bool   `yaml:"enable"`
			Addr   string `yaml:"addr"`
		} `yaml:"pprof"`
	} `yaml:"daemon"`

	Clientbound struct {
	}
}

func ParseFlags() *Configuration {

}
