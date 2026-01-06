package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm/logger"
)

// TestPostgresContainer holds the test container for database tests
type TestPostgresContainer struct {
	Container testcontainers.Container
	DSN       string
	Host      string
	Port      string
	ctx       context.Context
}

// setupTestPostgresContainer creates a PostgreSQL container for testing
func setupTestPostgresContainer(t *testing.T) *TestPostgresContainer {
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

	return &TestPostgresContainer{
		Container: container,
		DSN:       dsn,
		Host:      host,
		Port:      port.Port(),
		ctx:       ctx,
	}
}

// Cleanup terminates the container
func (tc *TestPostgresContainer) Cleanup(t *testing.T) {
	t.Helper()
	if tc.Container != nil {
		if err := tc.Container.Terminate(tc.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// GetPort returns the container port
func (tc *TestPostgresContainer) GetPort() nat.Port {
	port, _ := tc.Container.MappedPort(tc.ctx, "5432")
	return port
}

// =============================================================================
// Unit Tests (no database required)
// =============================================================================

func TestDefaultPoolConfig(t *testing.T) {
	t.Parallel()

	config := DefaultPoolConfig()

	assert.Equal(t, 25, config.MaxOpenConns, "MaxOpenConns should be 25")
	assert.Equal(t, 5, config.MaxIdleConns, "MaxIdleConns should be 5")
	assert.Equal(t, 5*time.Minute, config.ConnMaxLifetime, "ConnMaxLifetime should be 5 minutes")
	assert.Equal(t, 5*time.Minute, config.ConnMaxIdleTime, "ConnMaxIdleTime should be 5 minutes")
}

func TestPoolConfig_CustomValues(t *testing.T) {
	t.Parallel()

	config := PoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 3 * time.Minute,
	}

	assert.Equal(t, 100, config.MaxOpenConns)
	assert.Equal(t, 20, config.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, config.ConnMaxLifetime)
	assert.Equal(t, 3*time.Minute, config.ConnMaxIdleTime)
}

func TestClose_NilDB(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set DB to nil
	DB = nil

	// Close should return nil when DB is nil
	err := Close()
	assert.NoError(t, err, "Close should return nil when DB is nil")
}

func TestPing_NilDB(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Set DB to nil
	DB = nil

	// Ping should return an error when DB is nil
	err := Ping()
	require.Error(t, err, "Ping should return error when DB is nil")
	assert.Contains(t, err.Error(), "database not connected")
}

func TestPingDB_NilDB(t *testing.T) {
	t.Parallel()

	// PingDB should return an error when passed a nil db
	err := PingDB(nil)
	require.Error(t, err, "PingDB should return error when db is nil")
	assert.Contains(t, err.Error(), "database not connected")
}

// =============================================================================
// Integration Tests (require PostgreSQL container)
// =============================================================================

func TestConnect_Success(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err, "Connect should succeed with valid DSN")
	require.NotNil(t, db, "Connect should return a non-nil db")

	// Verify DB was set globally
	assert.Equal(t, db, DB, "Global DB should be set")

	// Clean up connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_ = sqlDB.Close()
}

func TestConnect_InvalidDSN(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Try to connect with invalid DSN
	db, err := Connect("host=invalid port=5432 user=invalid password=invalid dbname=invalid sslmode=disable", logger.Silent)

	require.Error(t, err, "Connect should fail with invalid DSN")
	assert.Nil(t, db, "Connect should return nil db on error")
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestConnectWithPool_Success(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	customConfig := PoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: 2 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}

	db, err := ConnectWithPool(tc.DSN, logger.Silent, customConfig)
	require.NoError(t, err, "ConnectWithPool should succeed with valid DSN")
	require.NotNil(t, db, "ConnectWithPool should return a non-nil db")

	// Verify the pool configuration was applied
	sqlDB, err := db.DB()
	require.NoError(t, err)

	stats := sqlDB.Stats()
	assert.Equal(t, 10, stats.MaxOpenConnections, "MaxOpenConns should be configured")

	// Clean up
	_ = sqlDB.Close()
}

func TestConnectWithPool_InvalidDSN(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	customConfig := PoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: 2 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}

	// Try to connect with invalid DSN
	db, err := ConnectWithPool("host=invalid port=5432 user=invalid password=invalid dbname=invalid sslmode=disable", logger.Silent, customConfig)

	require.Error(t, err, "ConnectWithPool should fail with invalid DSN")
	assert.Nil(t, db, "ConnectWithPool should return nil db on error")
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestConnectWithPool_DifferentLogLevels(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	logLevels := []struct {
		name  string
		level logger.LogLevel
	}{
		{"Silent", logger.Silent},
		{"Error", logger.Error},
		{"Warn", logger.Warn},
		{"Info", logger.Info},
	}

	for _, ll := range logLevels {
		t.Run(ll.name, func(t *testing.T) {
			db, err := ConnectWithPool(tc.DSN, ll.level, DefaultPoolConfig())
			require.NoError(t, err, "ConnectWithPool should succeed with %s log level", ll.name)
			require.NotNil(t, db)

			// Clean up
			sqlDB, _ := db.DB()
			_ = sqlDB.Close()
		})
	}
}

func TestClose_WithValidDB(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Connect first
	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close should succeed
	err = Close()
	assert.NoError(t, err, "Close should succeed with valid DB")

	// Verify connection is closed by trying to ping
	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Ping()
	assert.Error(t, err, "Ping should fail after Close")
}

func TestPing_WithValidDB(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Connect first
	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Ping should succeed
	err = Ping()
	assert.NoError(t, err, "Ping should succeed with valid DB connection")

	// Clean up
	_ = Close()
}

func TestPingDB_WithValidDB(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Connect first
	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db)

	// PingDB should succeed with the connected db
	err = PingDB(db)
	assert.NoError(t, err, "PingDB should succeed with valid DB connection")

	// Clean up
	_ = Close()
}

