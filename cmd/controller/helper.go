package main

import (
	"context"
	"fmt"
	"strconv"

	v1 "abucquet.com/garage-s3-operator/api/v1"
	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

func CreateGarageClient(kubeClient *kubernetes.Clientset, instance *v1.GarageS3Instance) (*garage.APIClient, context.Context, error) {
	// Setup Garage S3 client configuration
	configuration := garage.NewConfiguration()
	configuration.Host = instance.Spec.Url + ":" + strconv.Itoa(instance.Spec.Port)
	client := garage.NewAPIClient(configuration)

	// Retrieve admin token from Kubernetes Secret
	namespace := instance.ObjectMeta.Namespace
	secretName := instance.Spec.AdminTokenSecret
	adminToken, err := RetrieveAdminToken(kubeClient, namespace, secretName)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.WithValue(context.Background(), garage.ContextAccessToken, adminToken)
	return client, ctx, nil
}
