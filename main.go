package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/manzil-infinity180/k8s-custom-controller/controller"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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

	// Start the webhook server in a goroutine
	go func() {
		http.HandleFunc("/validate", ValidateDeployment)
		log.Println("Starting webhook server on :8000...")
		// local go for certs/tls.crt and certs/tls.key
		err := http.ListenAndServeTLS(":8000", "/certs/tls.crt", "/certs/tls.key", nil) // k8s
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

type Admitter struct {
	Request *admissionv1.AdmissionRequest
}

func parseRequest(r *http.Request) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type: %q should be %q",
			r.Header.Get("Content-Type"), "application/json")
	}
	bodybuf := new(bytes.Buffer)
	bodybuf.ReadFrom(r.Body)
	body := bodybuf.Bytes()
	if len(body) == 0 {
		return nil, fmt.Errorf("admission request body is empty")
	}
	var a admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, fmt.Errorf("could not parse admission review request: %v", err)
	}

	if a.Request == nil {
		return nil, fmt.Errorf("admission review can't be used: Request field is nil")
	}
	return &a, nil
}

func scanImageWithTrivy(image string) (bool, string, error) {
	// cmd := exec.Command("trivy", "image", "--quiet", "--severity", "HIGH,CRITICAL", "--format", "json", image)
	// out, err := cmd.Output()
	cmd := exec.Command(
		"trivy",
		"image",
		"--scanners", "vuln",
		"--server", "http://trivy-server-service.default.svc:8080", // [service_name].[namespace].svc:[port] (if not port 80)
		"--format", "json",
		image,
	)
	out, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("trivy scan failed for %s: %v", image, err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		return false, "", fmt.Errorf("failed to parse trivy output: %v", err)
	}
	// Check if vulnerabilities found
	vulns := []string{}
	log.Println("❗CVEs Found: ")
	if results, ok := result["Results"].([]interface{}); ok {
		for _, r := range results {
			rmap := r.(map[string]interface{})
			if vlist, ok := rmap["Vulnerabilities"].([]interface{}); ok {
				for _, v := range vlist {
					vmap := v.(map[string]interface{})
					severity := vmap["Severity"].(string)
					if severity == "HIGH" || severity == "CRITICAL" {
						msg := fmt.Sprintf("   - 🔥 %s\n", vmap["VulnerabilityID"].(string))
						//vulns = append(vulns, vmap["VulnerabilityID"].(string))
						vulns = append(vulns, msg)
					}
				}
			}
		}
	}
	if len(vulns) > 0 {
		return false, strings.Join(vulns, ","), nil
	}
	return true, "", nil
}
func ValidateDeployment(w http.ResponseWriter, r *http.Request) {
	log.Println("Received /validate request")
	in, err := parseRequest(r)
	if err != nil {
		log.Printf("Error parsing admission request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var dep appsv1.Deployment
	if err := json.Unmarshal(in.Request.Object.Raw, &dep); err != nil {
		log.Printf("Failed to unmarshal deployment: %v", err)
		http.Error(w, fmt.Sprintf("could not unmarshal deployment: %v", err), http.StatusBadRequest)
		return
	}
	images := []string{}
	denied := false
	var reasons []string
	BYPASS_CVE_DENIED := false
	// InitContainers
	for _, c := range dep.Spec.Template.Spec.InitContainers {
		for _, e := range c.Env {
			if e.Name == "BYPASS_CVE_DENIED" && (e.Value == "yes" || e.Value == "true") {
				BYPASS_CVE_DENIED = true
			}
		}
		images = append(images, c.Image)
	}
	// Containers
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == "BYPASS_CVE_DENIED" && (e.Value == "yes" || e.Value == "true") {
				BYPASS_CVE_DENIED = true
			}
		}
		images = append(images, c.Image)
	}
	for _, image := range images {
		log.Println("────────────────────────────────────────────────────")
		log.Printf("🛡️  Deployment Image Scanning Started : %s\n", image)
		if BYPASS_CVE_DENIED {
			log.Println("📦 BYPASS_CVE_DENIED: true/yes")
		} else {
			log.Println("📦 BYPASS_CVE_DENIED: default(false/no)")
		}
		ok, vulns, err := scanImageWithTrivy(image)
		if err != nil {
			log.Printf("Error scanning image %s: %v", image, err)
			continue
		}
		log.Println("────────────────────────────────────────────────────")
		if !ok {
			denied = true
			reasons = append(reasons, fmt.Sprintf("%s (CVE: %s)", image, vulns))
		}
	}
	message := "Images allowed"
	if denied {
		message = fmt.Sprintf("Denied images due to total CVEs across %v images: %v", len(images), reasons)
		log.Printf("Denied images due to CVEs: %v", reasons)
	}

	// look for BYPASS_CVE env - you need to skip
	if BYPASS_CVE_DENIED {
		log.Printf("It have CVE across all the %v images, but we are skipping as BYPASS_CVE_DENIED set true", len(images))
		denied = false
	}

	log.Printf("Validating Deployment: %s, Images: %v", dep.Name, images)
	response := admissionv1.AdmissionReview{
		TypeMeta: in.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     in.Request.UID,
			Allowed: !denied,
			Result: &metav1.Status{
				Message: message,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	jout, err := json.Marshal(response)
	if err != nil {
		e := fmt.Sprintf("could not parse admission response: %v", err)
		log.Println(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(jout); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
	log.Println("Admission response sent")
}
