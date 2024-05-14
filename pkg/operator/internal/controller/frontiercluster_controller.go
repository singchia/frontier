/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	frontierv1alpha1 "github.com/singchia/frontier/operator/api/v1alpha1"
	kubeclient "github.com/singchia/frontier/operator/pkg/kube/client"
	"github.com/singchia/frontier/operator/pkg/util/result"
	"github.com/singchia/frontier/operator/pkg/util/status"
	appsv1 "k8s.io/api/apps/v1"
)

// FrontierClusterReconciler reconciles a FrontierCluster object
type FrontierClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	client kubeclient.Client
}

func NewReconciler(mgr manager.Manager) *FrontierClusterReconciler {
	mgrClient := mgr.GetClient()
	return &FrontierClusterReconciler{
		Client: mgrClient,
		Scheme: mgr.GetScheme(),
		client: kubeclient.NewClient(mgrClient),
	}
}

//+kubebuilder:rbac:groups=frontier.singchia.io,resources=frontierclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=frontier.singchia.io,resources=frontierclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=frontier.singchia.io,resources=frontierclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;pods;secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FrontierCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *FrontierClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	frontiercluster := frontierv1alpha1.FrontierCluster{}
	if err := r.Get(ctx, req.NamespacedName, &frontiercluster); err != nil {
		log.Error(err, "get frontier cluster error")
		return result.Failed()
	}

	log.Info("Ensuring the service exists")
	if err := r.ensureService(ctx, frontiercluster); err != nil {
		return status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
			withMessage(Error, fmt.Sprintf("Error ensuring services: %s", err)).
			withFailedPhase())
	}

	log.Info("Ensuring the tls exists")
	if err := r.ensureTLS(ctx, frontiercluster); err != nil {
		return status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
			withMessage(Error, fmt.Sprintf("Error ensuring tls secret: %s", err)).
			withFailedPhase())
	}

	log.Info("Ensuring the deployment exists")
	ready, err := r.ensureDeployment(ctx, frontiercluster)
	if err != nil {
		return status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
			withMessage(Error, fmt.Sprintf("Error deploying Deployment: %s", err)).
			withFailedPhase())
	}

	if !ready {
		return status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
			withMessage(Info, "Deployment is not yet ready, retrying in 10 seconds").
			withPendingPhase(10))
	}

	res, err := status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().withRunningPhase())
	if err != nil {
		log.Error(err, "Error updating the status of FrontierCluster resource")
		return res, err
	}
	if res.RequeueAfter > 0 || res.Requeue {
		log.Info("Requeuing reconciliation")
		return res, nil
	}

	log.Info("Successfully finished reconciliation")
	// TODO record in annotations
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FrontierClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frontierv1alpha1.FrontierCluster{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
