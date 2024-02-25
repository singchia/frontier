package exchange

import (
	"github.com/singchia/frontier/pkg/api"
	"github.com/singchia/frontier/pkg/config"
)

type exchange struct {
	conf *config.Configuration

	Edgebound    api.Edgebound
	Servicebound api.Servicebound
	MQM          api.MQM
}

func NewExchange(conf *config.Configuration, mqm api.MQM) (api.Exchange, error) {
	return newExchange(conf, mqm)
}

func newExchange(conf *config.Configuration, mqm api.MQM) (*exchange, error) {
	exchange := &exchange{
		conf: conf,
		MQM:  mqm,
	}
	return exchange, nil
}

func (ex *exchange) AddEdgebound(edgebound api.Edgebound) {
	ex.Edgebound = edgebound
}

func (ex *exchange) AddServicebound(servicebound api.Servicebound) {
	ex.Servicebound = servicebound
}
