package caddy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Test NewReloader with default configuration
func TestNewReloader_Defaults(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "caddy", r.caddyBinary, "Expected default caddy binary 'caddy'")
	assert.Equal(t, DefaultAdminAPIURL, r.adminAPIURL, "Expected default admin API URL")
}

// Test NewReloader with custom configuration
func TestNewReloader_CustomConfig(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/usr/local/bin/caddy",
		AdminAPIURL: "http://localhost:3000",
	}

	r := NewReloader(cfg, logger)

	assert.Equal(t, "/usr/local/bin/caddy", r.caddyBinary, "Expected custom binary path")
	assert.Equal(t, "http://localhost:3000", r.adminAPIURL, "Expected custom admin API URL")
}

// Test GetStatus returns correct status information
func TestReloaderStatus(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	cfg := ReloaderConfig{
		CaddyBinary: "/usr/bin/caddy",
		AdminAPIURL: "http://localhost:2019",
	}

	r := NewReloader(cfg, logger)
	status := r.GetStatus()

	assert.Equal(t, "/usr/bin/caddy", status.CaddyBinary, "Status CaddyBinary mismatch")
	assert.Equal(t, "http://localhost:2019", status.AdminAPIURL, "Status AdminAPIURL mismatch")
	assert.Equal(t, 0, status.ReloadCount, "Initial reload count should be 0")
	assert.True(t, status.LastReload.IsZero(), "Initial last reload should be zero time")
}

// Test ValidationError implements error interface correctly
func TestValidationError(t *testing.T) {
	t.Parallel()
	err := &ValidationError{
		Message: "invalid syntax",
		Output:  "line 5: unexpected token",
	}

	expected := "configuration validation failed: invalid syntax"
	assert.Equal(t, expected, err.Error(), "ValidationError.Error() mismatch")
}

// Test extractValidationError with various inputs
func TestExtractValidationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Error with prefix",
			input:    "Error: invalid hostname",
			expected: "invalid hostname",
		},
		{
			name:     "error lowercase",
			input:    "error: missing closing brace",
			expected: "missing closing brace",
		},
		{
			name:     "Multi-line with error",
			input:    "Loading config...\nError: syntax error on line 10\nDone",
			expected: "syntax error on line 10",
		},
		{
			name:     "No error prefix",
			input:    "unexpected token at position 5",
			expected: "unexpected token at position 5",
		},
		{
			name:     "Empty output",
			input:    "",
			expected: "validation failed with unknown error",
		},
		{
			name:     "Only whitespace",
			input:    "   \n   \n   ",
			expected: "validation failed with unknown error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractValidationError(tc.input)
			assert.Equal(t, tc.expected, result, "extractValidationError result mismatch")
		})
	}
}

// Test ReloadResult fields
func TestReloadResult(t *testing.T) {
	t.Parallel()

	result := &ReloadResult{
		Success:     true,
		Message:     "Configuration reloaded successfully",
		ReloadCount: 5,
	}

	assert.True(t, result.Success, "Expected Success to be true")
	assert.Equal(t, "Configuration reloaded successfully", result.Message)
	assert.Equal(t, 5, result.ReloadCount)
}

// Test TestConnection with invalid binary
func TestTestConnection_InvalidBinary(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	r := NewReloader(ReloaderConfig{
		CaddyBinary: "/nonexistent/caddy/binary",
	}, logger)

	ctx := context.Background()
	err := r.TestConnection(ctx)
	assert.Error(t, err, "TestConnection should fail with invalid binary")
	assert.Contains(t, err.Error(), "caddy not available")
}

// Test TestConnection with context cancellation
func TestTestConnection_ContextCancelled(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	r := NewReloader(ReloaderConfig{
		CaddyBinary: "sleep", // Use sleep to simulate slow command
	}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := r.TestConnection(ctx)
	assert.Error(t, err, "TestConnection should fail with canceled context")
}

// Test ValidateJSON with invalid binary
func TestValidateJSON_InvalidBinary(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()
	r := NewReloader(ReloaderConfig{
		CaddyBinary: "/nonexistent/caddy/binary",
	}, logger)

	err := r.ValidateJSON("/some/config.json")
	assert.Error(t, err, "ValidateJSON should fail with invalid binary")

	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr, "error should be ValidationError")
}

// Test ReloadJSON with successful admin API response
func TestReloadJSON_AdminAPISuccess(t *testing.T) {
	t.Parallel()

	// Create a mock admin API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/load", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "caddy.json")
	configContent := `{"admin": {"listen": "localhost:2019"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "failed to write test config")

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "echo", // Use echo as a dummy validator (always succeeds)
		adminAPIURL: server.URL,
		httpClient:  server.Client(),
		logger:      logger,
	}

	ctx := context.Background()
	result, err := r.ReloadJSON(ctx, configPath)

	require.NoError(t, err, "ReloadJSON should not return error")
	require.NotNil(t, result, "result should not be nil")
	assert.True(t, result.Success, "result.Success should be true")
	assert.Equal(t, 1, result.ReloadCount, "reload count should be 1")
	assert.Greater(t, result.Duration, time.Duration(0), "duration should be positive")
}

// Test ReloadJSON with admin API error response
func TestReloadJSON_AdminAPIError(t *testing.T) {
	t.Parallel()

	// Create a mock admin API server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid configuration"))
	}))
	defer server.Close()

	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "caddy.json")
	configContent := `{"admin": {"listen": "localhost:2019"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "failed to write test config")

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "echo",
		adminAPIURL: server.URL,
		httpClient:  server.Client(),
		logger:      logger,
	}

	ctx := context.Background()
	result, err := r.ReloadJSON(ctx, configPath)

	assert.Error(t, err, "ReloadJSON should return error for bad status")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "400")
}

