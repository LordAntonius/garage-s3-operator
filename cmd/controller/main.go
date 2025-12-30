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
	"sigs.k8s.io/controller-runtime/pkg/healthz"
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
		fmt.Printf("Failed to get clientset: %v\n", err)
		return
	}

	// Set logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("Setup")

	// Start controller manager
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		return
	}
	// register simple health & readiness checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add healthz check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add readyz check")
		os.Exit(1)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&garageS3types.GarageS3Instance{}).
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "Unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}

}
