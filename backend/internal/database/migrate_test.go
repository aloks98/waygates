package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MigrateTestContainer holds the test container for migration tests
type MigrateTestContainer struct {
	Container   testcontainers.Container
	DatabaseURL string
	Host        string
	Port        string
	ctx         context.Context
}

// setupMigrateTestContainer creates a PostgreSQL container for migration testing
func setupMigrateTestContainer(t *testing.T) *MigrateTestContainer {
	t.Helper()

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
			"POSTGRES_DB":       "postgres", // Default postgres database for initial connection
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

	// Wait for postgres to be ready
	time.Sleep(2 * time.Second)

	// Use a unique database name for each test to avoid conflicts
	dbName := fmt.Sprintf("migrate_test_%d", time.Now().UnixNano())
	databaseURL := fmt.Sprintf("postgres://test:test@%s:%s/%s?sslmode=disable", host, port.Port(), dbName)

	return &MigrateTestContainer{
		Container:   container,
		DatabaseURL: databaseURL,
		Host:        host,
		Port:        port.Port(),
		ctx:         ctx,
	}
}

// Cleanup terminates the container
func (mtc *MigrateTestContainer) Cleanup(t *testing.T) {
	t.Helper()
	if mtc.Container != nil {
		if err := mtc.Container.Terminate(mtc.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// GetDatabaseURL returns a database URL with the given database name
func (mtc *MigrateTestContainer) GetDatabaseURL(dbName string) string {
	return fmt.Sprintf("postgres://test:test@%s:%s/%s?sslmode=disable", mtc.Host, mtc.Port, dbName)
}

// GetPostgresURL returns the URL to the default postgres database
func (mtc *MigrateTestContainer) GetPostgresURL() string {
	return fmt.Sprintf("postgres://test:test@%s:%s/postgres?sslmode=disable", mtc.Host, mtc.Port)
}

// =============================================================================
// Unit Tests (no database required)
// =============================================================================

func TestEnsureDatabase_InvalidURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		databaseURL string
		wantErr     bool
	}{
		{
			name:        "InvalidURLFormat",
			databaseURL: "not-a-valid-url",
			wantErr:     true,
		},
		{
			name:        "MissingDatabaseName",
			databaseURL: "postgres://test:test@localhost:5432/",
			wantErr:     true,
		},
		{
			name:        "MissingDatabaseNameNoSlash",
			databaseURL: "postgres://test:test@localhost:5432",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := EnsureDatabase(tt.databaseURL)
			if tt.wantErr {
				require.Error(t, err, "EnsureDatabase should fail with invalid URL: %s", tt.databaseURL)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Integration Tests (require PostgreSQL container)
// =============================================================================

func TestEnsureDatabase_CreateNewDatabase(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Use a unique database name
	dbName := fmt.Sprintf("new_test_db_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Ensure database creates it
	err := EnsureDatabase(databaseURL)
	require.NoError(t, err, "EnsureDatabase should create a new database")

	// Verify database exists by connecting to it
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = db.Ping()
	assert.NoError(t, err, "Should be able to connect to newly created database")
}

func TestEnsureDatabase_ExistingDatabase(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create database first
	dbName := fmt.Sprintf("existing_db_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// First call creates the database
	err := EnsureDatabase(databaseURL)
	require.NoError(t, err)

	// Second call should succeed (database already exists)
	err = EnsureDatabase(databaseURL)
	assert.NoError(t, err, "EnsureDatabase should succeed when database already exists")
}

func TestEnsureDatabase_ConnectionFailure(t *testing.T) {
	t.Parallel()

	// Try with an unreachable host
	databaseURL := "postgres://test:test@192.0.2.1:5432/testdb?sslmode=disable&connect_timeout=1"

	err := EnsureDatabase(databaseURL)
	require.Error(t, err, "EnsureDatabase should fail when cannot connect")
	// The error could be from connection or from the query, depending on when the timeout occurs
	assert.True(t, err != nil, "Should return an error when host is unreachable")
}

func TestRunMigrations_Success(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("migrations_test_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Navigate to project root (3 levels up from backend/internal/database)
	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrations(databaseURL)
	require.NoError(t, err, "RunMigrations should succeed")

	// Verify migrations ran by checking if tables exist
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Check if proxies table exists (from migration 000001)
	var tableExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'proxies')").Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "proxies table should exist after migrations")

	// Check if users table exists (from migration 000004)
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "users table should exist after migrations")
}

func TestRunMigrations_NoChange(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("migrations_nochange_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Run migrations first time
	err = RunMigrations(databaseURL)
	require.NoError(t, err)

	// Run migrations again - should return nil (ErrNoChange is handled)
	err = RunMigrations(databaseURL)
	assert.NoError(t, err, "RunMigrations should handle ErrNoChange gracefully")
}

func TestRunMigrations_InvalidMigrationPath(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("migrations_invalid_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to a directory where migrations don't exist
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Go to temp directory where no migrations exist
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run migrations should fail because migrations directory doesn't exist
	err = RunMigrations(databaseURL)
	require.Error(t, err, "RunMigrations should fail when migrations directory doesn't exist")
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestRollbackMigrations_Success(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("rollback_test_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// First run all migrations
	err = RunMigrations(databaseURL)
	require.NoError(t, err)

	// Get the version before rollback
	versionBefore, _, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err)
	require.Greater(t, versionBefore, uint(0), "Should have at least one migration applied")

	// Rollback 1 step
	err = RollbackMigrations(databaseURL, 1)
	require.NoError(t, err, "RollbackMigrations should succeed")

	// Get the version after rollback
	versionAfter, _, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err)

	// Version should have decreased
	assert.Less(t, versionAfter, versionBefore, "Version should decrease after rollback")
}

func TestRollbackMigrations_NoMigrationsApplied(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("rollback_nochange_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// First ensure database exists
	err = EnsureDatabase(databaseURL)
	require.NoError(t, err)

	// Try to rollback when no migrations have been applied
	// This may return an error or nil depending on migration state
	err = RollbackMigrations(databaseURL, 1)
	// We just verify it doesn't panic - the behavior depends on whether
	// the schema_migrations table exists
	t.Logf("RollbackMigrations with no migrations returned: %v", err)
}

func TestRollbackMigrations_InvalidMigrationPath(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("rollback_invalid_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Ensure database exists first
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Go to temp directory where no migrations exist
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Rollback should fail because migrations directory doesn't exist
	err = RollbackMigrations(databaseURL, 1)
	require.Error(t, err, "RollbackMigrations should fail when migrations directory doesn't exist")
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestGetMigrationVersion_Success(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("version_test_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrations(databaseURL)
	require.NoError(t, err)

	// Get version
	version, dirty, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err)
	assert.Greater(t, version, uint(0), "Version should be greater than 0 after migrations")
	assert.False(t, dirty, "Migration should not be in dirty state")
}

func TestGetMigrationVersion_NoMigrations(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("version_nomig_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Ensure database exists but don't run migrations
	err = EnsureDatabase(databaseURL)
	require.NoError(t, err)

	// Get version - should return 0 with no error (ErrNilVersion is handled)
	version, dirty, err := GetMigrationVersion(databaseURL)
	assert.NoError(t, err, "GetMigrationVersion should handle ErrNilVersion gracefully")
	assert.Equal(t, uint(0), version, "Version should be 0 when no migrations applied")
	assert.False(t, dirty, "Should not be dirty when no migrations applied")
}

func TestGetMigrationVersion_InvalidMigrationPath(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("version_invalid_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to temp directory where no migrations exist
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// GetMigrationVersion should fail because migrations directory doesn't exist
	_, _, err = GetMigrationVersion(databaseURL)
	require.Error(t, err, "GetMigrationVersion should fail when migrations directory doesn't exist")
	assert.Contains(t, err.Error(), "failed to create migrate instance")
}

func TestRollbackMigrations_MultipleSteps(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("rollback_multi_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Run all migrations
	err = RunMigrations(databaseURL)
	require.NoError(t, err)

	// Get initial version
	initialVersion, _, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err)
	require.Greater(t, initialVersion, uint(2), "Need at least 3 migrations to test multi-step rollback")

	// Rollback 2 steps
	err = RollbackMigrations(databaseURL, 2)
	require.NoError(t, err)

	// Get version after rollback
	afterVersion, _, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err)

	// Version should have decreased by 2
	assert.Equal(t, initialVersion-2, afterVersion, "Version should decrease by 2 after 2-step rollback")
}

func TestEnsureDatabase_SpecialCharactersInDbName(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Test with underscores (common pattern)
	dbName := fmt.Sprintf("test_db_with_underscores_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	err := EnsureDatabase(databaseURL)
	require.NoError(t, err, "EnsureDatabase should handle underscores in database name")

	// Verify database was created
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = db.Ping()
	assert.NoError(t, err)
}

func TestMigrationWorkflow_FullCycle(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Create a unique database for this test
	dbName := fmt.Sprintf("full_cycle_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Change to project root directory for migrations to be found
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	projectRoot := filepath.Join(origDir, "..", "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Step 1: EnsureDatabase
	err = EnsureDatabase(databaseURL)
	require.NoError(t, err, "Step 1: EnsureDatabase should succeed")

	// Step 2: Get initial version (should be 0)
	version, dirty, err := GetMigrationVersion(databaseURL)
	require.NoError(t, err, "Step 2: GetMigrationVersion should succeed")
	assert.Equal(t, uint(0), version, "Initial version should be 0")
	assert.False(t, dirty)

	// Step 3: Run migrations
	err = RunMigrations(databaseURL)
	require.NoError(t, err, "Step 3: RunMigrations should succeed")

	// Step 4: Get version after migrations
	version, dirty, err = GetMigrationVersion(databaseURL)
	require.NoError(t, err, "Step 4: GetMigrationVersion should succeed")
	assert.Greater(t, version, uint(0), "Version should be > 0 after migrations")
	assert.False(t, dirty)

	initialVersion := version

	// Step 5: Rollback 1 migration
	err = RollbackMigrations(databaseURL, 1)
	require.NoError(t, err, "Step 5: RollbackMigrations should succeed")

	// Step 6: Verify version decreased
	version, dirty, err = GetMigrationVersion(databaseURL)
	require.NoError(t, err, "Step 6: GetMigrationVersion should succeed")
	assert.Equal(t, initialVersion-1, version, "Version should decrease by 1")
	assert.False(t, dirty)

	// Step 7: Re-run migrations (should apply the rolled-back migration)
	err = RunMigrations(databaseURL)
	require.NoError(t, err, "Step 7: RunMigrations should succeed")

	// Step 8: Verify version is back to initial
	version, _, err = GetMigrationVersion(databaseURL)
	require.NoError(t, err, "Step 8: GetMigrationVersion should succeed")
	assert.Equal(t, initialVersion, version, "Version should be back to initial after re-running migrations")
}

func TestEnsureDatabase_SequentialCalls(t *testing.T) {
	mtc := setupMigrateTestContainer(t)
	defer mtc.Cleanup(t)

	// Use the same database name for multiple calls
	dbName := fmt.Sprintf("sequential_test_%d", time.Now().UnixNano())
	databaseURL := mtc.GetDatabaseURL(dbName)

	// Run EnsureDatabase multiple times sequentially - should all succeed
	for i := 0; i < 3; i++ {
		err := EnsureDatabase(databaseURL)
		require.NoError(t, err, "EnsureDatabase call %d should succeed", i+1)
	}

	// Verify database exists
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	err = db.Ping()
	assert.NoError(t, err, "Database should exist after multiple EnsureDatabase calls")
}
