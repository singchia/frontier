package config

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/IBM/sarama"
	armio "github.com/jumboframes/armorigo/io"
	"github.com/jumboframes/armorigo/log"
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
	// use with frontlas
	FrontiesID string `yaml:"fronties_id,omitempty"`
}

// edgebound
// Bypass is for the lagecy gateway, this will split
type Bypass struct {
	Enable  bool       `yaml:"enable"`
	Network string     `yaml:"network"`
	Addr    string     `yaml:"addr"` // addr to dial
	TLS     config.TLS `yaml:"tls"`  // certs to dial or ca to auth
}
type Edgebound struct {
	Listen       config.Listen `yaml:"listen"`
	Bypass       config.Dial   `yaml:"bypass"`
	BypassEnable bool          `yaml:"bypass_enable"`
	// alloc edgeID when no get_id function online
	EdgeIDAllocWhenNoIDServiceOn bool `yaml:"edgeid_alloc_when_no_idservice_on"`
}

// servicebound
type Servicebound struct {
	Listen config.Listen `yaml:"listen"`
}

type ControlPlane struct {
	Listen config.Listen `yaml:"listen"`
}

// message queue
type MQ struct {
	BroadCast bool `yaml:"broadcast"`
}

type Kafka struct {
	Enable bool     `yaml:"enable"`
	Addrs  []string `yaml:"addrs"`
	// Producer is the namespace for configuration related to producing messages,
	// used by the Producer.
	Producer struct {
		// topics to notify frontier which topics to allow to publish
		Topics []string
		Async  bool
		// The maximum permitted size of a message (defaults to 1000000). Should be
		// set equal to or smaller than the broker's `message.max.bytes`.
		MaxMessageBytes int
		// The level of acknowledgement reliability needed from the broker (defaults
		// to WaitForLocal). Equivalent to the `request.required.acks` setting of the
		// JVM producer.
		RequiredAcks sarama.RequiredAcks
		// The maximum duration the broker will wait the receipt of the number of
		// RequiredAcks (defaults to 10 seconds). This is only relevant when
		// RequiredAcks is set to WaitForAll or a number > 1. Only supports
		// millisecond resolution, nanoseconds will be truncated. Equivalent to
		// the JVM producer's `request.timeout.ms` setting.
		Timeout int
		// The type of compression to use on messages (defaults to no compression).
		// Similar to `compression.codec` setting of the JVM producer.
		Compression sarama.CompressionCodec
		// The level of compression to use on messages. The meaning depends
		// on the actual compression type used and defaults to default compression
		// level for the codec.
		CompressionLevel int
		// If enabled, the producer will ensure that exactly one copy of each message is
		// written.
		Idempotent bool

		// The following config options control how often messages are batched up and
		// sent to the broker. By default, messages are sent as fast as possible, and
		// all messages received while the current batch is in-flight are placed
		// into the subsequent batch.
		Flush struct {
			// The best-effort number of bytes needed to trigger a flush. Use the
			// global sarama.MaxRequestSize to set a hard upper limit.
			Bytes int
			// The best-effort number of messages needed to trigger a flush. Use
			// `MaxMessages` to set a hard upper limit.
			Messages int
			// The best-effort frequency of flushes. Equivalent to
			// `queue.buffering.max.ms` setting of JVM producer.
			Frequency int
			// The maximum number of messages the producer will send in a single
			// broker request. Defaults to 0 for unlimited. Similar to
			// `queue.buffering.max.messages` in the JVM producer.
			MaxMessages int
		}
		Retry struct {
			// The total number of times to retry sending a message (default 3).
			// Similar to the `message.send.max.retries` setting of the JVM producer.
			Max int
			// How long to wait for the cluster to settle between retries
			// (default 100ms). Similar to the `retry.backoff.ms` setting of the
			// JVM producer.
			Backoff int
		}
	}
}

