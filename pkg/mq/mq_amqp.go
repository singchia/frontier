package mq

import (
	"context"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

type mqAMQP struct {
	// TODO reconnect
	conn    *amqp.Connection
	channel *amqp.Channel

	// conf
	conf *config.AMQP
}

func newAMQP(config *config.Configuration) (*mqAMQP, error) {
	conf := config.MQM.AMQP
	aconf := initAMQPConfig(&conf)

	url := "amqp://" + conf.Addrs[0]
	conn, err := amqp.DialConfig(url, *aconf)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// exchanges declare
	if conf.Exchanges != nil {
		for _, elem := range conf.Exchanges {
			err = channel.ExchangeDeclare(elem.Name, elem.Kind, elem.Durable, elem.AutoDelete, elem.Internal, elem.NoWait, nil)
			if err != nil {
				klog.Errorf("exchange declare err: %s, name: %s, kind: %s", err, elem.Name, elem.Kind)
			}
		}
	}
	// queues declare
	if conf.Queues != nil {
		for _, elem := range conf.Queues {
			_, err = channel.QueueDeclare(elem.Name, elem.Durable, elem.AutoDelete, elem.Exclustive, elem.NoWait, nil)
			if err != nil {
				klog.Errorf("queue declare err: %s, name: %s", err, elem.Name)
			}
		}
	}
	// queue bindings
	if conf.QueueBindings != nil {
		for _, elem := range conf.QueueBindings {
			err = channel.QueueBind(elem.QueueName, elem.BindingKey, elem.ExchangeName, elem.NoWait, nil)
			if err != nil {
				klog.Errorf("queue bind err: %s, queue name: %s, binding key: %s, exchange name: %s", err, elem.QueueName, elem.BindingKey, elem.ExchangeName)
			}
		}
	}
	return &mqAMQP{
		conn:    conn,
		channel: channel,
	}, nil
}

func initAMQPConfig(conf *config.AMQP) *amqp.Config {
	aconf := &amqp.Config{}
	if conf.Vhost != "" {
		aconf.Vhost = conf.Vhost
	}
	if conf.ChannelMax != 0 {
		aconf.ChannelMax = conf.ChannelMax
	}
	if conf.FrameSize != 0 {
		aconf.FrameSize = conf.FrameSize
	}
	if conf.Heartbeat != 0 {
		aconf.Heartbeat = time.Duration(conf.Heartbeat) * time.Second
	}
	if conf.Locale != "" {
		aconf.Locale = conf.Locale
	}
	return aconf
}

func (mq *mqAMQP) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	message := opt.Origin.(geminio.Message)

	publishing := amqp.Publishing{
		ContentType:     mq.conf.Producer.ContentType,
		ContentEncoding: mq.conf.Producer.ContentEncoding,
		DeliveryMode:    mq.conf.Producer.DeliveryMode,
		Priority:        mq.conf.Producer.Priority,
		ReplyTo:         mq.conf.Producer.ReplyTo,
		Expiration:      mq.conf.Producer.Expiration,
		MessageId:       uuid.New().String(),
		Timestamp:       time.Now(),
		Type:            mq.conf.Producer.Type,
		UserId:          mq.conf.Producer.UserId,
		AppId:           mq.conf.Producer.AppId,
		Body:            data,
	}
	// TODO async or confirmation handle
	err := mq.channel.PublishWithContext(context.TODO(),
		mq.conf.Producer.Exchange, topic, mq.conf.Producer.Mandatory, mq.conf.Producer.Immediate, publishing)
	if err != nil {
		klog.Errorf("mq amqp producer, publish err: %s", err)
		message.Error(err)
		return err
	}
	message.Done()
	return nil
}

func (mq *mqAMQP) Close() error {
	mq.channel.Close()
	return mq.conn.Close()
}
