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

func NewExchange(conf *config.Configuration) (api.Exchange, error) {
	return newExchange(conf)
}

func newExchange(conf *config.Configuration) (*exchange, error) {
	exchange := &exchange{
		conf: conf,
	}
	return exchange, nil
}

func (ex *exchange) AddEdgebound(edgebound api.Edgebound) {
	ex.Edgebound = edgebound
}

func (ex *exchange) AddServicebound(servicebound api.Servicebound) {
	ex.Servicebound = servicebound
}

func (ex *exchange) AddMQM(mqm api.MQM) {
	ex.MQM = mqm
}
