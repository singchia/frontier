package http

import (
	"net"

	"github.com/gorilla/mux"
	"github.com/singchia/frontier/pkg/config"
)

type Rest struct {
	router *mux.Router

	// listener for http
	ln net.Listener
}

func NewRest(conf *config.Configuration) (*Rest, error) {
	listen := &conf.Http.Listen
	var (
		ln      net.Listener
		network string = listen.Network
		addr    string = listen.Addr
		err     error
	)

}
