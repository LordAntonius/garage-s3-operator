package main

import (
	"context"
	"time"

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

const instanceFinalizer = "garage.abucquet.com/finalizer"

const (
	instanceRequeueInterval      = 5 * time.Minute
	instanceErrorRequeueInterval = 30 * time.Second
)

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

func (r *instance_reconciler) AddFinalizer(ctx context.Context, instance *v1.GarageS3Instance) error {
	if instance.ObjectMeta.DeletionTimestamp == nil {
		has := false
		for _, f := range instance.ObjectMeta.Finalizers {
			if f == instanceFinalizer {
				has = true
				break
			}
		}
		if !has {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, instanceFinalizer)
			return r.Update(ctx, instance)
		}
	}
	return nil
}

func (r *instance_reconciler) HasChildren(ctx context.Context, instance *v1.GarageS3Instance) (bool, error) {
	accessKeyList := &v1.GarageS3AccessKeyList{}
	if err := r.List(ctx, accessKeyList); err != nil {
		return true, err
	}
	for _, key := range accessKeyList.Items {
		if key.Spec.InstanceRef.Name == instance.Name && key.Spec.InstanceRef.Namespace == instance.Namespace {
			return true, nil
		}
	}

	bucketList := &v1.GarageS3BucketList{}
	if err := r.List(ctx, bucketList); err != nil {
		return true, err
	}
	for _, bucket := range bucketList.Items {
		if bucket.Spec.InstanceRef.Name == instance.Name && bucket.Spec.InstanceRef.Namespace == instance.Namespace {
			return true, nil
		}
	}
	return false, nil
}

func (r *instance_reconciler) RemoveFinalizer(ctx context.Context, instance *v1.GarageS3Instance) error {
	orig := instance.DeepCopyObject().(client.Object)
	newFinalizers := []string{}
	for _, f := range instance.ObjectMeta.Finalizers {
		if f != instanceFinalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	instance.ObjectMeta.Finalizers = newFinalizers
	return r.Patch(ctx, instance, client.MergeFrom(orig))
}

func (r *instance_reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("GarageS3Instance", req.NamespacedName)

	instance := &v1.GarageS3Instance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add finalizer if not present
	if err := r.AddFinalizer(ctx, instance); err != nil {
		log.Error(err, "Failed to add finalizer to Garage S3 instance")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if instance.ObjectMeta.DeletionTimestamp != nil {
		// Check for child resources
		hasChildren, err := r.HasChildren(ctx, instance)
		if err != nil {
			log.Error(err, "Failed to check for child resources")
			return ctrl.Result{}, err
		}
		if hasChildren {
			log.Info("Cannot delete Garage S3 instance, child resources exist")
			r.UpdateStatus(ctx, metav1.ConditionFalse, "ChildResourcesExist", "Cannot delete Garage S3 instance, child resources exist", instance)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		// Remove finalizer
		if err := r.RemoveFinalizer(ctx, instance); err != nil {
			log.Error(err, "Failed to remove finalizer from Garage S3 instance")
			return ctrl.Result{}, err
		}
		log.Info("Deleted Garage S3 instance")
		return ctrl.Result{}, nil
	}

	// Create client to Garage S3 instance
	client, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client")
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageClientError", "Failed to create Garage S3 client", instance)
		return ctrl.Result{RequeueAfter: instanceErrorRequeueInterval}, err
	}

	// Test connection to Garage S3 instance and get status
	health, _, err := client.ClusterAPI.GetClusterHealth(apiCtx).Execute()
	if err != nil {
		log.Error(err, "Failed to connect to Garage S3 instance")
		r.UpdateStatus(ctx, metav1.ConditionFalse, "ConnectionError", "Failed to connect to Garage S3 instance", instance)
		return ctrl.Result{RequeueAfter: instanceErrorRequeueInterval}, err
	}
	log.Info("Connected to Garage S3 instance", "status", health.Status)

	// Update instance status with Connected condition
	r.UpdateStatus(ctx, metav1.ConditionTrue, "Connected", "Successfully connected to Garage S3 instance", instance)

	return ctrl.Result{RequeueAfter: instanceRequeueInterval}, nil
}
