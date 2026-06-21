package service

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// validReverseProxyFixture returns a *models.Proxy that passes models.Proxy.Validate():
// a valid Type, a non-empty Name, a valid Hostname, and at least one upstream.
func validReverseProxyFixture() *models.Proxy {
	return &models.Proxy{
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Fixture Proxy",
		Hostname: "fixture.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{"address": "http://localhost:8080"},
		},
	}
}

func importableProxy(name, hostname string) *models.Proxy {
	p := validReverseProxyFixture()
	p.Name = name
	p.Hostname = hostname
	return p
}

func TestImportProxies_DryRun_ClassifiesAndWritesNothing(t *testing.T) {
	created := 0
	repo := &MockProxyRepository{
		ExistingHostnamesFunc: func(_ []string) (map[string]bool, error) {
			return map[string]bool{"taken.example.com": true}, nil
		},
		CreateFunc: func(_ *models.Proxy) error { created++; return nil },
	}
	syncer := &MockProxySyncer{}
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: syncer})

	inputs := []ImportInput{
		{Proxy: importableProxy("ok", "fresh.example.com")},   // valid
		{Proxy: importableProxy("dupe", "taken.example.com")}, // conflict (DB)
		{Proxy: importableProxy("dup2", "fresh.example.com")}, // conflict (intra-batch)
		{Proxy: &models.Proxy{Type: "reverse_proxy"}},         // invalid (Validate fails)
		{DecodeError: "invalid item format: bad port"},        // invalid (decode)
	}
	report := svc.ImportProxies(inputs, true, 7)

	assert.Equal(t, 0, created, "dry-run must not create")
	assert.Equal(t, 5, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Importable)
	assert.Equal(t, 2, report.Summary.Conflicts)
	assert.Equal(t, 2, report.Summary.Invalid)
	assert.Equal(t, "valid", report.Items[0].Status)
	assert.Equal(t, "conflict", report.Items[1].Status)
	assert.Equal(t, "conflict", report.Items[2].Status)
	assert.Equal(t, "invalid", report.Items[3].Status)
	assert.Equal(t, "invalid", report.Items[4].Status)
	assert.NotEmpty(t, report.Items[4].Reason)
}

func TestImportProxies_Apply_CreatesValidSkipsConflicts(t *testing.T) {
	createdHosts := []string{}
	existing := map[string]bool{"taken.example.com": true}
	repo := &MockProxyRepository{
		// Import pre-check: batched existing-hostname lookup (called once upfront).
		ExistingHostnamesFunc: func(hostnames []string) (map[string]bool, error) {
			out := map[string]bool{}
			for _, h := range hostnames {
				if existing[h] {
					out[h] = true
				}
			}
			return out, nil
		},
		// CreateProxy's own per-item guard still runs on the apply path.
		HostnameExistsFunc: func(hostname string, _ int) (bool, error) { return existing[hostname], nil },
		CreateFunc: func(p *models.Proxy) error {
			createdHosts = append(createdHosts, p.Hostname)
			existing[p.Hostname] = true
			return nil
		},
	}
	syncer := &MockProxySyncer{} // SyncProxy no-op (nil func)
	svc := NewProxyService(ProxyServiceConfig{Repo: repo, SyncService: syncer})

	inputs := []ImportInput{
		{Proxy: importableProxy("a", "a.example.com")},
		{Proxy: importableProxy("b", "taken.example.com")}, // conflict → skipped
		{Proxy: importableProxy("c", "c.example.com")},
	}
	report := svc.ImportProxies(inputs, false, 7)

	assert.Equal(t, []string{"a.example.com", "c.example.com"}, createdHosts)
	assert.Equal(t, 2, report.Summary.Created)
	assert.Equal(t, 1, report.Summary.Conflicts)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, "created", report.Items[0].Status)
	assert.Equal(t, "skipped_conflict", report.Items[1].Status)
	assert.Equal(t, "created", report.Items[2].Status)
}

