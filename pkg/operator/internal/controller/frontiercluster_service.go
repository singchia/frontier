package controller

import (
	"context"

	"github.com/singchia/frontier/operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *FrontierClusterReconciler) ensureService(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	_ = log.FromContext(ctx)
	return nil
}
