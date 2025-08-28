package main

import (
	"fmt"
	"github.com/manzil-infinity180/k8s-custom-controller/utils"
	"github.com/manzil-infinity180/k8s-custom-controller/validator"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/manzil-infinity180/k8s-custom-controller/controller"
	"k8s.io/client-go/informers"
)

func main() {
	// add your context in the docker-compose.yml
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, assuming environment variables are set")
	}
	context := os.Getenv("CONTEXT")
	fmt.Println(context)
	clientset, _, err := utils.GetClientSetWithContext(context)
	if err != nil {
		fmt.Println()
		log.Printf("Error: %s", err.Error())
	}
	v := &validator.Validator{
		Clientset: clientset,
	}

	// Start the webhook server in a goroutine
	go func() {
		http.HandleFunc("/validate", v.ValidateDeployment)
		log.Println("Starting webhook server on :8000...")
		certPaths := []struct {
			cert string
			key  string
			desc string
		}{
			{"/certs/tls.crt", "/certs/tls.key", "Kubernetes-mounted certs"},
			{"certs/tls.crt", "certs/tls.key", "Local certs"},
		}
		// local go for certs/tls.crt and certs/tls.key
		var err error
		for _, cp := range certPaths {
			if _, statErr := os.Stat(cp.cert); statErr != nil {
				continue // Skip if cert not found
			}
			if _, statErr := os.Stat(cp.key); statErr != nil {
				continue // Skip if key not found
			}

			log.Printf("Using %s", cp.desc)
			err = http.ListenAndServeTLS(":8000", cp.cert, cp.key, nil)
			break
		}
		if err != nil {
			log.Fatalf("Failed to start webhook server: %v", err)

		}
	}()

	ch := make(chan struct{})
	// factory
	factory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	c := controller.NewController(clientset, factory.Apps().V1().Deployments())
	factory.Start(ch)
	c.Run(ch)
	fmt.Println(factory)
	factory.Apps().V1().Deployments().Informer()

	// Block forever
	select {}
}
