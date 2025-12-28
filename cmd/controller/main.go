package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func init() {

}

func getClientSet() (clientset *kubernetes.Clientset, err error) {
	var config *rest.Config
	// if kube config doesn't exist, try incluster config
	kubeconfigFilePath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfigFilePath); errors.Is(err, os.ErrNotExist) {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
		if err != nil {
			return nil, err
		}
	}

	// kubernetes client set
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func main() {

	// Retrieve Kubernetes clientset
	clienset, err := getClientSet()
	if err != nil {
		fmt.Printf("failed to get clientset: %v\n", err)
		return
	}
	fmt.Printf("clientset: %v\n", clienset)
}
