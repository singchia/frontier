package exchange

import (
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
)

type exchange struct {
	conf *config.Configuration

	Edgebound    apis.Edgebound
	Servicebound apis.Servicebound
	MQM          apis.MQM
}

func NewExchange(conf *config.Configuration, mqm apis.MQM) (apis.Exchange, error) {
	return newExchange(conf, mqm)
}

func newExchange(conf *config.Configuration, mqm apis.MQM) (*exchange, error) {
	exchange := &exchange{
		conf: conf,
		MQM:  mqm,
	}
	return exchange, nil
}

func (ex *exchange) AddEdgebound(edgebound apis.Edgebound) {
	ex.Edgebound = edgebound
}

func (ex *exchange) AddServicebound(servicebound apis.Servicebound) {
	ex.Servicebound = servicebound
}
