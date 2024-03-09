package config

import (
	"flag"
	"fmt"
	"io"
	"os"

	armio "github.com/jumboframes/armorigo/io"
	"github.com/jumboframes/armorigo/log"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// daemon related
type RLimit struct {
	Enable  bool `yaml:"enable"`
	NumFile int  `yaml:"nofile"`
}

type PProf struct {
	Enable         bool   `yaml:"enable"`
	Addr           string `yaml:"addr"`
	CPUProfileRate int    `yaml:"cpu_profile_rate"`
}

type Daemon struct {
	RLimit RLimit `yaml:"rlimit"`
	PProf  PProf  `yaml:"pprof"`
}

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

// edgebound
// Bypass is for the lagecy gateway, this will split
type Bypass struct {
	Enable  bool   `yaml:"enable"`
	Network string `yaml:"network"`
	Addr    string `yaml:"addr"` // addr to dial
	TLS     TLS    `yaml:"tls"`  // certs to dial or ca to auth
}
type Edgebound struct {
	Listen Listen `yaml:"listen"`
	Bypass Bypass `yaml:"bypass"`
	// alloc edgeID when no get_id function online
	EdgeIDAllocWhenNoIDServiceOn bool `yaml:"edgeid_alloc_when_no_idservice_on"`
}

// servicebound
type Servicebound struct {
	Listen Listen `yaml:"listen"`
}

type ControlPlane struct {
	Listen Listen `yaml:"listen"`
}

// message queue
type MQ struct {
	BroadCast bool `yaml:"broadcast"`
}

// exchange
type Exchange struct{}

type Dao struct {
	Debug bool `yaml:"debug"`
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	Edgebound Edgebound `yaml:"edgebound"`

	Servicebound Servicebound `yaml:"servicebound"`

	ControlPlane ControlPlane `yaml:"controlplane"`

	Dao Dao `yaml:"dao"`
}

// Configuration accepts config file and command-line, and command-line is more privileged.
func Parse() (*Configuration, error) {
	var (
		argConfigFile         = pflag.String("config", "", "config file, default not configured")
		argArmorigoLogLevel   = pflag.String("loglevel", "info", "log level for armorigo log")
		argDaemonRLimitNofile = pflag.Int("daemon-rlimit-nofile", -1, "SetRLimit for number of file of this daemon, default: -1 means ignore")

		config *Configuration
	)
	pflag.Lookup("daemon-rlimit-nofile").NoOptDefVal = "1048576"

	// set klog
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// sync the glog and klog flags.
	pflag.CommandLine.VisitAll(func(f1 *pflag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			if err := f2.Value.Set(value); err != nil {
				klog.Fatal(err, "failed to set flag")
				return
			}
		}
	})

	pflag.CommandLine.AddGoFlagSet(klogFlags)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	// armorigo log
	level, err := log.ParseLevel(*argArmorigoLogLevel)
	if err != nil {
		fmt.Println("parse log level err:", err)
		return nil, err
	}
	log.SetLevel(level)
	log.SetOutput(os.Stdout)

	// config file
	if *argConfigFile != "" {
		// TODO the command-line is prior to config file
		data, err := os.ReadFile(*argConfigFile)
		if err != nil {
			return nil, err
		}
		config = &Configuration{}
		if err = yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	if config == nil {
		config = &Configuration{}
	}
	// daemon
	config.Daemon.RLimit.NumFile = *argDaemonRLimitNofile
	if config.Daemon.PProf.CPUProfileRate == 0 {
		config.Daemon.PProf.CPUProfileRate = 10000
	}
	return config, nil
}

func genDefaultConfig(writer io.Writer) error {
	conf := &Configuration{
		Daemon: Daemon{
			RLimit: RLimit{
				NumFile: 1024,
			},
			PProf: PProf{
				Enable: true,
				Addr:   "0.0.0.0:6060",
			},
		},
		ControlPlane: ControlPlane{
			Listen: Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2430",
			},
		},
		Servicebound: Servicebound{
			Listen: Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2431",
				TLS: TLS{
					Enable: false,
					MTLS:   false,
					CACerts: []string{
						"ca1.cert",
						"ca2.cert",
					},
					Certs: []CertKey{
						{
							Cert: "servicebound.cert",
							Key:  "servicebound.key",
						},
					},
				},
			},
		},
		Edgebound: Edgebound{
			Listen: Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2432",
				TLS: TLS{
					Enable: false,
					MTLS:   false,
					CACerts: []string{
						"ca1.cert",
						"ca2.cert",
					},
					Certs: []CertKey{
						{
							Cert: "edgebound.cert",
							Key:  "edgebound.key",
						},
					},
				},
			},
			EdgeIDAllocWhenNoIDServiceOn: true,
			Bypass: Bypass{
				Enable:  false,
				Network: "tcp",
				Addr:    "192.168.1.10:8443",
				TLS: TLS{
					Enable: true,
					MTLS:   true,
					CACerts: []string{
						"ca1.cert",
					},
					Certs: []CertKey{
						{
							Cert: "frontier.cert",
							Key:  "frontier.key",
						},
					},
				},
			},
		},
		Dao: Dao{
			Debug: false,
		},
	}
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	_, err = armio.WriteAll(data, writer)
	if err != nil {
		return err
	}
	return nil
}
