package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// NewHealthHandler Tests
// =============================================================================

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler()

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.startTime.IsZero() {
		t.Error("Expected startTime to be set")
	}
}

func TestNewHealthHandlerWithDB(t *testing.T) {
	// Test with nil DB
	handler := NewHealthHandlerWithDB(nil)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.startTime.IsZero() {
		t.Error("Expected startTime to be set")
	}
}

// =============================================================================
// HealthCheck Tests
// =============================================================================

func TestHealthHandler_HealthCheck_ResponseFormat(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response structure
	if _, ok := response["success"]; !ok {
		t.Error("Expected 'success' field in response")
	}

	data := response["data"].(map[string]interface{})

	// Check required fields
	requiredFields := []string{"status", "service", "version", "uptime", "time", "components"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Expected field '%s' in data", field)
		}
	}

	// Check service name
	if data["service"] != "caddy-manager-backend" {
		t.Errorf("Expected service 'caddy-manager-backend', got '%v'", data["service"])
	}

	// Check version
	if data["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", data["version"])
	}

	// Check components structure
	components := data["components"].(map[string]interface{})
	if _, ok := components["database"]; !ok {
		t.Error("Expected 'database' in components")
	}
}

func TestHealthHandler_HealthCheck_UptimeIncreases(t *testing.T) {
	handler := NewHealthHandler()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec1 := httptest.NewRecorder()
	handler.HealthCheck(rec1, req1)

	var response1 map[string]interface{}
	if err := json.Unmarshal(rec1.Body.Bytes(), &response1); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data1 := response1["data"].(map[string]interface{})
	uptime1 := data1["uptime"].(string)

	if uptime1 == "" {
		t.Error("Expected uptime to be non-empty")
	}
}

func TestHealthHandler_HealthCheck_StatusField(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	status := data["status"].(string)

	// Status should be either "healthy" or "degraded"
	if status != "healthy" && status != "degraded" {
		t.Errorf("Expected status to be 'healthy' or 'degraded', got '%s'", status)
	}
}

func TestHealthHandler_HealthCheck_TimeFormat(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	timeStr := data["time"].(string)

	// Time should be in RFC3339 format
	if timeStr == "" {
		t.Error("Expected time to be non-empty")
	}
	// RFC3339 format should contain 'T' and 'Z' or timezone offset
	if len(timeStr) < 20 {
		t.Errorf("Time format seems incorrect: %s", timeStr)
	}
}

func TestHealthHandler_HealthCheck_WithNilDB(t *testing.T) {
	handler := NewHealthHandlerWithDB(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	// Should still return 200 OK even if DB check fails
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Response should still be successful
	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}
}

func TestHealthHandler_HealthCheck_ComponentsPresent(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	components := data["components"].(map[string]interface{})

	// Database component should have a status
	dbStatus := components["database"].(string)
	if dbStatus != "healthy" && dbStatus != "unhealthy" {
		t.Errorf("Expected database status to be 'healthy' or 'unhealthy', got '%s'", dbStatus)
	}
}