// Test ReloadJSON with nonexistent config file
func TestReloadJSON_FileNotFound(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "echo",
		adminAPIURL: "http://localhost:2019",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		logger:      logger,
	}

	ctx := context.Background()
	result, err := r.ReloadJSON(ctx, "/nonexistent/path/caddy.json")

	assert.Error(t, err, "ReloadJSON should return error for nonexistent file")
	assert.Nil(t, result, "result should be nil on error")
}

// Test ReloadJSON updates status correctly
func TestReloadJSON_UpdatesStatus(t *testing.T) {
	t.Parallel()

	// Create a mock admin API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "caddy.json")
	configContent := `{"admin": {"listen": "localhost:2019"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "failed to write test config")

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "echo",
		adminAPIURL: server.URL,
		httpClient:  server.Client(),
		logger:      logger,
	}

	// Check initial status
	status := r.GetStatus()
	assert.Equal(t, 0, status.ReloadCount, "initial reload count should be 0")
	assert.True(t, status.LastReload.IsZero(), "initial LastReload should be zero")

	ctx := context.Background()

	// First reload
	_, err = r.ReloadJSON(ctx, configPath)
	require.NoError(t, err)

	status = r.GetStatus()
	assert.Equal(t, 1, status.ReloadCount, "reload count should be 1")
	assert.False(t, status.LastReload.IsZero(), "LastReload should not be zero")

	firstReloadTime := status.LastReload

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Second reload
	_, err = r.ReloadJSON(ctx, configPath)
	require.NoError(t, err)

	status = r.GetStatus()
	assert.Equal(t, 2, status.ReloadCount, "reload count should be 2")
	assert.True(t, status.LastReload.After(firstReloadTime), "LastReload should be updated")
}

// Test ReloadJSON with context timeout
func TestReloadJSON_ContextTimeout(t *testing.T) {
	t.Parallel()

	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "caddy.json")
	configContent := `{"admin": {"listen": "localhost:2019"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "failed to write test config")

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "echo",
		adminAPIURL: server.URL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		logger:      logger,
	}

	// Create a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = r.ReloadJSON(ctx, configPath)
	assert.Error(t, err, "ReloadJSON should fail with context timeout")
}

// Test ReloadJSON with validation failure (invalid caddy binary simulating validation error)
func TestReloadJSON_ValidationFailure(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	r := &Reloader{
		caddyBinary: "/nonexistent/caddy", // Will fail validation
		adminAPIURL: "http://localhost:2019",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		logger:      logger,
	}

	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "caddy.json")
	configContent := `{"admin": {"listen": "localhost:2019"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "failed to write test config")

	ctx := context.Background()
	result, err := r.ReloadJSON(ctx, configPath)

	assert.Error(t, err, "ReloadJSON should fail when validation fails")
	assert.Nil(t, result, "result should be nil on validation failure")

	var validationErr *ValidationError
	assert.ErrorAs(t, err, &validationErr, "error should be ValidationError")
}

// Test DefaultAdminAPIURL constant
func TestDefaultAdminAPIURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://localhost:2019", DefaultAdminAPIURL)
}

// Test ReloaderStatus struct
func TestReloaderStatus_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := ReloaderStatus{
		CaddyBinary: "/usr/bin/caddy",
		AdminAPIURL: "http://localhost:2019",
		LastReload:  now,
		ReloadCount: 10,
	}

	assert.Equal(t, "/usr/bin/caddy", status.CaddyBinary)
	assert.Equal(t, "http://localhost:2019", status.AdminAPIURL)
	assert.Equal(t, now, status.LastReload)
	assert.Equal(t, 10, status.ReloadCount)
}

// Test ValidationError with empty values
func TestValidationError_EmptyValues(t *testing.T) {
	t.Parallel()

	err := &ValidationError{
		Message: "",
		Output:  "",
	}

	assert.Equal(t, "configuration validation failed: ", err.Error())
}

// Test ReloadResult with duration
func TestReloadResult_Duration(t *testing.T) {
	t.Parallel()

	result := &ReloadResult{
		Success:     true,
		Message:     "OK",
		Duration:    150 * time.Millisecond,
		ReloadCount: 1,
	}

	assert.Equal(t, 150*time.Millisecond, result.Duration)
}

// Test NewReloader creates http client with timeout
func TestNewReloader_HttpClientTimeout(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop()

	r := NewReloader(ReloaderConfig{}, logger)

	assert.NotNil(t, r.httpClient)
	assert.Equal(t, 30*time.Second, r.httpClient.Timeout)
}
