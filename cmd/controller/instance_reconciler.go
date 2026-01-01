package main

import (
	"context"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type instance_reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

func (r *instance_reconciler) UpdateStatus(ctx context.Context, status metav1.ConditionStatus, reason string, message string, instance *v1.GarageS3Instance) {
	cond := metav1.Condition{
		Type:    "Ready",
		Status:  status,
		Reason:  reason,
		Message: message,
	}
	// Replace existing Ready condition if present, preserve LastTransitionTime when status unchanged
	found := false
	for i := range instance.Status.Conditions {
		if instance.Status.Conditions[i].Type == "Ready" {
			if instance.Status.Conditions[i].Status == cond.Status {
				cond.LastTransitionTime = instance.Status.Conditions[i].LastTransitionTime
			} else {
				cond.LastTransitionTime = metav1.Now()
			}
			instance.Status.Conditions[i] = cond
			found = true
			break
		}
	}
	if !found {
		cond.LastTransitionTime = metav1.Now()
		instance.Status.Conditions = append(instance.Status.Conditions, cond)
	}

	if err := r.Status().Update(ctx, instance); err != nil {
		log.Log.Error(err, "Failed to update GarageS3Instance status")
	}
}

func (r *instance_reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("GarageS3Instance", req.NamespacedName)

	instance := &v1.GarageS3Instance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	// If object does not exist, it means deletion
	if err != nil {
		log.Info("Deleted GarageS3Instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)

		// TODO: delete associated resources
	}

	// Create client to Garage S3 instance
	client, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client")
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageClientError", "Failed to create Garage S3 client", instance)
		return ctrl.Result{}, err
	}

	// Test connection to Garage S3 instance and get status
	health, _, err := client.ClusterAPI.GetClusterHealth(apiCtx).Execute()
	if err != nil {
		log.Error(err, "Failed to connect to Garage S3 instance")
		r.UpdateStatus(ctx, metav1.ConditionFalse, "ConnectionError", "Failed to connect to Garage S3 instance", instance)
		return ctrl.Result{}, err
	}
	log.Info("Connected to Garage S3 instance", "status", health.Status)

	// Update instance status with Connected condition
	r.UpdateStatus(ctx, metav1.ConditionTrue, "Connected", "Successfully connected to Garage S3 instance", instance)

	return ctrl.Result{}, nil
}
