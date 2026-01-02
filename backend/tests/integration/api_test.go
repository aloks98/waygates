package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aloks98/goauth"
	sqlstore "github.com/aloks98/goauth/store/sql"
	"github.com/aloks98/homelab-proxy/backend/internal/api/routes"
	"github.com/aloks98/homelab-proxy/backend/internal/auth"
	"github.com/aloks98/homelab-proxy/backend/internal/config"
	"github.com/aloks98/homelab-proxy/backend/internal/models"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestEnvironment holds the test infrastructure
type TestEnvironment struct {
	PostgresContainer testcontainers.Container
	CaddyContainer    testcontainers.Container
	DB                *gorm.DB
	Router            http.Handler
	CaddyAdminURL     string
	AccessToken       string
	RefreshToken      string
}

// SetupTestEnvironment creates containers and initializes the test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	ctx := context.Background()
	env := &TestEnvironment{}

	// Start PostgreSQL container
	postgresReq := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: postgresReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	env.PostgresContainer = postgresContainer

	postgresHost, _ := postgresContainer.Host(ctx)
	postgresPort, _ := postgresContainer.MappedPort(ctx, "5432")

	// Start Caddy container with minimal config
	// IMPORTANT: auto_https off prevents Caddy from attempting ACME certificate requests
	// for test domains like test.example.com. This avoids:
	// 1. Unnecessary load on ACME servers (Let's Encrypt)
	// 2. Potential rate limiting issues
	// 3. Slow test execution due to failed ACME challenges
	caddyfileContent := `{
    admin 0.0.0.0:2019
    auto_https off
}

:80 {
    respond "Caddy is running" 200
}
`
	// Create temporary Caddyfile
	tmpDir := t.TempDir()
	caddyfilePath := tmpDir + "/Caddyfile"
	if err := os.WriteFile(caddyfilePath, []byte(caddyfileContent), 0644); err != nil {
		t.Fatalf("Failed to write Caddyfile: %v", err)
	}

	caddyReq := testcontainers.ContainerRequest{
		Image:        "caddy:2.10.2-alpine",
		ExposedPorts: []string{"80/tcp", "2019/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      caddyfilePath,
				ContainerFilePath: "/etc/caddy/Caddyfile",
				FileMode:          0644,
			},
		},
		Cmd: []string{"caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"},
		WaitingFor: wait.ForHTTP("/config/").
			WithPort("2019/tcp").
			WithStartupTimeout(30 * time.Second),
	}

	caddyContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: caddyReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start caddy container: %v", err)
	}
	env.CaddyContainer = caddyContainer

	caddyHost, _ := caddyContainer.Host(ctx)
	caddyAdminPort, _ := caddyContainer.MappedPort(ctx, "2019")
	env.CaddyAdminURL = fmt.Sprintf("http://%s:%s", caddyHost, caddyAdminPort.Port())

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		postgresHost, postgresPort.Port())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	env.DB = db

	// Run migrations
	if err := db.AutoMigrate(&models.User{}, &models.Proxy{}); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Get raw SQL DB for goauth
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get SQL DB: %v", err)
	}

	// Create goauth instance
	goauthInstance, err := createGoAuthInstance(sqlDB)
	if err != nil {
		t.Fatalf("Failed to create goauth instance: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Security: config.SecurityConfig{
			BcryptCost:  4, // Low cost for faster tests
			CORSOrigins: []string{"*"},
		},
		Caddy: config.CaddyConfig{
			AdminURL: env.CaddyAdminURL,
			Timeout:  10 * time.Second,
		},
	}

	// Create router
	testLogger := zap.NewNop()
	router := routes.SetupRoutes(cfg, db, testLogger, goauthInstance)
	env.Router = router

	return env
}

