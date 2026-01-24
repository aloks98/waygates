package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ContainerTestEnv holds the container-based test infrastructure
type ContainerTestEnv struct {
	Network           *testcontainers.DockerNetwork
	PostgresContainer testcontainers.Container
	WaygatesContainer testcontainers.Container
	BaseURL           string
	AccessToken       string
	RefreshToken      string
	ctx               context.Context
}

// SetupContainerEnvironment creates containers and initializes the test environment
func SetupContainerEnvironment(t *testing.T) *ContainerTestEnv {
	if testing.Short() {
		t.Skip("Skipping container-based integration test in short mode")
	}

	ctx := context.Background()
	env := &ContainerTestEnv{ctx: ctx}

	// Create a Docker network for containers to communicate
	testNetwork, err := network.New(ctx,
		network.WithDriver("bridge"),
	)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}
	env.Network = testNetwork

	// Start PostgreSQL container
	postgresReq := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Networks:     []string{testNetwork.Name},
		NetworkAliases: map[string][]string{
			testNetwork.Name: {"postgres"},
		},
		Env: map[string]string{
			"POSTGRES_USER":     "waygates",
			"POSTGRES_PASSWORD": "waygates",
			"POSTGRES_DB":       "waygates",
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

	// Get the path to test JSON config
	// This config disables TLS automation for testing
	testJSONConfig := findTestJSONConfig(t)

	// Use pre-built image (run `docker build -t waygates-test:latest .` before running tests)
	// This avoids issues with platform variables in Docker BuildKit
	waygatesReq := testcontainers.ContainerRequest{
		Image:        "waygates-test:latest",
		ExposedPorts: []string{"8080/tcp", "80/tcp", "443/tcp"},
		Networks:     []string{testNetwork.Name},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      testJSONConfig,
				ContainerFilePath: "/etc/caddy/caddy.json",
				FileMode:          0o644,
			},
		},
		Env: map[string]string{
			// Database
			"DB_HOST":     "postgres",
			"DB_PORT":     "5432",
			"DB_USER":     "waygates",
			"DB_PASSWORD": "waygates",
			"DB_NAME":     "waygates",
			// JWT
			"JWT_SECRET":         "test-secret-key-that-is-at-least-32-characters-long",
			"JWT_ACCESS_EXPIRY":  "15m",
			"JWT_REFRESH_EXPIRY": "24h",
			// Security
			"BCRYPT_COST":              "4", // Low cost for faster tests
			"CORS_ORIGINS":             "*",
			"RBAC_PATH":                "/app/backend/rbac.yaml",
			"CADDY_EMAIL":              "test@example.com",
			"CADDY_DISABLE_AUTO_HTTPS": "true", // Disable TLS for testing
			// Cloudflare (not used when auto_https is disabled, but needed for validation)
			"CLOUDFLARE_EMAIL":     "test@example.com",
			"CLOUDFLARE_API_TOKEN": "dummy-token-for-testing",
			// Logging
			"LOG_LEVEL":  "debug",
			"LOG_FORMAT": "console",
			// UI
			"UI_ENABLED": "false",
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/api/health").WithPort("8080/tcp").WithStartupTimeout(120*time.Second),
			// Wait for Caddy to log "Caddy is ready!" from entrypoint
			wait.ForLog("Caddy is ready!").WithStartupTimeout(60*time.Second),
		),
	}

	waygatesContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: waygatesReq,
		Started:          true,
	})
	if err != nil {
		// Clean up postgres if waygates fails to start
		_ = postgresContainer.Terminate(ctx)
		_ = testNetwork.Remove(ctx)
		t.Fatalf("Failed to start waygates container: %v", err)
	}
	env.WaygatesContainer = waygatesContainer

	// Get the mapped port for the API
	mappedPort, err := waygatesContainer.MappedPort(ctx, nat.Port("8080/tcp"))
	if err != nil {
		env.Cleanup(t)
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	host, err := waygatesContainer.Host(ctx)
	if err != nil {
		env.Cleanup(t)
		t.Fatalf("Failed to get host: %v", err)
	}

	env.BaseURL = fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
	t.Logf("Waygates API running at: %s", env.BaseURL)

	return env
}

