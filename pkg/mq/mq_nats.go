package mq

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"k8s.io/klog/v2"
)

type mqNats struct {
	jetstream bool

	conn *nats.Conn
	js   jetstream.JetStream

	// conf
	conf *config.Nats
}

func newNats(config *config.Configuration) (*mqNats, error) {
	conf := config.MQM.Nats
	if conf.Addrs == nil || len(conf.Addrs) == 0 {
		return nil, apis.ErrEmptyAddress
	}
	var (
		conn *nats.Conn
		js   jetstream.JetStream
		err  error
	)
	// dial
	conn, err = nats.Connect(getNatsURL(conf.Addrs))
	if err != nil {
		klog.Errorf("new mq nats err: %s", err)
		return nil, err
	}
	if conf.JetStream.Enable {
		js, err = jetstream.New(conn)
		if err != nil {
			klog.Errorf("new jetstream err: %s", err)
			return nil, err
		}
		_, err = js.CreateStream(context.TODO(), jetstream.StreamConfig{
			Name:     conf.JetStream.Name,
			Subjects: conf.JetStream.Subjects,
		})
		if err != nil {
			klog.Errorf("jetstream create stream err: %s", err)
			return nil, err
		}
	}
	return &mqNats{
		jetstream: conf.JetStream.Enable,
		conn:      conn,
		js:        js,
		conf:      &conf,
	}, nil
}

func getNatsURL(addrs []string) string {
	url := ""
	for i, elem := range addrs {
		if !strings.HasPrefix(elem, "nats://") {
			elem = "nats://" + elem
		}
		if i != len(addrs)-1 {
			url += elem + ","
		}
	}
	return url
}

func (mq *mqNats) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	if mq.jetstream {
		_, err := mq.js.Publish(context.TODO(), topic, data)
		return err
	}
	err := mq.conn.Publish(topic, data)
	return err
}

func (mq *mqNats) Close() {
	mq.conn.Close()
}