// createGoAuthInstance creates a goauth instance for testing
func createGoAuthInstance(db *sql.DB) (*goauth.Auth[*auth.CustomClaims], error) {
	// Create a simple in-memory RBAC config for testing matching the real format
	rbacConfig := `version: 1

permission_groups:
  - name: "Proxies"
    permissions:
      - key: "proxies:read"
        name: "View Proxies"
        description: "View proxy configurations"
      - key: "proxies:create"
        name: "Create Proxies"
        description: "Create new proxy configurations"
      - key: "proxies:update"
        name: "Update Proxies"
        description: "Modify existing proxy configurations"
      - key: "proxies:delete"
        name: "Delete Proxies"
        description: "Remove proxy configurations"

role_templates:
  - key: "admin"
    name: "Administrator"
    description: "Full access to all features"
    permissions:
      - "*"

  - key: "operator"
    name: "Operator"
    description: "Manage proxy configurations"
    permissions:
      - "proxies:*"
`
	tmpFile, err := os.CreateTemp("", "rbac-*.yaml")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.WriteString(rbacConfig); err != nil {
		return nil, err
	}
	tmpFile.Close()

	// Use goauth with SQL store
	store, err := sqlstore.New(&sqlstore.Config{
		Dialect:     sqlstore.PostgreSQL,
		DB:          db,
		TablePrefix: "goauth_",
	})
	if err != nil {
		return nil, err
	}

	return goauth.New[*auth.CustomClaims](
		goauth.WithSecret("test-secret-key-that-is-at-least-32-characters-long"),
		goauth.WithStore(store),
		goauth.WithAccessTokenTTL(15*time.Minute),
		goauth.WithRefreshTokenTTL(24*time.Hour),
		goauth.WithRBACFromFile(tmpFile.Name()),
		goauth.WithAutoMigrate(true),
		goauth.WithRoleSyncOnStartup(true),
	)
}

// Cleanup cleans up the test environment
func (env *TestEnvironment) Cleanup(t *testing.T) {
	ctx := context.Background()

	if env.PostgresContainer != nil {
		if err := env.PostgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate postgres container: %v", err)
		}
	}

	if env.CaddyContainer != nil {
		if err := env.CaddyContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate caddy container: %v", err)
		}
	}
}

// RegisterAndLogin creates a test user and logs in
func (env *TestEnvironment) RegisterAndLogin(t *testing.T) {
	// Register user
	registerBody := map[string]string{
		"name":     "Test User",
		"username": "testuser",
		"email":    "test@example.com",
		"password": "testpassword123",
	}
	body, _ := json.Marshal(registerBody)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: %d - %s", rec.Code, rec.Body.String())
	}

	// Login
	loginBody := map[string]string{
		"identifier": "testuser",
		"password":   "testpassword123",
	}
	body, _ = json.Marshal(loginBody)

	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Failed to login: %d - %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	env.AccessToken = loginResp.Data.AccessToken
	env.RefreshToken = loginResp.Data.RefreshToken
}

// MakeAuthenticatedRequest makes an authenticated HTTP request
func (env *TestEnvironment) MakeAuthenticatedRequest(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.AccessToken)

	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	return rec
}

// GetCaddyConfig fetches the current Caddy configuration
func (env *TestEnvironment) GetCaddyConfig(t *testing.T) map[string]interface{} {
	resp, err := http.Get(env.CaddyAdminURL + "/config/")
	if err != nil {
		t.Fatalf("Failed to get Caddy config: %v", err)
	}
	defer resp.Body.Close()

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode Caddy config: %v", err)
	}

	return config
}

