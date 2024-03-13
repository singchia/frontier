package mq

import (
	"context"
	"encoding/binary"

	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/options"
	"k8s.io/klog/v2"
)

type mqService struct {
	end geminio.End
}

func NewMQServiceFromEnd(end geminio.End) apis.MQ {
	return &mqService{end}
}

func (mq *mqService) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	message := opt.Origin.(geminio.Message)
	edgeID := opt.EdgeID
	tail := make([]byte, 8)
	binary.BigEndian.PutUint64(tail, edgeID)

	// we record the edgeID to service
	custom := message.Custom()
	if custom == nil {
		custom = tail
	} else {
		custom = append(custom, tail...)
	}
	// new message
	mopt := options.NewMessage()
	mopt.SetCustom(custom)
	mopt.SetTopic(topic)
	mopt.SetCnss(message.Cnss())
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
