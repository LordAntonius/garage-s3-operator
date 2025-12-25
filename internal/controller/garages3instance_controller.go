/*
Copyright 2025.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	garages3operatorfrv1alpha1 "github.com/LordAntonius/garage-s3-operator/api/v1alpha1"
)

// GarageS3InstanceReconciler reconciles a GarageS3Instance object
type GarageS3InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=garage.s3operator.fr,resources=garages3instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=garage.s3operator.fr,resources=garages3instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=garage.s3operator.fr,resources=garages3instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GarageS3Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *GarageS3InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Reconciling GarageS3Instance", "NamespacedName", req.NamespacedName)

	// Get instance
	var garageS3Instance garages3operatorfrv1alpha1.GarageS3Instance
	if err := r.Get(ctx, req.NamespacedName, &garageS3Instance); err != nil {
		// Is a deletion event
		logger.Info("Deleting : GarageS3Instance", req.NamespacedName.Name)

		// TODO: Add finalizer logic for deleting related buckets and access keys in Garage S3 instance
		return ctrl.Result{}, nil
	}
	logger.Info("Fetched GarageS3Instance", "Spec", garageS3Instance.Spec)

	// Try to connect to Garage S3 instance

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GarageS3InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&garages3operatorfrv1alpha1.GarageS3Instance{}).
		Named("garages3instance").
		Complete(r)
}
