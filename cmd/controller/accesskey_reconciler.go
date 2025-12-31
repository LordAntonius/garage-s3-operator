package main

import (
	"context"
	"time"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// accessKeyReconciler is a reconciler for GarageS3AccessKey resources.
type accessKeyReconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

// AccessKeyExists checks if an access key with the given name exists in Garage S3.
func (r *accessKeyReconciler) AccessKeyExists(apiCtx context.Context, garageClient *garage.APIClient, keyName string) (string, error) {
	keys, _, err := garageClient.AccessKeyAPI.ListKeys(apiCtx).Execute()
	if err != nil {
		return "", err
	}

	for _, key := range keys {
		if key.Name == keyName {
			return key.Id, nil
		}
	}
	return "", nil
}

// Return Secret Access Key
func (r *accessKeyReconciler) GetSecretAccessKey(ctx context.Context, namespace string, secretName string) bool {

	_, err := r.kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

func (r *accessKeyReconciler) GenerateCreateKeyBody(ak v1.GarageS3AccessKey, keyname string) (garage.UpdateKeyRequestBody, error) {

	// Create Access Key in Garage S3
	keyPerm := garage.KeyPerm{
		CreateBucket: &ak.Spec.CanCreateBucket,
	}
	var expiration garage.NullableTime
	if ak.Spec.Expiration == "" {
		expiration = *garage.NewNullableTime(nil)
	} else {
		// Parse expiration time in RFC3339
		parsed, err := time.Parse(time.RFC3339, ak.Spec.Expiration)
		if err != nil {
			return garage.UpdateKeyRequestBody{}, err
		}
		expiration = *garage.NewNullableTime(&parsed)
	}

	keyReq := garage.UpdateKeyRequestBody{
		Name:         *garage.NewNullableString(&keyname),
		Allow:        *garage.NewNullableKeyPerm(&keyPerm),
		Deny:         *garage.NewNullableKeyPerm(nil),
		Expiration:   expiration,
		NeverExpires: &ak.Spec.NeverExpires,
	}

	return keyReq, nil
}

func (r *accessKeyReconciler) UpdateStatus(ctx context.Context, secretName string, status metav1.ConditionStatus, reason string, message string, instance *v1.GarageS3AccessKey) {
	instance.Status.Secret = secretName
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
		log.Log.Error(err, "Failed to update GarageS3AccessKey status")
	}
}

// Reconcile performs reconciliation for GarageS3AccessKey.
func (r *accessKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("GarageS3AccessKey", req.NamespacedName)

	const finalizerName = "garage-s3-operator.abucquet.com/finalizer"

	// Fetch the GarageS3AccessKey instance
	ak := &v1.GarageS3AccessKey{}
	if err := r.Get(ctx, req.NamespacedName, ak); err != nil {
		// If the object no longer exists, nothing to do
		if client.IgnoreNotFound(err) == nil {
			log.Info("GarageS3AccessKey not found, nothing to do")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get GarageS3AccessKey")
		return ctrl.Result{}, err
	}

	// If the object is being deleted, handle finalizer cleanup
	if !ak.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(ak, finalizerName) {
			// Perform external cleanup: delete the access key in Garage S3 if present
			instanceRef := ak.Spec.InstanceRef
			if instanceRef.Name != "" && instanceRef.Namespace != "" {
				instance := &v1.GarageS3Instance{}
				if err := r.Get(ctx, client.ObjectKey{Name: instanceRef.Name, Namespace: instanceRef.Namespace}, instance); err != nil {
					// If instance not found, ignore — nothing to cleanup remotely
				} else {
					garageClient, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
					if err == nil {
						accessKeyID, err := r.AccessKeyExists(apiCtx, garageClient, ak.Name)
						if err == nil && accessKeyID != "" {
							if _, err := garageClient.AccessKeyAPI.DeleteKey(apiCtx).Id(accessKeyID).Execute(); err != nil {
								log.Error(err, "Failed to delete external access key during finalizer cleanup", "accessKeyId", accessKeyID)
								return ctrl.Result{}, err
							}
							log.Info("Deleted external access key during finalizer cleanup", "accessKeyId", accessKeyID)
						}
					} else {
						log.Error(err, "Failed to create Garage client for finalizer cleanup", "InstanceRef", instanceRef)
						return ctrl.Result{}, err
					}
				}
			}

			// Remove finalizer and update resource
			controllerutil.RemoveFinalizer(ak, finalizerName)
			if err := r.Update(ctx, ak); err != nil {
				log.Error(err, "Failed to remove finalizer from GarageS3AccessKey")
				return ctrl.Result{}, err
			}
		}
		// Nothing more to do for deleted object
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is present on non-deleted objects
	if !controllerutil.ContainsFinalizer(ak, finalizerName) {
		controllerutil.AddFinalizer(ak, finalizerName)
		if err := r.Update(ctx, ak); err != nil {
			log.Error(err, "Failed to add finalizer to GarageS3AccessKey")
			return ctrl.Result{}, err
		}
	}

	keyName := ak.ObjectMeta.Name

	// Fetch the associated GarageS3Instance and create Garage Client
	instanceRef := ak.Spec.InstanceRef
	instance := &v1.GarageS3Instance{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      instanceRef.Name,
		Namespace: instanceRef.Namespace,
	}, instance); err != nil {
		log.Error(err, "Failed to get associated GarageS3Instance", "InstanceRef", instanceRef)
		r.UpdateStatus(ctx, "", metav1.ConditionFalse, "InstanceNotFound", "Associated GarageS3Instance not found", ak)
		return ctrl.Result{}, err
	}
	garageClient, apiCtx, err := CreateGarageClient(r.kubeClient, instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client for associated instance", "InstanceRef", instanceRef)
		r.UpdateStatus(ctx, "", metav1.ConditionFalse, "GarageClientError", "Could not create Garage S3 client", ak)
		return ctrl.Result{}, err
	}

	// Check if Access Key already exists in Garage S3
	var secretKey string
	accessKey, err := r.AccessKeyExists(apiCtx, garageClient, keyName)
	if err != nil {
		log.Error(err, "Failed to check if Access Key exists", "KeyName", keyName)
		r.UpdateStatus(ctx, "", metav1.ConditionUnknown, "UnknownGarageState", "Failed to check if Access Key exists", ak)
		return ctrl.Result{}, err
	}
	if accessKey == "" {
		// Create Access Key in Garage S3
		req, err := r.GenerateCreateKeyBody(*ak, keyName)
		if err != nil {
			log.Error(err, "Failed to generate Access Key creation body", "KeyName", keyName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "SyntaxError", "Error in json spec", ak)
			return ctrl.Result{}, err
		}
		keyInfo, _, err := garageClient.AccessKeyAPI.CreateKey(apiCtx).Body(req).Execute()
		if err != nil {
			log.Error(err, "Failed to create Access Key in Garage S3", "KeyName", keyName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "GarageClientError", "Failed to create Access Key in Garage S3", ak)
			return ctrl.Result{}, err
		}

		accessKey = keyInfo.AccessKeyId
		secretKey = keyInfo.GetSecretAccessKey()

		log.Info("Created Access Key in Garage S3", "KeyName", keyName, "AccessKey", keyInfo)
	} else {
		// Update Access Key in Garage S3
		cReq, err := r.GenerateCreateKeyBody(*ak, keyName)
		if err != nil {
			log.Error(err, "Failed to generate Access Key update body", "KeyName", keyName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "SyntaxError", "Error in json spec", ak)
			return ctrl.Result{}, err
		}
		req := garage.UpdateKeyRequestBody{
			Name:         cReq.Name,
			Allow:        cReq.Allow,
			Deny:         cReq.Deny,
			Expiration:   cReq.Expiration,
			NeverExpires: cReq.NeverExpires,
		}
		keyInfo, _, err := garageClient.AccessKeyAPI.UpdateKey(apiCtx).Id(accessKey).UpdateKeyRequestBody(req).Execute()
		if err != nil {
			log.Error(err, "Failed to update Access Key in Garage S3", "KeyName", keyName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "GarageClientError", "Failed to update Access Key in Garage S3", ak)
			return ctrl.Result{}, err
		}

		accessKey = keyInfo.AccessKeyId
		secretKey = keyInfo.GetSecretAccessKey()
		if secretKey == "" {
			res, _, err := garageClient.AccessKeyAPI.GetKeyInfo(apiCtx).Id(accessKey).ShowSecretKey(true).Execute()
			if err != nil {
				log.Error(err, "Failed to retrieve Secret Key for Access Key", "KeyName", keyName)
				r.UpdateStatus(ctx, "", metav1.ConditionFalse, "GarageClientError", "Failed to retrieve Secret Key for Access Key", ak)
				return ctrl.Result{}, err
			}
			secretKey = res.GetSecretAccessKey()
		}

		log.Info("Updated Access Key in Garage S3", "KeyName", keyName, "AccessKey", keyInfo)
	}

	// Check if the corresponding Kubernetes Secret exists
	secretName := ak.Name + "-gs3ak"
	secret, err := r.kubeClient.CoreV1().Secrets(ak.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		// Create Kubernetes Secret with Access Key and Secret Key
		secretData := map[string][]byte{
			"AWS_ACCESS_KEY": []byte(accessKey),
			"AWS_SECRET_KEY": []byte(secretKey),
		}

		sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ak.Namespace,
				Annotations: map[string]string{
					"managed-by": "garage-s3-operator",
				},
			},
			Data: secretData,
		}
		// set owner reference so Secret is garbage-collected with the GarageS3AccessKey
		if err := controllerutil.SetControllerReference(ak, sec, r.scheme); err != nil {
			log.Error(err, "Failed to set owner reference on Secret", "SecretName", secretName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "OwnerReferenceError", "Failed to set owner reference on Secret", ak)
			return ctrl.Result{}, err
		}

		_, err := r.kubeClient.CoreV1().Secrets(ak.Namespace).Create(ctx, sec, metav1.CreateOptions{})
		if err != nil {
			log.Error(err, "Failed to create Kubernetes Secret for Access Key", "SecretName", secretName)
			r.UpdateStatus(ctx, "", metav1.ConditionFalse, "KubernetesError", "Failed to create Kubernetes Secret for Access Key", ak)
			return ctrl.Result{}, err
		}
		log.Info("Created Kubernetes Secret for Access Key", "SecretName", secretName)
	} else {
		// Update existing Kubernetes Secret if needed
		updated := false
		// ensure managed-by annotation is present
		if secret.ObjectMeta.Annotations == nil {
			secret.ObjectMeta.Annotations = map[string]string{}
		}
		if secret.ObjectMeta.Annotations["managed-by"] != "garage-s3-operator" {
			secret.ObjectMeta.Annotations["managed-by"] = "garage-s3-operator"
			updated = true
		}

		if string(secret.Data["AWS_ACCESS_KEY"]) != accessKey {
			secret.Data["AWS_ACCESS_KEY"] = []byte(accessKey)
			updated = true
		}
		if string(secret.Data["AWS_SECRET_KEY"]) != secretKey {
			secret.Data["AWS_SECRET_KEY"] = []byte(secretKey)
			updated = true
		}
		if updated {
			_, err := r.kubeClient.CoreV1().Secrets(ak.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				log.Error(err, "Failed to update Kubernetes Secret for Access Key", "SecretName", secretName)
				r.UpdateStatus(ctx, "", metav1.ConditionFalse, "KubernetesError", "Failed to update Kubernetes Secret for Access Key", ak)
				return ctrl.Result{}, err
			}
			log.Info("Updated Kubernetes Secret for Access Key", "SecretName", secretName)
		}
	}

	r.UpdateStatus(ctx, secretName, metav1.ConditionTrue, "Ready", "Access Key is ready", ak)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *accessKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.GarageS3AccessKey{}).
		Complete(r)
}
