package webhook

import (
	"encoding/json"
	"log"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/manzil-infinity180/k8s-custom-controller/pkg/types"
)

// Validator handles deployment validation logic
type Validator struct {
	scanner *Scanner
}

// NewValidator creates a new Validator instance
func NewValidator(scanner *Scanner) *Validator {
	return &Validator{
		scanner: scanner,
	}
}

// ValidateDeployment validates a deployment by scanning its images
func (v *Validator) ValidateDeployment(admissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionReview {
	req := admissionReview.Request

	var deployment appsv1.Deployment
	if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
		return v.createErrorResponse(admissionReview, "failed to unmarshal deployment", err)
	}

	images := v.extractImages(deployment)
	bypassCVE := v.shouldBypassCVE(deployment)

	if bypassCVE {
		log.Println("📦 BYPASS_CVE_DENIED: true/yes")
	} else {
		log.Println("📦 BYPASS_CVE_DENIED: default(false/no)")
	}

	// Scan images
	scanResults := v.scanner.ScanImages(images)

	// Build validation output
	validationOutput := types.ValidationOutput{
		Deployment: deployment.Name,
		Namespace:  deployment.Namespace,
		Images:     scanResults,
		Decision:   "ALLOWED",
	}

	// Check if any image has critical CVEs
	denied := v.hasCriticalCVEs(scanResults)
	if denied {
		validationOutput.Decision = "DENIED"
	}

	// Override decision if bypass is enabled
	if bypassCVE {
		log.Printf("Images have CVEs, but allowing due to BYPASS_CVE_DENIED=true")
		validationOutput.Decision = "ALLOWED"
		denied = false
	}

	return v.createResponse(admissionReview, !denied, validationOutput)
}

// extractImages extracts all container images from a deployment
func (v *Validator) extractImages(deployment appsv1.Deployment) []string {
	var images []string

	// Extract from init containers
	for _, container := range deployment.Spec.Template.Spec.InitContainers {
		images = append(images, container.Image)
	}

	// Extract from regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	return images
}

// shouldBypassCVE checks if CVE validation should be bypassed
func (v *Validator) shouldBypassCVE(deployment appsv1.Deployment) bool {
	// Check init containers
	for _, container := range deployment.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.Name == "BYPASS_CVE_DENIED" && (env.Value == "yes" || env.Value == "true") {
				return true
			}
		}
	}

	// Check regular containers
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "BYPASS_CVE_DENIED" && (env.Value == "yes" || env.Value == "true") {
				return true
			}
		}
	}

	return false
}

// hasCriticalCVEs checks if any image has critical CVEs
func (v *Validator) hasCriticalCVEs(results []types.ImageScanResult) bool {
	for _, result := range results {
		if result.CriticalCVEs > 0 {
			return true
		}
	}
	return false
}

// createResponse creates an admission response
func (v *Validator) createResponse(admissionReview *admissionv1.AdmissionReview, allowed bool, validationOutput types.ValidationOutput) *admissionv1.AdmissionReview {
	outputJSON, _ := json.MarshalIndent(validationOutput, "", "  ")

	return &admissionv1.AdmissionReview{
		TypeMeta: admissionReview.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     admissionReview.Request.UID,
			Allowed: allowed,
			Result: &metav1.Status{
				Message: string(outputJSON),
			},
		},
	}
}

// createErrorResponse creates an error admission response
func (v *Validator) createErrorResponse(admissionReview *admissionv1.AdmissionReview, message string, err error) *admissionv1.AdmissionReview {
	log.Printf("Error: %s: %v", message, err)

	return &admissionv1.AdmissionReview{
		TypeMeta: admissionReview.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     admissionReview.Request.UID,
			Allowed: false,
			Result: &metav1.Status{
				Code:    400,
				Message: message + ": " + err.Error(),
			},
		},
	}
}