type AMQP struct {
	Enable bool `yaml:"enable"`
	// TODO we don't support multiple addresses for now
	Addrs []string `yaml:"addrs"`
	// Vhost specifies the namespace of permissions, exchanges, queues and
	// bindings on the server.  Dial sets this to the path parsed from the URL.
	Vhost string
	// 0 max channels means 2^16 - 1
	ChannelMax int
	// 0 max bytes means unlimited
	FrameSize int
	// less than 1s uses the server's interval
	Heartbeat int
	// Connection locale that we expect to always be en_US
	// Even though servers must return it as per the AMQP 0-9-1 spec,
	// we are not aware of it being used other than to satisfy the spec requirements
	Locale string
	// exchange to declare
	Exchanges []struct {
		// exchange name to declare
		Name string
		// direct topic fanout headers, default direct
		Kind       string
		Durable    bool
		AutoDelete bool
		Internal   bool
		NoWait     bool
	}
	// queues to declare, default nil
	Queues []struct {
		Name       string
		Durable    bool
		AutoDelete bool
		Exclustive bool
		NoWait     bool
	}
	// queue bindings to exchange, default nil
	QueueBindings []struct {
		QueueName    string
		ExchangeName string
		BindingKey   string
		NoWait       bool
	}
	Producer struct {
		RoutingKeys []string // topics
		Exchange    string
		Mandatory   bool
		Immediate   bool

		// message related
		Headers map[string]interface{}
		// properties
		ContentType     string // MIME content type
		ContentEncoding string // MIME content encoding
		DeliveryMode    uint8  // Transient (0 or 1) or Persistent (2)
		Priority        uint8  // 0 to 9
		ReplyTo         string // address to to reply to (ex: RPC)
		Expiration      string // message expiration spec
		Type            string // message type name
		UserId          string // creating user id - ex: "guest"
		AppId           string // creating application id

	}
}

type Nats struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	Producer struct {
		Subjects []string // topics
	}
	JetStream struct {
		// using jetstream instead of nats
		Enable   bool   `yaml:"enable"`
		Name     string `yaml:"name"`
		Producer struct {
			Subjects []string
		}
	}
}

type NSQ struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	Producer struct {
		Topics []string
	}
}

type Redis struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	DB       int      `yaml:"db"`
	Password string   `yaml:"password"`
	Producer struct {
		Channels []string
	}
}

type MQM struct {
	Kafka Kafka `yaml:"kafka"`
	AMQP  AMQP  `yaml:"amqp"`
	Nats  Nats  `yaml:"nats"`
	NSQ   NSQ   `yaml:"nsq"`
	Redis Redis `yaml:"redis"`
}

// exchange
type Exchange struct{}

type Dao struct {
	Debug bool `yaml:"debug"`
}

// frontlas
type Frontlas struct {
	Dial config.Dial
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	Edgebound Edgebound `yaml:"edgebound"`

	Servicebound Servicebound `yaml:"servicebound"`

	ControlPlane ControlPlane `yaml:"controlplane"`

	Dao Dao `yaml:"dao"`

	Frontlas Frontlas

	MQM MQM `yaml:"mqm"`
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
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2430",
			},
		},
		Servicebound: Servicebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2431",
				TLS: config.TLS{
					Enable: false,
					MTLS:   false,
					CACerts: []string{
						"ca1.cert",
						"ca2.cert",
					},
					Certs: []config.CertKey{
						{
							Cert: "servicebound.cert",
							Key:  "servicebound.key",
						},
					},
				},
			},
		},
		Edgebound: Edgebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:2432",
				TLS: config.TLS{
					Enable: false,
					MTLS:   false,
					CACerts: []string{
						"ca1.cert",
						"ca2.cert",
					},
					Certs: []config.CertKey{
						{
							Cert: "edgebound.cert",
							Key:  "edgebound.key",
						},
					},
				},
			},
			EdgeIDAllocWhenNoIDServiceOn: true,
			BypassEnable:                 false,
			Bypass: config.Dial{
				Network: "tcp",
				Addr:    "192.168.1.10:8443",
				TLS: config.TLS{
					Enable: true,
					MTLS:   true,
					CACerts: []string{
						"ca1.cert",
					},
					Certs: []config.CertKey{
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
