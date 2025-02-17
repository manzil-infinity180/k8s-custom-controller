package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsInformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type controller struct {
	clientset      kubernetes.Interface
	depLister      appsListers.DeploymentLister
	depCacheSycned cache.InformerSynced
	//queue          workqueue.TypedRateLimitingInterface[any]
	queue workqueue.RateLimitingInterface
}

func NewController(clientset kubernetes.Interface, depinfomer appsInformer.DeploymentInformer) *controller {
	c := &controller{
		clientset:      clientset,
		depLister:      depinfomer.Lister(),
		depCacheSycned: depinfomer.Informer().HasSynced,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ekspose"),
	}

	depinfomer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		})
	return c
}

func (c *controller) Run(ch <-chan struct{}) {
	fmt.Sprintf("starting controller")
	if !cache.WaitForCacheSync(ch, c.depCacheSycned) {
		fmt.Println("waiting for cache to be synced")
	}

	go wait.Until(c.worker, 1*time.Second, ch)
	<-ch
}

func (c *controller) worker() {
	for c.processItem() {

	}
}

func (c *controller) processItem() bool {
	item, shutdown := c.queue.Get()

	if shutdown {
		return false
	}
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("key and err, %s\n", err.Error())
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("spliting namespace and name, %s\n", err.Error())
	}
	err = c.syncDeployment(namespace, name)
	if err != nil {
		fmt.Printf("sync deployment, %s\n", err.Error())
		return false
	}
	return true
}

func (c *controller) syncDeployment(ns, name string) error {
	ctx := context.Background()
	dep, err := c.depLister.Deployments(ns).Get(name)
	if err != nil {
		fmt.Printf("getting deployment from lister %s\n", err.Error())
	}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: depLabels(*dep),
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}

	_, err = c.clientset.CoreV1().Services(ns).Create(ctx, &service, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("sync deployment, %s\n", err.Error())
	}

	return nil
}

func depLabels(dep appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}
func (c *controller) handleAdd(obj interface{}) {
	fmt.Println("hello add is called")
	c.queue.Add(obj)
}

func (c *controller) handleDel(obj interface{}) {
	fmt.Println("hello del is called")
	c.queue.Add(obj)
}
