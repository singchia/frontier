package mq

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

type mqKafka struct {
	asyncProducer sarama.AsyncProducer
	producer      sarama.SyncProducer
}

func newKafka(config *config.Configuration) (*mqKafka, error) {
	conf := config.MQM.Kafka
	if conf.Addrs == nil || len(conf.Addrs) == 0 {
		return nil, apis.ErrEmptyAddress
	}
	sconf := initKafkaConfig(&conf)

	var (
		asyncProducer sarama.AsyncProducer
		producer      sarama.SyncProducer
		err           error
	)

	// TODO not enabled
	if conf.Producer.Async {
		// dial
		asyncProducer, err = sarama.NewAsyncProducer(conf.Addrs, sconf)
		if err != nil {
			klog.Errorf("new mq kafka async producer err: %s, addr: %v", err, conf.Addrs)
			return nil, err
		}
		// handle error
		go func() {
			for msg := range asyncProducer.Errors() {
				message, ok := msg.Msg.Metadata.(geminio.Message)
				if !ok {
					klog.Errorf("mq kafka producer, errors channel return wrong type: %v", msg.Msg.Metadata)
					continue
				}
				message.Error(msg.Err)
				// TODO metrics
			}
		}()
		// handle success
		go func() {
			for msg := range asyncProducer.Successes() {
				message, ok := msg.Metadata.(geminio.Message)
				if !ok {
					klog.Errorf("mq kafka producer, success channel return wrong type: %v", msg.Metadata)
					continue
				}
				message.Done()
				// TODO metrics
			}
		}()
		return &mqKafka{
			asyncProducer: asyncProducer,
		}, nil
	}

	producer, err = sarama.NewSyncProducer(conf.Addrs, sconf)
	if err != nil {
		klog.Errorf("new mq kafka sync producer err: %s, addr: %v", err, conf.Addrs)
		return nil, err
	}
	return &mqKafka{
		producer: producer,
	}, nil
}

func initKafkaConfig(conf *config.Kafka) *sarama.Config {
	sconf := sarama.NewConfig()
	sconf.Producer.Return.Successes = true
	sconf.Producer.Return.Errors = true
	if conf.Producer.MaxMessageBytes != 0 {
		sconf.Producer.MaxMessageBytes = conf.Producer.MaxMessageBytes
	}
	if conf.Producer.RequiredAcks != 0 {
		sconf.Producer.RequiredAcks = conf.Producer.RequiredAcks
	}
	if conf.Producer.Timeout != 0 {
		sconf.Producer.Timeout = time.Duration(conf.Producer.Timeout) * time.Second
	}
	if conf.Producer.Idempotent {
		sconf.Producer.Idempotent = conf.Producer.Idempotent
	}
	// compression
	if conf.Producer.Compression != 0 {
		sconf.Producer.Compression = conf.Producer.Compression
	}
	if conf.Producer.CompressionLevel != 0 {
		sconf.Producer.CompressionLevel = conf.Producer.CompressionLevel
	}
	// retry
	if conf.Producer.Retry.Backoff != 0 {
		sconf.Producer.Retry.Backoff = time.Duration(conf.Producer.Retry.Backoff) * time.Second
	}
	if conf.Producer.Retry.Max != 0 {
		sconf.Producer.Retry.Max = conf.Producer.Retry.Max
	}
	// flush
	if conf.Producer.Flush.Bytes != 0 {
		sconf.Producer.Flush.Bytes = conf.Producer.Flush.Bytes
	}
	if conf.Producer.Flush.Frequency != 0 {
		sconf.Producer.Flush.Frequency = time.Duration(conf.Producer.Flush.Frequency) * time.Second
	}
	if conf.Producer.Flush.MaxMessages != 0 {
		sconf.Producer.Flush.MaxMessages = conf.Producer.Flush.MaxMessages
	}
	if conf.Producer.Flush.Messages != 0 {
		sconf.Producer.Flush.Messages = conf.Producer.Flush.Messages
	}
	return sconf
}

func (mq *mqKafka) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}

	// TODO we can add yaegi handler here to let user to do some transfer works
	_, _, err := mq.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (mq *mqKafka) Close() error {
	return mq.producer.Close()
}
