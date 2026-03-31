package k8s

import (
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ConfigManager handles Kubernetes configuration operations
type ConfigManager struct{}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

// ListContexts returns the current context and available contexts
func (cm *ConfigManager) ListContexts() (string, []string, error) {
	config, err := cm.getKubeConfig()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	currentContext := config.CurrentContext
	var contexts []string

	for name := range config.Contexts {
		if strings.Contains(name, "wds") {
			contexts = append(contexts, name)
		}
	}

	return currentContext, contexts, nil
}

// getKubeConfig loads the kubeconfig file
func (cm *ConfigManager) getKubeConfig() (*api.Config, error) {
	kubeconfig := getKubeConfigPath()
	if kubeconfig == "" {
		return nil, fmt.Errorf("kubeconfig path not found")
	}

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfig, err)
	}

	return config, nil
}

// ValidateContext validates if a context exists in the kubeconfig
func (cm *ConfigManager) ValidateContext(contextName string) error {
	config, err := cm.getKubeConfig()
	if err != nil {
		return err
	}

	if contextName == "" {
		contextName = config.CurrentContext
	}

	if config.Contexts[contextName] == nil {
		return fmt.Errorf("context '%s' not found in kubeconfig", contextName)
	}

	return nil
}
