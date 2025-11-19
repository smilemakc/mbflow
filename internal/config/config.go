package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	LogLevel    string
	DatabaseDSN string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseDSN: getEnv("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) GetPortInt() int {
	p, _ := strconv.Atoi(c.Port)
	return p
}
