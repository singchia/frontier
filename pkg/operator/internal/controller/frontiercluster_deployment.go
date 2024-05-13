package controller

import (
	"context"

	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/kube/deployment"
)

func (r *FrontierClusterReconciler) ensureDeployment(ctx context.Context, fc *v1alpha1.FrontierCluster) error {
	labels := map[string]string{
		"app": fc.AppName(),
	}

	sbService, _, _ := fc.FrontierServiceboundServicePort()
	deployment.Builder().
		SetName(fc.FrontierDeploymentName()).
		SetNamespace(fc.Namespace).
		SetServiceName(sbService).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetReplicas(fc.FrontierReplicas()).
		SetPodTemplateSpec()
}
