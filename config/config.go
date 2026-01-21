package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	MongoDBURI          string
	MongoDBDatabase     string
	JWTSecret           string
	AutoCompleteMinutes int
}

func LoadConfig() *Config {
	autoCompleteMinutes := 10 // default
	if minutes := os.Getenv("AUTO_COMPLETE_MINUTES"); minutes != "" {
		if m, err := strconv.Atoi(minutes); err == nil {
			autoCompleteMinutes = m
		}
	}

	return &Config{
		Port:                getEnv("PORT", "8080"),
		MongoDBURI:          getEnv("MONGODB_URI", "mongodb://admin:password123@localhost:27017"),
		MongoDBDatabase:     getEnv("MONGODB_DATABASE", "taskdb"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AutoCompleteMinutes: autoCompleteMinutes,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
