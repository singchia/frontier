package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
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
	Enable  bool `yaml:"enable" json:"enable"`
	NumFile int  `yaml:"nofile" json:"nofile"`
}

type PProf struct {
	Enable         bool   `yaml:"enable" json:"enable"`
	Addr           string `yaml:"addr" json:"addr"`
	CPUProfileRate int    `yaml:"cpu_profile_rate" json:"cpu_profile_rate"`
}

type Daemon struct {
	RLimit RLimit `yaml:"rlimit,omitempty" json:"rlimit"`
	PProf  PProf  `yaml:"pprof,omitempty" json:"pprof"`
	// use with frontlas
	FrontierID string `yaml:"frontier_id,omitempty" json:"frontier_id"`
}

// edgebound
// Bypass is for the lagecy gateway, this will split
type Bypass struct {
	Enable  bool       `yaml:"enable" json:"enable"`
	Network string     `yaml:"network" json:"network"`
	Addr    string     `yaml:"addr" json:"addr"` // addr to dial
	TLS     config.TLS `yaml:"tls" json:"tls"`   // certs to dial or ca to auth
}
type Edgebound struct {
	Listen       config.Listen `yaml:"listen" json:"listen"`
	Bypass       config.Dial   `yaml:"bypass,omitempty" json:"bypass"`
	BypassEnable bool          `yaml:"bypass_enable,omitempty" json:"bypass_enable"`
	// alloc edgeID when no get_id function online
	EdgeIDAllocWhenNoIDServiceOn bool `yaml:"edgeid_alloc_when_no_idservice_on" json:"edgeid_alloc_when_no_idservice_on"`
}

// servicebound
type Servicebound struct {
	Listen config.Listen `yaml:"listen" json:"listen"`
}

type ControlPlane struct {
	Enable bool          `yaml:"enable" json:"enable"`
	Listen config.Listen `yaml:"listen" json:"listen"`
}

type Kafka struct {
	Enable bool     `yaml:"enable" json:"enable"`
	Addrs  []string `yaml:"addrs" json:"addrs"`
	// Producer is the namespace for configuration related to producing messages,
	// used by the Producer.
	Producer struct {
		// topics to notify frontier which topics to allow to publish
		Topics []string `yaml:"topics" json:"topics"`
		Async  bool     `yaml:"async" json:"async"`
		// The maximum permitted size of a message (defaults to 1000000). Should be
		// set equal to or smaller than the broker's `message.max.bytes`.
		MaxMessageBytes int `yaml:"max_message_bytes,omitempty" json:"max_message_bytes"`
		// The level of acknowledgement reliability needed from the broker (defaults
		// to WaitForLocal). Equivalent to the `request.required.acks` setting of the
		// JVM producer.
		RequiredAcks sarama.RequiredAcks `yaml:"required_acks,omitempty" json:"required_acks"`
		// The maximum duration the broker will wait the receipt of the number of
		// RequiredAcks (defaults to 10 seconds). This is only relevant when
		// RequiredAcks is set to WaitForAll or a number > 1. Only supports
		// millisecond resolution, nanoseconds will be truncated. Equivalent to
		// the JVM producer's `request.timeout.ms` setting.
		Timeout int `yaml:"timeout,omitempty" json:"timeout"`
		// The type of compression to use on messages (defaults to no compression).
		// Similar to `compression.codec` setting of the JVM producer.
		Compression sarama.CompressionCodec `yaml:"compression,omitempty" json:"compression"`
		// The level of compression to use on messages. The meaning depends
		// on the actual compression type used and defaults to default compression
		// level for the codec.
		CompressionLevel int `yaml:"compression_level,omitempty" json:"compression_level"`
		// If enabled, the producer will ensure that exactly one copy of each message is
		// written.
		Idempotent bool `yaml:"idepotent,omitempty" json:"idempotent"`

		// The following config options control how often messages are batched up and
		// sent to the broker. By default, messages are sent as fast as possible, and
		// all messages received while the current batch is in-flight are placed
		// into the subsequent batch.
		Flush struct {
			// The best-effort number of bytes needed to trigger a flush. Use the
			// global sarama.MaxRequestSize to set a hard upper limit.
			Bytes int `yaml:"bytes,omitempty" json:"bytes"`
			// The best-effort number of messages needed to trigger a flush. Use
			// `MaxMessages` to set a hard upper limit.
			Messages int `yaml:"messages,omitempty" json:"messages"`
			// The best-effort frequency of flushes. Equivalent to
			// `queue.buffering.max.ms` setting of JVM producer.
			Frequency int `yaml:"frequency,omitempty" json:"frequency"`
			// The maximum number of messages the producer will send in a single
			// broker request. Defaults to 0 for unlimited. Similar to
			// `queue.buffering.max.messages` in the JVM producer.
			MaxMessages int `yaml:"max_messages,omitempty" json:"max_messages"`
		} `yaml:"flush,omitempty" json:"flush"`
		Retry struct {
			// The total number of times to retry sending a message (default 3).
			// Similar to the `message.send.max.retries` setting of the JVM producer.
			Max int `yaml:"max,omitempty" json:"max"`
			// How long to wait for the cluster to settle between retries
			// (default 100ms). Similar to the `retry.backoff.ms` setting of the
			// JVM producer.
			Backoff int `yaml:"backoff,omitempty" json:"backoff"`
		} `yaml:"retry" json:"retry"`
	} `yaml:"producer" json:"producer"`
}

