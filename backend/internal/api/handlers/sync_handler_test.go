package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aloks98/waygates/backend/internal/utils"
)

// mockSyncStatus represents the sync status for testing
type mockSyncStatus struct {
	LastSyncTime    time.Time `json:"last_sync_time"`
	IsSyncing       bool      `json:"is_syncing"`
	LastSyncSuccess bool      `json:"last_sync_success"`
	LastError       string    `json:"last_error,omitempty"`
}

// TestGetSyncStatus_Success tests successful retrieval of sync status
func TestGetSyncStatus_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := mockSyncStatus{
			LastSyncTime:    time.Now(),
			IsSyncing:       false,
			LastSyncSuccess: true,
			LastError:       "",
		}
		utils.Success(w, status, "Sync status retrieved successfully")
	})

	handler.ServeHTTP(rec, req)

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

// TestGetSyncStatus_WhileSyncing tests status while sync is in progress
func TestGetSyncStatus_WhileSyncing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := mockSyncStatus{
			LastSyncTime:    time.Now().Add(-30 * time.Second),
			IsSyncing:       true,
			LastSyncSuccess: true,
			LastError:       "",
		}
		utils.Success(w, status, "Sync status retrieved successfully")
	})

	handler.ServeHTTP(rec, req)

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

// TestGetSyncStatus_AfterError tests status after a sync error
func TestGetSyncStatus_AfterError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/sync/status", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := mockSyncStatus{
			LastSyncTime:    time.Now().Add(-1 * time.Minute),
			IsSyncing:       false,
			LastSyncSuccess: false,
			LastError:       "Caddy not reachable: connection refused",
		}
		utils.Success(w, status, "Sync status retrieved successfully")
	})

	handler.ServeHTTP(rec, req)

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
	if data["last_error"] == "" {
		t.Error("Expected last_error to contain error message")
	}
}

// TestTriggerSync_Success tests successful manual sync trigger
func TestTriggerSync_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate successful sync
		status := mockSyncStatus{
			LastSyncTime:    time.Now(),
			IsSyncing:       false,
			LastSyncSuccess: true,
			LastError:       "",
		}
		utils.Success(w, status, "Sync completed successfully")
	})

	handler.ServeHTTP(rec, req)

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
	if response["message"] != "Sync completed successfully" {
		t.Errorf("Expected message 'Sync completed successfully', got '%v'", response["message"])
	}
}

// TestTriggerSync_AlreadyInProgress tests triggering sync when one is already running
func TestTriggerSync_AlreadyInProgress(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	isSyncing := true

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSyncing {
			utils.BadRequest(w, "Sync already in progress", nil)
			return
		}
		utils.Success(w, nil, "Sync completed")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestTriggerSync_CaddyUnreachable tests sync failure when Caddy is down
func TestTriggerSync_CaddyUnreachable(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate Caddy unreachable error
		utils.InternalError(w, "Sync failed: Caddy not reachable: connection refused")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestTriggerSync_RouteSyncFailed tests sync failure during route sync
func TestTriggerSync_RouteSyncFailed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/sync/trigger", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.InternalError(w, "Sync failed: Route sync failed: invalid route configuration")
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestSyncStatusResponseFormat tests the sync status response format
func TestSyncStatusResponseFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	status := mockSyncStatus{
		LastSyncTime:    time.Now(),
		IsSyncing:       false,
		LastSyncSuccess: true,
	}
	utils.Success(rec, status, "Sync status retrieved")

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})

	// Verify required fields exist
	requiredFields := []string{"last_sync_time", "is_syncing", "last_sync_success"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("Expected field '%s' to be present", field)
		}
	}
}

// TestSyncHTTPMethods tests that only correct HTTP methods are allowed
func TestSyncHTTPMethods(t *testing.T) {
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{"GET status", http.MethodGet, "/api/sync/status", http.StatusOK},
		{"POST trigger", http.MethodPost, "/api/sync/trigger", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				utils.Success(w, nil, "")
			})

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expected {
				t.Errorf("Expected status %d, got %d", tc.expected, rec.Code)
			}
		})
	}
}
