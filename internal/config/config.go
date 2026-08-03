package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	App      AppConfig
	SMTP     SMTPConfig
}

// SMTPConfig holds email configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
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
	Name            string
	Version         string
	DefaultTimezone string // fallback when gym has no timezone configured
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
			// A SQLite file has one effective writer: a big pool only multiplies
			// lock collisions. 8 covers the operators, the check-in kiosk, the
			// biometric service and the background jobs.
			MaxIdleConns: getIntEnv("DB_MAX_IDLE_CONNS", 8),
			MaxOpenConns: getIntEnv("DB_MAX_OPEN_CONNS", 8),
			// Recycling a handle to a local file buys nothing and forces the
			// per-connection PRAGMAs to be re-applied.
			MaxLifetime: getDurationEnv("DB_MAX_LIFETIME", 0),
			LogLevel:    getLogLevelEnv("DB_LOG_LEVEL", logger.Warn),
		},
		JWT: JWTConfig{
			AccessSecret:      getEnv("JWT_ACCESS_SECRET", "your-super-secret-access-key-change-in-production"),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", "your-super-secret-refresh-key-change-in-production"),
			AccessExpiration:  getDurationEnv("JWT_ACCESS_EXPIRATION", 24*time.Hour),     // Cambiado a 24 horas
			RefreshExpiration: getDurationEnv("JWT_REFRESH_EXPIRATION", 30*24*time.Hour), // Cambiado a 30 días
			Issuer:            getEnv("JWT_ISSUER", "gym-go"),
		},
		App: AppConfig{
			Name:            getEnv("APP_NAME", "Gym-Go"),
			Version:         getEnv("APP_VERSION", "1.0.0"),
			DefaultTimezone: getEnv("DEFAULT_TIMEZONE", "America/Bogota"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getIntEnv("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", ""),
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

// getLogLevelEnv reads a GORM log level by name: silent, error, warn or info.
// "info" logs every SQL statement — useful to count the queries a request makes.
func getLogLevelEnv(key string, defaultValue logger.LogLevel) logger.LogLevel {
	switch strings.ToLower(os.Getenv(key)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return defaultValue
	}
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
