package mq

import (
	"sync"

	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/frontier/pkg/config"
)

type mq interface {
	api.MQ
	Close() error
}

type mqManager struct {
	mtx  sync.RWMutex
	conf *config.Configuration
	// mqs
	mqs map[string][]mq
}

func newMQManager(conf *config.Configuration) (*mqManager, error) {
	mqm := &mqManager{
		mqs:  make(map[string][]mq),
		conf: conf,
	}
	return mqm, nil
}
