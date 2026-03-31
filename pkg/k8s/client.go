package k8s

import (
	"fmt"
	"os"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientManager handles Kubernetes client creation and management
type ClientManager struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewClientManager creates a new ClientManager
func NewClientManager() *ClientManager {
	return &ClientManager{}
}

// GetClients retrieves Kubernetes clientset and dynamic client for a specified context
func (cm *ClientManager) GetClients(contextName string) (*kubernetes.Clientset, dynamic.Interface, error) {
	if cm.clientset != nil && cm.dynamicClient != nil {
		return cm.clientset, cm.dynamicClient, nil
	}

	config, err := cm.getRestConfig(contextName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	cm.clientset = clientset
	cm.dynamicClient = dynamicClient

	return clientset, dynamicClient, nil
}

// getRestConfig creates a rest.Config for the given context
func (cm *ClientManager) getRestConfig(contextName string) (*rest.Config, error) {
	kubeconfig := getKubeConfigPath()

	// Check if kubeconfig file exists (local development)
	if _, err := os.Stat(kubeconfig); err == nil {
		return cm.getLocalConfig(kubeconfig, contextName)
	}

	// Try in-cluster config
	return rest.InClusterConfig()
}

// getLocalConfig creates config from local kubeconfig file
func (cm *ClientManager) getLocalConfig(kubeconfig, contextName string) (*rest.Config, error) {
	rawConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	if contextName == "" {
		contextName = rawConfig.CurrentContext
	}

	if rawConfig.Contexts[contextName] == nil {
		return nil, fmt.Errorf("context '%s' not found", contextName)
	}

	clientConfig := clientcmd.NewDefaultClientConfig(
		*rawConfig,
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		},
	)

	return clientConfig.ClientConfig()
}

// getKubeConfigPath returns the path to kubeconfig file
func getKubeConfigPath() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	if home := homeDir(); home != "" {
		return fmt.Sprintf("%s/.kube/config", home)
	}

	return ""
}

// homeDir retrieves the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}
