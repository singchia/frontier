package exchange

import (
	"github.com/singchia/frontier/pkg/api"
)

type exchange struct {
	Edgebound    api.Edgebound
	Servicebound api.Servicebound
	MQ           api.MQ
}
