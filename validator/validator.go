package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
)

type Validator struct {
	Clientset kubernetes.Interface
	//cveScanner *CVEScanner
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

func (v *Validator) ValidateDeployment(w http.ResponseWriter, r *http.Request) {
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