// Cleanup cleans up the test environment
func (env *ContainerTestEnv) Cleanup(t *testing.T) {
	ctx := context.Background()

	if env.WaygatesContainer != nil {
		// Get logs for debugging if needed
		logs, err := env.WaygatesContainer.Logs(ctx)
		if err == nil {
			logBytes, _ := io.ReadAll(logs)
			if len(logBytes) > 0 && t.Failed() {
				t.Logf("Waygates container logs:\n%s", string(logBytes))
			}
			_ = logs.Close()
		}

		if err := env.WaygatesContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate waygates container: %v", err)
		}
	}

	if env.PostgresContainer != nil {
		if err := env.PostgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate postgres container: %v", err)
		}
	}

	if env.Network != nil {
		if err := env.Network.Remove(ctx); err != nil {
			t.Logf("Failed to remove network: %v", err)
		}
	}
}

// RegisterAndLogin creates a test user and logs in
func (env *ContainerTestEnv) RegisterAndLogin(t *testing.T) {
	// Register user
	registerBody := map[string]string{
		"name":     "Test User",
		"username": "testuser",
		"email":    "test@example.com",
		"password": "testpassword123",
	}
	body, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatalf("Failed to marshal register body: %v", err)
	}

	resp, err := http.Post(env.BaseURL+"/api/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to register user: %d - %s", resp.StatusCode, string(respBody))
	}

	// Login
	loginBody := map[string]string{
		"identifier": "testuser",
		"password":   "testpassword123",
	}
	body, err = json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("Failed to marshal login body: %v", err)
	}

	resp, err = http.Post(env.BaseURL+"/api/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to login: %d - %s", resp.StatusCode, string(respBody))
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	env.AccessToken = loginResp.Data.AccessToken
	env.RefreshToken = loginResp.Data.RefreshToken
}

// MakeAuthenticatedRequest makes an authenticated HTTP request
func (env *ContainerTestEnv) MakeAuthenticatedRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, env.BaseURL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.AccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}

// ExecInContainer executes a command inside the waygates container
func (env *ContainerTestEnv) ExecInContainer(_ *testing.T, cmd []string) (string, error) {
	exitCode, reader, err := env.WaygatesContainer.Exec(env.ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("exec failed: %w", err)
	}

	output, _ := io.ReadAll(reader)
	if exitCode != 0 {
		return string(output), fmt.Errorf("command exited with code %d: %s", exitCode, string(output))
	}

	return string(output), nil
}

// ProxyExistsInJSONConfig checks if a proxy hostname exists in the JSON config
func (env *ContainerTestEnv) ProxyExistsInJSONConfig(t *testing.T, hostname string) bool {
	config, err := env.ReadJSONConfig(t)
	if err != nil {
		return false
	}
	// Check if hostname appears in the config
	return strings.Contains(config, hostname)
}

// ReadJSONConfig reads the Caddy JSON config from the container
func (env *ContainerTestEnv) ReadJSONConfig(t *testing.T) (string, error) {
	return env.ExecInContainer(t, []string{"cat", "/etc/caddy/caddy.json"})
}

// WaitForProxyInConfig waits for a proxy hostname to appear in the JSON config
func (env *ContainerTestEnv) WaitForProxyInConfig(t *testing.T, hostname string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if env.ProxyExistsInJSONConfig(t, hostname) {
			return true
		}
		time.Sleep(pollInterval)
	}
	return false
}

// WaitForProxyRemovedFromConfig waits for a proxy hostname to be removed from the JSON config
func (env *ContainerTestEnv) WaitForProxyRemovedFromConfig(t *testing.T, hostname string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if !env.ProxyExistsInJSONConfig(t, hostname) {
			return true
		}
		time.Sleep(pollInterval)
	}
	return false
}

