package config

import (
	"flag"
	"io"
	"os"

	armio "github.com/jumboframes/armorigo/io"
	"github.com/singchia/frontier/pkg/config"
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

// for rest and grpc
type ControlPlane struct {
	Listen config.Listen `yaml:"listen"`
}

// TODO tls support
type Redis struct {
	Mode string `yaml:"mode"` // standalone, sentinel or cluster

	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string `yaml:"username,omitempty"`

	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string `yaml:"password,omitempty"`

	// Protocol 2 or 3. Use the version to negotiate RESP version with redis-server.
	// Default is 3.
	Protocol int `yaml:"protocol,omitempty"`

	// ClientName will execute the `CLIENT SETNAME ClientName` command for each conn.
	ClientName string `yaml:"clientname,omitempty"`

	// connection retry settings
	MaxRetries      int `yaml:"max_retries,omitempty"`
	MinRetryBackoff int `yaml:"min_retry_backoff,omitempty"`
	MaxRetryBackoff int `yaml:"max_retry_backoff,omitempty"`
	// connection r/w settings
	DialTimeout  int `yaml:"dial_timeout,omitempty"`
	ReadTimeout  int `yaml:"read_timeout,omitempty"`
	WriteTimeout int `yaml:"write_timeout,omitempty"`
	// connection pool settings
	PoolFIFO         bool   `yaml:"pool_fifo,omitempty"`
	PoolSize         int    `yaml:"pool_size,omitempty"` // applies per cluster node and not for the whole cluster
	PoolTimeout      int    `yaml:"pool_timeout,omitempty"`
	MinIdleConns     int    `yaml:"min_idle_conns,omitempty"`
	MaxIdleConns     int    `yaml:"max_idle,omitempty"`
	MaxActiveConns   int    `yaml:"max_active_conns,omitempty"` // applies per cluster node and not for the whole cluster
	ConnMaxIdleTime  int    `yaml:"conn_max_idle_time,omitempty"`
	ConnMaxLifetime  int    `yaml:"conn_max_life_time,omitempty"`
	DisableIndentity bool   `yaml:"disable_identity,omitempty"` // Disable set-lib on connect. Default is false.
	IdentitySuffix   string `yaml:"identity_suffix,omitempty"`  // Add suffix to client name. Default is empty.

	Standalone struct {
		// The network type, either tcp or unix.
		// Default is tcp.
		Network string `yaml:"network"`
		// host:port address.
		Addr string `yaml:"addr"`
		// CredentialsProvider allows the username and password to be updated
		// before reconnecting. It should return the current username and password.
		DB int `yaml:"db"`
	} `yaml:"standalone,omitempty"`
	Sentinel struct {
		Addrs      []string `yaml:"addrs"`
		MasterName string   `yaml:"master_name"`
		DB         int      `yaml:"db"`

		// route settings
		// Allows routing read-only commands to the closest master or replica node.
		// This option only works with NewFailoverClusterClient.
		RouteByLatency bool `yaml:"route_by_latency,omitempty"`
		// Allows routing read-only commands to the random master or replica node.
		// This option only works with NewFailoverClusterClient.
		RouteRandomly bool `yaml:"route_randomly,omitempty"`
		// Route all commands to replica read-only nodes.
		ReplicaOnly bool `yaml:"replica_only,omitempty"`

		// Use replicas disconnected with master when cannot get connected replicas
		// Now, this option only works in RandomReplicaAddr function.
		UseDisconnectedReplicas bool `yaml:"use_disconnected_replicas,omitempty"`
	} `yaml:"sentinel,omitempty"`
	Cluster struct {
		Addrs []string `yaml:"addrs"`
		// The maximum number of retries before giving up. Command is retried
		// on network errors and MOVED/ASK redirects.
		// Default is 3 retries.
		MaxRedirects int `yaml:"max_redirects,omitempty"`
		// Allows routing read-only commands to the closest master or slave node.
		// It automatically enables ReadOnly.
		RouteByLatency bool `yaml:"route_by_latency,omitempty"`
		// Allows routing read-only commands to the random master or slave node.
		// It automatically enables ReadOnly.
		RouteRandomly bool `yaml:"route_randomly,omitempty"`
	} `yaml:"cluster,omitempty"`
}

type FrontierManager struct {
	Listen     config.Listen `yaml:"listen"`
	Expiration struct {
		ServiceMeta int `yaml:"service_meta"` // service meta expiration in redis, in seconds, default 86400s
		EdgeMeta    int `yaml:"edge_meta"`    // edge meta expiration in redis, in seconds, default 86400s
	} `yaml:"expiration,omitempty"`
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	ControlPlane ControlPlane `yaml:"control_plane"`

	FrontierManager FrontierManager `yaml:"frontier_plane"`

	Redis Redis `yaml:"redis"`
}

func Parse() (*Configuration, error) {
	var (
		argConfigFile         = pflag.String("config", "", "config file, default not configured")
		argDaemonRLimitNofile = pflag.Int("daemon-rlimit-nofile", -1, "SetRLimit for number of file of this daemon, default: -1 means ignore")
		// TODO more command-line args

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
				Addr:   "0.0.0.0:6061",
			},
		},
		ControlPlane: ControlPlane{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30021",
			},
		},
		FrontierManager: FrontierManager{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30022",
			},
		},
		Redis: Redis{
			Mode: "standalone",
		},
	}
	conf.Redis.Standalone.Network = "tcp"
	conf.Redis.Standalone.Addr = "127.0.0.1:6379"
	conf.Redis.Standalone.DB = 0
	conf.FrontierManager.Expiration.EdgeMeta = 30
	conf.FrontierManager.Expiration.ServiceMeta = 30

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
