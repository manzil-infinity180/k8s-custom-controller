package controller

import (
	"fmt"
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
			AddFunc:    handleAdd,
			DeleteFunc: handleDel,
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

}

func handleAdd(obj interface{}) {
	fmt.Println("hello add is called")
}

func handleDel(obj interface{}) {
	fmt.Println("hello del is called")
}
