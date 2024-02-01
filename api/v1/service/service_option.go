package service

import (
	"github.com/jumboframes/armorigo/log"

	"github.com/singchia/go-timer/v2"
)

type Logger log.Logger
type Timer timer.Timer

type serviceOption struct {
	logger Logger
	tmr    Timer
}

type ServiceOption func(*serviceOption)

func OptionServiceLog(logger Logger) ServiceOption {
	return func(opt *serviceOption) {
		opt.logger = logger
	}
}

// Pre set timer for the sdk
func OptionServiceTimer(tmr Timer) ServiceOption {
	return func(opt *serviceOption) {
		opt.tmr = tmr
	}
}
