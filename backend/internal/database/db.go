package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// PoolConfig holds database connection pool configuration
type PoolConfig struct {
	MaxOpenConns    int           // Maximum number of open connections (default: 25)
	MaxIdleConns    int           // Maximum number of idle connections (default: 5)
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection (default: 5 minutes)
	ConnMaxIdleTime time.Duration // Maximum idle time for a connection (default: 5 minutes)
}

// DefaultPoolConfig returns sensible defaults for connection pooling
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// Connect establishes a database connection to PostgreSQL
func Connect(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	return ConnectWithPool(dsn, logLevel, DefaultPoolConfig())
}

// ConnectWithPool establishes a database connection with custom pool configuration
func ConnectWithPool(dsn string, logLevel logger.LogLevel, poolConfig PoolConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	// Store globally for easy access
	DB = db

	return db, nil
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// Ping checks if the database connection is healthy
func Ping() error {
	if DB == nil {
		return fmt.Errorf("database not connected")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return sqlDB.Ping()
}

// PingDB checks if a specific database instance is healthy
func PingDB(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not connected")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	return sqlDB.Ping()
}
