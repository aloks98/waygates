package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/aloks98/waygates/backend/internal/config"
)

// testDB holds the test database connection and container
type testDB struct {
	Container testcontainers.Container
	DB        *sql.DB
	DSN       string
	Host      string
	Port      string
	ctx       context.Context
}

// setupTestDB creates a PostgreSQL container for integration testing
func setupTestDB(t *testing.T) *testDB {
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
			"POSTGRES_DB":       "goauth_test",
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

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=goauth_test sslmode=disable", host, port.Port())

	// Wait for postgres to be ready
	time.Sleep(2 * time.Second)

	// Connect with database/sql
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	migrateURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", "test", "test", host, port.Port(), "goauth_test")
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

	return &testDB{
		Container: container,
		DB:        db,
		DSN:       dsn,
		Host:      host,
		Port:      port.Port(),
		ctx:       ctx,
	}
}

// cleanup terminates the container
func (tdb *testDB) cleanup(t *testing.T) {
	if tdb.DB != nil {
		_ = tdb.DB.Close()
	}
	if tdb.Container != nil {
		if err := tdb.Container.Terminate(tdb.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// createTestRBACFile creates a temporary RBAC config file for testing
func createTestRBACFile(t *testing.T) string {
	t.Helper()

	rbacContent := `
roles:
  - key: admin
    label: Administrator
    permissions:
      - users:read
      - users:write
      - users:delete
  - key: viewer
    label: Viewer
    permissions:
      - users:read
permission_groups:
  - name: Users
    permissions:
      - key: users:read
        label: Read Users
      - key: users:write
        label: Write Users
      - key: users:delete
        label: Delete Users
`

	tmpDir := t.TempDir()
	rbacPath := filepath.Join(tmpDir, "rbac.yaml")
	err := os.WriteFile(rbacPath, []byte(rbacContent), 0644)
	require.NoError(t, err)

	return rbacPath
}

// =============================================================================
// Integration Tests for NewAuth
// =============================================================================

func TestNewAuth_Integration_Success(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.cleanup(t)

	rbacPath := createTestRBACFile(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-must-be-at-least-32-bytes-long-for-security",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: rbacPath,
		},
	}

	auth, err := NewAuth(cfg, tdb.DB)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Verify the auth instance is functional
	adapter := auth.Adapter()
	require.NotNil(t, adapter)

	// Clean up auth resources
	if auth.Auth != nil {
		_ = auth.Auth.Close()
	}
}

func TestNewAuth_Integration_InvalidSecret(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.cleanup(t)

	rbacPath := createTestRBACFile(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "short", // Too short
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: rbacPath,
		},
	}

	auth, err := NewAuth(cfg, tdb.DB)
	// goauth should reject short secrets
	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "failed to create goauth instance")
}

func TestNewAuth_Integration_InvalidRBACPath(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.cleanup(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-must-be-at-least-32-bytes-long-for-security",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: "/nonexistent/path/to/rbac.yaml",
		},
	}

	auth, err := NewAuth(cfg, tdb.DB)
	// Should fail because RBAC file doesn't exist
	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "failed to create goauth instance")
}

func TestNewAuth_Integration_EmptyRBACPath(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.cleanup(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-must-be-at-least-32-bytes-long-for-security",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: "", // Empty - RBAC disabled
		},
	}

	// With empty RBAC path, goauth should initialize without RBAC
	auth, err := NewAuth(cfg, tdb.DB)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Clean up auth resources
	if auth.Auth != nil {
		_ = auth.Auth.Close()
	}
}

func TestNewAuth_Integration_FullLifecycle(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.cleanup(t)

	rbacPath := createTestRBACFile(t)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret-key-must-be-at-least-32-bytes-long-for-security",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			RBACPath: rbacPath,
		},
	}

	// Create auth instance
	auth, err := NewAuth(cfg, tdb.DB)
	require.NoError(t, err)
	require.NotNil(t, auth)
	defer func() {
		if auth.Auth != nil {
			_ = auth.Auth.Close()
		}
	}()

	ctx := context.Background()
	adapter := auth.Adapter()

	// Test token generation and validation
	tokenPair, err := auth.Auth.GenerateTokenPair(ctx, "user123", nil)
	require.NoError(t, err)
	require.NotEmpty(t, tokenPair.AccessToken)

	// Test token validation through adapter
	claims, err := adapter.ValidateAccessToken(ctx, tokenPair.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Extract user ID
	userID := adapter.ExtractUserID(claims)
	assert.Equal(t, "user123", userID)

	// Test permission setting directly (without role assignment to avoid role sync issues)
	err = auth.Auth.SetPermissions(ctx, "user123", []string{"users:read", "proxies:read"})
	require.NoError(t, err)

	hasPerm, err := adapter.HasPermission(ctx, "user123", "users:read")
	require.NoError(t, err)
	assert.True(t, hasPerm)

	hasPerm, err = adapter.HasPermission(ctx, "user123", "users:delete")
	require.NoError(t, err)
	assert.False(t, hasPerm)

	// Test HasAllPermissions
	hasAll, err := adapter.HasAllPermissions(ctx, "user123", []string{"users:read", "proxies:read"})
	require.NoError(t, err)
	assert.True(t, hasAll)

	// Test HasAnyPermission
	hasAny, err := adapter.HasAnyPermission(ctx, "user123", []string{"users:delete", "users:read"})
	require.NoError(t, err)
	assert.True(t, hasAny)
}