// ReadJSONResponse reads response body and unmarshals to v
func (env *ContainerTestEnv) ReadJSONResponse(t *testing.T, resp *http.Response, v interface{}) []byte {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			t.Fatalf("Failed to unmarshal response: %v\nBody: %s", err, string(body))
		}
	}
	return body
}

// WaitForSyncComplete waits for sync to complete by polling sync status
func (env *ContainerTestEnv) WaitForSyncComplete(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/sync/status", nil)
		var status struct {
			Data struct {
				IsSyncing bool `json:"is_syncing"`
			} `json:"data"`
		}
		env.ReadJSONResponse(t, resp, &status)
		_ = resp.Body.Close()

		if !status.Data.IsSyncing {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Logf("Warning: sync did not complete within %v timeout", timeout)
}

// findTestJSONConfig locates the test JSON config
func findTestJSONConfig(t *testing.T) string {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Try relative path from test directory
	testdataPath := filepath.Join(cwd, "testdata", "caddy.json.test")
	if _, err := os.Stat(testdataPath); err == nil {
		return testdataPath
	}

	// Try from project root (when running from project root)
	projectPath := filepath.Join(cwd, "backend", "tests", "integration", "testdata", "caddy.json.test")
	if _, err := os.Stat(projectPath); err == nil {
		return projectPath
	}

	t.Fatalf("Could not find test JSON config. Checked:\n- %s\n- %s", testdataPath, projectPath)
	return ""
}

// TestIntegration_ProxyLifecycle tests the full proxy lifecycle
// NOTE: Subtests in this function share state (createdProxyID) and MUST run sequentially.
// Do NOT add t.Parallel() to any subtest as they depend on the order of execution.
func TestIntegration_ProxyLifecycle(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// proxyState holds state shared between sequential subtests
	type proxyState struct {
		id              int
		hostname        string
		updatedHostname string
	}
	state := &proxyState{
		hostname:        "test.example.com",
		updatedHostname: "updated.example.com",
	}

	// Test 1: Create a proxy
	t.Run("CreateProxy", func(t *testing.T) {
		proxy := map[string]interface{}{
			"type":     "reverse_proxy",
			"name":     "Test Backend",
			"hostname": state.hostname,
			"upstreams": []map[string]interface{}{
				{
					"host":   "httpbin.org",
					"port":   80,
					"scheme": "http",
				},
			},
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Data.ID == 0 {
			t.Error("Expected proxy ID to be set")
		}
		state.id = response.Data.ID
		if response.Data.Hostname != state.hostname {
			t.Errorf("Expected hostname '%s', got '%s'", state.hostname, response.Data.Hostname)
		}

		t.Logf("Created proxy with ID: %d", response.Data.ID)

		// Verify proxy appears in JSON config (with polling)
		if !env.WaitForProxyInConfig(t, state.hostname, 5*time.Second) {
			t.Errorf("Expected hostname %s to appear in JSON config", state.hostname)
		} else {
			content, err := env.ReadJSONConfig(t)
			if err != nil {
				t.Errorf("Failed to read JSON config: %v", err)
			} else {
				if !strings.Contains(content, state.hostname) {
					t.Error("JSON config should contain hostname")
				}
				if !strings.Contains(content, "reverse_proxy") {
					t.Error("JSON config should contain reverse_proxy handler")
				}
				t.Logf("JSON config contains proxy for: %s", state.hostname)
			}
		}
	})

	// Test 2: List proxies
	t.Run("ListProxies", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/proxies", nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				Items []struct {
					ID       int    `json:"id"`
					Name     string `json:"name"`
					Hostname string `json:"hostname"`
				} `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response.Data.Items) != 1 {
			t.Errorf("Expected 1 proxy, got %d", len(response.Data.Items))
		}
	})

	// Test 3: Get proxy by ID
	t.Run("GetProxy", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}
	})

	// Test 4: Update proxy
	t.Run("UpdateProxy", func(t *testing.T) {
		updateData := map[string]interface{}{
			"type":     "reverse_proxy",
			"name":     "Updated Backend",
			"hostname": state.updatedHostname,
			"upstreams": []map[string]interface{}{
				{
					"host":   "httpbin.org",
					"port":   443,
					"scheme": "https",
				},
			},
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, fmt.Sprintf("/api/proxies/%d", state.id), updateData)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Verify the update
		resp = env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ = io.ReadAll(resp.Body)
		var response struct {
			Data struct {
				Name     string `json:"name"`
				Hostname string `json:"hostname"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Data.Name != "Updated Backend" {
			t.Errorf("Expected name 'Updated Backend', got '%s'", response.Data.Name)
		}
		if response.Data.Hostname != state.updatedHostname {
			t.Errorf("Expected hostname '%s', got '%s'", state.updatedHostname, response.Data.Hostname)
		}

		// Verify proxy appears in JSON config with updated hostname (with polling)
		if !env.WaitForProxyInConfig(t, state.updatedHostname, 5*time.Second) {
			t.Errorf("Expected updated hostname %s to appear in JSON config", state.updatedHostname)
		} else {
			content, err := env.ReadJSONConfig(t)
			if err != nil {
				t.Errorf("Failed to read JSON config: %v", err)
			} else {
				if !strings.Contains(content, state.updatedHostname) {
					t.Error("JSON config should contain updated hostname")
				}
				t.Logf("JSON config updated for: %s", state.updatedHostname)
			}
		}
	})

	// Test 5: Disable proxy
	t.Run("DisableProxy", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/proxies/%d/disable", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Verify proxy is removed from JSON config (disabled proxies are not included)
		if !env.WaitForProxyRemovedFromConfig(t, state.updatedHostname, 5*time.Second) {
			t.Error("Proxy hostname should not be in JSON config after disable")
		}
		t.Log("Proxy disabled - removed from JSON config")
	})

	// Test 6: Enable proxy
	t.Run("EnableProxy", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, fmt.Sprintf("/api/proxies/%d/enable", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Verify proxy is back in JSON config after enable
		if !env.WaitForProxyInConfig(t, state.updatedHostname, 5*time.Second) {
			t.Error("Proxy hostname should be in JSON config after enable")
		}
		t.Log("Proxy enabled - added back to JSON config")
	})

	// Test 7: Delete proxy
	t.Run("DeleteProxy", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Verify proxy is deleted
		resp = env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 after delete, got %d", resp.StatusCode)
		}

		// Verify proxy is removed from JSON config (with polling)
		if !env.WaitForProxyRemovedFromConfig(t, state.updatedHostname, 5*time.Second) {
			t.Error("Proxy hostname should not be in JSON config after delete")
		}
		t.Log("Proxy deleted - removed from JSON config")
	})
}

