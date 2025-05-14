package main

import (
	"fmt"
	"github.com/manzil-infinity180/k8s-custom-controller/controller"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

// homeDir retrieves the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

// GetClientSetWithContext retrieves a Kubernetes clientset and dynamic client for a specified context
func GetClientSetWithContext(contextName string) (*kubernetes.Clientset, dynamic.Interface, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}

	// Load the kubeconfig file
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Check if the specified context exists
	ctxContext := config.Contexts[contextName]
	if ctxContext == nil {
		return nil, nil, fmt.Errorf("failed to find context '%s'", contextName)
	}

	// Create config for the specified context
	clientConfig := clientcmd.NewDefaultClientConfig(
		*config,
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		},
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create restconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return clientset, dynamicClient, nil
}
func main() {
	// add your context here
	clientset, _, err := GetClientSetWithContext("cluster1")
	if err != nil {
		fmt.Println()
		fmt.Errorf("%s", err.Error())
	}

	ch := make(chan struct{})
	// factory
	factory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	c := controller.NewController(clientset, factory.Apps().V1().Deployments())
	factory.Start(ch)
	c.Run(ch)
	fmt.Println(factory)
	factory.Apps().V1().Deployments().Informer()
}
