package mq

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
)

type mqRedis struct {
	rdb *redis.Client

	// conf
	conf *config.Redis
}

func newRedis(config *config.Configuration) (*mqRedis, error) {
	conf := config.MQM.Redis
	if conf.Addrs == nil || len(conf.Addrs) == 0 {
		return nil, apis.ErrEmptyAddress
	}
	ropt := &redis.Options{
		Addr:     conf.Addrs[0],
		DB:       conf.DB,
		Password: conf.Password,
	}
	rdb := redis.NewClient(ropt)
	_, err := rdb.Ping(context.TODO()).Result()
	if err != nil {
		return nil, err
	}
	return &mqRedis{
		rdb:  rdb,
		conf: &conf,
	}, nil
}

func (mq *mqRedis) Produce(topic string, data []byte, opts ...apis.OptionProduce) error {
	opt := &apis.ProduceOption{}
	for _, fun := range opts {
		fun(opt)
	}
	err := mq.rdb.Publish(context.TODO(), topic, data).Err()
	return err
}

func (mq *mqRedis) Close() error {
	mq.rdb.Close()
	return nil
}