func TestPingDB_AfterClose(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Connect first
	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close the connection
	err = Close()
	require.NoError(t, err)

	// PingDB should fail after close
	err = PingDB(db)
	assert.Error(t, err, "PingDB should fail after connection is closed")
}

func TestConnect_GlobalDBIsSet(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Ensure DB is nil before test
	DB = nil

	// Connect
	db, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify global DB is set
	assert.NotNil(t, DB, "Global DB should be set after Connect")
	assert.Equal(t, db, DB, "Global DB should match returned db")

	// Clean up
	_ = Close()
}

func TestConnectWithPool_PoolConfigApplied(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	testCases := []struct {
		name   string
		config PoolConfig
	}{
		{
			name: "MinimalPool",
			config: PoolConfig{
				MaxOpenConns:    1,
				MaxIdleConns:    1,
				ConnMaxLifetime: 1 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
			},
		},
		{
			name: "LargePool",
			config: PoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    25,
				ConnMaxLifetime: 30 * time.Minute,
				ConnMaxIdleTime: 15 * time.Minute,
			},
		},
		{
			name:   "DefaultPool",
			config: DefaultPoolConfig(),
		},
	}

	for _, tc2 := range testCases {
		t.Run(tc2.name, func(t *testing.T) {
			db, err := ConnectWithPool(tc.DSN, logger.Silent, tc2.config)
			require.NoError(t, err)
			require.NotNil(t, db)

			sqlDB, err := db.DB()
			require.NoError(t, err)

			stats := sqlDB.Stats()
			assert.Equal(t, tc2.config.MaxOpenConns, stats.MaxOpenConnections)

			// Clean up
			_ = sqlDB.Close()
		})
	}
}

func TestConnect_MultipleConnections(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// First connection
	db1, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db1)

	firstDB := DB

	// Second connection (should replace the first)
	db2, err := Connect(tc.DSN, logger.Silent)
	require.NoError(t, err)
	require.NotNil(t, db2)

	// Global DB should now point to the second connection
	assert.Equal(t, db2, DB, "Global DB should be updated to the new connection")
	assert.NotEqual(t, firstDB, DB, "Global DB should be different from first connection")

	// Clean up both connections
	sqlDB1, _ := db1.DB()
	_ = sqlDB1.Close()

	sqlDB2, _ := db2.DB()
	_ = sqlDB2.Close()
}

func TestConnect_UnreachableHost(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Try to connect to an unreachable host
	// Using a non-routable IP address to ensure it fails
	db, err := Connect("host=192.0.2.1 port=5432 user=test password=test dbname=test sslmode=disable connect_timeout=1", logger.Silent)

	require.Error(t, err, "Connect should fail with unreachable host")
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestConnect_WrongPort(t *testing.T) {
	t.Parallel()

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Try to connect to the wrong port
	db, err := Connect("host=localhost port=59999 user=test password=test dbname=test sslmode=disable connect_timeout=1", logger.Silent)

	require.Error(t, err, "Connect should fail with wrong port")
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestConnect_InvalidCredentials(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Try to connect with wrong credentials
	invalidDSN := fmt.Sprintf("host=%s port=%s user=wronguser password=wrongpass dbname=waygates_test sslmode=disable", tc.Host, tc.Port)

	db, err := Connect(invalidDSN, logger.Silent)

	require.Error(t, err, "Connect should fail with invalid credentials")
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

func TestConnect_NonExistentDatabase(t *testing.T) {
	tc := setupTestPostgresContainer(t)
	defer tc.Cleanup(t)

	// Save original DB value
	originalDB := DB
	defer func() { DB = originalDB }()

	// Try to connect to a non-existent database
	invalidDSN := fmt.Sprintf("host=%s port=%s user=test password=test dbname=nonexistent_db sslmode=disable", tc.Host, tc.Port)

	db, err := Connect(invalidDSN, logger.Silent)

	require.Error(t, err, "Connect should fail with non-existent database")
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to database")
}
