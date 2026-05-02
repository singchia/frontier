package config

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	armio "github.com/jumboframes/armorigo/io"
	"github.com/singchia/frontier/pkg/config"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// daemon related
type RLimit struct {
	Enable  bool `yaml:"enable" json:"enable"`
	NumFile int  `yaml:"nofile" json:"num_file"`
}

type PProf struct {
	Enable         bool   `yaml:"enable" json:"enable"`
	Addr           string `yaml:"addr" json:"addr"`
	CPUProfileRate int    `yaml:"cpu_profile_rate" json:"cpu_profile_rate"`
}

type Daemon struct {
	RLimit RLimit `yaml:"rlimit" json:"rlimit"`
	PProf  PProf  `yaml:"pprof" json:"pprof"`
}

// for rest and grpc
type ControlPlane struct {
	Listen config.Listen `yaml:"listen" json:"listen"`
}

// TODO tls support
type Redis struct {
	Mode string `yaml:"mode" json:"mode"` // standalone, sentinel or cluster

	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string `yaml:"username,omitempty" json:"username"`

	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string `yaml:"password,omitempty" json:"password"`

	// Protocol 2 or 3. Use the version to negotiate RESP version with redis-server.
	// Default is 3.
	Protocol int `yaml:"protocol,omitempty" json:"protocol"`

	// ClientName will execute the `CLIENT SETNAME ClientName` command for each conn.
	ClientName string `yaml:"clientname,omitempty" json:"client_name"`

	// connection retry settings
	MaxRetries      int `yaml:"max_retries,omitempty" json:"max_retries"`
	MinRetryBackoff int `yaml:"min_retry_backoff,omitempty" json:"min_retry_backoff"`
	MaxRetryBackoff int `yaml:"max_retry_backoff,omitempty" json:"max_retry_backoff"`
	// connection r/w settings
	DialTimeout  int `yaml:"dial_timeout,omitempty" json:"dial_timeout"`
	ReadTimeout  int `yaml:"read_timeout,omitempty" json:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout,omitempty" json:"write_timeout"`
	// connection pool settings
	PoolFIFO         bool   `yaml:"pool_fifo,omitempty" json:"pool_fifo"`
	PoolSize         int    `yaml:"pool_size,omitempty" json:"pool_size"` // applies per cluster node and not for the whole cluster
	PoolTimeout      int    `yaml:"pool_timeout,omitempty" json:"pool_timeout"`
	MinIdleConns     int    `yaml:"min_idle_conns,omitempty" json:"min_idle_conns"`
	MaxIdleConns     int    `yaml:"max_idle,omitempty" json:"max_idle_conns"`
	MaxActiveConns   int    `yaml:"max_active_conns,omitempty" json:"max_active_conns"` // applies per cluster node and not for the whole cluster
	ConnMaxIdleTime  int    `yaml:"conn_max_idle_time,omitempty" json:"conn_max_idle_time"`
	ConnMaxLifetime  int    `yaml:"conn_max_life_time,omitempty" json:"conn_max_lifetime"`
	DisableIndentity bool   `yaml:"disable_identity,omitempty" json:"disable_indentity"` // Disable set-lib on connect. Default is false.
	IdentitySuffix   string `yaml:"identity_suffix,omitempty" json:"identity_suffix"`    // Add suffix to client name. Default is empty.

	Standalone struct {
		// The network type, either tcp or unix.
		// Default is tcp.
		Network string `yaml:"network" json:"network"`
		// host:port address.
		Addr string `yaml:"addr" json:"addr"`
		// CredentialsProvider allows the username and password to be updated
		// before reconnecting. It should return the current username and password.
		DB int `yaml:"db" json:"db"`
	} `yaml:"standalone,omitempty" json:"standalone"`
	Sentinel struct {
		Addrs      []string `yaml:"addrs" json:"addrs"`
		MasterName string   `yaml:"master_name" json:"master_name"`
		DB         int      `yaml:"db" json:"db"`

		// route settings
		// Allows routing read-only commands to the closest master or replica node.
		// This option only works with NewFailoverClusterClient.
		RouteByLatency bool `yaml:"route_by_latency,omitempty" json:"route_by_latency"`
		// Allows routing read-only commands to the random master or replica node.
		// This option only works with NewFailoverClusterClient.
		RouteRandomly bool `yaml:"route_randomly,omitempty" json:"route_randomly"`
		// Route all commands to replica read-only nodes.
		ReplicaOnly bool `yaml:"replica_only,omitempty" json:"replica_only"`

		// Use replicas disconnected with master when cannot get connected replicas
		// Now, this option only works in RandomReplicaAddr function.
		UseDisconnectedReplicas bool `yaml:"use_disconnected_replicas,omitempty" json:"use_disconnected_replicas"`
	} `yaml:"sentinel,omitempty" json:"sentinel"`
	Cluster struct {
		Addrs []string `yaml:"addrs" json:"addrs"`
		// The maximum number of retries before giving up. Command is retried
		// on network errors and MOVED/ASK redirects.
		// Default is 3 retries.
		MaxRedirects int `yaml:"max_redirects,omitempty" json:"max_redirects"`
		// Allows routing read-only commands to the closest master or slave node.
		// It automatically enables ReadOnly.
		RouteByLatency bool `yaml:"route_by_latency,omitempty" json:"route_by_latency"`
		// Allows routing read-only commands to the random master or slave node.
		// It automatically enables ReadOnly.
		RouteRandomly bool `yaml:"route_randomly,omitempty" json:"route_randomly"`
	} `yaml:"cluster,omitempty" json:"cluster"`
}

