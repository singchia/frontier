package server

import (
	"net"

	nethttp "net/http"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
)

func NewHTTPServer(ln net.Listener, clustersvc v1.ClusterServiceHTTPServer, healthsvc v1.HealthServer) *http.Server {
	// new server
	opts := []http.ServerOption{
		http.Middleware(recovery.Recovery()),
		http.Listener(ln),
	}
	opts = append(opts, http.ResponseEncoder(responseEncoder))
	srv := http.NewServer(opts...)
	v1.RegisterClusterServiceHTTPServer(srv, clustersvc)
	v1.RegisterHealthHTTPServer(srv, healthsvc)
	return srv
}

func responseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if v == nil {
		return nil
	}
	healthCheckResponse, ok := v.(*v1.HealthCheckResponse)
	if ok {
		if healthCheckResponse.Status == v1.HealthCheckResponse_SERVING {
			w.WriteHeader(nethttp.StatusOK)
		} else {
			w.WriteHeader(nethttp.StatusExpectationFailed)
		}
	}
	return nil
}
