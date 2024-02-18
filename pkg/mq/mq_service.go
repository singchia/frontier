package mq

import (
	"context"
	"encoding/binary"

	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/options"
	"k8s.io/klog/v2"
)

type mqService struct {
	end geminio.End
}

func NewMQServiceFromEnd(end geminio.End) api.MQ {
	return &mqService{end}
}

func (mq *mqService) Produce(topic string, data []byte, opts ...api.OptionProduce) error {
	opt := &api.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	msg := opt.Origin.(geminio.Message)
	edgeID := opt.EdgeID
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)

	// we record the edgeID to service
	custom := msg.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	// new message
	mopt := options.NewMessage()
	mopt.SetCustom(custom)
	mopt.SetTopic(topic)
	newmsg := mq.end.NewMessage(data, mopt)
	err := mq.end.Publish(context.TODO(), newmsg)
	if err != nil {
		klog.Errorf("mq service, publish err: %s, edgeID: %d, serviceID: %d", err, edgeID, mq.end.ClientID())
		return err
	}
	return nil
}

func (mq *mqService) Close() error {
	return nil
}
