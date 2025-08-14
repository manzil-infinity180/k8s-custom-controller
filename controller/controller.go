package controller

// https://danielms.site/zet/2024/client-go-kubernetes-deploymentservice-and-ingress/
import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsInformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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
	fmt.Println("starting controller")
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
	// we do not process the item again
	defer c.queue.Forget(item)
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
		fmt.Println("────────────────────────────────────────────────────")
		fmt.Printf("sync deployment, %s\n", err.Error())
		fmt.Println("────────────────────────────────────────────────────")
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
	// Pretty-print the deployment object as JSON
	// jsonData, err := json.MarshalIndent(dep, "", "  ")
	// if err != nil {
	// 	fmt.Printf(`{"error": "Failed to marshal deployment", "details": "%s"}`, err.Error())
	// 	// return
	// }
	// fmt.Println(string(jsonData))

	// if NO_AUTO_CREATION inside '- env' is true or yes then we will not create the auto svc and ingress
	NO_AUTO_CREATION := false
	for _, c := range dep.Spec.Template.Spec.InitContainers {
		for _, e := range c.Env {
			if e.Name == "NO_AUTO_CREATION" && (e.Value == "yes" || e.Value == "true") {
				NO_AUTO_CREATION = true
			}
		}
	}
	// Containers
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == "NO_AUTO_CREATION" && (e.Value == "yes" || e.Value == "true") {
				NO_AUTO_CREATION = true
			}
		}
	}
	if NO_AUTO_CREATION {
		fmt.Printf("Skipping the auto svc and ingress creation as NO_AUTO_CREATION set is true/yes \n")
		return nil
	}
	fmt.Printf("Started creating auto svc and ingresses \n")
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: ns,
			//Labels: map[string]string{
			//
			//},
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
	labels := service.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labelKey := "rahulxf.io/workload"
	if _, exist := labels[labelKey]; !exist {
		labels[labelKey] = name
		service.SetLabels(labels)
	}

	_, err = c.clientset.CoreV1().Services(ns).Create(ctx, &service, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("────────────────────────────────────────────────────")
		fmt.Printf("sync deployment, %s\n", err.Error())
		fmt.Println("────────────────────────────────────────────────────")
	}

	err = c.createIngress(ns, name)
	if err != nil {
		fmt.Println("────────────────────────────────────────────────────")
		fmt.Printf("sync deployment, %s\n", err.Error())
		fmt.Println("────────────────────────────────────────────────────")
	}
	return nil
}

func (c *controller) createIngress(ns, name string) error {
	ctx := context.Background()
	pathType := "Prefix"
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
			//Labels: map[string]string{
			//
			//},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				networkingv1.IngressRule{
					Host: "demo.local",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								networkingv1.HTTPIngressPath{
									Path:     fmt.Sprintf("/%s", name),
									PathType: (*networkingv1.PathType)(&pathType),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
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

	labels := ingress.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labelKey := "rahulxf.io/workload"
	if _, exist := labels[labelKey]; !exist {
		labels[labelKey] = name
		ingress.SetLabels(labels)
	}
	_, err := c.clientset.NetworkingV1().Ingresses(ns).Create(ctx, &ingress, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil

}

func depLabels(dep appsv1.Deployment) map[string]string {
	return dep.Spec.Template.Labels
}

// Almost working
func (c *controller) handleAdd(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		fmt.Println("\n❌ Not a Deployment object")
		return
	}

	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println("📦 Deployment Added")
	fmt.Printf("🔤 Name:      %s\n", deployment.Name)
	fmt.Printf("📂 Namespace: %s\n", deployment.Namespace)
	fmt.Printf("🆔 UID:       %s\n", deployment.UID)
	fmt.Printf("🕓 Created:   %s\n", deployment.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Println("────────────────────────────────────────────────────")

	c.queue.Add(obj)
}

// Not tested
func (c *controller) handleDel(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		fmt.Println("\n❌ Not a Deployment")
		return
	}

	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println("📦 Deployment DELETED")
	fmt.Printf("🔤 Name:      %s\n", deployment.Name)
	fmt.Printf("📂 Namespace: %s\n", deployment.Namespace)
	fmt.Printf("🆔 UID:       %s\n", deployment.UID)
	fmt.Printf("🕓 Deleted:   %s\n", deployment.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Println("────────────────────────────────────────────────────")

	//c.queue.Add(obj)
}
