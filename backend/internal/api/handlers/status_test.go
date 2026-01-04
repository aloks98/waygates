package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewStatusHandler Tests
// =============================================================================

func TestNewStatusHandler(t *testing.T) {
	mockReloader := &mocks.MockReloader{}
	mockUserRepo := &mocks.MockUserRepository{}

	handler := NewStatusHandler(mockReloader, mockUserRepo)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.reloader != mockReloader {
		t.Error("Expected reloader to be set")
	}
	if handler.userRepo != mockUserRepo {
		t.Error("Expected userRepo to be set")
	}
}

// =============================================================================
// GetStatus Tests
// =============================================================================

func TestStatusHandler_GetStatus_AllHealthy(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 5, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})
	if data["caddy_status"] != "healthy" {
		t.Errorf("Expected caddy_status 'healthy', got '%v'", data["caddy_status"])
	}
	if data["user_setup_complete"] != true {
		t.Errorf("Expected user_setup_complete true, got '%v'", data["user_setup_complete"])
	}
}

func TestStatusHandler_GetStatus_CaddyUnhealthy(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return errors.New("connection refused")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["caddy_status"] != "unhealthy" {
		t.Errorf("Expected caddy_status 'unhealthy', got '%v'", data["caddy_status"])
	}
}

func TestStatusHandler_GetStatus_NoUsers(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["user_setup_complete"] != false {
		t.Errorf("Expected user_setup_complete false, got '%v'", data["user_setup_complete"])
	}
}

func TestStatusHandler_GetStatus_UserCountError(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, errors.New("database error")
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestStatusHandler_GetStatus_BothUnhealthy(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return errors.New("caddy not running")
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 0, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["caddy_status"] != "unhealthy" {
		t.Errorf("Expected caddy_status 'unhealthy', got '%v'", data["caddy_status"])
	}
	if data["user_setup_complete"] != false {
		t.Errorf("Expected user_setup_complete false, got '%v'", data["user_setup_complete"])
	}
}

func TestStatusHandler_GetStatus_ResponseFormat(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response structure
	if _, ok := response["success"]; !ok {
		t.Error("Expected 'success' field in response")
	}
	if _, ok := response["message"]; !ok {
		t.Error("Expected 'message' field in response")
	}
	if _, ok := response["data"]; !ok {
		t.Error("Expected 'data' field in response")
	}

	// Check data structure
	data := response["data"].(map[string]interface{})
	if _, ok := data["caddy_status"]; !ok {
		t.Error("Expected 'caddy_status' in data")
	}
	if _, ok := data["user_setup_complete"]; !ok {
		t.Error("Expected 'user_setup_complete' in data")
	}
}

func TestStatusHandler_GetStatus_SuccessMessage(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Status retrieved successfully" {
		t.Errorf("Expected message 'Status retrieved successfully', got '%v'", response["message"])
	}
}

func TestStatusHandler_GetStatus_CaddyTimeout(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			// Simulate context being canceled/timeout
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return errors.New("timeout")
			}
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	// Caddy should be marked unhealthy on timeout
	if data["caddy_status"] != "unhealthy" {
		t.Errorf("Expected caddy_status 'unhealthy' on timeout, got '%v'", data["caddy_status"])
	}
}

func TestStatusHandler_GetStatus_ManyUsers(t *testing.T) {
	mockReloader := &mocks.MockReloader{
		TestConnectionFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockUserRepo := &mocks.MockUserRepository{
		CountFunc: func() (int64, error) {
			return 1000, nil
		},
	}
	handler := NewStatusHandler(mockReloader, mockUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["user_setup_complete"] != true {
		t.Errorf("Expected user_setup_complete true with 1000 users, got '%v'", data["user_setup_complete"])
	}
}
