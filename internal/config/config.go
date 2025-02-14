package config

import (
    "os"
)

type Config struct {
    ServerPort string
    LogLevel   string
}

func LoadConfig() *Config {
    return &Config{
        ServerPort: getEnv("SERVER_PORT", "8000"),
        LogLevel:   getEnv("LOG_LEVEL", "info"),
    }
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}
