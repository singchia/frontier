package client

import (
	"context"

	"github.com/singchia/frontier/operator/pkg/kube/configmap"
	"github.com/singchia/frontier/operator/pkg/kube/secret"
	"github.com/singchia/frontier/operator/pkg/kube/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	k8sclient.Client

	configmap.GetUpdateCreateDeleter
	secret.GetUpdateCreateDeleter
	service.GetUpdateCreateDeleter
}

type client struct {
	k8sclient.Client
}

func NewClient(c k8sclient.Client) Client {
	return client{
		Client: c,
	}
}

// wrapper for service
// GetService provides a thin wrapper and client.Client to access corev1.Service types
func (c client) GetService(ctx context.Context, objectKey k8sclient.ObjectKey) (corev1.Service, error) {
	s := corev1.Service{}
	if err := c.Get(ctx, objectKey, &s); err != nil {
		return corev1.Service{}, err
	}
	return s, nil
}

// UpdateService provides a thin wrapper and client.Client to update corev1.Service types
func (c client) UpdateService(ctx context.Context, service corev1.Service) error {
	return c.Update(ctx, &service)
}

// CreateService provides a thin wrapper and client.Client to create corev1.Service types
func (c client) CreateService(ctx context.Context, service corev1.Service) error {
	return c.Create(ctx, &service)
}

// DeleteService provides a thin wrapper around client.Client to delete corev1.Service types
func (c client) DeleteService(ctx context.Context, objectKey k8sclient.ObjectKey) error {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
		},
	}
	return c.Delete(ctx, &svc)
}

// wrapper for secret
// GetSecret provides a thin wrapper and client.Client to access corev1.Secret types
func (c client) GetSecret(ctx context.Context, objectKey k8sclient.ObjectKey) (corev1.Secret, error) {
	s := corev1.Secret{}
	if err := c.Get(ctx, objectKey, &s); err != nil {
		return corev1.Secret{}, err
	}
	return s, nil
}

// UpdateSecret provides a thin wrapper and client.Client to update corev1.Secret types
func (c client) UpdateSecret(ctx context.Context, secret corev1.Secret) error {
	return c.Update(ctx, &secret)
}

// CreateSecret provides a thin wrapper and client.Client to create corev1.Secret types
func (c client) CreateSecret(ctx context.Context, secret corev1.Secret) error {
	return c.Create(ctx, &secret)
}

// DeleteSecret provides a thin wrapper and client.Client to delete corev1.Secret types
func (c client) DeleteSecret(ctx context.Context, key k8sclient.ObjectKey) error {
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
	return c.Delete(ctx, &s)
}

// wrapper for configmap
// GetConfigMap provides a thin wrapper and client.client to access corev1.ConfigMap types
func (c client) GetConfigMap(ctx context.Context, objectKey k8sclient.ObjectKey) (corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	if err := c.Get(ctx, objectKey, &cm); err != nil {
		return corev1.ConfigMap{}, err
	}
	return cm, nil
}

// UpdateConfigMap provides a thin wrapper and client.Client to update corev1.ConfigMap types
func (c client) UpdateConfigMap(ctx context.Context, cm corev1.ConfigMap) error {
	return c.Update(ctx, &cm)
}

// CreateConfigMap provides a thin wrapper and client.Client to create corev1.ConfigMap types
func (c client) CreateConfigMap(ctx context.Context, cm corev1.ConfigMap) error {
	return c.Create(ctx, &cm)
}

// DeleteConfigMap deletes the configmap of the given object key
func (c client) DeleteConfigMap(ctx context.Context, key k8sclient.ObjectKey) error {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
	return c.Delete(ctx, &cm)
}
