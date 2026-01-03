package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file source for migrations
	_ "github.com/lib/pq"                                      // postgres driver for database
)

// EnsureDatabase creates the database if it doesn't exist
func EnsureDatabase(databaseURL string) error {
	// Parse the URL to extract database name
	u, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return fmt.Errorf("database name not found in URL")
	}

	// Connect to the default 'postgres' database to create our target database
	u.Path = "/postgres"
	postgresURL := u.String()

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Check if database exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !exists {
		// Create the database (can't use parameterized query for CREATE DATABASE)
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
	}

	return nil
}

// RunMigrations runs all pending database migrations
func RunMigrations(databaseURL string) error {
	// Ensure database exists first
	if err := EnsureDatabase(databaseURL); err != nil {
		return fmt.Errorf("failed to ensure database exists: %w", err)
	}

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

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
func RollbackMigrations(databaseURL string, steps int) error {

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

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
func GetMigrationVersion(databaseURL string) (version uint, dirty bool, err error) {

	// Create migrate instance
	m, err := migrate.New(
		"file://backend/migrations",
		databaseURL,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	version, dirty, err = m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}