// TestIntegration_ProxyLifecycle tests the full proxy lifecycle
func TestIntegration_ProxyLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	// Register and login
	env.RegisterAndLogin(t)

	// Verify Caddy is accessible
	resp, err := http.Get(env.CaddyAdminURL + "/config/")
	if err != nil {
		t.Fatalf("Caddy admin API not accessible: %v", err)
	}
	resp.Body.Close()
	t.Log("Caddy admin API is accessible")

	// Test 1: Create a proxy
	t.Run("CreateProxy", func(t *testing.T) {
		proxy := map[string]interface{}{
			"type":     "reverse_proxy",
			"name":     "Test Backend",
			"hostname": "test.example.com",
			"upstreams": []map[string]interface{}{
				{
					"host":   "httpbin.org",
					"port":   80,
					"scheme": "http",
				},
			},
		}

		rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if resp.Data.ID == 0 {
			t.Error("Expected proxy ID to be set")
		}
		if resp.Data.Hostname != "test.example.com" {
			t.Errorf("Expected hostname 'test.example.com', got '%s'", resp.Data.Hostname)
		}

		t.Logf("Created proxy with ID: %d", resp.Data.ID)

		// Verify Caddy config was updated
		caddyConfig := env.GetCaddyConfig(t)
		t.Logf("Caddy config after create: %+v", caddyConfig)

		// Check if route was added
		if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
			if httpApp, ok := apps["http"].(map[string]interface{}); ok {
				if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
					if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
						if routes, ok := srv0["routes"].([]interface{}); ok {
							found := false
							for _, r := range routes {
								if route, ok := r.(map[string]interface{}); ok {
									if id, ok := route["@id"].(string); ok && id == "proxy_1" {
										found = true
										break
									}
								}
							}
							if !found {
								t.Error("Expected route 'proxy_1' in Caddy config")
							} else {
								t.Log("Route 'proxy_1' found in Caddy config")
							}
						}
					}
				}
			}
		}
	})

	// Test 2: List proxies
	t.Run("ListProxies", func(t *testing.T) {
		rec := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/proxies", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Proxies []struct {
					ID       int    `json:"id"`
					Name     string `json:"name"`
					Hostname string `json:"hostname"`
				} `json:"proxies"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(resp.Data.Proxies) != 1 {
			t.Errorf("Expected 1 proxy, got %d", len(resp.Data.Proxies))
		}
	})

	// Test 3: Get proxy by ID
	t.Run("GetProxy", func(t *testing.T) {
		rec := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/proxies/1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Test 4: Update proxy
	t.Run("UpdateProxy", func(t *testing.T) {
		updateData := map[string]interface{}{
			"type":     "reverse_proxy",
			"name":     "Updated Backend",
			"hostname": "updated.example.com",
			"upstreams": []map[string]interface{}{
				{
					"host":   "httpbin.org",
					"port":   443,
					"scheme": "https",
				},
			},
		}

		rec := env.MakeAuthenticatedRequest(t, http.MethodPut, "/api/proxies/1", updateData)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify the update
		rec = env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/proxies/1", nil)
		var resp struct {
			Data struct {
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
			} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Data.Name != "Updated Backend" {
			t.Errorf("Expected name 'Updated Backend', got '%s'", resp.Data.Name)
		}
		if resp.Data.Hostname != "updated.example.com" {
			t.Errorf("Expected hostname 'updated.example.com', got '%s'", resp.Data.Hostname)
		}
	})

	// Test 5: Disable proxy
	t.Run("DisableProxy", func(t *testing.T) {
		rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies/1/disable", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify route was removed from Caddy
		caddyConfig := env.GetCaddyConfig(t)
		if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
			if httpApp, ok := apps["http"].(map[string]interface{}); ok {
				if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
					if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
						if routes, ok := srv0["routes"].([]interface{}); ok {
							for _, r := range routes {
								if route, ok := r.(map[string]interface{}); ok {
									if id, ok := route["@id"].(string); ok && id == "proxy_1" {
										t.Error("Route 'proxy_1' should have been removed from Caddy config")
									}
								}
							}
						}
					}
				}
			}
		}
		t.Log("Route removed from Caddy after disable")
	})

	// Test 6: Enable proxy
	t.Run("EnableProxy", func(t *testing.T) {
		rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies/1/enable", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify route was added back to Caddy
		caddyConfig := env.GetCaddyConfig(t)
		if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
			if httpApp, ok := apps["http"].(map[string]interface{}); ok {
				if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
					if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
						if routes, ok := srv0["routes"].([]interface{}); ok {
							found := false
							for _, r := range routes {
								if route, ok := r.(map[string]interface{}); ok {
									if id, ok := route["@id"].(string); ok && id == "proxy_1" {
										found = true
										break
									}
								}
							}
							if !found {
								t.Error("Expected route 'proxy_1' to be restored in Caddy config")
							} else {
								t.Log("Route 'proxy_1' restored in Caddy config after enable")
							}
						}
					}
				}
			}
		}
	})

	// Test 7: Delete proxy
	t.Run("DeleteProxy", func(t *testing.T) {
		rec := env.MakeAuthenticatedRequest(t, http.MethodDelete, "/api/proxies/1", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify proxy is deleted
		rec = env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/proxies/1", nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 after delete, got %d", rec.Code)
		}

		// Verify route was removed from Caddy
		caddyConfig := env.GetCaddyConfig(t)
		if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
			if httpApp, ok := apps["http"].(map[string]interface{}); ok {
				if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
					if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
						if routes, ok := srv0["routes"].([]interface{}); ok {
							for _, r := range routes {
								if route, ok := r.(map[string]interface{}); ok {
									if id, ok := route["@id"].(string); ok && id == "proxy_1" {
										t.Error("Route 'proxy_1' should have been removed after delete")
									}
								}
							}
						}
					}
				}
			}
		}
		t.Log("Route removed from Caddy after delete")
	})
}

// TestIntegration_RedirectProxy tests redirect proxy type
func TestIntegration_RedirectProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create redirect proxy
	proxy := map[string]interface{}{
		"type":     "redirect",
		"name":     "Redirect Test",
		"hostname": "redirect.example.com",
		"redirect": map[string]interface{}{
			"target":         "https://target.example.com",
			"status_code":    301,
			"preserve_path":  true,
			"preserve_query": true,
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has the redirect handler
	caddyConfig := env.GetCaddyConfig(t)

	foundRedirect := false
	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								// Only check the route with our proxy ID
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "static_response" {
													foundRedirect = true
													t.Log("Found static_response handler for redirect")
													if statusCode, ok := handler["status_code"].(float64); ok {
														if int(statusCode) != 301 {
															t.Errorf("Expected status code 301, got %v", statusCode)
														} else {
															t.Log("Correct status code 301 confirmed")
														}
													}
													// Check Location header
													if headers, ok := handler["headers"].(map[string]interface{}); ok {
														if location, ok := headers["Location"].([]interface{}); ok && len(location) > 0 {
															t.Logf("Location header: %v", location[0])
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if !foundRedirect {
		t.Error("Expected to find redirect handler in Caddy config")
	}
}

// TestIntegration_StaticProxy tests static file server proxy type
func TestIntegration_StaticProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create static proxy
	proxy := map[string]interface{}{
		"type":     "static",
		"name":     "Static Test",
		"hostname": "static.example.com",
		"static": map[string]interface{}{
			"root_path":  "/var/www/html",
			"index_file": "index.html",
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has the file_server handler
	caddyConfig := env.GetCaddyConfig(t)

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if handles, ok := route["handle"].([]interface{}); ok {
									for _, h := range handles {
										if handler, ok := h.(map[string]interface{}); ok {
											if handler["handler"] == "file_server" {
												t.Log("Found file_server handler for static proxy")
												if root, ok := handler["root"].(string); ok {
													if root != "/var/www/html" {
														t.Errorf("Expected root '/var/www/html', got '%s'", root)
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// TestIntegration_HostnameConflict tests that duplicate hostnames are rejected
func TestIntegration_HostnameConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create first proxy
	proxy := map[string]interface{}{
		"type":     "reverse_proxy",
		"name":     "First Proxy",
		"hostname": "conflict.example.com",
		"upstreams": []map[string]interface{}{
			{"host": "backend1", "port": 8080, "scheme": "http"},
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Try to create second proxy with same hostname
	proxy["name"] = "Second Proxy"
	rec = env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict for duplicate hostname, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestIntegration_ReverseProxy_LoadBalancing tests reverse proxy with multiple upstreams and load balancing
func TestIntegration_ReverseProxy_LoadBalancing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create proxy with multiple upstreams and load balancing
	proxy := map[string]interface{}{
		"type":     "reverse_proxy",
		"name":     "Load Balanced Backend",
		"hostname": "lb.example.com",
		"upstreams": []map[string]interface{}{
			{"host": "backend1.internal", "port": 8080, "scheme": "http"},
			{"host": "backend2.internal", "port": 8080, "scheme": "http"},
			{"host": "backend3.internal", "port": 8080, "scheme": "http"},
		},
		"load_balancing": map[string]interface{}{
			"strategy": "round_robin",
			"health_checks": map[string]interface{}{
				"enabled":             true,
				"path":                "/health",
				"interval":            "30s",
				"timeout":             "10s",
				"unhealthy_threshold": 3,
				"healthy_threshold":   2,
			},
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config
	caddyConfig := env.GetCaddyConfig(t)

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "reverse_proxy" {
													// Check upstreams count
													if upstreams, ok := handler["upstreams"].([]interface{}); ok {
														if len(upstreams) != 3 {
															t.Errorf("Expected 3 upstreams, got %d", len(upstreams))
														} else {
															t.Log("Found 3 upstreams for load balanced proxy")
														}
													}

													// Check load balancing config
													if lb, ok := handler["load_balancing"].(map[string]interface{}); ok {
														if policy, ok := lb["selection_policy"].(map[string]interface{}); ok {
															if policy["policy"] == "round_robin" {
																t.Log("Correct load balancing policy: round_robin")
															} else {
																t.Errorf("Expected round_robin policy, got %v", policy["policy"])
															}
														}
													}

													// Check health checks
													if hc, ok := handler["health_checks"].(map[string]interface{}); ok {
														if active, ok := hc["active"].(map[string]interface{}); ok {
															if path, ok := active["path"].(string); ok && path == "/health" {
																t.Log("Correct health check path: /health")
															}
															if interval, ok := active["interval"].(string); ok && interval == "30s" {
																t.Log("Correct health check interval: 30s")
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// TestIntegration_ReverseProxy_HTTPSUpstream tests reverse proxy with HTTPS upstream
func TestIntegration_ReverseProxy_HTTPSUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create proxy with HTTPS upstream and TLS skip verify
	proxy := map[string]interface{}{
		"type":                    "reverse_proxy",
		"name":                    "HTTPS Backend",
		"hostname":                "secure.example.com",
		"tls_insecure_skip_verify": true,
		"upstreams": []map[string]interface{}{
			{"host": "secure-backend.internal", "port": 443, "scheme": "https"},
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has transport with TLS
	caddyConfig := env.GetCaddyConfig(t)

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "reverse_proxy" {
													// Check transport config
													if transport, ok := handler["transport"].(map[string]interface{}); ok {
														t.Log("Found transport config for HTTPS upstream")
														if tls, ok := transport["tls"].(map[string]interface{}); ok {
															if skipVerify, ok := tls["insecure_skip_verify"].(bool); ok && skipVerify {
																t.Log("TLS insecure_skip_verify is enabled")
															} else {
																t.Error("Expected insecure_skip_verify to be true")
															}
														}
													} else {
														t.Error("Expected transport config for HTTPS upstream")
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// TestIntegration_ReverseProxy_CustomHeaders tests reverse proxy with custom headers
func TestIntegration_ReverseProxy_CustomHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create proxy with custom headers
	proxy := map[string]interface{}{
		"type":     "reverse_proxy",
		"name":     "Custom Headers Backend",
		"hostname": "headers.example.com",
		"upstreams": []map[string]interface{}{
			{"host": "backend.internal", "port": 8080, "scheme": "http"},
		},
		"custom_headers": map[string]interface{}{
			"X-Custom-Header":  "custom-value",
			"X-Another-Header": "another-value",
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has headers
	caddyConfig := env.GetCaddyConfig(t)

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "reverse_proxy" {
													if headers, ok := handler["headers"].(map[string]interface{}); ok {
														if request, ok := headers["request"].(map[string]interface{}); ok {
															if set, ok := request["set"].(map[string]interface{}); ok {
																if _, hasCustom := set["X-Custom-Header"]; hasCustom {
																	t.Log("Custom header X-Custom-Header found")
																}
																if _, hasAnother := set["X-Another-Header"]; hasAnother {
																	t.Log("Custom header X-Another-Header found")
																}
																// Check standard headers are also present
																if _, hasRealIP := set["X-Real-IP"]; hasRealIP {
																	t.Log("Standard X-Real-IP header found")
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// TestIntegration_Redirect_StatusCodes tests redirect proxy with different status codes
func TestIntegration_Redirect_StatusCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name       string
		statusCode int
		hostname   string
	}{
		{"Permanent Redirect 301", 301, "redirect301.example.com"},
		{"Temporary Redirect 302", 302, "redirect302.example.com"},
		{"See Other 303", 303, "redirect303.example.com"},
		{"Temporary Redirect 307", 307, "redirect307.example.com"},
		{"Permanent Redirect 308", 308, "redirect308.example.com"},
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxy := map[string]interface{}{
				"type":     "redirect",
				"name":     tc.name,
				"hostname": tc.hostname,
				"redirect": map[string]interface{}{
					"target":      "https://target.example.com",
					"status_code": tc.statusCode,
				},
			}

			rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

			if rec.Code != http.StatusCreated {
				t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
			}

			// Verify Caddy config has correct status code
			caddyConfig := env.GetCaddyConfig(t)
			expectedRouteID := fmt.Sprintf("proxy_%d", i+1)

			if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
				if httpApp, ok := apps["http"].(map[string]interface{}); ok {
					if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
						if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
							if routes, ok := srv0["routes"].([]interface{}); ok {
								for _, r := range routes {
									if route, ok := r.(map[string]interface{}); ok {
										if routeID, ok := route["@id"].(string); ok && routeID == expectedRouteID {
											if handles, ok := route["handle"].([]interface{}); ok {
												for _, h := range handles {
													if handler, ok := h.(map[string]interface{}); ok {
														if handler["handler"] == "static_response" {
															if statusCode, ok := handler["status_code"].(float64); ok {
																if int(statusCode) != tc.statusCode {
																	t.Errorf("Expected status code %d, got %d", tc.statusCode, int(statusCode))
																} else {
																	t.Logf("Correct status code %d for %s", tc.statusCode, tc.name)
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestIntegration_Redirect_PreservePath tests redirect with and without path preservation
func TestIntegration_Redirect_PreservePath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	testCases := []struct {
		name          string
		hostname      string
		preservePath  bool
		preserveQuery bool
		expectedURL   string
	}{
		{
			name:          "No preservation",
			hostname:      "nopreserve.example.com",
			preservePath:  false,
			preserveQuery: false,
			expectedURL:   "https://target.example.com",
		},
		{
			name:          "Preserve path only",
			hostname:      "preservepath.example.com",
			preservePath:  true,
			preserveQuery: false,
			expectedURL:   "https://target.example.com{http.request.uri.path}",
		},
		{
			name:          "Preserve both",
			hostname:      "preserveboth.example.com",
			preservePath:  true,
			preserveQuery: true,
			expectedURL:   "https://target.example.com{http.request.uri.path}?{http.request.uri.query}",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxy := map[string]interface{}{
				"type":     "redirect",
				"name":     tc.name,
				"hostname": tc.hostname,
				"redirect": map[string]interface{}{
					"target":         "https://target.example.com",
					"status_code":    302,
					"preserve_path":  tc.preservePath,
					"preserve_query": tc.preserveQuery,
				},
			}

			rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

			if rec.Code != http.StatusCreated {
				t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
			}

			// Verify Caddy config has correct Location header
			caddyConfig := env.GetCaddyConfig(t)
			expectedRouteID := fmt.Sprintf("proxy_%d", i+1)

			if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
				if httpApp, ok := apps["http"].(map[string]interface{}); ok {
					if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
						if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
							if routes, ok := srv0["routes"].([]interface{}); ok {
								for _, r := range routes {
									if route, ok := r.(map[string]interface{}); ok {
										if routeID, ok := route["@id"].(string); ok && routeID == expectedRouteID {
											if handles, ok := route["handle"].([]interface{}); ok {
												for _, h := range handles {
													if handler, ok := h.(map[string]interface{}); ok {
														if handler["handler"] == "static_response" {
															if headers, ok := handler["headers"].(map[string]interface{}); ok {
																if location, ok := headers["Location"].([]interface{}); ok && len(location) > 0 {
																	if location[0] != tc.expectedURL {
																		t.Errorf("Expected Location '%s', got '%s'", tc.expectedURL, location[0])
																	} else {
																		t.Logf("Correct Location header for %s", tc.name)
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestIntegration_Static_SPA tests static file server with SPA try_files
func TestIntegration_Static_SPA(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create SPA static proxy with try_files
	proxy := map[string]interface{}{
		"type":     "static",
		"name":     "SPA Application",
		"hostname": "spa.example.com",
		"static": map[string]interface{}{
			"root_path":  "/var/www/spa",
			"index_file": "index.html",
			"try_files":  []string{"/index.html"},
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has subroute with rewrite
	caddyConfig := env.GetCaddyConfig(t)

	foundSubroute := false
	foundFileServer := false

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "subroute" {
													foundSubroute = true
													t.Log("Found subroute handler for SPA rewrite")
												}
												if handler["handler"] == "file_server" {
													foundFileServer = true
													if root, ok := handler["root"].(string); ok {
														if root != "/var/www/spa" {
															t.Errorf("Expected root '/var/www/spa', got '%s'", root)
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if !foundSubroute {
		t.Error("Expected subroute handler for SPA try_files")
	}
	if !foundFileServer {
		t.Error("Expected file_server handler")
	}
}

// TestIntegration_Static_Templates tests static file server with template rendering
func TestIntegration_Static_Templates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create static proxy with template rendering
	proxy := map[string]interface{}{
		"type":     "static",
		"name":     "Template Site",
		"hostname": "templates.example.com",
		"static": map[string]interface{}{
			"root_path":          "/var/www/templates",
			"index_file":         "index.html",
			"template_rendering": true,
		},
	}

	rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify Caddy config has templates handler
	caddyConfig := env.GetCaddyConfig(t)

	foundTemplates := false
	foundFileServer := false

	if apps, ok := caddyConfig["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if srv0, ok := servers["srv0"].(map[string]interface{}); ok {
					if routes, ok := srv0["routes"].([]interface{}); ok {
						for _, r := range routes {
							if route, ok := r.(map[string]interface{}); ok {
								if routeID, ok := route["@id"].(string); ok && routeID == "proxy_1" {
									if handles, ok := route["handle"].([]interface{}); ok {
										for _, h := range handles {
											if handler, ok := h.(map[string]interface{}); ok {
												if handler["handler"] == "templates" {
													foundTemplates = true
													t.Log("Found templates handler")
													if fileRoot, ok := handler["file_root"].(string); ok {
														if fileRoot != "/var/www/templates" {
															t.Errorf("Expected file_root '/var/www/templates', got '%s'", fileRoot)
														}
													}
												}
												if handler["handler"] == "file_server" {
													foundFileServer = true
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if !foundTemplates {
		t.Error("Expected templates handler for template_rendering")
	}
	if !foundFileServer {
		t.Error("Expected file_server handler")
	}
}

// TestIntegration_ProxyValidation tests proxy validation errors
func TestIntegration_ProxyValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	testCases := []struct {
		name           string
		proxy          map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "Missing hostname",
			proxy: map[string]interface{}{
				"type": "reverse_proxy",
				"name": "Test",
				"upstreams": []map[string]interface{}{
					{"host": "backend", "port": 8080, "scheme": "http"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject proxy without hostname",
		},
		{
			name: "Missing name",
			proxy: map[string]interface{}{
				"type":     "reverse_proxy",
				"hostname": "test.example.com",
				"upstreams": []map[string]interface{}{
					{"host": "backend", "port": 8080, "scheme": "http"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject proxy without name",
		},
		{
			name: "Invalid proxy type",
			proxy: map[string]interface{}{
				"type":     "invalid_type",
				"name":     "Test",
				"hostname": "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject invalid proxy type",
		},
		{
			name: "Reverse proxy without upstreams",
			proxy: map[string]interface{}{
				"type":     "reverse_proxy",
				"name":     "Test",
				"hostname": "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject reverse_proxy without upstreams",
		},
		{
			name: "Redirect without config",
			proxy: map[string]interface{}{
				"type":     "redirect",
				"name":     "Test",
				"hostname": "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject redirect without redirect config",
		},
		{
			name: "Static without config",
			proxy: map[string]interface{}{
				"type":     "static",
				"name":     "Test",
				"hostname": "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject static without static config",
		},
		{
			name: "Hostname with scheme",
			proxy: map[string]interface{}{
				"type":     "reverse_proxy",
				"name":     "Test",
				"hostname": "https://test.example.com",
				"upstreams": []map[string]interface{}{
					{"host": "backend", "port": 8080, "scheme": "http"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject hostname with scheme",
		},
		{
			name: "Hostname with port",
			proxy: map[string]interface{}{
				"type":     "reverse_proxy",
				"name":     "Test",
				"hostname": "test.example.com:8080",
				"upstreams": []map[string]interface{}{
					{"host": "backend", "port": 8080, "scheme": "http"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject hostname with port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", tc.proxy)

			if rec.Code != tc.expectedStatus {
				t.Errorf("%s: Expected %d, got %d: %s", tc.description, tc.expectedStatus, rec.Code, rec.Body.String())
			} else {
				t.Logf("Correctly rejected: %s", tc.description)
			}
		})
	}
}
