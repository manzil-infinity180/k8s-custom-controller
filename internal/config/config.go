package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/manzil-infinity180/k8s-custom-controller/pkg/types"
)

// Load loads configuration from environment variables
func Load() (*types.ControllerConfig, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Not an error if .env doesn't exist
	}

	config := &types.ControllerConfig{
		Context:        getEnv("CONTEXT", ""),
		TrivyServerURL: getEnv("TRIVY_SERVER_URL", "http://trivy-server-service.default.svc:8080"),
		WebhookPort:    getEnvAsInt("WEBHOOK_PORT", 8000),
		CertPath:       getEnv("CERT_PATH", ""),
		KeyPath:        getEnv("KEY_PATH", ""),
	}

	return config, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets environment variable as integer with default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
