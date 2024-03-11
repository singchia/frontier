package mq

import (
	"github.com/IBM/sarama"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

type mqKafka struct {
	producer sarama.AsyncProducer
}

func newMQKafka(conf *config.Configuration) (*mqKafka, error) {
	kafka := conf.MQM.Kafka
	sconf := initConfig(&kafka)

	producer, err := sarama.NewAsyncProducer(kafka.Addrs, sconf)
	if err != nil {
		klog.Errorf("new mq kafka err: %s", err)
		return nil, err
	}

	// handle error
	go func() {
		for msg := range producer.Errors() {
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
		for msg := range producer.Successes() {
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
		producer: producer,
	}, nil
}

func initConfig(kafka *config.Kafka) *sarama.Config {
	sconf := sarama.NewConfig()
	sconf.Producer.Return.Successes = true
	sconf.Producer.Return.Errors = true
	if kafka.Producer.MaxMessageBytes != 0 {
		sconf.Producer.MaxMessageBytes = kafka.Producer.MaxMessageBytes
	}
	if kafka.Producer.RequiredAcks != 0 {
		sconf.Producer.RequiredAcks = kafka.Producer.RequiredAcks
	}
	if kafka.Producer.Timeout != 0 {
		sconf.Producer.Timeout = kafka.Producer.Timeout
	}
	if kafka.Producer.Idempotent {
		sconf.Producer.Idempotent = kafka.Producer.Idempotent
	}
	// compression
	if kafka.Producer.Compression != 0 {
		sconf.Producer.Compression = kafka.Producer.Compression
	}
	if kafka.Producer.CompressionLevel != 0 {
		sconf.Producer.CompressionLevel = kafka.Producer.CompressionLevel
	}
	// retry
	if kafka.Producer.Retry.Backoff != 0 {
		sconf.Producer.Retry.Backoff = kafka.Producer.Retry.Backoff
	}
	if kafka.Producer.Retry.Max != 0 {
		sconf.Producer.Retry.Max = kafka.Producer.Retry.Max
	}
	// flush
	if kafka.Producer.Flush.Bytes != 0 {
		sconf.Producer.Flush.Bytes = kafka.Producer.Flush.Bytes
	}
	if kafka.Producer.Flush.Frequency != 0 {
		sconf.Producer.Flush.Frequency = kafka.Producer.Flush.Frequency
	}
	if kafka.Producer.Flush.MaxMessages != 0 {
		sconf.Producer.Flush.MaxMessages = kafka.Producer.Flush.MaxMessages
	}
	if kafka.Producer.Flush.Messages != 0 {
		sconf.Producer.Flush.Messages = kafka.Producer.Flush.Messages
	}
	return sconf
}

func (mq *mqKafka) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	message := opt.Origin.(geminio.Message)

	mq.producer.Input() <- &sarama.ProducerMessage{
		Topic:    topic,
		Value:    sarama.ByteEncoder(data),
		Metadata: message,
	}
	return nil
}

func (mq *mqKafka) Close() error {
	return mq.producer.Close()
}
