package service

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Getter interface {
	GetService(ctx context.Context, objectKey client.ObjectKey) (corev1.Service, error)
}

type Updater interface {
	UpdateService(ctx context.Context, service corev1.Service) error
}

type Creator interface {
	CreateService(ctx context.Context, service corev1.Service) error
}

type Deleter interface {
	DeleteService(ctx context.Context, objectKey client.ObjectKey) error
}

type GetDeleter interface {
	Getter
	Deleter
}

type GetUpdater interface {
	Getter
	Updater
}

type GetUpdateCreator interface {
	Getter
	Updater
	Creator
}

type GetUpdateCreateDeleter interface {
	Getter
	Updater
	Creator
	Deleter
}

// CreateOrUpdate creates the Service if it doesn't exist, other wise it updates it
func CreateOrUpdate(ctx context.Context, getUpdateCreator GetUpdateCreator, service corev1.Service) error {
	oldservice, err := getUpdateCreator.GetService(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace})
	if err != nil {
		if ServiceNotExist(err) {
			return getUpdateCreator.CreateService(ctx, service)
		}
		return err
	}
	service.ResourceVersion = oldservice.ResourceVersion
	return getUpdateCreator.UpdateService(ctx, service)
}

func ServiceNotExist(err error) bool {
	if err == nil {
		return false
	}
	return apiErrors.IsNotFound(err) || strings.Contains(err.Error(), "service not found")
}