// TestIntegration_RedirectProxy tests redirect proxy type
func TestIntegration_RedirectProxy(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	hostname := "redirect.example.com"
	redirectTarget := "https://target.example.com"

	// Create redirect proxy
	proxy := map[string]interface{}{
		"type":     "redirect",
		"name":     "Redirect Test",
		"hostname": hostname,
		"redirect": map[string]interface{}{
			"target":         redirectTarget,
			"status_code":    301,
			"preserve_path":  true,
			"preserve_query": true,
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify proxy appears in JSON config (with polling)
	if !env.WaitForProxyInConfig(t, hostname, 5*time.Second) {
		t.Fatalf("Expected hostname %s to appear in JSON config", hostname)
	}

	content, err := env.ReadJSONConfig(t)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	if !strings.Contains(content, hostname) {
		t.Error("JSON config should contain hostname")
	}
	// In JSON config, redirects use static_response handler with Location header
	if !strings.Contains(content, "static_response") {
		t.Error("JSON config should contain static_response handler for redirect")
	}
	if !strings.Contains(content, "target.example.com") {
		t.Error("JSON config should contain redirect target")
	}
	// Check for 301 status code in JSON config
	if !strings.Contains(content, "301") {
		t.Error("JSON config should contain 301 status code")
	}
	t.Logf("Redirect proxy JSON config created for: %s", hostname)
}

// TestIntegration_StaticProxy tests static file server proxy type
func TestIntegration_StaticProxy(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	hostname := "static.example.com"

	// Create static proxy
	proxy := map[string]interface{}{
		"type":     "static",
		"name":     "Static Test",
		"hostname": hostname,
		"static": map[string]interface{}{
			"root_path":  "/var/www/html",
			"index_file": "index.html",
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify proxy appears in JSON config (with polling)
	if !env.WaitForProxyInConfig(t, hostname, 5*time.Second) {
		t.Fatalf("Expected hostname %s to appear in JSON config", hostname)
	}

	content, err := env.ReadJSONConfig(t)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	if !strings.Contains(content, hostname) {
		t.Error("JSON config should contain hostname")
	}
	if !strings.Contains(content, "file_server") {
		t.Error("JSON config should contain file_server handler")
	}
	if !strings.Contains(content, "/var/www/html") {
		t.Error("JSON config should contain root path")
	}
	t.Logf("Static proxy JSON config created for: %s", hostname)
}

// TestIntegration_HostnameConflict tests that duplicate hostnames are rejected
func TestIntegration_HostnameConflict(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	hostname := "conflict.example.com"

	// Create first proxy
	proxy := map[string]interface{}{
		"type":     "reverse_proxy",
		"name":     "First Proxy",
		"hostname": hostname,
		"upstreams": []map[string]interface{}{
			{"host": "backend1", "port": 8080, "scheme": "http"},
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	// Try to create second proxy with same hostname
	proxy["name"] = "Second Proxy"
	resp = env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 409 Conflict for duplicate hostname, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestIntegration_ProxyValidation tests proxy validation errors
func TestIntegration_ProxyValidation(t *testing.T) {
	env := SetupContainerEnvironment(t)
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", tc.proxy)
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.expectedStatus {
				respBody, _ := io.ReadAll(resp.Body)
				t.Errorf("%s: Expected %d, got %d: %s", tc.description, tc.expectedStatus, resp.StatusCode, string(respBody))
			} else {
				t.Logf("Correctly rejected: %s", tc.description)
			}
		})
	}
}

// TestIntegration_SettingsAPI tests settings API
func TestIntegration_SettingsAPI(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Test 1: Get all settings
	t.Run("GetAllSettings", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/settings", nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool              `json:"success"`
			Data    map[string]string `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		// Verify default settings exist
		if response.Data["not_found.mode"] != "default" {
			t.Errorf("Expected not_found.mode to be 'default', got '%s'", response.Data["not_found.mode"])
		}
		t.Logf("Settings retrieved: %+v", response.Data)
	})

	// Test 2: Get 404 settings
	t.Run("Get404Settings", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/settings/404", nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				Mode        string `json:"mode"`
				RedirectURL string `json:"redirect_url"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Data.Mode != "default" {
			t.Errorf("Expected mode 'default', got '%s'", response.Data.Mode)
		}
		t.Logf("404 settings: mode=%s, redirect_url=%s", response.Data.Mode, response.Data.RedirectURL)
	})

	// Test 3: Update 404 mode to redirect
	t.Run("Update404ToRedirect", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"mode":         "redirect",
			"redirect_url": "https://example.com/404",
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, "/api/settings/404", updateReq)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Trigger a sync to update the JSON config
		syncResp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/sync/trigger", nil)
		_ = syncResp.Body.Close()
		env.WaitForSyncComplete(t, 5*time.Second)

		// Verify JSON config was updated with redirect for 404
		content, err := env.ReadJSONConfig(t)
		if err != nil {
			t.Fatalf("Failed to read JSON config: %v", err)
		}

		// In JSON config, the 404 redirect should appear as a static_response with Location header
		// or a redirect handler in the catch-all server
		if !strings.Contains(content, "https://example.com/404") {
			t.Error("JSON config should contain redirect URL for 404 mode")
		}
		t.Logf("JSON config updated with 404 redirect mode")
	})

	// Test 4: Update 404 mode back to default
	t.Run("Update404ToDefault", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"mode":         "default",
			"redirect_url": "",
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, "/api/settings/404", updateReq)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		// Trigger a sync to update the JSON config
		syncResp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/sync/trigger", nil)
		_ = syncResp.Body.Close()
		env.WaitForSyncComplete(t, 5*time.Second)

		// Verify JSON config was updated with default 404 response
		content, err := env.ReadJSONConfig(t)
		if err != nil {
			t.Fatalf("Failed to read JSON config: %v", err)
		}

		// In JSON config, the default 404 should not contain the redirect URL anymore
		if strings.Contains(content, "https://example.com/404") {
			t.Error("JSON config should not contain redirect URL for default mode")
		}
		t.Logf("JSON config updated with default 404 mode")
	})
}

