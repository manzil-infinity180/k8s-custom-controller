package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/manzil-infinity180/k8s-custom-controller/controller"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	"os"
	"strings"
	"time"
)

// homeDir retrieves the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

func ListContexts() (string, []string, error) {
	config, err := getKubeConfig()
	if err != nil {
		return "", nil, err
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
func getKubeConfig() (*api.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
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

func main() {
	// add your context in the docker-compose.yml
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, assuming environment variables are set")
	}
	context := os.Getenv("CONTEXT")
	fmt.Println(context)
	clientset, _, err := GetClientSetWithContext(context)
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
