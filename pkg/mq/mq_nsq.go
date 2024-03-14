package mq

import (
	"github.com/nsqio/go-nsq"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
)

type mqNSQ struct {
	producer *nsq.Producer
	// conf
	conf *config.NSQ
}

func newNSQ(config *config.Configuration) (*mqNSQ, error) {
	conf := config.MQM.NSQ
	if conf.Addrs == nil || len(conf.Addrs) == 0 {
		return nil, apis.ErrEmptyAddress
	}
	nconf := nsq.NewConfig()
	producer, err := nsq.NewProducer(conf.Addrs[0], nconf)
	if err != nil {
		return nil, err
	}
	return &mqNSQ{
		producer: producer,
		conf:     &conf,
	}, nil
}

func (mq *mqNSQ) ProducerTopics() []string {
	return mq.conf.Producer.Topics
}

func (mq *mqNSQ) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	err := mq.producer.Publish(topic, data)
	return err
}

func (mq *mqNSQ) Close() error {
	mq.producer.Stop()
	return nil
}
