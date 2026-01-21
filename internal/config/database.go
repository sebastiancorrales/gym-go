package config

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Database holds database connection
type Database struct {
	DB *gorm.DB
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DatabasePath string
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
	LogLevel     logger.LogLevel
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DatabasePath: "gym-go.db",
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		MaxLifetime:  time.Hour,
		LogLevel:     logger.Info,
	}
}

// NewDatabase creates a new database connection
func NewDatabase(config *DatabaseConfig) (*Database, error) {
	if config == nil {
		config = DefaultDatabaseConfig()
	}

	// Use pure Go SQLite driver for cross-platform compatibility
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        config.DatabasePath,
	}, &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.MaxLifetime)

	// Enable foreign keys for SQLite
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	log.Println("✅ Database connection established successfully")

	return &Database{DB: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck checks database connection health
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
