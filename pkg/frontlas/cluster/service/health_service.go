package service

import (
	"context"

	v1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
)

type Readiness interface {
	Ready() bool
}

func (cs *ClusterService) Check(context.Context, *v1.HealthCheckRequest) (*v1.HealthCheckResponse, error) {
	ready := cs.readiness.Ready()
	status := v1.HealthCheckResponse_NOT_SERVING
	if ready {
		status = v1.HealthCheckResponse_SERVING
	}
	return &v1.HealthCheckResponse{Status: status}, nil
}

func (cs *ClusterService) Watch(_ *v1.HealthCheckRequest, stream v1.Health_WatchServer) error {
	ready := cs.readiness.Ready()
	status := v1.HealthCheckResponse_NOT_SERVING
	if ready {
		status = v1.HealthCheckResponse_SERVING
	}
	return stream.Send(&v1.HealthCheckResponse{Status: status})
}
