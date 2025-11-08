package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs all pending database migrations
func RunMigrations(dbType, dsn string) error {
	// Construct database URL
	var databaseURL string
	if dbType == "sqlite" {
		databaseURL = fmt.Sprintf("sqlite3://%s", dsn)
	} else if dbType == "postgres" {
		databaseURL = fmt.Sprintf("postgres://%s", dsn)
	} else {
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			// No migrations to run - this is fine
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// RollbackMigrations rolls back the last migration
func RollbackMigrations(dbType, dsn string, steps int) error {
	// Construct database URL
	var databaseURL string
	if dbType == "sqlite" {
		databaseURL = fmt.Sprintf("sqlite3://%s", dsn)
	} else if dbType == "postgres" {
		databaseURL = fmt.Sprintf("postgres://%s", dsn)
	} else {
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Rollback migrations
	if err := m.Steps(-steps); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	return nil
}

// GetMigrationVersion returns the current migration version
func GetMigrationVersion(dbType, dsn string) (uint, bool, error) {
	// Construct database URL
	var databaseURL string
	if dbType == "sqlite" {
		databaseURL = fmt.Sprintf("sqlite3://%s", dsn)
	} else if dbType == "postgres" {
		databaseURL = fmt.Sprintf("postgres://%s", dsn)
	} else {
		return 0, false, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}
