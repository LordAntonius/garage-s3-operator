package main

import (
	"context"
	"fmt"
	"time"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AccessKeyPerm struct {
	Name        string
	AccessKeyID string
	Owner       bool
	Read        bool
	Write       bool
}

type bucket_reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

const bucketFinalizer = "garage.abucquet.com/bucket-finalizer"

const (
	bucketRequeueInterval      = 5 * time.Minute
	bucketErrorRequeueInterval = 30 * time.Second
)

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

func (r *bucket_reconciler) GetAccessKeyForName(name string, namespace string) (string, error) {
	// Look for GarageS3AccessKey with the given name, in the same namespace as the bucket
	accessKey := &v1.GarageS3AccessKey{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, accessKey)
	if err != nil {
		return "", err
	}
	// Retrieve the AccessKeyID from the associated secret
	secretName := accessKey.Status.Secret
	secret, err := r.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	// AccessKeyID is stored in "AWS_ACCESS_KEY" key in the secret data
	accessKeyID := string(secret.Data["AWS_ACCESS_KEY"])
	return accessKeyID, nil
}

func (r *bucket_reconciler) GetAllBucketPermInfo(bucket *v1.GarageS3Bucket) ([]AccessKeyPerm, error) {
	var perms []AccessKeyPerm
	oneNotFoundErr := false
	for _, p := range bucket.Spec.Permissions {
		accessKeyID, err := r.GetAccessKeyForName(p.AccessKeyName, bucket.Namespace)
		if err != nil {
			oneNotFoundErr = true
			continue
		}
		perm := AccessKeyPerm{
			Name:        p.AccessKeyName,
			AccessKeyID: accessKeyID,
			Owner:       p.Owner,
			Read:        p.Read,
			Write:       p.Write,
		}
		perms = append(perms, perm)
	}
	if oneNotFoundErr {
		return perms, fmt.Errorf("one or more AccessKeys not found for bucket permissions")
	}
	return perms, nil
}

// ptrBoolVal returns the value of a *bool, treating nil as false.
func ptrBoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
func boolPtr(b bool) *bool { v := b; return &v }

func (r *bucket_reconciler) GetBucketPermissionChangeRequests(bucket *v1.GarageS3Bucket, bucketInfo *garage.GetBucketInfoResponse) ([]garage.BucketKeyPermChangeRequest, []garage.BucketKeyPermChangeRequest, error) {

	accessKeyInfos, err := r.GetAllBucketPermInfo(bucket) // Error if one or more AccessKeys not found

	var allowRequests []garage.BucketKeyPermChangeRequest
	var denyRequests []garage.BucketKeyPermChangeRequest

	for _, ak := range accessKeyInfos {
		found := false
		foundPerm := garage.ApiBucketKeyPerm{}
		for _, existingPerm := range bucketInfo.Keys {
			// Found existing permission for this access key
			if ak.AccessKeyID == existingPerm.AccessKeyId {
				found = true
				foundPerm = existingPerm.Permissions
				break
			}
		}
		if !found {
			// Not found, create request
			// make local copies so each pointer refers to a distinct value
			owner := ak.Owner
			read := ak.Read
			write := ak.Write
			req := garage.BucketKeyPermChangeRequest{
				AccessKeyId: ak.AccessKeyID,
				BucketId:    bucketInfo.Id,
				Permissions: garage.ApiBucketKeyPerm{
					Owner: &owner,
					Read:  &read,
					Write: &write,
				},
			}
			allowRequests = append(allowRequests, req)
		} else {
			// Found, check for updates
			add := false
			rem := false
			allowReq := garage.BucketKeyPermChangeRequest{
				AccessKeyId: ak.AccessKeyID,
				BucketId:    bucketInfo.Id,
				Permissions: garage.ApiBucketKeyPerm{
					Owner: boolPtr(false),
					Read:  boolPtr(false),
					Write: boolPtr(false),
				},
			}
			denyReq := garage.BucketKeyPermChangeRequest{
				AccessKeyId: ak.AccessKeyID,
				BucketId:    bucketInfo.Id,
				Permissions: garage.ApiBucketKeyPerm{
					Owner: boolPtr(false),
					Read:  boolPtr(false),
					Write: boolPtr(false),
				},
			}
			if ak.Owner != ptrBoolVal(foundPerm.Owner) {
				if ak.Owner {
					allowReq.Permissions.Owner = boolPtr(true)
					add = true
				} else {
					denyReq.Permissions.Owner = boolPtr(true)
					rem = true
				}
			}
			if ak.Read != ptrBoolVal(foundPerm.Read) {
				if ak.Read {
					allowReq.Permissions.Read = boolPtr(true)
					add = true
				} else {
					denyReq.Permissions.Read = boolPtr(true)
					rem = true
				}
			}
			if ak.Write != ptrBoolVal(foundPerm.Write) {
				if ak.Write {
					allowReq.Permissions.Write = boolPtr(true)
					add = true
				} else {
					denyReq.Permissions.Write = boolPtr(true)
					rem = true
				}
			}
			if add {
				allowRequests = append(allowRequests, allowReq)
			}
			if rem {
				denyRequests = append(denyRequests, denyReq)
			}
		}
	}

	// Run through existing permissions to find any to remove
	for _, existingPerm := range bucketInfo.Keys {
		found := false
		for _, ak := range accessKeyInfos {
			if ak.AccessKeyID == existingPerm.AccessKeyId {
				found = true
				break
			}
		}
		if !found {
			// Existing permission not found in desired permissions, remove it
			req := garage.BucketKeyPermChangeRequest{
				AccessKeyId: existingPerm.AccessKeyId,
				BucketId:    bucketInfo.Id,
				Permissions: existingPerm.Permissions,
			}
			denyRequests = append(denyRequests, req)
		}
	}

	return allowRequests, denyRequests, err
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

func (r *bucket_reconciler) BucketCleanup(ctx context.Context, bucket *v1.GarageS3Bucket) error {

	// Being deleted: perform finalization then remove finalizer
	if controllerutil.ContainsFinalizer(bucket, bucketFinalizer) {
		// Try to delete the bucket on Garage S3
		instanceRef := bucket.Spec.InstanceRef
		instance := &v1.GarageS3Instance{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      instanceRef.Name,
			Namespace: instanceRef.Namespace,
		}, instance); err != nil {
			// If associated instance can't be found, log and continue to remove finalizer
			return fmt.Errorf("failed to get associated GarageS3Instance while finalizing; will remove finalizer to avoid blocking deletion: %w", err)
		} else {
			garageClient, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
			if err != nil {
				return fmt.Errorf("failed to create Garage S3 client while finalizing; requeueing: %w", err)
			}
			// Check bucket presence and delete if exists
			bucketID, err := r.BucketExists(apiCtx, garageClient, bucket.Name)
			if err != nil {
				return fmt.Errorf("failed to check bucket existence while finalizing; requeueing: %w", err)
			}
			if bucketID != "" {
				resp, err := garageClient.BucketAPI.DeleteBucket(apiCtx).Id(bucketID).Execute()
				if err != nil {
					switch resp.StatusCode {
					case 400:
						return fmt.Errorf("failed to delete bucket in Garage S3 during finalization; requeueing: %w", err)
					case 404:
						// Ignore bucket not found
					default:
						return fmt.Errorf("failed to delete bucket in Garage S3 during finalization; requeueing: %w", err)
					}
				}
			}
		}
		// Remove finalizer so Kubernetes can delete the object
		controllerutil.RemoveFinalizer(bucket, bucketFinalizer)
		if err := r.Update(ctx, bucket); err != nil {
			return err
		}
	}
	return nil
}

func (r *bucket_reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("GarageS3Bucket", req.NamespacedName)

	// Fetching instance in K8s
	bucket := &v1.GarageS3Bucket{}
	if err := r.Get(ctx, req.NamespacedName, bucket); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Finalizer handling: add when not deleting, run cleanup when deleting
	if bucket.ObjectMeta.DeletionTimestamp.IsZero() {
		// Not being deleted: ensure finalizer present
		if !controllerutil.ContainsFinalizer(bucket, bucketFinalizer) {
			controllerutil.AddFinalizer(bucket, bucketFinalizer)
			if err := r.Update(ctx, bucket); err != nil {
				log.Error(err, "Failed to add finalizer to GarageS3Bucket")
				r.UpdateStatus(ctx, metav1.ConditionFalse, "KubernetesError", "Failed to add finalizer", bucket)
				return ctrl.Result{}, err
			}
		}
	} else {
		// Being deleted: perform finalization then remove finalizer
		if err := r.BucketCleanup(ctx, bucket); err != nil {
			log.Error(err, "Failed to finalize GarageS3Bucket")
			r.UpdateStatus(ctx, metav1.ConditionFalse, "FinalizationError", "Failed to finalize bucket", bucket)
			return ctrl.Result{}, err
		}
		log.Info("Deleted bucket", "BucketName", bucket.Name)
		return ctrl.Result{}, nil
	}

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
		return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
	}
	garageClient, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client for associated instance", "InstanceRef", instanceRef)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageClientError", "Could not create Garage S3 client", bucket)
		return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
	}

	// Check if bucket exists in Garage S3
	bucketID, err := r.BucketExists(apiCtx, garageClient, bucket.Name)
	if err != nil {
		log.Error(err, "Failed to check if bucket exists in Garage S3", "BucketName", bucket.Name)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when calling Garage S3 API", bucket)
		return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
	}

	// Create bucket if not exists
	var bucketInfo *garage.GetBucketInfoResponse = nil
	if bucketID == "" {
		bucketInfo, err = r.CreateBucket(apiCtx, garageClient, &bucket.Name)
		if err != nil {
			log.Error(err, "Failed to create bucket in Garage S3", "BucketName", bucket.Name)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when creating bucket in Garage S3", bucket)
			return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
		}
		log.Info("Created bucket in Garage S3", "BucketName", bucket.Name, "BucketID", bucketInfo.Id)
	} else {
		// Bucket exists, let's sync its info
		bucketInfo, _, err = garageClient.BucketAPI.GetBucketInfo(apiCtx).Id(bucketID).Execute()
		if err != nil {
			log.Error(err, "Failed to get bucket info from Garage S3", "BucketName", bucket.Name, "BucketID", bucketID)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when retrieving bucket info from Garage S3", bucket)
			return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
		}
	}

	// Update Bucket parameters
	updateBucketReq := garage.UpdateBucketRequestBody{
		Quotas:        r.GetBucketQuota(bucket),
		WebsiteAccess: r.GetBucketWebsiteAccess(bucket),
	}
	garageClient.BucketAPI.UpdateBucket(apiCtx).Id(bucketInfo.Id).UpdateBucketRequestBody(updateBucketReq).Execute()

	// Update Bucket aliases
	aliases := bucket.Spec.AdditionalAliases
	aliases = append(aliases, bucket.Name) // Ensure main alias is present
	for _, alias := range bucketInfo.GlobalAliases {
		// Remove aliases that are not in the spec
		found := false
		for _, desiredAlias := range aliases {
			if alias == desiredAlias {
				found = true
				break
			}
		}
		if !found {
			aliasReq := garage.RemoveBucketAliasRequest{
				GlobalAlias: alias,
				BucketId:    bucketInfo.Id,
			}
			_, _, err := garageClient.BucketAliasAPI.RemoveBucketAlias(apiCtx).RemoveBucketAliasRequest(aliasReq).Execute()
			if err != nil {
				log.Error(err, "Failed to remove bucket global alias in Garage S3", "BucketName", bucket.Name, "BucketID", bucketInfo.Id, "Alias", alias)
				r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when removing bucket alias in Garage S3", bucket)
				return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
			}
		}
	}
	for _, desiredAlias := range aliases {
		// Add aliases that are in the spec but not in Garage S3
		found := false
		for _, alias := range bucketInfo.GlobalAliases {
			if alias == desiredAlias {
				found = true
				break
			}
		}
		if !found {
			aliasReq := garage.AddBucketAliasRequest{
				GlobalAlias: desiredAlias,
				BucketId:    bucketInfo.Id,
			}
			_, _, err := garageClient.BucketAliasAPI.AddBucketAlias(apiCtx).AddBucketAliasRequest(aliasReq).Execute()
			if err != nil {
				log.Error(err, "Failed to add bucket global alias in Garage S3", "BucketName", bucket.Name, "BucketID", bucketInfo.Id, "Alias", desiredAlias)
				r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when adding bucket alias in Garage S3", bucket)
				return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
			}
		}
	}

	// Handle permissions
	allowReq, denyReq, err := r.GetBucketPermissionChangeRequests(bucket, bucketInfo)
	// In case of error, some AccessKeys were not found, but we can still process the others
	// error means it is needed to requeue later
	for _, req := range allowReq {
		_, _, err := garageClient.PermissionAPI.AllowBucketKey(apiCtx).Body(req).Execute()
		if err != nil {
			log.Error(err, "Failed to allow bucket key permission", "BucketName", bucket.Name, "BucketID", bucketInfo.Id, "AccessKeyID", req.AccessKeyId)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when updating bucket permissions in Garage S3", bucket)
			return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
		}
	}
	for _, req := range denyReq {
		_, _, err := garageClient.PermissionAPI.DenyBucketKey(apiCtx).Body(req).Execute()
		if err != nil {
			log.Error(err, "Failed to deny bucket key permission", "BucketName", bucket.Name, "BucketID", bucketInfo.Id, "AccessKeyID", req.AccessKeyId)
			r.UpdateStatus(ctx, metav1.ConditionFalse, "GarageAPIError", "Error when updating bucket permissions in Garage S3", bucket)
			return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
		}
	}

	if err != nil {
		log.Error(err, "One or more access keys not found for bucket permissions, will retry", "BucketName", bucket.Name)
		r.UpdateStatus(ctx, metav1.ConditionFalse, "PermissionsIncomplete", "One or more access keys not found for bucket permissions", bucket)
		return ctrl.Result{RequeueAfter: bucketErrorRequeueInterval}, err
	}

	r.UpdateStatus(ctx, metav1.ConditionTrue, "Ready", "Bucket is ready", bucket)
	return ctrl.Result{RequeueAfter: bucketRequeueInterval}, nil
}
