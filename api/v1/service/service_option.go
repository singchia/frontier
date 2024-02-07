package service

import (
	"github.com/jumboframes/armorigo/log"

	"github.com/singchia/geminio/delegate"
	"github.com/singchia/go-timer/v2"
)

type Logger log.Logger
type Timer timer.Timer

type Delegate delegate.ClientDelegate

type serviceOption struct {
	logger Logger
	tmr    Timer
	// to tell frontier which topics to receive, default no receiving
	topics []string
	// to tell frontier what service we are
	service string
	// delegate to know online offline stuff
	delegate Delegate
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

func OptionServiceReceiveTopics(topics []string) ServiceOption {
	return func(opt *serviceOption) {
		opt.topics = topics
	}
}

func OptionServiceName(service string) ServiceOption {
	return func(opt *serviceOption) {
		opt.service = service
	}
}

// delegations for the service own connection, streams and more
func OptionServiceDelegate(delegate Delegate) ServiceOption {
	return func(opt *serviceOption) {
		opt.delegate = delegate
	}
}
