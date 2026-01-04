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

func TestSyncHandler_GetStatus_Success(t *testing.T) {
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

	handler := NewSyncHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
}

func TestSyncHandler_GetStatus_WhileSyncing(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       true,
				LastSyncTime:    time.Now().Add(-time.Minute),
				LastSyncSuccess: true,
				LastError:       "",
				SyncCount:       5,
			}
		},
	}

	handler := NewSyncHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["is_syncing"] != true {
		t.Error("Expected is_syncing to be true")
	}
}

func TestSyncHandler_Trigger_Success(t *testing.T) {
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
				SyncCount:       5,
			}
		},
	}

	handler := NewSyncHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	w := httptest.NewRecorder()

	handler.Trigger(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
}

func TestSyncHandler_Trigger_Error(t *testing.T) {
	mockService := &mocks.MockSyncService{
		FullSyncFunc: func() error {
			return errors.New("sync failed")
		},
	}

	handler := NewSyncHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	w := httptest.NewRecorder()

	handler.Trigger(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestSyncHandler_GetStatus_WithError(t *testing.T) {
	mockService := &mocks.MockSyncService{
		GetStatusFunc: func() service.SyncStatus {
			return service.SyncStatus{
				IsSyncing:       false,
				LastSyncTime:    time.Now().Add(-time.Hour),
				LastSyncSuccess: false,
				LastError:       "Previous sync failed",
				SyncCount:       0,
			}
		},
	}

	handler := NewSyncHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	w := httptest.NewRecorder()

	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data := response["data"].(map[string]interface{})
	if data["last_error"] != "Previous sync failed" {
		t.Errorf("Expected last_error 'Previous sync failed', got '%s'", data["last_error"])
	}
}
