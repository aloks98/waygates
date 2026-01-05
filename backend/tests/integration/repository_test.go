package integration

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
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// PostgresTestContainer holds the test container
type PostgresTestContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	DSN       string
	ctx       context.Context
}

// SetupPostgresContainer creates a PostgreSQL container for testing
func SetupPostgresContainer(t *testing.T) *PostgresTestContainer {
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

	// Connect with GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations using migrate library directly
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

	return &PostgresTestContainer{
		Container: container,
		DB:        db,
		DSN:       dsn,
		ctx:       ctx,
	}
}

// Cleanup terminates the container
func (p *PostgresTestContainer) Cleanup(t *testing.T) {
	if p.Container != nil {
		if err := p.Container.Terminate(p.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// createTestUser creates a test user and returns their ID
func createTestUser(t *testing.T, db *gorm.DB) int {
	// Generate unique username/email using current time
	suffix := time.Now().UnixNano()
	user := &models.User{
		Name:         "Test User",
		Username:     fmt.Sprintf("testuser_%d", suffix),
		Email:        fmt.Sprintf("test_%d@example.com", suffix),
		PasswordHash: "$2a$10$fakehashfortest", // Fake hash for testing
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user.ID
}

// TestProxyRepository_CRUD tests basic CRUD operations
func TestProxyRepository_CRUD(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	// Create test user first (for foreign key)
	userID := createTestUser(t, tc.DB)

	repo := repository.NewProxyRepository(tc.DB)

	// Create proxy and track its ID for subsequent tests
	proxy := &models.Proxy{
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Test Proxy",
		Hostname:  "test.example.com",
		IsActive:  true,
		CreatedBy: userID,
		Upstreams: models.JSONField{
			"host":   "backend",
			"port":   8080,
			"scheme": "http",
		},
	}

	// Test Create
	t.Run("Create", func(t *testing.T) {
		err := repo.Create(proxy)
		if err != nil {
			t.Fatalf("Failed to create proxy: %v", err)
		}

		if proxy.ID == 0 {
			t.Error("Expected proxy ID to be set after create")
		}
	})

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		fetched, err := repo.GetByID(proxy.ID)
		if err != nil {
			t.Fatalf("Failed to get proxy: %v", err)
		}

		if fetched.Name != "Test Proxy" {
			t.Errorf("Expected name 'Test Proxy', got '%s'", fetched.Name)
		}
		if fetched.Hostname != "test.example.com" {
			t.Errorf("Expected hostname 'test.example.com', got '%s'", fetched.Hostname)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		proxies, total, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("Failed to list proxies: %v", err)
		}

		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
		if len(proxies) != 1 {
			t.Errorf("Expected 1 proxy, got %d", len(proxies))
		}
	})

	// Test HostnameExists
	t.Run("HostnameExists", func(t *testing.T) {
		exists, err := repo.HostnameExists("test.example.com", 0)
		if err != nil {
			t.Fatalf("Failed to check hostname: %v", err)
		}
		if !exists {
			t.Error("Expected hostname to exist")
		}

		exists, err = repo.HostnameExists("nonexistent.example.com", 0)
		if err != nil {
			t.Fatalf("Failed to check hostname: %v", err)
		}
		if exists {
			t.Error("Expected hostname not to exist")
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		fetched, _ := repo.GetByID(proxy.ID)
		fetched.Name = "Updated Proxy"

		err := repo.Update(fetched)
		if err != nil {
			t.Fatalf("Failed to update proxy: %v", err)
		}

		updated, _ := repo.GetByID(proxy.ID)
		if updated.Name != "Updated Proxy" {
			t.Errorf("Expected name 'Updated Proxy', got '%s'", updated.Name)
		}
	})

	// Test UpdateStatus
	t.Run("UpdateStatus", func(t *testing.T) {
		err := repo.UpdateStatus(proxy.ID, false)
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		fetched, _ := repo.GetByID(proxy.ID)
		if fetched.IsActive {
			t.Error("Expected proxy to be inactive")
		}
	})

	// Test GetStats
	t.Run("GetStats", func(t *testing.T) {
		stats, err := repo.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.Total != 1 {
			t.Errorf("Expected total 1, got %d", stats.Total)
		}
		if stats.Inactive != 1 {
			t.Errorf("Expected inactive 1, got %d", stats.Inactive)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(proxy.ID)
		if err != nil {
			t.Fatalf("Failed to delete proxy: %v", err)
		}

		_, err = repo.GetByID(proxy.ID)
		if err == nil {
			t.Error("Expected error when getting deleted proxy")
		}
	})
}

// TestProxyRepository_List_Filters tests list with various filters
func TestProxyRepository_List_Filters(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	// Create test user first (for foreign key)
	userID := createTestUser(t, tc.DB)

	repo := repository.NewProxyRepository(tc.DB)

	// Create test data - note: all initially active due to DB default, then disable one
	proxies := []*models.Proxy{
		{Type: models.ProxyTypeReverseProxy, Name: "API Backend", Hostname: "api.example.com", CreatedBy: userID},
		{Type: models.ProxyTypeRedirect, Name: "Old Site Redirect", Hostname: "old.example.com", CreatedBy: userID, RedirectConfig: models.JSONField{"target": "https://new.example.com"}},
		{Type: models.ProxyTypeStatic, Name: "Docs Site", Hostname: "docs.example.com", CreatedBy: userID, StaticConfig: models.JSONField{"root_path": "/var/www"}},
	}

	for _, p := range proxies {
		if err := repo.Create(p); err != nil {
			t.Fatalf("Failed to create proxy: %v", err)
		}
	}

	// Disable the third proxy (Docs Site) for status filter tests
	if err := repo.UpdateStatus(proxies[2].ID, false); err != nil {
		t.Fatalf("Failed to update proxy status: %v", err)
	}

	t.Run("FilterByType", func(t *testing.T) {
		result, total, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Types: []string{string(models.ProxyTypeReverseProxy)},
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if total != 1 {
			t.Errorf("Expected 1 reverse_proxy, got %d", total)
		}
		if len(result) != 1 || result[0].Type != models.ProxyTypeReverseProxy {
			t.Error("Expected only reverse_proxy type")
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		result, total, err := repo.List(repository.ProxyListParams{
			Page:   1,
			Limit:  10,
			Status: "active",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if total != 2 {
			t.Errorf("Expected 2 active proxies, got %d", total)
		}
		for _, p := range result {
			if !p.IsActive {
				t.Error("Expected all proxies to be active")
			}
		}
	})

	t.Run("Search", func(t *testing.T) {
		result, total, err := repo.List(repository.ProxyListParams{
			Page:   1,
			Limit:  10,
			Search: "API",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if total != 1 {
			t.Errorf("Expected 1 result for 'API', got %d", total)
		}
		if len(result) != 1 || result[0].Name != "API Backend" {
			t.Error("Expected to find 'API Backend'")
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		result, total, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 2,
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if total != 3 {
			t.Errorf("Expected total 3, got %d", total)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 results for page 1, got %d", len(result))
		}

		// Page 2
		result, _, err = repo.List(repository.ProxyListParams{
			Page:  2,
			Limit: 2,
		})
		if err != nil {
			t.Fatalf("Failed to list page 2: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1 result for page 2, got %d", len(result))
		}
	})
}

// TestSettingsRepository tests settings repository
func TestSettingsRepository(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	repo := repository.NewSettingsRepository(tc.DB)

	t.Run("SetAndGet", func(t *testing.T) {
		err := repo.Set("test_key", "test_value")
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		setting, err := repo.Get("test_key")
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if setting.Value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", setting.Value)
		}
	})

	t.Run("GetValue", func(t *testing.T) {
		value := repo.GetValue("test_key", "default")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}

		value = repo.GetValue("nonexistent", "default")
		if value != "default" {
			t.Errorf("Expected 'default', got '%s'", value)
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		_ = repo.Set("key1", "value1")
		_ = repo.Set("key2", "value2")

		all, err := repo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all: %v", err)
		}

		if len(all) < 2 {
			t.Errorf("Expected at least 2 settings, got %d", len(all))
		}
	})

	t.Run("Upsert", func(t *testing.T) {
		err := repo.Set("upsert_key", "initial")
		if err != nil {
			t.Fatalf("Failed to set: %v", err)
		}

		err = repo.Set("upsert_key", "updated")
		if err != nil {
			t.Fatalf("Failed to upsert: %v", err)
		}

		value := repo.GetValue("upsert_key", "")
		if value != "updated" {
			t.Errorf("Expected 'updated', got '%s'", value)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_ = repo.Set("to_delete", "value")

		err := repo.Delete("to_delete")
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		value := repo.GetValue("to_delete", "default")
		if value != "default" {
			t.Errorf("Expected 'default' after delete, got '%s'", value)
		}
	})

	t.Run("NotFoundSettings", func(t *testing.T) {
		settings, err := repo.GetNotFoundSettings()
		if err != nil {
			t.Fatalf("Failed to get 404 settings: %v", err)
		}
		if settings.Mode != "default" {
			t.Errorf("Expected default mode, got '%s'", settings.Mode)
		}

		err = repo.SetNotFoundSettings(&models.NotFoundSettings{
			Mode:        "redirect",
			RedirectURL: "https://example.com",
		})
		if err != nil {
			t.Fatalf("Failed to set 404 settings: %v", err)
		}

		settings, err = repo.GetNotFoundSettings()
		if err != nil {
			t.Fatalf("Failed to get updated 404 settings: %v", err)
		}
		if settings.Mode != "redirect" {
			t.Errorf("Expected redirect mode, got '%s'", settings.Mode)
		}
		if settings.RedirectURL != "https://example.com" {
			t.Errorf("Expected 'https://example.com', got '%s'", settings.RedirectURL)
		}
	})
}

// TestUserRepository tests user repository
func TestUserRepository(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	repo := repository.NewUserRepository(tc.DB)

	t.Run("Create", func(t *testing.T) {
		user := &models.User{
			Name:         "Test User",
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "$2a$10$fakehashfortest",
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set after create")
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		user := &models.User{
			Name:         "GetByID User",
			Username:     "getbyid",
			Email:        "getbyid@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if fetched.Name != "GetByID User" {
			t.Errorf("Expected name 'GetByID User', got '%s'", fetched.Name)
		}
		if fetched.Username != "getbyid" {
			t.Errorf("Expected username 'getbyid', got '%s'", fetched.Username)
		}
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		_, err := repo.GetByID(999999)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	t.Run("GetByEmail", func(t *testing.T) {
		user := &models.User{
			Name:         "Email User",
			Username:     "emailuser",
			Email:        "email@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByEmail("email@example.com")
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		if fetched.Email != "email@example.com" {
			t.Errorf("Expected email 'email@example.com', got '%s'", fetched.Email)
		}
	})

	t.Run("GetByUsernameOrEmail_ByUsername", func(t *testing.T) {
		user := &models.User{
			Name:         "Username User",
			Username:     "usernameonly",
			Email:        "usernameonly@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByUsernameOrEmail("usernameonly")
		if err != nil {
			t.Fatalf("Failed to get user by username: %v", err)
		}

		if fetched.Username != "usernameonly" {
			t.Errorf("Expected username 'usernameonly', got '%s'", fetched.Username)
		}
	})

	t.Run("GetByUsernameOrEmail_ByEmail", func(t *testing.T) {
		user := &models.User{
			Name:         "Email Only User",
			Username:     "emailonly",
			Email:        "onlyemail@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		fetched, err := repo.GetByUsernameOrEmail("onlyemail@example.com")
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		if fetched.Email != "onlyemail@example.com" {
			t.Errorf("Expected email 'onlyemail@example.com', got '%s'", fetched.Email)
		}
	})

	t.Run("GetByUsernameOrEmail_NotFound", func(t *testing.T) {
		_, err := repo.GetByUsernameOrEmail("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	t.Run("Count", func(t *testing.T) {
		// Get initial count
		initialCount, err := repo.Count()
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		// Create a new user
		user := &models.User{
			Name:         "Count User",
			Username:     "countuser",
			Email:        "count@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		// Count should have increased
		newCount, err := repo.Count()
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		if newCount != initialCount+1 {
			t.Errorf("Expected count %d, got %d", initialCount+1, newCount)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		user := &models.User{
			Name:         "Delete User",
			Username:     "deleteuser",
			Email:        "delete@example.com",
			PasswordHash: "$2a$10$fakehash",
		}
		_ = repo.Create(user)

		err := repo.Delete(user.ID)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		_, err = repo.GetByID(user.ID)
		if err == nil {
			t.Error("Expected error when getting deleted user")
		}
	})
}

// TestProxyRepository_Sorting tests proxy list sorting
func TestProxyRepository_Sorting(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	userID := createTestUser(t, tc.DB)
	repo := repository.NewProxyRepository(tc.DB)

	// Create test proxies with different names
	proxies := []*models.Proxy{
		{Type: models.ProxyTypeReverseProxy, Name: "Zebra Service", Hostname: "zebra.example.com", CreatedBy: userID},
		{Type: models.ProxyTypeReverseProxy, Name: "Alpha Service", Hostname: "alpha.example.com", CreatedBy: userID},
		{Type: models.ProxyTypeReverseProxy, Name: "Middle Service", Hostname: "middle.example.com", CreatedBy: userID},
	}

	for _, p := range proxies {
		if err := repo.Create(p); err != nil {
			t.Fatalf("Failed to create proxy: %v", err)
		}
	}

	t.Run("SortByNameASC", func(t *testing.T) {
		result, _, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "asc",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(result) < 3 {
			t.Fatalf("Expected at least 3 proxies, got %d", len(result))
		}
		if result[0].Name != "Alpha Service" {
			t.Errorf("Expected first 'Alpha Service', got '%s'", result[0].Name)
		}
	})

	t.Run("SortByNameDESC", func(t *testing.T) {
		result, _, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "desc",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(result) < 3 {
			t.Fatalf("Expected at least 3 proxies, got %d", len(result))
		}
		if result[0].Name != "Zebra Service" {
			t.Errorf("Expected first 'Zebra Service', got '%s'", result[0].Name)
		}
	})

	t.Run("SortByHostname", func(t *testing.T) {
		result, _, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "hostname",
			Order: "asc",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(result) < 3 {
			t.Fatalf("Expected at least 3 proxies, got %d", len(result))
		}
		if result[0].Hostname != "alpha.example.com" {
			t.Errorf("Expected first 'alpha.example.com', got '%s'", result[0].Hostname)
		}
	})

	t.Run("InvalidSortField_UsesDefault", func(t *testing.T) {
		// Should use default sort (created_at DESC) when invalid field provided
		result, _, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "invalid_field",
			Order: "asc",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		// Just verify it doesn't error - default sort is created_at DESC
		if len(result) < 3 {
			t.Fatalf("Expected at least 3 proxies, got %d", len(result))
		}
	})

	t.Run("InvalidSortOrder_UsesDefault", func(t *testing.T) {
		// Should use default order (DESC) when invalid order provided
		result, _, err := repo.List(repository.ProxyListParams{
			Page:  1,
			Limit: 10,
			Sort:  "name",
			Order: "INVALID",
		})
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		// Just verify it doesn't error
		if len(result) < 3 {
			t.Fatalf("Expected at least 3 proxies, got %d", len(result))
		}
	})
}

// TestProxyRepository_GetByHostname tests getting proxy by hostname
func TestProxyRepository_GetByHostname(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	userID := createTestUser(t, tc.DB)
	repo := repository.NewProxyRepository(tc.DB)

	// Create a test proxy
	proxy := &models.Proxy{
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Hostname Test",
		Hostname:  "hostname-test.example.com",
		CreatedBy: userID,
	}
	if err := repo.Create(proxy); err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	t.Run("Found", func(t *testing.T) {
		fetched, err := repo.GetByHostname("hostname-test.example.com")
		if err != nil {
			t.Fatalf("Failed to get by hostname: %v", err)
		}

		if fetched.ID != proxy.ID {
			t.Errorf("Expected ID %d, got %d", proxy.ID, fetched.ID)
		}
		if fetched.Name != "Hostname Test" {
			t.Errorf("Expected name 'Hostname Test', got '%s'", fetched.Name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := repo.GetByHostname("nonexistent.example.com")
		if err == nil {
			t.Error("Expected error for non-existent hostname")
		}
	})
}

// TestProxyRepository_Stats tests statistics with various proxy states
func TestProxyRepository_Stats(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	userID := createTestUser(t, tc.DB)
	repo := repository.NewProxyRepository(tc.DB)

	// Create proxies of different types and states
	proxies := []*models.Proxy{
		{Type: models.ProxyTypeReverseProxy, Name: "Active RP 1", Hostname: "active-rp-1.example.com", CreatedBy: userID, IsActive: true},
		{Type: models.ProxyTypeReverseProxy, Name: "Active RP 2", Hostname: "active-rp-2.example.com", CreatedBy: userID, IsActive: true},
		{Type: models.ProxyTypeRedirect, Name: "Active Redirect", Hostname: "active-redirect.example.com", CreatedBy: userID, IsActive: true, RedirectConfig: models.JSONField{"target": "https://target.com"}},
		{Type: models.ProxyTypeStatic, Name: "Inactive Static", Hostname: "inactive-static.example.com", CreatedBy: userID, IsActive: false, StaticConfig: models.JSONField{"root_path": "/var/www"}},
	}

	for _, p := range proxies {
		if err := repo.Create(p); err != nil {
			t.Fatalf("Failed to create proxy: %v", err)
		}
	}

	// Need to explicitly set IsActive=false for the last one as GORM may default it
	if err := repo.UpdateStatus(proxies[3].ID, false); err != nil {
		t.Fatalf("Failed to set inactive: %v", err)
	}

	stats, err := repo.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 4 {
		t.Errorf("Expected total 4, got %d", stats.Total)
	}
	if stats.Active != 3 {
		t.Errorf("Expected active 3, got %d", stats.Active)
	}
	if stats.Inactive != 1 {
		t.Errorf("Expected inactive 1, got %d", stats.Inactive)
	}
	if stats.ByType[models.ProxyTypeReverseProxy] != 2 {
		t.Errorf("Expected 2 reverse_proxy, got %d", stats.ByType[models.ProxyTypeReverseProxy])
	}
	if stats.ByType[models.ProxyTypeRedirect] != 1 {
		t.Errorf("Expected 1 redirect, got %d", stats.ByType[models.ProxyTypeRedirect])
	}
	if stats.ByType[models.ProxyTypeStatic] != 1 {
		t.Errorf("Expected 1 static, got %d", stats.ByType[models.ProxyTypeStatic])
	}
}

// TestProxyService_Integration tests proxy service with real database
func TestProxyService_Integration(t *testing.T) {
	tc := SetupPostgresContainer(t)
	defer tc.Cleanup(t)

	// Create test user first (for foreign key)
	userID := createTestUser(t, tc.DB)

	logger := zap.NewNop()
	proxyRepo := repository.NewProxyRepository(tc.DB)

	// Note: We can't fully test ProxyService without SyncService mocking
	// But we can test the repository-dependent parts

	t.Run("ProxyRepository_GetByID_NotFound", func(t *testing.T) {
		_, err := proxyRepo.GetByID(999)
		if err == nil {
			t.Error("Expected error for non-existent proxy")
		}
	})

	t.Run("ProxyRepository_HostnameExists_Exclude", func(t *testing.T) {
		// Create a proxy
		proxy := &models.Proxy{
			Type:      models.ProxyTypeReverseProxy,
			Name:      "Exclude Test",
			Hostname:  "exclude.example.com",
			IsActive:  true,
			CreatedBy: userID,
		}
		_ = proxyRepo.Create(proxy)

		// Check hostname exists (should be true)
		exists, _ := proxyRepo.HostnameExists("exclude.example.com", 0)
		if !exists {
			t.Error("Expected hostname to exist")
		}

		// Check hostname exists excluding own ID (should be false)
		exists, _ = proxyRepo.HostnameExists("exclude.example.com", proxy.ID)
		if exists {
			t.Error("Expected hostname not to exist when excluding own ID")
		}
	})

	_ = logger // Used for future service tests
}