type AMQP struct {
	Enable bool `yaml:"enable" json:"enable"`
	// TODO we don't support multiple addresses for now
	Addrs []string `yaml:"addrs" json:"addrs"`
	// Vhost specifies the namespace of permissions, exchanges, queues and
	// bindings on the server.  Dial sets this to the path parsed from the URL.
	Vhost string `yaml:"vhost,omitempty" json:"vhost"`
	// 0 max channels means 2^16 - 1
	ChannelMax int `yaml:"channel_max,omitempty" json:"channel_max"`
	// 0 max bytes means unlimited
	FrameSize int `yaml:"frame_size,omitempty" json:"frame_size"`
	// less than 1s uses the server's interval
	Heartbeat int `yaml:"heartbeat,omitempty" json:"heartbeat"`
	// Connection locale that we expect to always be en_US
	// Even though servers must return it as per the AMQP 0-9-1 spec,
	// we are not aware of it being used other than to satisfy the spec requirements
	Locale string `yaml:"locale,omitempty" json:"locale"`
	// exchange to declare
	Exchanges []struct {
		// exchange name to declare
		Name string `yaml:"name" json:"name"`
		// direct topic fanout headers, default direct
		Kind       string `yaml:"kind,omitempty" json:"kind"`
		Durable    bool   `yaml:"durable,omitempty" json:"durable"`
		AutoDelete bool   `yaml:"auto_delete,omitempty" json:"auto_delete"`
		Internal   bool   `yaml:"internal,omitempty" json:"internal"`
		NoWait     bool   `yaml:"nowait,omitempty" json:"noWait"`
	} `yaml:"exchanges,omitempty" json:"exchanges"`
	// queues to declare, default nil
	Queues []struct {
		Name       string `yaml:"name" json:"name"`
		Durable    bool   `yaml:"durable,omitempty" json:"durable"`
		AutoDelete bool   `yaml:"auto_delete,omitempty" json:"auto_delete"`
		Exclustive bool   `yaml:"exclustive,omitempty" json:"exclustive"`
		NoWait     bool   `yaml:"nowait,omitempty" json:"noWait"`
	} `json:"queues"`
	// queue bindings to exchange, default nil
	QueueBindings []struct {
		QueueName    string `yaml:"queue_name" json:"queue_name"`
		ExchangeName string `yaml:"exchange_name,omitempty" json:"exchange_name"`
		BindingKey   string `yaml:"binding_key,omitempty" json:"binding_key"`
		NoWait       bool   `yaml:"nowait,omitempty" json:"nowait"`
	} `json:"queueBindings"`
	Producer struct {
		RoutingKeys []string `yaml:"routing_keys" json:"routing_keys"` // topics
		Exchange    string   `yaml:"exchange" json:"exchange"`
		Mandatory   bool     `yaml:"mandatory,omitempty" json:"mandatory"`
		Immediate   bool     `yaml:"immediate,omitempty" json:"immediate"`
		// message related
		Headers map[string]interface{} `yaml:"headers,omitempty" json:"headers"`
		// properties
		ContentType     string `yaml:"content_type,omitempty" json:"content_type"`         // MIME content type
		ContentEncoding string `yaml:"content_encoding,omitempty" json:"content_encoding"` // MIME content encoding
		DeliveryMode    uint8  `yaml:"delivery_mode,omitempty" json:"delivery_mode"`       // Transient (0 or 1) or Persistent (2)
		Priority        uint8  `yaml:"priority,omitempty" json:"priority"`                 // 0 to 9
		ReplyTo         string `yaml:"reply_to,omitempty" json:"reply_to"`                 // address to to reply to (ex: RPC)
		Expiration      string `yaml:"expiration,omitempty" json:"expiration"`             // message expiration spec
		Type            string `yaml:"type,omitempty" json:"type"`                         // message type name
		UserId          string `yaml:"user_id,omitempty" json:"user_id"`                   // creating user id - ex: "guest"
		AppId           string `yaml:"app_id,omitempty" json:"app_id"`                     // creating application id
	} `yaml:"producer,omitempty" json:"producer"`
}

