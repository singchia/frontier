package controller

import (
	"context"
	"fmt"

	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/kube/service"
)

func (r *FrontierClusterReconciler) ensureService(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	// servicebound
	sbServiceName, sbServiceType, port := fc.FrontierServiceboundServicePort()
	label := map[string]string{
		"app": sbServiceName,
	}
	sbService := service.Builder().
		SetName(sbServiceName).
		SetNamespace(fc.Namespace).
		SetSelector(label).
		SetLabels(label).
		SetServiceType(sbServiceType).
		SetClusterIP("None").
		SetPublishNotReadyAddresses(true).
		SetOwnerReferences(fc.GetOwnerReferences()).
		AddPort(&port).Build()

	if err := service.CreateOrUpdate(ctx, r.client, sbService); err != nil {
		return fmt.Errorf("Could not ensure servicebound service: %s", err)
	}

	// edgebound
	ebServiceName, ebServiceType, port := fc.FrontierEdgeboundServicePort()
	label = map[string]string{
		"app": ebServiceName,
	}
	ebService := service.Builder().
		SetName(ebServiceName).
		SetNamespace(fc.Namespace).
		SetSelector(label).
		SetLabels(label).
		SetServiceType(ebServiceType).
		SetClusterIP("None").
		SetPublishNotReadyAddresses(true).
		SetOwnerReferences(fc.GetOwnerReferences()).
		AddPort(&port).Build()

	if err := service.CreateOrUpdate(ctx, r.client, ebService); err != nil {
		return fmt.Errorf("Could not ensure edgebound service: %s", err)
	}

	// controlplane
	cpServiceName, cpServiceType, port := fc.FrontlasControlPlaneServicePort()
	label = map[string]string{
		"app": cpServiceName,
	}
	cpService := service.Builder().
		SetName(cpServiceName).
		SetNamespace(fc.Namespace).
		SetSelector(label).
		SetLabels(label).
		SetServiceType(cpServiceType).
		SetClusterIP("None").
		SetPublishNotReadyAddresses(true).
		SetOwnerReferences(fc.GetOwnerReferences()).
		AddPort(&port).Build()

	if err := service.CreateOrUpdate(ctx, r.client, cpService); err != nil {
		return fmt.Errorf("Could not ensure controlplane service: %s", err)
	}
	return nil
}
