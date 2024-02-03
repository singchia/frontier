package config

import (
	"flag"
	"io"
	"os"
	"strconv"

	armio "github.com/jumboframes/armorigo/io"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// daemon related
type RLimit struct {
	NumFile int `yaml:"nofile"`
}

type PProf struct {
	Addr string `yaml:"addr"`
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

type Log struct {
	LogDir           string `yaml:"log_dir"`
	LogFile          string `yaml:"log_file"`
	LogFileMaxSizeMB uint64 `yaml:"log_file_max_size"`
	ToStderr         bool   `yaml:"logtostderr"`
	AlsoToStderr     bool   `yaml:"alsologtostderr"`
	Verbosity        int32  `yaml:"verbosity"`
	AddDirHeader     bool   `yaml:"add_dir_header"`
	SkipHeaders      bool   `yaml:"skip_headers"`
	OneOutput        bool   `yaml:"one_output"`
	SkipLogHeaders   bool   `yaml:"skip_log_headers"`
	StderrThreshold  int32  `yaml:"stderrthreshold"`
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	Edgebound Edgebound `yaml:"edgebound"`

	Servicebound Servicebound `yaml:"servicebound"`

	Log Log `yaml:"log"`
}

// Configuration accepts config file and command-line, and command-line is more privileged.
func ParseFlags() (*Configuration, error) {
	var (
		argConfigFile         = pflag.String("config", "", "config file, default not configured")
		argDaemonRLimitNofile = pflag.Int("daemon-rlimit-nofile", -1, "SetRLimit for number of file of this daemon, default: -1 means ignore")

		config *Configuration
	)
	pflag.Lookup("daemon-rlimit-nofile").NoOptDefVal = "1048576"

	// config file
	if *argConfigFile != "" {
		data, err := os.ReadFile(*argConfigFile)
		if err != nil {
			return nil, err
		}
		config = &Configuration{}
		if err = yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	// set klog
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Set("log_dir", config.Log.LogDir)
	klogFlags.Set("log_file", config.Log.LogFile)
	klogFlags.Set("log_file_max_file", strconv.FormatUint(config.Log.LogFileMaxSizeMB, 10))
	klogFlags.Set("logtostderr", strconv.FormatBool(config.Log.ToStderr))
	klogFlags.Set("alsologtostderr", strconv.FormatBool(config.Log.AlsoToStderr))
	klogFlags.Set("verbosity", strconv.FormatInt(int64(config.Log.Verbosity), 10))
	klogFlags.Set("add_dir_header", strconv.FormatBool(config.Log.AddDirHeader))
	klogFlags.Set("skip_headers", strconv.FormatBool(config.Log.SkipHeaders))
	klogFlags.Set("one_output", strconv.FormatBool(config.Log.OneOutput))
	klogFlags.Set("skip_log_headers", strconv.FormatBool(config.Log.SkipLogHeaders))
	klogFlags.Set("stderrthreshold", strconv.FormatInt(int64(config.Log.StderrThreshold), 10))

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

	if config == nil {
		config = &Configuration{}
	}
	config.Daemon.RLimit.NumFile = *argDaemonRLimitNofile

	return config, nil
}

func genDefaultConfig(writer io.Writer) error {
	conf := &Configuration{
		Daemon: Daemon{
			RLimit: RLimit{
				NumFile: 1024,
			},
			PProf: PProf{
				Addr: "0.0.0.0:6060",
			},
		},
		Edgebound: Edgebound{
			Listen: Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2432",
				TLS: TLS{
					Enable: true,
					MTLS:   true,
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
				Enable:  true,
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
		Servicebound: Servicebound{
			Listen: Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2431",
				TLS: TLS{
					Enable: true,
					MTLS:   true,
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
		Log: Log{
			LogDir:           "/app/log",
			LogFile:          "frontier.log",
			LogFileMaxSizeMB: 100,
			ToStderr:         false,
			AlsoToStderr:     false,
			Verbosity:        4,
			AddDirHeader:     true,
			SkipHeaders:      true,
			OneOutput:        true,
			SkipLogHeaders:   true,
			StderrThreshold:  1024,
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