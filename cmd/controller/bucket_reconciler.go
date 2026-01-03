package main

import (
	"context"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type bucket_reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

// Returns the bucket ID if the bucket exists, or empty string if not found
func (r *bucket_reconciler) BucketExists(apiCtx context.Context, garageClient *garage.APIClient, bucketName string) (string, error) {
	buckets, _, err := garageClient.BucketAPI.ListBuckets(apiCtx).Execute()
	if err != nil {
		return "", err
	}

	for _, bucket := range buckets {
		for _, alias := range bucket.GlobalAliases {
			if alias == bucketName {
				return bucket.Id, nil
			}
		}
		for _, alias := range bucket.LocalAliases {
			if alias.GetAlias() == bucketName {
				return bucket.Id, nil
			}
		}
	}
	return "", nil
}

func (r *bucket_reconciler) CreateBucket(apiCtx context.Context, garageClient *garage.APIClient, bucketName *string) (*garage.GetBucketInfoResponse, error) {

	request := garage.CreateBucketRequest{
		GlobalAlias: *garage.NewNullableString(bucketName),
		LocalAlias:  *garage.NewNullableCreateBucketLocalAlias(nil),
	}
	info, _, err := garageClient.BucketAPI.CreateBucket(apiCtx).CreateBucketRequest(request).Execute()
	return info, err
}

func (r *bucket_reconciler) GetBucketQuota(bucket *v1.GarageS3Bucket) garage.NullableApiBucketQuotas {
	if bucket.Spec.Quota == nil {
		return *garage.NewNullableApiBucketQuotas(nil)
	}

	quota := garage.ApiBucketQuotas{
		MaxObjects: *garage.NewNullableInt64(bucket.Spec.Quota.MaxObjects),
		MaxSize:    *garage.NewNullableInt64(bucket.Spec.Quota.MaxBytes),
	}
	return *garage.NewNullableApiBucketQuotas(&quota)
}

func (r *bucket_reconciler) GetBucketWebsiteAccess(bucket *v1.GarageS3Bucket) garage.NullableUpdateBucketWebsiteAccess {
	if bucket.Spec.WebsiteAccess == nil {
		return *garage.NewNullableUpdateBucketWebsiteAccess(nil)
	}

	wa := garage.UpdateBucketWebsiteAccess{
		Enabled:       bucket.Spec.WebsiteAccess.Enabled,
		IndexDocument: *garage.NewNullableString(&bucket.Spec.WebsiteAccess.IndexDocument),
		ErrorDocument: *garage.NewNullableString(&bucket.Spec.WebsiteAccess.ErrorDocument),
	}
	return *garage.NewNullableUpdateBucketWebsiteAccess(&wa)
}

func (r *bucket_reconciler) UpdateStatus(ctx context.Context, status metav1.ConditionStatus, reason string, message string, instance *v1.GarageS3Bucket) {

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
		log.Log.Error(err, "Failed to update GarageS3Bucket status")
	}
}

func (r *bucket_reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("GarageS3Bucket", req.NamespacedName)

	// Fetching instance in K8s
	bucket := &v1.GarageS3Bucket{}
	if err := r.Get(ctx, req.NamespacedName, bucket); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: implement finalizer logic here
	log.Info("TODO: implement GarageS3Bucket finalizer logic")

	// Create client to Garage S3 instance
	// Fetch the associated GarageS3Instance and create Garage Client
	instanceRef := bucket.Spec.InstanceRef
	instance := &v1.GarageS3Instance{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      instanceRef.Name,
		Namespace: instanceRef.Namespace,
	}, instance); err != nil {
		log.Error(err, "Failed to get associated GarageS3Instance", "InstanceRef", instanceRef)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "InstanceNotFound", "Associated GarageS3Instance not found", bucket)
		return ctrl.Result{}, err
	}
	garageClient, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client for associated instance", "InstanceRef", instanceRef)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageClientError", "Could not create Garage S3 client", bucket)
		return ctrl.Result{}, err
	}

	// Check if bucket exists in Garage S3
	bucketID, err := r.BucketExists(apiCtx, garageClient, bucket.Name)
	if err != nil {
		log.Error(err, "Failed to check if bucket exists in Garage S3", "BucketName", bucket.Name)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when calling Garage S3 API", bucket)
		return ctrl.Result{}, err
	}

	// Create bucket if not exists
	var bucketInfo *garage.GetBucketInfoResponse = nil
	if bucketID == "" {
		bucketInfo, err = r.CreateBucket(apiCtx, garageClient, &bucket.Name)
		if err != nil {
			log.Error(err, "Failed to create bucket in Garage S3", "BucketName", bucket.Name)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when creating bucket in Garage S3", bucket)
			return ctrl.Result{}, err
		}
		log.Info("Created bucket in Garage S3", "BucketName", bucket.Name, "BucketID", bucketInfo.Id)
	} else {
		// Bucket exists, let's sync its info
		bucketInfo, _, err = garageClient.BucketAPI.GetBucketInfo(apiCtx).Id(bucketID).Execute()
		if err != nil {
			log.Error(err, "Failed to get bucket info from Garage S3", "BucketName", bucket.Name, "BucketID", bucketID)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when retrieving bucket info from Garage S3", bucket)
			return ctrl.Result{}, err
		}
	}

	// Update Bucket parameters
	ubReq := garage.UpdateBucketRequestBody{
		Quotas:        r.GetBucketQuota(bucket),
		WebsiteAccess: r.GetBucketWebsiteAccess(bucket),
	}
	garageClient.BucketAPI.UpdateBucket(apiCtx).Id(bucketInfo.Id).UpdateBucketRequestBody(ubReq).Execute()

	return ctrl.Result{}, nil
}
