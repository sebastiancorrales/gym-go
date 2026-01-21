package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	App      AppConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Environment  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessSecret      string
	RefreshSecret     string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
	Issuer            string
}

// AppConfig holds application configuration
type AppConfig struct {
	Name    string
	Version string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "localhost"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			Environment:  getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			DatabasePath: getEnv("DATABASE_PATH", "gym-go.db"),
			MaxIdleConns: getIntEnv("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns: getIntEnv("DB_MAX_OPEN_CONNS", 100),
			MaxLifetime:  getDurationEnv("DB_MAX_LIFETIME", time.Hour),
		},
		JWT: JWTConfig{
			AccessSecret:      getEnv("JWT_ACCESS_SECRET", "your-super-secret-access-key-change-in-production"),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", "your-super-secret-refresh-key-change-in-production"),
			AccessExpiration:  getDurationEnv("JWT_ACCESS_EXPIRATION", 15*time.Minute),
			RefreshExpiration: getDurationEnv("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
			Issuer:            getEnv("JWT_ISSUER", "gym-go"),
		},
		App: AppConfig{
			Name:    getEnv("APP_NAME", "Gym-Go"),
			Version: getEnv("APP_VERSION", "1.0.0"),
		},
	}
}

// getEnv gets environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets integer environment variable or returns default
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv gets duration environment variable or returns default
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}



