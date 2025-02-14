package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host        string
	Port        string
	LogLevel    string
	BaseURL     string
	RedisURL    string
	RedisActive bool
	TCPPort     int
	TestMode    bool
}

func LoadConfig() *Config {
	// Get the Replit domain from environment
	replitSlug := os.Getenv("REPL_SLUG")
	replitOwner := os.Getenv("REPL_OWNER")
	baseURL := ""

	if replitSlug != "" && replitOwner != "" {
		baseURL = "https://" + replitSlug + "." + replitOwner + ".repl.co"
	}

	// Get TCP port from environment or use default
	tcpPort := 5023
	if portStr := os.Getenv("TCP_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			tcpPort = port
		}
	}

	return &Config{
		Host:        getEnv("HOST", "0.0.0.0"),
		Port:        getEnv("PORT", "8000"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		BaseURL:     baseURL,
		RedisURL:    getEnv("REDIS_URL", ""),
		RedisActive: strings.ToLower(getEnv("REDIS_ACTIVE", "false")) == "true",
		TCPPort:     tcpPort,
		TestMode:    strings.ToLower(getEnv("TEST_MODE", "false")) == "true",
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}