type FrontierManager struct {
	Listen     config.Listen `yaml:"listen" json:"listen"`
	Expiration struct {
		ServiceMeta int `yaml:"service_meta" json:"service_meta"` // service meta expiration in redis, in seconds, default 86400s
		EdgeMeta    int `yaml:"edge_meta" json:"edge_meta"`       // edge meta expiration in redis, in seconds, default 86400s
	} `yaml:"expiration,omitempty" json:"expiration"`
}

// Observability 控制 /healthz、/readyz、/metrics HTTP 端点。
// 注：原有 /cluster/v1/health gRPC-gateway 端点保留兼容。
type Observability struct {
	Enable bool   `yaml:"enable" json:"enable"`
	Addr   string `yaml:"addr" json:"addr"` // 默认 0.0.0.0:9092
}

type Configuration struct {
	Log config.Log `yaml:"log,omitempty" json:"log"`

	Daemon Daemon `yaml:"daemon" json:"daemon"`

	ControlPlane ControlPlane `yaml:"control_plane" json:"control_plane"`

	FrontierManager FrontierManager `yaml:"frontier_plane" json:"frontier_manager"`

	Redis Redis `yaml:"redis" json:"redis"`

	Observability Observability `yaml:"observability,omitempty" json:"observability"`
}

func Parse() (*Configuration, error) {
	var (
		argConfigFile         = pflag.String("config", "", "config file, default not configured")
		argLogLevel           = pflag.String("loglevel", "", "log level: debug, info, warn, error")
		argLogOutput          = pflag.String("log-output", "", "log output: stdout, stderr, file, both")
		argLogFormat          = pflag.String("log-format", "", "log format: text, json")
		argLogFile            = pflag.String("log-file", "", "log file path, used when output is file or both")
		argDaemonRLimitNofile = pflag.Int("daemon-rlimit-nofile", -1, "SetRLimit for number of file of this daemon, default: -1 means ignore")
		// TODO more command-line args

		conf *Configuration
	)
	pflag.Lookup("daemon-rlimit-nofile").NoOptDefVal = "1048576"
	pflag.Parse()

	// config file
	if *argConfigFile != "" {
		data, err := os.ReadFile(*argConfigFile)
		if err != nil {
			return nil, err
		}
		conf = &Configuration{}
		if err = yaml.Unmarshal(data, conf); err != nil {
			return nil, err
		}
	}

	if conf == nil {
		conf = &Configuration{}
	}

	// env overrides for log (priority: flag > env > yaml > default)
	config.ApplyLogEnvOverrides(&conf.Log)

	// flag overrides for log (highest priority)
	if *argLogLevel != "" {
		conf.Log.Level = *argLogLevel
	}
	if *argLogOutput != "" {
		conf.Log.Output = *argLogOutput
	}
	if *argLogFormat != "" {
		conf.Log.Format = *argLogFormat
	}
	if *argLogFile != "" {
		conf.Log.File.Path = *argLogFile
	}

	// setup unified logging
	if err := config.SetupLogging(&conf.Log, "frontlas"); err != nil {
		return nil, err
	}

	// daemon
	conf.Daemon.RLimit.NumFile = *argDaemonRLimitNofile
	if conf.Daemon.PProf.CPUProfileRate == 0 {
		conf.Daemon.PProf.CPUProfileRate = 10000
	}
	// env overrides for network
	cpPort := os.Getenv("FRONTLAS_CONTROLPLANE_PORT")
	if cpPort != "" {
		host, _, err := net.SplitHostPort(conf.ControlPlane.Listen.Addr)
		if err != nil {
			return nil, err
		}
		conf.ControlPlane.Listen.Addr = net.JoinHostPort(host, cpPort)
	}
	fpPort := os.Getenv("FRONTLAS_FRONTIERPLANE_PORT")
	if fpPort != "" {
		host, _, err := net.SplitHostPort(conf.FrontierManager.Listen.Addr)
		if err != nil {
			return nil, err
		}
		conf.FrontierManager.Listen.Addr = net.JoinHostPort(host, fpPort)
	}
	redisType := os.Getenv("REDIS_TYPE")
	redisAddrs := os.Getenv("REDIS_ADDRS")
	redisUser := os.Getenv("REDIS_USER")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB")
	redisMasterName := os.Getenv("MASTER_NAME")
	switch redisType {
	case "standalone":
		addrs := strings.Split(redisAddrs, ",")
		db, err := strconv.Atoi(redisDB)
		if err != nil {
			return nil, err
		}
		conf.Redis.Standalone.DB = db
		conf.Redis.Standalone.Addr = addrs[0]
		conf.Redis.Username = redisUser
		conf.Redis.Password = redisPassword
		conf.Redis.Mode = redisType
	case "sentinel":
		addrs := strings.Split(redisAddrs, ",")
		conf.Redis.Sentinel.Addrs = addrs
		conf.Redis.Sentinel.MasterName = redisMasterName
		conf.Redis.Username = redisUser
		conf.Redis.Password = redisPassword
		conf.Redis.Mode = redisType
	case "cluster":
		addrs := strings.Split(redisAddrs, ",")
		conf.Redis.Cluster.Addrs = addrs
		conf.Redis.Username = redisUser
		conf.Redis.Password = redisPassword
		conf.Redis.Mode = redisType
	}
	return conf, nil
}

func genAllConfig(writer io.Writer) error {
	conf := &Configuration{
		Log: config.Log{
			Level:  "info",
			Output: "stdout",
			Format: "text",
			File: config.LogFile{
				Path:       "/var/log/frontier/frontlas.log",
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   false,
			},
		},
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
				Addr:    "0.0.0.0:40011",
			},
		},
		FrontierManager: FrontierManager{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:40012",
			},
		},
		Redis: Redis{
			Mode: "standalone",
		},
	}
	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	newConf := map[string]interface{}{}
	err = yaml.Unmarshal(data, &newConf)
	if err != nil {
		return err
	}
	data, err = yaml.Marshal(newConf)
	if err != nil {
		return err
	}
	_, err = armio.WriteAll(data, writer)
	if err != nil {
		return err
	}
	return nil
}

func genMinConfig(writer io.Writer) error {
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
				Addr:    "0.0.0.0:40011",
			},
		},
		FrontierManager: FrontierManager{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:40012",
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
