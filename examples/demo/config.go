package main

import (
	"os"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	App struct {
		Port int
		Env  string
	}
	Database struct {
		Host     string
		Port     int
		Name     string
		User     string
		Password string
	}
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	cfg := &Config{}
	
	// App configuration
	cfg.App.Port = getEnvInt("APP_PORT", 8080)
	cfg.App.Env = getEnv("APP_ENV", "development")
	
	// Database configuration (if needed)
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvInt("DB_PORT", 5432)
	cfg.Database.Name = getEnv("DB_NAME", "app")
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	
	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
