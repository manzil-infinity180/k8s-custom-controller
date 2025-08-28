package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
)

// Server handles admission webhook HTTP server
type Server struct {
	validator *Validator
	port      int
}

// NewServer creates a new webhook server
func NewServer(validator *Validator, port int) *Server {
	return &Server{
		validator: validator,
		port:      port,
	}
}

// Start starts the webhook server
func (s *Server) Start() error {
	http.HandleFunc("/validate", s.handleValidation)
	log.Printf("Starting webhook server on :%d...", s.port)

	certPaths := []struct {
		cert string
		key  string
		desc string
	}{
		{"/certs/tls.crt", "/certs/tls.key", "Kubernetes-mounted certs"},
		{"certs/tls.crt", "certs/tls.key", "Local certs"},
	}

	for _, cp := range certPaths {
		if !s.certExists(cp.cert, cp.key) {
			continue
		}

		log.Printf("Using %s", cp.desc)
		return http.ListenAndServeTLS(fmt.Sprintf(":%d", s.port), cp.cert, cp.key, nil)
	}

	return fmt.Errorf("no valid certificate found")
}

// handleValidation handles the /validate endpoint
func (s *Server) handleValidation(w http.ResponseWriter, r *http.Request) {
	log.Println("Received /validate request")

	admissionReview, err := s.parseRequest(r)
	if err != nil {
		log.Printf("Error parsing admission request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := s.validator.ValidateDeployment(admissionReview)

	w.Header().Set("Content-Type", "application/json")
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(responseJSON); err != nil {
		log.Printf("Failed to write response: %v", err)
	}

	log.Println("Admission response sent")
}

// parseRequest parses the HTTP request into an AdmissionReview
func (s *Server) parseRequest(r *http.Request) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type: %q should be %q",
			r.Header.Get("Content-Type"), "application/json")
	}

	bodyBuf := new(bytes.Buffer)
	if _, err := bodyBuf.ReadFrom(r.Body); err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	body := bodyBuf.Bytes()
	if len(body) == 0 {
		return nil, fmt.Errorf("admission request body is empty")
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		return nil, fmt.Errorf("could not parse admission review request: %w", err)
	}

	if admissionReview.Request == nil {
		return nil, fmt.Errorf("admission review request field is nil")
	}

	return &admissionReview, nil
}

// certExists checks if certificate files exist
func (s *Server) certExists(certPath, keyPath string) bool {
	if _, err := os.Stat(certPath); err != nil {
		return false
	}
	if _, err := os.Stat(keyPath); err != nil {
		return false
	}
	return true
}
