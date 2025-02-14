package config

import (
	"os"
	"strings"
)

type Config struct {
	Host     string
	Port     string
	LogLevel string
	BaseURL  string
}

func LoadConfig() *Config {
	// Get the Replit domain from environment
	replitSlug := os.Getenv("REPL_SLUG")
	replitOwner := os.Getenv("REPL_OWNER")
	baseURL := ""

	if replitSlug != "" && replitOwner != "" {
		baseURL = "https://" + replitSlug + "." + replitOwner + ".repl.co"
	}

	return &Config{
		Host:     getEnv("HOST", "0.0.0.0"),
		Port:     getEnv("PORT", "8000"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		BaseURL:  baseURL,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}