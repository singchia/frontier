package config

import (
	"flag"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

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

type Clientbound struct {
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
	OneOutput        bool   `yaml:"ont_output"`
	SkipLogHeaders   bool   `yaml:"skip_log_headers"`
	StderrThreshold  int32  `yaml:"stderrthreshold"`
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	Clientbound Clientbound `yaml:"clientbound"`

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
