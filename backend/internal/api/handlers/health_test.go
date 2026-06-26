package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/version"
)

// =============================================================================
// NewHealthHandler Tests
// =============================================================================

func TestNewHealthHandler(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	require.NotNil(t, handler, "handler should be created")
	assert.False(t, handler.startTime.IsZero(), "startTime should be set")
}

func TestNewHealthHandlerWithDB(t *testing.T) {
	t.Parallel()
	// Test with nil DB
	handler := NewHealthHandlerWithDB(nil)

	require.NotNil(t, handler, "handler should be created")
	assert.False(t, handler.startTime.IsZero(), "startTime should be set")
}

// =============================================================================
// HealthCheck Tests
// =============================================================================

func TestHealthHandler_HealthCheck_ResponseFormat(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status     string            `json:"status"`
			Service    string            `json:"service"`
			Version    string            `json:"version"`
			Uptime     string            `json:"uptime"`
			Time       string            `json:"time"`
			Components map[string]string `json:"components"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Check response structure
	assert.True(t, response.Success, "expected success to be true")

	// Check required fields are present
	assert.NotEmpty(t, response.Data.Status, "expected status to be set")
	assert.NotEmpty(t, response.Data.Service, "expected service to be set")
	assert.NotEmpty(t, response.Data.Version, "expected version to be set")
	assert.NotEmpty(t, response.Data.Uptime, "expected uptime to be set")
	assert.NotEmpty(t, response.Data.Time, "expected time to be set")
	assert.NotNil(t, response.Data.Components, "expected components to be set")

	// Check service name
	assert.Equal(t, "caddy-manager-backend", response.Data.Service)

	// Check version reflects the resolved build version (stamped value, or
	// "dev-<commit>" when unstamped)
	assert.Equal(t, version.String(), response.Data.Version)

	// Check components structure
	_, hasDatabase := response.Data.Components["database"]
	assert.True(t, hasDatabase, "expected 'database' in components")
}

func TestHealthHandler_HealthCheck_UptimeIncreases(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec1 := httptest.NewRecorder()
	handler.HealthCheck(rec1, req1)

	var response1 struct {
		Data struct {
			Uptime string `json:"uptime"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &response1), "failed to parse response")

	assert.NotEmpty(t, response1.Data.Uptime, "expected uptime to be non-empty")
}

func TestHealthHandler_HealthCheck_StatusField(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Status should be either "healthy" or "degraded"
	assert.Contains(t, []string{"healthy", "degraded"}, response.Data.Status,
		"expected status to be 'healthy' or 'degraded', got '%s'", response.Data.Status)
}

func TestHealthHandler_HealthCheck_TimeFormat(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response struct {
		Data struct {
			Time string `json:"time"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Time should be in RFC3339 format
	assert.NotEmpty(t, response.Data.Time, "expected time to be non-empty")
	// RFC3339 format should contain 'T' and be at least 20 chars
	assert.GreaterOrEqual(t, len(response.Data.Time), 20, "time format seems incorrect: %s", response.Data.Time)
}

func TestHealthHandler_HealthCheck_WithNilDB(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandlerWithDB(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	// Should still return 200 OK even if DB check fails
	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Response should still be successful
	assert.True(t, response.Success, "expected success to be true")
}

func TestHealthHandler_HealthCheck_ComponentsPresent(t *testing.T) {
	t.Parallel()
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response struct {
		Data struct {
			Components map[string]string `json:"components"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	// Database component should have a status
	dbStatus, hasDatabase := response.Data.Components["database"]
	require.True(t, hasDatabase, "expected 'database' in components")
	assert.Contains(t, []string{"healthy", "unhealthy"}, dbStatus,
		"expected database status to be 'healthy' or 'unhealthy', got '%s'", dbStatus)
}
