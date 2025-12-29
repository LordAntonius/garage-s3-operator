package main

import (
	"context"
	"fmt"
	"strconv"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

func RetrieveAdminToken(kubeClient *kubernetes.Clientset, namespace string, secretName string) (string, error) {
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	tokenBytes, exists := secret.Data["token"]
	if !exists {
		return "", fmt.Errorf("token key not found in secret %s", secretName)
	}
	return string(tokenBytes), nil
}

// CreateGarageClient creates a Garage API client for the given instance
func (r *reconciler) CreateGarageClient(instance *v1.GarageS3Instance) (*garage.APIClient, context.Context, error) {

	// Setup Garage S3 client configuration
	configuration := garage.NewConfiguration()
	configuration.Host = instance.Spec.Url + ":" + strconv.Itoa(instance.Spec.Port)
	client := garage.NewAPIClient(configuration)

	// Retrieve admin token from Kubernetes Secret (use reconciler's kubeClient)
	namespace := instance.ObjectMeta.Namespace
	secretName := instance.Spec.AdminTokenSecret
	adminToken, err := RetrieveAdminToken(r.kubeClient, namespace, secretName)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.WithValue(context.Background(), garage.ContextAccessToken, adminToken)
	return client, ctx, nil
}

func (r *reconciler) SetGarageS3InstanceStatus(status string, message string, instance *v1.GarageS3Instance) {
	condition := v1.GarageS3Condition{
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	instance.Status.Conditions = append(instance.Status.Conditions, condition)
	err := r.Status().Update(context.Background(), instance)
	if err != nil {
		log.Log.Error(err, "Failed to update GarageS3Instance status")
	}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
	client, apiCtx, err := r.CreateGarageClient(instance)
	if err != nil {
		log.Error(err, "Failed to create Garage S3 client")
		message := fmt.Sprintf("Failed to create Garage S3 client: %v", err)
		r.SetGarageS3InstanceStatus("error", message, instance)
		return ctrl.Result{}, err
	}

	// Test connection to Garage S3 instance and get status
	health, _, err := client.ClusterAPI.GetClusterHealth(apiCtx).Execute()
	if err != nil {
		log.Error(err, "Failed to connect to Garage S3 instance")
		message := fmt.Sprintf("Failed to connect to Garage S3 instance: %v", err)
		r.SetGarageS3InstanceStatus("error", message, instance)
		return ctrl.Result{}, err
	}
	log.Info("Connected to Garage S3 instance", "status", health.Status)

	// Update instance status with Connected condition
	message := fmt.Sprintf("Nodes: %d/%d", health.StorageNodesUp, health.StorageNodes)
	r.SetGarageS3InstanceStatus(health.Status, message, instance)

	return ctrl.Result{}, nil
}
