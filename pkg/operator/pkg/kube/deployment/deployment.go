package deployment

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	notFound = -1
)

type Getter interface {
	GetDeployment(ctx context.Context, objectKey client.ObjectKey) (appsv1.Deployment, error)
}

type Updater interface {
	UpdateDeployment(ctx context.Context, deployment appsv1.Deployment) (appsv1.Deployment, error)
}

type Creator interface {
	CreateDeployment(ctx context.Context, deployment appsv1.Deployment) error
}

type Deleter interface {
	DeleteDeployment(ctx context.Context, objectKey client.ObjectKey) error
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

// CreateOrUpdate creates the given Deployment if it doesn't exist,
// or updates it if it does.
func CreateOrUpdate(ctx context.Context, getUpdateCreator GetUpdateCreator, deployment appsv1.Deployment) (appsv1.Deployment, error) {
	_, err := getUpdateCreator.GetDeployment(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return appsv1.Deployment{}, getUpdateCreator.CreateDeployment(ctx, deployment)
		}
		return appsv1.Deployment{}, err
	}
	return getUpdateCreator.UpdateDeployment(ctx, deployment)
}

// GetAndUpdate applies the provided function to the most recent version of the object
func GetAndUpdate(ctx context.Context, getUpdater GetUpdater, nsName types.NamespacedName, updateFunc func(*appsv1.Deployment)) (appsv1.Deployment, error) {
	deployment, err := getUpdater.GetDeployment(ctx, nsName)
	if err != nil {
		return appsv1.Deployment{}, err
	}
	// apply the function on the most recent version of the resource
	updateFunc(&deployment)
	return getUpdater.UpdateDeployment(ctx, deployment)
}

func IsReady(deployment appsv1.Deployment, expectedReplicas int) bool {
	allUpdated := int32(expectedReplicas) == deployment.Status.UpdatedReplicas
	allReady := int32(expectedReplicas) == deployment.Status.ReadyReplicas
	return allUpdated && allReady
}
