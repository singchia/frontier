package mq

import (
	"sync"
	"sync/atomic"

	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/geminio"
	"k8s.io/klog/v2"
)

type mqManager struct {
	conf *config.Configuration
	// mqs
	mtx     sync.RWMutex
	mqs     map[string][]api.MQ // key: topic, value: mqs
	mqindex map[string]*uint64  // for round robin
}

func NewMQM(conf *config.Configuration) (api.MQM, error) {
	return newMQManager(conf)
}

func newMQManager(conf *config.Configuration) (*mqManager, error) {
	mqm := &mqManager{
		mqs:     make(map[string][]api.MQ),
		mqindex: make(map[string]*uint64),
		conf:    conf,
	}
	return mqm, nil
}

func (mqm *mqManager) AddMQ(topics []string, mq api.MQ) {
	mqm.mtx.Lock()
	defer mqm.mtx.Unlock()

	for _, topic := range topics {
		mqs, ok := mqm.mqs[topic]
		if !ok {
			klog.V(2).Infof("mq manager, add topic: %s mq succeed", topic)
			mqm.mqs[topic] = []api.MQ{mq}
			mqm.mqindex[topic] = new(uint64)
			continue
		}
		for _, exist := range mqs {
			if exist == mq {
				klog.V(2).Infof("mq manager, add topic: %s mq existed", topic)
				continue
			}
			// special handle for service, a deep comparison
			left, ok := exist.(*mqService)
			if ok {
				right, ok := mq.(*mqService)
				if ok && left.end == right.end {
					klog.V(2).Infof("mq manager, add topic: %s service mq existed", topic)
					continue
				}
			}
		}
		mqs = append(mqs, mq)
		mqm.mqs[topic] = mqs
		klog.V(2).Infof("mq mqnager, add topic: %s mq succeed", topic)
	}
}

func (mqm *mqManager) AddMQByEnd(topics []string, end geminio.End) {
	mq := NewMQServiceFromEnd(end)
	mqm.AddMQ(topics, mq)
}

func (mqm *mqManager) DelMQ(mq api.MQ) {
	mqm.mtx.Lock()
	defer mqm.mtx.Unlock()

	for topic, mqs := range mqm.mqs {
		news := []api.MQ{}
		for _, exist := range mqs {
			if exist == mq {
				klog.V(3).Infof("mq manager, del topic: %s mq succeed", topic)
				continue
			}
			news = append(news, exist)
		}
		if len(news) == 0 {
			// delete array of this topic
			delete(mqm.mqs, topic)
			delete(mqm.mqindex, topic)
			continue
		}
		mqm.mqs[topic] = news
	}
}

// special handle for service, a deep comparison
func (mqm *mqManager) DelMQByEnd(end geminio.End) {
	mqm.mtx.Lock()
	defer mqm.mtx.Unlock()

	for topic, mqs := range mqm.mqs {
		news := []api.MQ{}
		for _, exist := range mqs {
			left, ok := exist.(*mqService)
			if ok {
				if ok && left.end == end {
					klog.V(3).Infof("mq manager, del topic: %s service mq succeed", topic)
					continue
				}
			}
			news = append(news, exist)
		}
		if len(news) == 0 {
			delete(mqm.mqs, topic)
			delete(mqm.mqindex, topic)
			continue
		}
		mqm.mqs[topic] = news
	}
}

func (mqm *mqManager) GetMQ(topic string) api.MQ {
	mqm.mtx.RLock()
	defer mqm.mtx.RUnlock()

	mqs, ok := mqm.mqs[topic]
	if !ok {
		return nil
	}
	index := mqm.mqindex[topic]
	newindex := atomic.AddUint64(index, 1)

	i := newindex % uint64(len(mqs))
	return mqs[i]
}

func (mqm *mqManager) GetMQs(topic string) []api.MQ {
	mqm.mtx.RLock()
	defer mqm.mtx.RUnlock()
	return mqm.mqs[topic]
}

func (mqm *mqManager) Produce(topic string, data []byte, opts ...api.OptionProduce) error {
	mq := mqm.GetMQ(topic)
	if mq == nil {
		mq = mqm.GetMQ("*")
		if mq == nil {
			err := api.ErrTopicNotOnline
			klog.Errorf("mq manager, get mq nil, topic: %s err: %s", topic, err)
			return err
		}
	}
	err := mq.Produce(topic, data, opts...)
	if err != nil {
		klog.Errorf("mq manager, produce topic: %s message err: %s", topic, err)
		return err
	}
	klog.V(3).Infof("mq manager, produce topic: %s message succeed", topic)
	return nil
}

func (mqm *mqManager) Close() error {
	mqm.mtx.RLock()
	defer mqm.mtx.RUnlock()

	var reterr error

	for topic, mqs := range mqm.mqs {
		for _, mq := range mqs {
			err := mq.Close()
			if err != nil {
				klog.Errorf("mq manager, close mq err: %s, topic: %s", err, topic)
				reterr = err
			}
		}
	}
	return reterr
}
