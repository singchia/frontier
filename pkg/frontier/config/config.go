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
	FrontierID string `yaml:"frontier_id,omitempty"`
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
	Enable bool          `yaml:"enable"`
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
		Topics []string `yaml:"topics"`
		Async  bool     `yaml:"async"`
		// The maximum permitted size of a message (defaults to 1000000). Should be
		// set equal to or smaller than the broker's `message.max.bytes`.
		MaxMessageBytes int `yaml:"max_message_bytes,omitempty"`
		// The level of acknowledgement reliability needed from the broker (defaults
		// to WaitForLocal). Equivalent to the `request.required.acks` setting of the
		// JVM producer.
		RequiredAcks sarama.RequiredAcks `yaml:"required_acks,omitempty"`
		// The maximum duration the broker will wait the receipt of the number of
		// RequiredAcks (defaults to 10 seconds). This is only relevant when
		// RequiredAcks is set to WaitForAll or a number > 1. Only supports
		// millisecond resolution, nanoseconds will be truncated. Equivalent to
		// the JVM producer's `request.timeout.ms` setting.
		Timeout int `yaml:"timeout,omitempty"`
		// The type of compression to use on messages (defaults to no compression).
		// Similar to `compression.codec` setting of the JVM producer.
		Compression sarama.CompressionCodec `yaml:"compression,omitempty"`
		// The level of compression to use on messages. The meaning depends
		// on the actual compression type used and defaults to default compression
		// level for the codec.
		CompressionLevel int `yaml:"compression_level,omitempty"`
		// If enabled, the producer will ensure that exactly one copy of each message is
		// written.
		Idempotent bool `yaml:"idepotent,omitempty"`

		// The following config options control how often messages are batched up and
		// sent to the broker. By default, messages are sent as fast as possible, and
		// all messages received while the current batch is in-flight are placed
		// into the subsequent batch.
		Flush struct {
			// The best-effort number of bytes needed to trigger a flush. Use the
			// global sarama.MaxRequestSize to set a hard upper limit.
			Bytes int `yaml:"bytes,omitempty"`
			// The best-effort number of messages needed to trigger a flush. Use
			// `MaxMessages` to set a hard upper limit.
			Messages int `yaml:"messages,omitempty"`
			// The best-effort frequency of flushes. Equivalent to
			// `queue.buffering.max.ms` setting of JVM producer.
			Frequency int `yaml:"frequency,omitempty"`
			// The maximum number of messages the producer will send in a single
			// broker request. Defaults to 0 for unlimited. Similar to
			// `queue.buffering.max.messages` in the JVM producer.
			MaxMessages int `yaml:"max_messages,omitempty"`
		} `yaml:"flush,omitempty"`
		Retry struct {
			// The total number of times to retry sending a message (default 3).
			// Similar to the `message.send.max.retries` setting of the JVM producer.
			Max int `yaml:"max,omitempty"`
			// How long to wait for the cluster to settle between retries
			// (default 100ms). Similar to the `retry.backoff.ms` setting of the
			// JVM producer.
			Backoff int `yaml:"back_off,omitempty"`
		} `yaml:"retry"`
	} `yaml:"producer"`
}

type AMQP struct {
	Enable bool `yaml:"enable"`
	// TODO we don't support multiple addresses for now
	Addrs []string `yaml:"addrs"`
	// Vhost specifies the namespace of permissions, exchanges, queues and
	// bindings on the server.  Dial sets this to the path parsed from the URL.
	Vhost string `yaml:"vhost,omitempty"`
	// 0 max channels means 2^16 - 1
	ChannelMax int `yaml:"channel_max,omitempty"`
	// 0 max bytes means unlimited
	FrameSize int `yaml:"frame_size,omitempty"`
	// less than 1s uses the server's interval
	Heartbeat int `yaml:"heartbeat,omitempty"`
	// Connection locale that we expect to always be en_US
	// Even though servers must return it as per the AMQP 0-9-1 spec,
	// we are not aware of it being used other than to satisfy the spec requirements
	Locale string `yaml:"locale,omitempty"`
	// exchange to declare
	Exchanges []struct {
		// exchange name to declare
		Name string `yaml:"name"`
		// direct topic fanout headers, default direct
		Kind       string `yaml:"kind,omitempty"`
		Durable    bool   `yaml:"durable,omitempty"`
		AutoDelete bool   `yaml:"auto_delete,omitempty"`
		Internal   bool   `yaml:"internal,omitempty"`
		NoWait     bool   `yaml:"nowait,omitempty"`
	} `yaml:"exchanges,omitempty"`
	// queues to declare, default nil
	Queues []struct {
		Name       string `yaml:"name"`
		Durable    bool   `yaml:"durable,omitempty"`
		AutoDelete bool   `yaml:"auto_delete,omitempty"`
		Exclustive bool   `yaml:"exclustive,omitempty"`
		NoWait     bool   `yaml:"nowait,omitempty"`
	}
	// queue bindings to exchange, default nil
	QueueBindings []struct {
		QueueName    string `yaml:"queue_name"`
		ExchangeName string `yaml:"exchange_name,omitempty"`
		BindingKey   string `yaml:"binding_key,omitempty"`
		NoWait       bool   `yaml:"nowait,omitempty"`
	}
	Producer struct {
		RoutingKeys []string `yaml:"routing_keys"` // topics
		Exchange    string   `yaml:"exchange"`
		Mandatory   bool     `yaml:"mandatory,omitempty"`
		Immediate   bool     `yaml:"immediate,omitempty"`
		// message related
		Headers map[string]interface{} `yaml:"headers,omitempty"`
		// properties
		ContentType     string `yaml:"content_type,omitempty"`     // MIME content type
		ContentEncoding string `yaml:"content_encoding,omitempty"` // MIME content encoding
		DeliveryMode    uint8  `yaml:"delivery_mode,omitempty"`    // Transient (0 or 1) or Persistent (2)
		Priority        uint8  `yaml:"priority,omitempty"`         // 0 to 9
		ReplyTo         string `yaml:"reply_to,omitempty"`         // address to to reply to (ex: RPC)
		Expiration      string `yaml:"expiration,omitempty"`       // message expiration spec
		Type            string `yaml:"type,omitempty"`             // message type name
		UserId          string `yaml:"user_id,omitempty"`          // creating user id - ex: "guest"
		AppId           string `yaml:"app_id,omitempty"`           // creating application id
	} `yaml:"producer,omitempty"`
}

