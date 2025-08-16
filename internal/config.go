package internal

import (
	"os"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Web      WebConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type WebConfig struct {
	SessionSecret string
}

type DatabaseConfig struct {
	URI  string
	Name string
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "localhost"),
		},
		Database: DatabaseConfig{
			URI:  getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Name: getEnv("MONGODB_DATABASE", "family_budget"),
		},
		Web: WebConfig{
			SessionSecret: getEnv("SESSION_SECRET", "your-super-secret-session-key-change-in-production"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
