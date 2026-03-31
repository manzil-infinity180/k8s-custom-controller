package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsInformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller handles deployment events and creates associated resources
type Controller struct {
	clientset      kubernetes.Interface
	depLister      appsListers.DeploymentLister
	depCacheSynced cache.InformerSynced
	queue          workqueue.RateLimitingInterface
	handlers       *EventHandlers
}

// NewController creates a new Controller instance
func NewController(clientset kubernetes.Interface, depInformer appsInformer.DeploymentInformer) *Controller {
	handlers := NewEventHandlers()

	c := &Controller{
		clientset:      clientset,
		depLister:      depInformer.Lister(),
		depCacheSynced: depInformer.Informer().HasSynced,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
		handlers:       handlers,
	}

	// Register event handlers
	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDelete,
			UpdateFunc: c.handleUpdate,
		},
	)

	return c
}

// Run starts the controller
func (c *Controller) Run(stopCh <-chan struct{}) error {
	fmt.Println("Starting controller")

	if !cache.WaitForCacheSync(stopCh, c.depCacheSynced) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	fmt.Println("Cache synced, starting worker")
	go wait.Until(c.worker, time.Second, stopCh)

	<-stopCh
	fmt.Println("Shutting down controller")
	c.queue.ShutDown()

	return nil
}

// worker processes items from the work queue
func (c *Controller) worker() {
	for c.processNextItem() {
		// Continue processing items
	}
}

// processNextItem processes the next item in the queue
func (c *Controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("Error getting key: %v\n", err)
		c.queue.Forget(item)
		return true
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("Error splitting key: %v\n", err)
		c.queue.Forget(item)
		return true
	}

	err = c.syncDeployment(namespace, name)
	if err != nil {
		fmt.Println("────────────────────────────────────────────────────")
		fmt.Printf("Error syncing deployment: %v\n", err)
		fmt.Println("────────────────────────────────────────────────────")

		// Retry with exponential backoff
		c.queue.AddRateLimited(key)
		return true
	}

	c.queue.Forget(item)
	return true
}

// syncDeployment synchronizes the deployment state
func (c *Controller) syncDeployment(namespace, name string) error {
	ctx := context.Background()

	deployment, err := c.depLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment was deleted, nothing to do
			fmt.Printf("Deployment %s/%s not found, assuming it was deleted\n", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get deployment from lister: %w", err)
	}

	// Check if auto-creation should be skipped
	if c.shouldSkipAutoCreation(deployment) {
		fmt.Println("Skipping auto service and ingress creation (NO_AUTO_CREATION=true)")
		return nil
	}

	fmt.Printf("Creating auto service and ingress for deployment %s/%s\n", namespace, name)

	// Create service
	if err := c.createService(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Create ingress
	if err := c.createIngress(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create ingress: %w", err)
	}

	return nil
}

// shouldSkipAutoCreation checks if auto-creation should be skipped
func (c *Controller) shouldSkipAutoCreation(deployment *appsv1.Deployment) bool {
	// Check init containers
	for _, container := range deployment.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.Name == "NO_AUTO_CREATION" && (env.Value == "yes" || env.Value == "true") {
				return true
			}
		}
	}

	// Check regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "NO_AUTO_CREATION" && (env.Value == "yes" || env.Value == "true") {
				return true
			}
		}
	}

	return false
}

// createService creates a service for the deployment
func (c *Controller) createService(ctx context.Context, deployment *appsv1.Deployment) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels:    c.getServiceLabels(deployment.Name),
		},
		Spec: corev1.ServiceSpec{
			Selector: c.getDeploymentLabels(deployment),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}

	_, err := c.clientset.CoreV1().Services(deployment.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Printf("Service %s/%s already exists, skipping creation\n", deployment.Namespace, deployment.Name)
			return nil
		}
		return fmt.Errorf("failed to create service: %w", err)
	}

	fmt.Printf("✅ Service created: %s/%s\n", deployment.Namespace, deployment.Name)
	return nil
}

// createIngress creates an ingress for the deployment
func (c *Controller) createIngress(ctx context.Context, deployment *appsv1.Deployment) error {
	pathType := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
			Labels: c.getIngressLabels(deployment.Name),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "demo.local",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     fmt.Sprintf("/%s", deployment.Name),
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: deployment.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.NetworkingV1().Ingresses(deployment.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Printf("Ingress %s/%s already exists, skipping creation\n", deployment.Namespace, deployment.Name)
			return nil
		}
		return fmt.Errorf("failed to create ingress: %w", err)
	}

	fmt.Printf("✅ Ingress created: %s/%s\n", deployment.Namespace, deployment.Name)
	return nil
}

// getDeploymentLabels extracts labels from deployment
func (c *Controller) getDeploymentLabels(deployment *appsv1.Deployment) map[string]string {
	labels := deployment.Spec.Template.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	return labels
}

// getServiceLabels returns labels for the service
func (c *Controller) getServiceLabels(name string) map[string]string {
	return map[string]string{
		"rahulxf.io/workload": name,
		"app":                 name,
		"component":           "service",
	}
}

// getIngressLabels returns labels for the ingress
func (c *Controller) getIngressLabels(name string) map[string]string {
	return map[string]string{
		"rahulxf.io/workload": name,
		"app":                 name,
		"component":           "ingress",
	}
}

// Event handler methods

// handleAdd handles deployment add events
func (c *Controller) handleAdd(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		fmt.Println("❌ Object is not a Deployment")
		return
	}

	c.handlers.HandleAdd(deployment)
	c.queue.Add(obj)
}

// handleUpdate handles deployment update events
func (c *Controller) handleUpdate(oldObj, newObj interface{}) {
	oldDeployment, ok1 := oldObj.(*appsv1.Deployment)
	newDeployment, ok2 := newObj.(*appsv1.Deployment)

	if !ok1 || !ok2 {
		fmt.Println("❌ Objects are not Deployments")
		return
	}

	c.handlers.HandleUpdate(oldDeployment, newDeployment)
	c.queue.Add(newObj)
}

// handleDelete handles deployment delete events
func (c *Controller) handleDelete(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		// Handle DeletedFinalStateUnknown
		if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			deployment, ok = deletedFinalStateUnknown.Obj.(*appsv1.Deployment)
			if !ok {
				fmt.Println("❌ DeletedFinalStateUnknown object is not a Deployment")
				return
			}
		} else {
			fmt.Println("❌ Object is not a Deployment")
			return
		}
	}

	c.handlers.HandleDelete(deployment)
	// Note: We don't add delete events to the queue for processing
	// If you want to clean up associated resources, you can do it here
	// or add the delete event to the queue and handle it in syncDeployment
}

// Utility methods for testing and debugging

// GetClientset returns the Kubernetes clientset (for testing purposes)
func (c *Controller) GetClientset() kubernetes.Interface {
	return c.clientset
}

// GetQueue returns the work queue (for testing purposes)
func (c *Controller) GetQueue() workqueue.RateLimitingInterface {
	return c.queue
}

// GetLister returns the deployment lister (for testing purposes)
func (c *Controller) GetLister() appsListers.DeploymentLister {
	return c.depLister
}

// Stop gracefully stops the controller
func (c *Controller) Stop() {
	fmt.Println("Stopping controller...")
	c.queue.ShutDown()
}
