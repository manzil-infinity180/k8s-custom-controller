package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func scanImageWithTrivy(image string) (bool, string, error) {
	// cmd := exec.Command("trivy", "image", "--quiet", "--severity", "HIGH,CRITICAL", "--format", "json", image)
	// out, err := cmd.Output()
	// TODO: Need to make it dynamic to support all trivy server (will go with env)
	cmd := exec.Command(
		"trivy",
		"image",
		"--scanners", "vuln",
		"--severity", "CRITICAL", // only critical CVEs
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
					// skipping for High CVE > Checking only for CRITICAL
					if severity == "CRITICAL" {
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
