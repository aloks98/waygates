package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewSyncHandler Tests
// =============================================================================

func TestNewSyncHandler(t *testing.T) {
	mockService := &mocks.MockSyncService{}
	handler := NewSyncHandler(mockService, nil)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.syncService != mockService {
		t.Error("Expected sync service to be set")
	}
}

// =============================================================================
// GetStatus Tests
// =============================================================================

func TestSyncHandler_Unit_GetStatus_Success(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now(),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       10,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
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
	if data["is_syncing"].(bool) {
		t.Error("Expected is_syncing to be false")
	}
	if !data["last_sync_success"].(bool) {
		t.Error("Expected last_sync_success to be true")
	}
}

func TestSyncHandler_Unit_GetStatus_WhileSyncing(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       true,
				LastSyncTime:    time.Now().Add(-30 * time.Second),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       5,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
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
	if !data["is_syncing"].(bool) {
		t.Error("Expected is_syncing to be true")
	}
}

func TestSyncHandler_GetStatus_AfterError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now().Add(-1 * time.Minute),
				LastSyncSuccess: false,
				LastError:       "Caddy not reachable: connection refused",
				SyncCount:       3,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
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
	if data["last_sync_success"].(bool) {
		t.Error("Expected last_sync_success to be false")
	}
	if data["last_error"] != "Caddy not reachable: connection refused" {
		t.Errorf("Expected last_error to be 'Caddy not reachable: connection refused', got '%v'", data["last_error"])
	}
}

func TestSyncHandler_GetStatus_ZeroSyncCount(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Time{},
				LastSyncSuccess: false,
				LastError:       "",
				SyncCount:       0,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
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
	if data["sync_count"].(float64) != 0 {
		t.Errorf("Expected sync_count to be 0, got %v", data["sync_count"])
	}
}

// =============================================================================
// Trigger Tests
// =============================================================================

func TestSyncHandler_Unit_Trigger_Success(t *testing.T) {
	syncCalled := false
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			syncCalled = true
			return nil
		},
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now(),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       1,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !syncCalled {
		t.Error("Expected FullSync to be called")
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}
	if response["message"] != "Sync completed successfully" {
		t.Errorf("Expected message 'Sync completed successfully', got '%v'", response["message"])
	}
}

func TestSyncHandler_Trigger_SyncError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("Caddy not reachable: connection refused")
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"].(bool) {
		t.Error("Expected success to be false")
	}
}

func TestSyncHandler_Trigger_DatabaseError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("database connection lost")
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestSyncHandler_Trigger_ReturnsUpdatedStatus(t *testing.T) {
	statusCallCount := 0
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return nil
		},
		GetStatusFunc: func() service.SyncStatus {
			statusCallCount++
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now(),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       statusCallCount,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// GetStatus should be called once after successful sync
	if statusCallCount != 1 {
		t.Errorf("Expected GetStatus to be called once, but was called %d times", statusCallCount)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["sync_count"].(float64) != 1 {
		t.Errorf("Expected sync_count to be 1, got %v", data["sync_count"])
	}
}

func TestSyncHandler_Trigger_RouteConfigError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("invalid route configuration: missing upstream")
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorData := response["error"].(map[string]interface{})
	if errorData["message"] != "Sync failed: invalid route configuration: missing upstream" {
		t.Errorf("Expected error message to contain sync failure reason, got '%v'", errorData["message"])
	}
}

func TestSyncHandler_Trigger_FileSystemError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("failed to write proxy file: permission denied")
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestSyncHandler_StatusResponseFormat(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now(),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       5,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
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
	requiredFields := []string{"is_syncing", "last_sync_time", "last_sync_success", "sync_count"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Expected field '%s' in data", field)
		}
	}
}

func TestSyncHandler_TriggerResponseFormat(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return nil
		},
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now(),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       1,
			}
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Successful trigger should include status data
	if _, ok := response["data"]; !ok {
		t.Error("Expected 'data' field in response after successful trigger")
	}
}

func TestSyncHandler_TriggerErrorResponseFormat(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("sync failed")
		},
	}
	handler := NewSyncHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler.Trigger(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Error response should have error field
	if _, ok := response["error"]; !ok {
		t.Error("Expected 'error' field in error response")
	}

	errorData := response["error"].(map[string]interface{})
	if _, ok := errorData["message"]; !ok {
		t.Error("Expected 'message' field in error data")
	}
}
