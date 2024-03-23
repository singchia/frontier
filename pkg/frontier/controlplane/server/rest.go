package server

import (
	"net"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/singchia/frontier/api/controlplane/frontier/v1"
)

func NewHTTPServer(ln net.Listener, svc v1.ControlPlaneServer) *http.Server {
	// new server
	opts := []http.ServerOption{
		http.Middleware(recovery.Recovery()),
		http.Listener(ln),
	}
	srv := http.NewServer(opts...)
	v1.RegisterControlPlaneHTTPServer(srv, svc)
	return srv
}