type Nats struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	Producer struct {
		Subjects []string `yaml:"subjects"` // topics
	} `yaml:"producer,omitempty"`
	JetStream struct {
		// using jetstream instead of nats
		Enable   bool   `yaml:"enable"`
		Name     string `yaml:"name"`
		Producer struct {
			Subjects []string `yaml:"subjects"`
		} `yaml:"producer,omitempty"`
	} `yaml:"jetstream,omitempty"`
}

type NSQ struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	Producer struct {
		Topics []string `yaml:"topics"`
	} `yaml:"producer"`
}

type Redis struct {
	Enable   bool     `yaml:"enable"`
	Addrs    []string `yaml:"addrs"`
	DB       int      `yaml:"db"`
	Password string   `yaml:"password"`
	Producer struct {
		Channels []string `yaml:"channels"`
	} `yaml:"producer"`
}

type MQM struct {
	Kafka Kafka `yaml:"kafka,omitempty"`
	AMQP  AMQP  `yaml:"amqp,omitempty"`
	Nats  Nats  `yaml:"nats,omitempty"`
	NSQ   NSQ   `yaml:"nsq,omitempty"`
	Redis Redis `yaml:"redis,omitempty"`
}

// exchange
type Exchange struct{}

type Dao struct {
	Debug bool `yaml:"debug"`
}

// frontlas
type Frontlas struct {
	Enable  bool        `yaml:"enable"`
	Dial    config.Dial `yaml:"dial"`
	Metrics struct {
		Enable   bool `yaml:"enable"`
		Interval int  `yaml:"interval"` // for stats
	}
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon"`

	Edgebound Edgebound `yaml:"edgebound"`

	Servicebound Servicebound `yaml:"servicebound"`

	ControlPlane ControlPlane `yaml:"controlplane"`

	Dao Dao `yaml:"dao"`

	Frontlas Frontlas `yaml:"frontlas"`

	MQM MQM `yaml:"mqm"`
}

// Configuration accepts config file and command-line, and command-line is more privileged.
func Parse() (*Configuration, error) {
	var (
		argConfigFile         = pflag.String("config", "", "config file, default not configured")
		argArmorigoLogLevel   = pflag.String("loglevel", "info", "log level for armorigo log")
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
				NumFile: 102400,
			},
			PProf: PProf{
				Enable: true,
				Addr:   "0.0.0.0:6060",
			},
		},
		// default listen on 30010
		ControlPlane: ControlPlane{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30010",
			},
		},
		// default listen on 30011
		Servicebound: Servicebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30011",
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
		// default listen on 30012
		Edgebound: Edgebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30012",
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
		Frontlas: Frontlas{
			Enable: false,
			Dial: config.Dial{
				Network: "tcp",
				Addr:    "127.0.0.1:30022",
				TLS: config.TLS{
					Enable: false,
					MTLS:   false,
				},
			},
		},
		MQM: MQM{
			Kafka: Kafka{
				Enable: false,
			},
			AMQP: AMQP{
				Enable: false,
			},
			Nats: Nats{
				Enable: false,
			},
			NSQ: NSQ{
				Enable: false,
			},
			Redis: Redis{
				Enable: false,
			},
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
