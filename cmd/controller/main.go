package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	garageS3types "abucquet.com/garage-s3-operator/api/v1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(garageS3types.AddToScheme(scheme))
}

func getClientSetAndConfig() (clientset *kubernetes.Clientset, config *rest.Config, err error) {

	// if kube config doesn't exist, try incluster config
	kubeconfigFilePath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfigFilePath); errors.Is(err, os.ErrNotExist) {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
		if err != nil {
			return nil, nil, err
		}
	}

	// kubernetes client set
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, config, err
	}
	return clientset, config, nil
}

func main() {

	// Retrieve Kubernetes clientset
	clientset, config, err := getClientSetAndConfig()
	if err != nil {
		fmt.Printf("failed to get clientset: %v\n", err)
		return
	}

	// Set logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("setup")

	// Start controller manager
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&garageS3types.GarageS3Instance{}).
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
