package main

import (
	"fmt"
	"log"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/manzil-infinity180/k8s-custom-controller/internal/config"
	"github.com/manzil-infinity180/k8s-custom-controller/pkg/controller"
	"github.com/manzil-infinity180/k8s-custom-controller/pkg/k8s"
	"github.com/manzil-infinity180/k8s-custom-controller/pkg/types"
	"github.com/manzil-infinity180/k8s-custom-controller/pkg/webhook"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Using context: %s\n", cfg.Context)

	// Initialize Kubernetes clients
	clientManager := k8s.NewClientManager()
	clientset, _, err := clientManager.GetClients(cfg.Context)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes clients: %v", err)
	}

	// Start webhook server in a goroutine
	go func() {
		if err := startWebhookServer(cfg); err != nil {
			log.Fatalf("Failed to start webhook server: %v", err)
		}
	}()

	// Start the controller
	if err := startController(clientset); err != nil {
		log.Fatalf("Failed to start controller: %v", err)
	}
}

// startWebhookServer initializes and starts the admission webhook server
func startWebhookServer(cfg *types.ControllerConfig) error {
	// Initialize scanner
	scanner := webhook.NewScanner(cfg.TrivyServerURL)

	// Initialize validator
	validator := webhook.NewValidator(scanner)

	// Initialize and start server
	server := webhook.NewServer(validator, cfg.WebhookPort)
	return server.Start()
}

// startController initializes and starts the deployment controller
func startController(clientset kubernetes.Interface) error {
	// Create informer factory
	factory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	// Create controller
	ctrl := controller.NewController(
		clientset,
		factory.Apps().V1().Deployments(),
	)

	// Start informers
	stopCh := make(chan struct{})
	factory.Start(stopCh)

	// Run controller
	fmt.Println("Starting deployment controller...")
	return ctrl.Run(stopCh)
}
