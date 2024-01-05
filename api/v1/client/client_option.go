package client

import (
	"github.com/jumboframes/armorigo/log"

	"github.com/singchia/go-timer/v2"
)

type Logger log.Logger
type Timer timer.Timer

type clientOption struct {
	logger   Logger
	tmr      Timer
	clientID *uint64
	meta     []byte
}

type ClientOption func(*clientOption)

func OptionClientLog(logger Logger) ClientOption {
	return func(opt *clientOption) {
		opt.logger = logger
	}
}

// Pre set timer for the sdk
func OptionClientTimer(tmr Timer) ClientOption {
	return func(opt *clientOption) {
		opt.tmr = tmr
	}
}

// Pre set ClientID for the sdk
func OptionClientID(clientID uint64) ClientOption {
	return func(opt *clientOption) {
		opt.clientID = &clientID
	}
}

func OptionClientMeta(meta []byte) ClientOption {
	return func(opt *clientOption) {
		opt.meta = meta
	}
}
