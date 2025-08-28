package utils

import (
	"fmt"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
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
	var (
		config        *rest.Config
		err           error
		clientset     *kubernetes.Clientset
		dynamicClient dynamic.Interface
	)

	// Try to use kubeconfig first (local dev)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}

	if _, err := os.Stat(kubeconfig); err == nil {
		// kubeconfig file exists → local development
		rawConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load kubeconfig: %v", err)
		}
		if contextName == "" {
			contextName = rawConfig.CurrentContext
		}
		ctxContext := rawConfig.Contexts[contextName]
		if ctxContext == nil {
			return nil, nil, fmt.Errorf("failed to find context '%s'", contextName)
		}
		clientConfig := clientcmd.NewDefaultClientConfig(
			*rawConfig,
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			},
		)
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create restconfig: %v", err)
		}
	} else {
		// kubeconfig not found → try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get in-cluster config: %v", err)
		}
	}

	// Create clients
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}
	return clientset, dynamicClient, nil
}
