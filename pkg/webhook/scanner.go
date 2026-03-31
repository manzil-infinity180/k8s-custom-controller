package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/manzil-infinity180/k8s-custom-controller/pkg/types"
)

// Scanner handles image vulnerability scanning with Trivy
type Scanner struct {
	serverURL string
}

// NewScanner creates a new Scanner instance
func NewScanner(serverURL string) *Scanner {
	return &Scanner{
		serverURL: serverURL,
	}
}

// ScanImage scans an image for vulnerabilities using Trivy
func (s *Scanner) ScanImage(image string) types.ScanResult {
	log.Printf("🛡️  Scanning image: %s", image)

	cmd := exec.Command(
		"trivy",
		"image",
		"--scanners", "vuln",
		"--severity", "CRITICAL",
		"--server", s.serverURL,
		"--format", "json",
		image,
	)

	output, err := cmd.Output()
	if err != nil {
		return types.ScanResult{
			Safe:  false,
			Error: fmt.Errorf("trivy scan failed for %s: %w", image, err),
		}
	}

	return s.parseTrivyOutput(output)
}

// parseTrivyOutput parses the JSON output from Trivy
func (s *Scanner) parseTrivyOutput(output []byte) types.ScanResult {
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return types.ScanResult{
			Safe:  false,
			Error: fmt.Errorf("failed to parse trivy output: %w", err),
		}
	}

	var vulnerabilities []types.CVE
	var count int

	log.Println("❗CVEs Found: ")

	results, ok := result["Results"].([]interface{})
	if !ok {
		return types.ScanResult{Safe: true}
	}

	for _, r := range results {
		rmap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		vlist, ok := rmap["Vulnerabilities"].([]interface{})
		if !ok {
			continue
		}

		for _, v := range vlist {
			vmap, ok := v.(map[string]interface{})
			if !ok {
				continue
			}

			severity, ok := vmap["Severity"].(string)
			if !ok {
				continue
			}

			if strings.EqualFold(severity, "CRITICAL") {
				count++

				vulnID, _ := vmap["VulnerabilityID"].(string)
				primaryURL, _ := vmap["PrimaryURL"].(string)

				vulnerabilities = append(vulnerabilities, types.CVE{
					ID:  vulnID,
					URL: primaryURL,
				})
			}
		}
	}

	safe := len(vulnerabilities) == 0
	return types.ScanResult{
		Safe:            safe,
		Vulnerabilities: vulnerabilities,
		Count:           count,
	}
}

// ScanImages scans multiple images and returns results
func (s *Scanner) ScanImages(images []string) []types.ImageScanResult {
	var results []types.ImageScanResult

	for _, image := range images {
		log.Println("────────────────────────────────────────────────────")

		scanResult := s.ScanImage(image)

		// Convert CVE slice to map slice for compatibility
		var cvesMaps []map[string]string
		for _, cve := range scanResult.Vulnerabilities {
			cvesMaps = append(cvesMaps, map[string]string{
				"id":  cve.ID,
				"url": cve.URL,
			})
		}

		imgResult := types.ImageScanResult{
			Name:         image,
			CriticalCVEs: scanResult.Count,
			CVEs:         cvesMaps,
		}

		results = append(results, imgResult)
		log.Println("────────────────────────────────────────────────────")
	}

	return results
}