// TestIntegration_SyncAPI tests sync API
func TestIntegration_SyncAPI(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Test 1: Get sync status
	t.Run("GetSyncStatus", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/sync/status", nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool `json:"success"`
			Data    struct {
				LastSyncTime    string `json:"last_sync_time"`
				IsSyncing       bool   `json:"is_syncing"`
				LastSyncSuccess bool   `json:"last_sync_success"`
				LastError       string `json:"last_error"`
				SyncCount       int    `json:"sync_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}
		t.Logf("Sync status: syncing=%v, last_success=%v, count=%d",
			response.Data.IsSyncing, response.Data.LastSyncSuccess, response.Data.SyncCount)
	})

	// Test 2: Trigger sync
	t.Run("TriggerSync", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/sync/trigger", nil)
		defer func() { _ = resp.Body.Close() }()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
		}

		var response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}
		t.Logf("Sync trigger response: %s", response.Message)

		// Wait for sync to complete (with polling)
		env.WaitForSyncComplete(t, 5*time.Second)

		// Verify JSON config was generated
		content, err := env.ReadJSONConfig(t)
		if err != nil {
			t.Fatalf("Failed to read JSON config: %v", err)
		}

		// Check for valid JSON structure
		if !strings.Contains(content, `"apps"`) {
			t.Error("JSON config should contain 'apps' key")
		}
		if !strings.Contains(content, `"http"`) {
			t.Error("JSON config should contain 'http' app")
		}
		t.Logf("Caddy JSON config:\n%s", content)
	})
}

// TestIntegration_ReverseProxy_LoadBalancing tests reverse proxy with load balancing
func TestIntegration_ReverseProxy_LoadBalancing(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	hostname := "lb.example.com"

	// Create proxy with multiple upstreams and load balancing
	proxy := map[string]interface{}{
		"type":     "reverse_proxy",
		"name":     "Load Balanced Backend",
		"hostname": hostname,
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

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify proxy appears in JSON config (with polling)
	if !env.WaitForProxyInConfig(t, hostname, 5*time.Second) {
		t.Fatalf("Expected hostname %s to appear in JSON config", hostname)
	}

	content, err := env.ReadJSONConfig(t)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	// Check for multiple upstreams in JSON config
	if !strings.Contains(content, "backend1.internal") {
		t.Error("JSON config should contain backend1")
	}
	if !strings.Contains(content, "backend2.internal") {
		t.Error("JSON config should contain backend2")
	}
	if !strings.Contains(content, "backend3.internal") {
		t.Error("JSON config should contain backend3")
	}

	// Check for reverse_proxy handler
	if !strings.Contains(content, "reverse_proxy") {
		t.Error("JSON config should contain reverse_proxy handler")
	}

	// Check for load balancing policy (in JSON config it's under load_balancing or selection_policy)
	if !strings.Contains(content, "round_robin") {
		t.Error("JSON config should contain round_robin load balancing policy")
	}

	// Check for health checks (in JSON config it's under health_checks or active_health_checks)
	if !strings.Contains(content, "/health") {
		t.Error("JSON config should contain health check path")
	}

	t.Logf("Load balanced proxy JSON config created for: %s", hostname)
}

// TestIntegration_ReverseProxy_BlockExploits tests reverse proxy with block exploits enabled
func TestIntegration_ReverseProxy_BlockExploits(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	hostname := "secure.example.com"

	// Create proxy with block exploits enabled
	proxy := map[string]interface{}{
		"type":           "reverse_proxy",
		"name":           "Secure Backend",
		"hostname":       hostname,
		"block_exploits": true,
		"upstreams": []map[string]interface{}{
			{"host": "backend.internal", "port": 8080, "scheme": "http"},
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Data struct {
			ID            int  `json:"id"`
			BlockExploits bool `json:"block_exploits"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify block_exploits is set in the response
	if !response.Data.BlockExploits {
		t.Error("Expected block_exploits to be true in response")
	}

	// Verify proxy appears in JSON config (with polling)
	if !env.WaitForProxyInConfig(t, hostname, 5*time.Second) {
		t.Fatalf("Expected hostname %s to appear in JSON config", hostname)
	}

	content, err := env.ReadJSONConfig(t)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	// Verify the proxy is in the config
	if !strings.Contains(content, hostname) {
		t.Error("JSON config should contain hostname")
	}

	// In JSON config, block_exploits adds security routes that block common exploit patterns
	// Check for the reverse_proxy handler
	if !strings.Contains(content, "reverse_proxy") {
		t.Error("JSON config should contain reverse_proxy handler")
	}

	// Verify the proxy was created with block_exploits by checking the API response
	getResp := env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/proxies/%d", response.Data.ID), nil)
	defer func() { _ = getResp.Body.Close() }()

	var getResponse struct {
		Data struct {
			BlockExploits bool `json:"block_exploits"`
		} `json:"data"`
	}
	env.ReadJSONResponse(t, getResp, &getResponse)

	if !getResponse.Data.BlockExploits {
		t.Error("Expected block_exploits to be true when retrieving proxy")
	}

	t.Logf("Secure proxy JSON config created for: %s with block_exploits enabled", hostname)
}

// TestIntegration_AuthErrors tests authentication error handling
func TestIntegration_AuthErrors(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	// Test 1: Missing Authorization header
	t.Run("MissingAuthHeader", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/api/proxies", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		// Intentionally not setting Authorization header

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			respBody, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 401 Unauthorized for missing auth header, got %d: %s", resp.StatusCode, string(respBody))
		}
		t.Log("Correctly rejected request without Authorization header")
	})

	// Test 2: Invalid/malformed token
	t.Run("InvalidToken", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/api/proxies", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid-token-that-is-not-a-valid-jwt")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			respBody, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 401 Unauthorized for invalid token, got %d: %s", resp.StatusCode, string(respBody))
		}
		t.Log("Correctly rejected request with invalid token")
	})

	// Test 3: Malformed Authorization header (missing Bearer prefix)
	t.Run("MalformedAuthHeader", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/api/proxies", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "some-token-without-bearer-prefix")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			respBody, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 401 Unauthorized for malformed auth header, got %d: %s", resp.StatusCode, string(respBody))
		}
		t.Log("Correctly rejected request with malformed Authorization header")
	})

	// Test 4: Empty Bearer token
	t.Run("EmptyBearerToken", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, env.BaseURL+"/api/proxies", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer ")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			respBody, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 401 Unauthorized for empty bearer token, got %d: %s", resp.StatusCode, string(respBody))
		}
		t.Log("Correctly rejected request with empty Bearer token")
	})
}
