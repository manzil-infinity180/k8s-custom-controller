package types

import admissionv1 "k8s.io/api/admission/v1"

// ImageScanResult represents the result of scanning a single image
type ImageScanResult struct {
	Name         string              `json:"name"`
	CriticalCVEs int                 `json:"critical_cves"`
	CVEs         []map[string]string `json:"cves"`
}

// ValidationOutput represents the complete validation response
type ValidationOutput struct {
	Deployment string            `json:"deployment"`
	Namespace  string            `json:"namespace"`
	Images     []ImageScanResult `json:"images"`
	Decision   string            `json:"decision"`
}

// Admitter wraps admission request for processing
type Admitter struct {
	Request *admissionv1.AdmissionRequest
}

// CVE represents a Common Vulnerabilities and Exposures entry
type CVE struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// ScanResult represents the result of a security scan
type ScanResult struct {
	Safe            bool  `json:"safe"`
	Vulnerabilities []CVE `json:"vulnerabilities"`
	Count           int   `json:"count"`
	Error           error `json:"error,omitempty"`
}

// ControllerConfig holds configuration for the controller components
type ControllerConfig struct {
	Context        string `json:"context"`
	TrivyServerURL string `json:"trivy_server_url"`
	WebhookPort    int    `json:"webhook_port"`
	CertPath       string `json:"cert_path"`
	KeyPath        string `json:"key_path"`
}