type Nats struct {
	Enable   bool     `yaml:"enable" json:"enable"`
	Addrs    []string `yaml:"addrs" json:"addrs"`
	Producer struct {
		Subjects []string `yaml:"subjects" json:"subjects"` // topics
	} `yaml:"producer,omitempty" json:"producer"`
	JetStream struct {
		// using jetstream instead of nats
		Enable   bool   `yaml:"enable" json:"enable"`
		Name     string `yaml:"name" json:"name"`
		Producer struct {
			Subjects []string `yaml:"subjects" json:"subjects"`
		} `yaml:"producer,omitempty" json:"producer"`
	} `yaml:"jetstream,omitempty" json:"jetStream"`
}

type NSQ struct {
	Enable   bool     `yaml:"enable" json:"enable"`
	Addrs    []string `yaml:"addrs" json:"addrs"`
	Producer struct {
		Topics []string `yaml:"topics" json:"topics"`
	} `yaml:"producer" json:"producer"`
}

type Redis struct {
	Enable   bool     `yaml:"enable" json:"enable"`
	Addrs    []string `yaml:"addrs" json:"addrs"`
	DB       int      `yaml:"db" json:"db"`
	Password string   `yaml:"password" json:"password"`
	Producer struct {
		Channels []string `yaml:"channels" json:"channels"`
	} `yaml:"producer" json:"producer"`
}

type MQM struct {
	Kafka Kafka `yaml:"kafka,omitempty" json:"kafka"`
	AMQP  AMQP  `yaml:"amqp,omitempty" json:"amqp"`
	Nats  Nats  `yaml:"nats,omitempty" json:"nats"`
	NSQ   NSQ   `yaml:"nsq,omitempty" json:"nsq"`
	Redis Redis `yaml:"redis,omitempty" json:"redis"`
}

// exchange
type Exchange struct {
	HashBy string `yaml:"hashby" json:"hashby"` // default edgeid, options: srcip random
}

type Dao struct {
	Debug   bool   `yaml:"debug,omitempty" json:"debug"`
	Backend string `yaml:"backend,omitempty" json:"backend"` // default buntdb
}

// frontlas
type Frontlas struct {
	Enable  bool        `yaml:"enable" json:"enable"`
	Dial    config.Dial `yaml:"dial" json:"dial"`
	Metrics struct {
		Enable   bool `yaml:"enable" json:"enable"`
		Interval int  `yaml:"interval" json:"interval"` // for stats
	} `yaml:"metrics" json:"metrics"`
}

type Configuration struct {
	Daemon Daemon `yaml:"daemon,omitempty" json:"daemon"`

	Edgebound Edgebound `yaml:"edgebound" json:"edgebound"`

	Servicebound Servicebound `yaml:"servicebound" json:"servicebound"`

	ControlPlane ControlPlane `yaml:"controlplane,omitempty" json:"controlplane"`

	Exchange Exchange `yaml:"exchange,omitempty" json:"exchange"`

	Dao Dao `yaml:"dao,omitempty" json:"dao"`

	Frontlas Frontlas `yaml:"frontlas,omitempty" json:"frontlas"`

	MQM MQM `yaml:"mqm,omitempty" json:"mqm"`
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
	// env
	sbPort := os.Getenv("FRONTIER_SERVICEBOUND_PORT")
	if sbPort != "" {
		host, _, err := net.SplitHostPort(config.Servicebound.Listen.Addr)
		if err != nil {
			return nil, err
		}
		config.Servicebound.Listen.Addr = net.JoinHostPort(host, sbPort)
	}
	ebPort := os.Getenv("FRONTIER_EDGEBOUND_PORT")
	if ebPort != "" {
		host, _, err := net.SplitHostPort(config.Edgebound.Listen.Addr)
		if err != nil {
			return nil, err
		}
		config.Edgebound.Listen.Addr = net.JoinHostPort(host, ebPort)
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName != "" {
		config.Daemon.FrontierID = "frontier-" + nodeName
	}
	frontlasAddr := os.Getenv("FRONTLAS_ADDR")
	if frontlasAddr != "" {
		config.Frontlas.Enable = true
		config.Frontlas.Dial.Addr = frontlasAddr
	}
	return config, nil
}

func genAllConfig(writer io.Writer) error {
	conf := &Configuration{
		Daemon: Daemon{
			RLimit: RLimit{
				Enable:  true,
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
			Debug:   false,
			Backend: "buntdb",
		},
		Frontlas: Frontlas{
			Enable: false,
			Dial: config.Dial{
				Network: "tcp",
				Addr:    "127.0.0.1:40012",
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
		// default listen on 30011
		Servicebound: Servicebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30011",
			},
		},
		// default listen on 30012
		Edgebound: Edgebound{
			Listen: config.Listen{
				Network: "tcp",
				Addr:    "0.0.0.0:30012",
			},
			EdgeIDAllocWhenNoIDServiceOn: true,
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
