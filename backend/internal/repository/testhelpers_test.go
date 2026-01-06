package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aloks98/waygates/backend/internal/models"
)

// TestDB holds the test database connection and container
type TestDB struct {
	Container testcontainers.Container
	DB        *gorm.DB
	DSN       string
	ctx       context.Context
}

// SetupTestDB creates a PostgreSQL container for testing
// This function is designed to be called from test files to get a real database
func SetupTestDB(t *testing.T) *TestDB {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "waygates_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to get host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to get port: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=waygates_test sslmode=disable", host, port.Port())

	// Wait for postgres to be ready
	time.Sleep(2 * time.Second)

	// Connect with GORM (silent logger for cleaner test output)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	migrateURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", "test", "test", host, port.Port(), "waygates_test")
	m, err := migrate.New(
		"file://../../migrations",
		migrateURL,
	)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestDB{
		Container: container,
		DB:        db,
		DSN:       dsn,
		ctx:       ctx,
	}
}

// Cleanup terminates the container
func (tdb *TestDB) Cleanup(t *testing.T) {
	if tdb.Container != nil {
		if err := tdb.Container.Terminate(tdb.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// CleanTables truncates all test tables for clean state between tests
func (tdb *TestDB) CleanTables(t *testing.T) {
	// Delete in correct order due to foreign keys
	tdb.DB.Exec("DELETE FROM audit_logs")
	tdb.DB.Exec("DELETE FROM proxies")
	tdb.DB.Exec("DELETE FROM settings")
	tdb.DB.Exec("DELETE FROM users")
}

// CreateTestUser creates a test user and returns the user
func CreateTestUser(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()
	suffix := time.Now().UnixNano()
	user := &models.User{
		Name:         "Test User",
		Username:     fmt.Sprintf("testuser_%d", suffix),
		Email:        fmt.Sprintf("test_%d@example.com", suffix),
		PasswordHash: "$2a$10$fakehashfortest",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// CreateTestProxy creates a test proxy with the given parameters
func CreateTestProxy(t *testing.T, db *gorm.DB, userID int, name, hostname, proxyType string) *models.Proxy {
	t.Helper()
	proxy := &models.Proxy{
		Type:      proxyType,
		Name:      name,
		Hostname:  hostname,
		IsActive:  true,
		CreatedBy: userID,
	}

	// Add type-specific configuration
	switch proxyType {
	case models.ProxyTypeReverseProxy:
		proxy.Upstreams = []interface{}{
			map[string]interface{}{
				"host":   "backend",
				"port":   8080,
				"scheme": "http",
			},
		}
	case models.ProxyTypeRedirect:
		proxy.RedirectConfig = models.JSONField{
			"target":      "https://target.example.com",
			"status_code": 301,
		}
	case models.ProxyTypeStatic:
		proxy.StaticConfig = models.JSONField{
			"root_path":  "/var/www",
			"index_file": "index.html",
		}
	}

	if err := db.Create(proxy).Error; err != nil {
		t.Fatalf("Failed to create test proxy: %v", err)
	}
	return proxy
}

// CreateTestSetting creates a test setting
func CreateTestSetting(t *testing.T, db *gorm.DB, key, value string) *models.Setting {
	t.Helper()
	setting := &models.Setting{
		Key:   key,
		Value: value,
	}
	if err := db.Create(setting).Error; err != nil {
		t.Fatalf("Failed to create test setting: %v", err)
	}
	return setting
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}

// TimePtr returns a pointer to a time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// CreateTestAuditLog creates a test audit log entry with the given parameters
func CreateTestAuditLog(t *testing.T, db *gorm.DB, userID *int, action, status string) *models.AuditLog {
	t.Helper()
	resourceType := "proxy"
	resourceID := 1
	resourceName := "test-proxy"
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0 Test Agent"

	log := &models.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		ResourceName: &resourceName,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Status:       status,
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("Failed to create test audit log: %v", err)
	}
	return log
}

// CreateTestAuditLogWithDetails creates a test audit log with custom details
func CreateTestAuditLogWithDetails(t *testing.T, db *gorm.DB, opts AuditLogTestOptions) *models.AuditLog {
	t.Helper()
	log := &models.AuditLog{
		UserID:       opts.UserID,
		Action:       opts.Action,
		ResourceType: opts.ResourceType,
		ResourceID:   opts.ResourceID,
		ResourceName: opts.ResourceName,
		Details:      opts.Details,
		IPAddress:    opts.IPAddress,
		UserAgent:    opts.UserAgent,
		Status:       opts.Status,
		ErrorMessage: opts.ErrorMessage,
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("Failed to create test audit log: %v", err)
	}
	return log
}

// AuditLogTestOptions holds options for creating test audit logs
type AuditLogTestOptions struct {
	UserID       *int
	Action       string
	ResourceType *string
	ResourceID   *int
	ResourceName *string
	Details      models.JSONField
	IPAddress    *string
	UserAgent    *string
	Status       string
	ErrorMessage *string
}