// importTestDB holds the integration test database connection and container.
type importTestDB struct {
	container testcontainers.Container
	db        *gorm.DB
	ctx       context.Context
}

// setupImportTestDB spins up a PostgreSQL container, runs migrations, and returns
// a connected *gorm.DB. It mirrors the repository package's SetupTestDB helper
// (which lives in a _test.go file and is therefore not importable here).
func setupImportTestDB(t *testing.T) *importTestDB {
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

	// Wait for postgres to be ready.
	time.Sleep(2 * time.Second)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	migrateURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", "test", "test", host, port.Port(), "waygates_test")
	m, err := migrate.New("file://../../migrations", migrateURL)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &importTestDB{container: container, db: db, ctx: ctx}
}

func (tdb *importTestDB) cleanup(t *testing.T) {
	t.Helper()
	if tdb.container != nil {
		if err := tdb.container.Terminate(tdb.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// createImportTestUser creates a user so created proxies satisfy the CreatedBy FK.
func createImportTestUser(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()
	suffix := time.Now().UnixNano()
	user := &models.User{
		Name:         "Import Test User",
		Username:     fmt.Sprintf("importuser_%d", suffix),
		Email:        fmt.Sprintf("import_%d@example.com", suffix),
		PasswordHash: "$2a$10$fakehashfortest",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestImportProxies_Integration(t *testing.T) {
	tdb := setupImportTestDB(t)
	defer tdb.cleanup(t)

	user := createImportTestUser(t, tdb.db)

	repo := repository.NewProxyRepository(tdb.db)
	svc := NewProxyService(ProxyServiceConfig{
		Repo:        repo,
		SyncService: &MockProxySyncer{}, // no-op SyncProxy
	})

	// Seed an existing proxy that should conflict with the import.
	existing := &models.Proxy{
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Existing",
		Hostname:  "existing.example.com",
		IsActive:  true,
		CreatedBy: user.ID,
		Upstreams: []interface{}{
			map[string]interface{}{"address": "http://localhost:8080"},
		},
	}
	require.NoError(t, repo.Create(existing))

	inputs := []ImportInput{
		{Proxy: importableProxy("existing-dupe", "existing.example.com")}, // conflict
		{Proxy: importableProxy("new1", "new1.example.com")},              // importable
	}

	countProxies := func() int64 {
		var n int64
		require.NoError(t, tdb.db.Model(&models.Proxy{}).Count(&n).Error)
		return n
	}

	// Dry-run: reports 1 conflict + 1 importable, writes nothing.
	dry := svc.ImportProxies(inputs, true, user.ID)
	assert.Equal(t, 2, dry.Summary.Total)
	assert.Equal(t, 1, dry.Summary.Importable)
	assert.Equal(t, 1, dry.Summary.Conflicts)
	assert.Equal(t, 0, dry.Summary.Created)
	assert.Equal(t, ImportStatusConflict, dry.Items[0].Status)
	assert.Equal(t, ImportStatusValid, dry.Items[1].Status)
	assert.Equal(t, int64(1), countProxies(), "dry-run must not write to the DB")

	// Apply: creates new1 only; existing untouched.
	apply := svc.ImportProxies(inputs, false, user.ID)
	assert.Equal(t, 2, apply.Summary.Total)
	assert.Equal(t, 1, apply.Summary.Created)
	assert.Equal(t, 1, apply.Summary.Conflicts)
	assert.Equal(t, 0, apply.Summary.Failed)
	assert.Equal(t, ImportStatusSkippedConflict, apply.Items[0].Status)
	assert.Equal(t, ImportStatusCreated, apply.Items[1].Status)
	assert.Equal(t, int64(2), countProxies(), "apply should create exactly one new proxy")

	// The existing proxy is unchanged.
	fetched, err := repo.GetByHostname("existing.example.com")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, fetched.ID)
	assert.Equal(t, "Existing", fetched.Name)

	// The newly imported proxy exists.
	created, err := repo.GetByHostname("new1.example.com")
	require.NoError(t, err)
	assert.Equal(t, "new1", created.Name)
}
