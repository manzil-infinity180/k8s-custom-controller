# Informer implementation with the NewSharedInformerFactoryWithOptions
# Implementation of watch workload (in this case deployment)
# Informer and websocket implementation for specific workload (deployments)
# Kubernetes custom controller from scratch in go

```go
package main

// websocket
router.GET("/ws", func(ctx *gin.Context) {
	deployment.HandleDeploymentLogs(ctx.Writer, ctx.Request)
})

```

```go

package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/katamyra/kubestellarUI/wds"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type DeploymentUpdate struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

func HandleDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}
	defer conn.Close()

	namespace := r.URL.Query().Get("namespace")
	deploymentName := r.URL.Query().Get("deployment")

	if namespace == "" || deploymentName == "" {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Error: Missing namespace or deployment name")); err != nil {
			log.Printf("Failed to write message: %v", err)
		}
		return
	}
    // nothing but just getting the kubeclient 
	clientset, err := wds.GetClientSetKubeConfig()
	if err != nil {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Error: Failed to create Kubernetes clientset - "+err.Error())); err != nil {
			log.Printf("Failed to send WebSocket message: %v", err)
		}
		return
	}

	sendInitialLogs(conn, clientset, namespace, deploymentName)

	// Use an informer to watch the deployment
	watchDeploymentWithInformer(conn, clientset, namespace, deploymentName)

	// WE ARE USING INFORMER
	//watchDeploymentChanges(conn, clientset, namespace, deploymentName)
}

func sendInitialLogs(conn *websocket.Conn, clientset *kubernetes.Clientset, namespace, deploymentName string) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Error: Failed to fetch deployment - "+err.Error())); err != nil {
			log.Printf("Failed to send WebSocket message: %v", err)
		}
		return
	}

	logs := getDeploymentLogs(deployment)
	for _, logLine := range logs {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(logLine)); err != nil {
			log.Println("Error writing to WebSocket:", err)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func watchDeploymentWithInformer(conn *websocket.Conn, clientset *kubernetes.Clientset, namespace, deploymentName string) {
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0,
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("metadata.name=%s", deploymentName)
		}),
	)
	informer := factory.Apps().V1().Deployments().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok1 := oldObj.(*v1.Deployment)
			newDeployment, ok2 := newObj.(*v1.Deployment)
			if !ok1 || !ok2 || newDeployment.Name != deploymentName {
				return
			}
			updateHandler(conn, oldDeployment, newDeployment)
		},
	})
	// make(chan struct{}) creates an empty struct channel.
	//This channel is used to signal the informer to stop.
	// struct{} is a zero-memory type, meaning it doesn’t allocate memory.
	// We don’t need to send any actual data through this channel, just a signal.
	stopCh := make(chan struct{})
	defer close(stopCh)

	go informer.Run(stopCh)

	// Keep the connection open

	//This is a blocking operation that:
	//
	//Prevents the function from exiting.
	//	Keeps the WebSocket connection open.
	//💡 Why does this work?
	//
	//select {} waits indefinitely because there are no case statements.
	//The function never returns, so the informer keeps running.

	/*
		1️⃣ Create a channel (stopCh) for stopping the informer.
		2️⃣ Run the informer in a goroutine (go informer.Run(stopCh)).
		3️⃣ Block forever (select {}) to keep the function running.
		4️⃣ If the function returns (not in this case), defer close(stopCh) stops the informer.
	*/
	select {}

}

func updateHandler(conn *websocket.Conn, oldDeployment, newDeployment *v1.Deployment) {
	var logs []DeploymentUpdate
	if *oldDeployment.Spec.Replicas != *newDeployment.Spec.Replicas {
		logs = append(logs, DeploymentUpdate{
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   fmt.Sprintf("Deployment %s updated - Replicas changed: %d", newDeployment.Name, *newDeployment.Spec.Replicas),
		})
	}
	oldImage := oldDeployment.Spec.Template.Spec.Containers[0].Image
	newImage := newDeployment.Spec.Template.Spec.Containers[0].Image
	if oldImage != newImage {
		logs = append(logs, DeploymentUpdate{
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   fmt.Sprintf("Deployment %s updated - Image changed: %s", newDeployment.Name, newImage),
		})
	}
	for _, logLine := range logs {
		jsonMessage, _ := json.Marshal(logLine)
		conn.WriteMessage(websocket.TextMessage, jsonMessage)
	}
}

// Watches deployment changes and sends updates
// Keeping it for reference - NOT USEFUL
func watchDeploymentChanges(conn *websocket.Conn, clientset *kubernetes.Clientset, namespace, deploymentName string) {
	options := metav1.ListOptions{
		// remove this line it will become universal for all the deployment
		// it will listen for all deployment inside namespace
		FieldSelector: fmt.Sprintf("metadata.name=%s", deploymentName),
	}
	watcher, err := clientset.AppsV1().Deployments(namespace).Watch(context.Background(), options)
	if err != nil {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Error: Failed to watch deployment - "+err.Error())); err != nil {
			log.Printf("Failed to send WebSocket message: %v", err)
		}
		return
	}

	defer watcher.Stop()

	// preserving the replicas and image for next call
	var lastReplicas *int32
	var lastImage string

	for event := range watcher.ResultChan() {
		deployment, ok := event.Object.(*v1.Deployment)
		if !ok {
			continue
		}

		var logs []DeploymentUpdate
		message := fmt.Sprintf("Deployment %s changed: %s", deployment.Name, event.Type)
		log.Println(message)

		if lastReplicas == nil || *lastReplicas != *deployment.Spec.Replicas {
			message = fmt.Sprintf("Deployment %s updated - Replicas changed: %d", deployment.Name, *deployment.Spec.Replicas)
			lastReplicas = deployment.Spec.Replicas
			logs = append(logs, DeploymentUpdate{
				Timestamp: time.Now().Format(time.RFC3339),
				Message:   message,
			})
		}

		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			currentImage := deployment.Spec.Template.Spec.Containers[0].Image
			if lastImage == "" || lastImage != currentImage {
				message = fmt.Sprintf("Deployment %s updated - Image changed: %s", deployment.Name, currentImage)
				logs = append(logs, DeploymentUpdate{
					Timestamp: time.Now().Format(time.RFC3339),
					Message:   message,
				})
				lastImage = currentImage
			}
		}

		for _, logLine := range logs {
			jsonMessage, _ := json.Marshal(logLine)
			if err := conn.WriteMessage(websocket.TextMessage, jsonMessage); err != nil {
				log.Println("Error writing to WebSocket:", err)
				return
			}
		}
	}
}

func getDeploymentLogs(deployment *v1.Deployment) []string {
	baseTime := time.Now().Format(time.RFC3339)

	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	logs := []string{
		fmt.Sprintf("[%v] INFO: Deployment workload %v initiated ", baseTime, deployment.Name),
		fmt.Sprintf("[%v] INFO: Workload created with replicas: %d, image: %v ", baseTime, replicas, deployment.Spec.Template.Spec.Containers[0].Image),
		fmt.Sprintf("[%v] INFO: Namespace %v successfully updated  ", baseTime, deployment.Namespace),
		fmt.Sprintf("[%v] INFO: Available Replicas: %d ", baseTime, deployment.Status.AvailableReplicas),
	}

	// Check if Conditions slice has elements before accessing it
	if len(deployment.Status.Conditions) > 0 {
		condition := deployment.Status.Conditions[0]
		logs = append(logs,
			fmt.Sprintf("[%v] INFO: Conditions: %s ", baseTime, condition.Type),
			fmt.Sprintf("[%v] INFO: LastUpdateTime : %s ", baseTime, condition.LastUpdateTime.Time),
			fmt.Sprintf("[%v] INFO: LastTransitionTime : %s ", baseTime, condition.LastTransitionTime.Time),
			fmt.Sprintf("[%v] INFO: Message: %s ", baseTime, condition.Message),
		)
	} else {
		logs = append(logs, fmt.Sprintf("[%v] INFO: No conditions available", baseTime))
	}

	return logs
}

```


```go
package wds

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

/*
Load the KubeConfig file and return the kubernetes clientset which gives you access to play with the k8s api
*/
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
func GetClientSetKubeConfig() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}

	// Load the kubeconfig file
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load kubeconfig"})
		return nil, fmt.Errorf("failed to load kubeconfig")
	}

	// Use WDS1 context specifically
	ctxContext := config.Contexts["wds1"]
	if ctxContext == nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ctxConfig"})
		return nil, fmt.Errorf("failed to create ctxConfig")
	}

	// Create config for WDS cluster
	clientConfig := clientcmd.NewDefaultClientConfig(
		*config,
		&clientcmd.ConfigOverrides{
			CurrentContext: "wds1",
		},
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create restconfig")
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client")
	}
	return clientset, nil
}

```