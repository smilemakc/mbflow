package config

import (
	"os"
	"strconv"
)

// Config represents the application configuration.
// This is an infrastructure component that loads configuration from environment variables.
type Config struct {
	Port        string
	LogLevel    string
	DatabaseDSN string
}

// Load creates a new Config instance by reading environment variables.
func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseDSN: getEnv("DATABASE_DSN", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// GetPortInt returns the port as an integer.
func (c *Config) GetPortInt() int {
	p, _ := strconv.Atoi(c.Port)
	return p
}
