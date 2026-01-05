package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewAuditHandler Tests
// =============================================================================

func TestNewAuditHandler(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.auditService != mockService {
		t.Error("Expected audit service to be set")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestAuditHandler_List_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return &models.AuditLogListResponse{
				Items: []models.AuditLog{
					{ID: 1, Action: "proxy.create", Status: "success"},
					{ID: 2, Action: "proxy.update", Status: "success"},
				},
				Total:      2,
				Page:       1,
				Limit:      20,
				TotalPages: 1,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

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
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
}

func TestAuditHandler_List_WithFilters(t *testing.T) {
	var capturedParams repository.AuditLogListParams
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			capturedParams = params
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      10,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?page=2&limit=10&action=proxy.create&status=success&search=test", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedParams.Page != 2 {
		t.Errorf("Expected page 2, got %d", capturedParams.Page)
	}
	if capturedParams.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", capturedParams.Limit)
	}
	if len(capturedParams.Actions) != 1 || capturedParams.Actions[0] != "proxy.create" {
		t.Errorf("Expected actions ['proxy.create'], got %v", capturedParams.Actions)
	}
	if capturedParams.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", capturedParams.Status)
	}
	if capturedParams.Search != "test" {
		t.Errorf("Expected search 'test', got '%s'", capturedParams.Search)
	}
}

func TestAuditHandler_List_WithMultipleActions(t *testing.T) {
	var capturedParams repository.AuditLogListParams
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			capturedParams = params
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?action=proxy.create,proxy.update,proxy.delete", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(capturedParams.Actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(capturedParams.Actions))
	}
	expectedActions := []string{"proxy.create", "proxy.update", "proxy.delete"}
	for i, action := range expectedActions {
		if capturedParams.Actions[i] != action {
			t.Errorf("Expected action[%d] '%s', got '%s'", i, action, capturedParams.Actions[i])
		}
	}
}

func TestAuditHandler_List_WithExclusionFilters(t *testing.T) {
	var capturedParams repository.AuditLogListParams
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			capturedParams = params
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	// New format: field=operator:value
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?action=not_in:proxy.delete,auth.logout&resource_type=not:system&status=not:failure", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(capturedParams.ActionsExclude) != 2 {
		t.Errorf("Expected 2 excluded actions, got %d", len(capturedParams.ActionsExclude))
	}
	if capturedParams.ActionsExclude[0] != "proxy.delete" {
		t.Errorf("Expected excluded action 'proxy.delete', got '%s'", capturedParams.ActionsExclude[0])
	}
	if len(capturedParams.ResourceTypesExclude) != 1 || capturedParams.ResourceTypesExclude[0] != "system" {
		t.Errorf("Expected excluded resource type 'system', got %v", capturedParams.ResourceTypesExclude)
	}
	if capturedParams.StatusExclude != "failure" {
		t.Errorf("Expected excluded status 'failure', got '%s'", capturedParams.StatusExclude)
	}
}

func TestAuditHandler_List_WithIPAddressFilters(t *testing.T) {
	testCases := []struct {
		name                 string
		query                string
		expectedIP           string
		expectedIPNot        string
		expectedIPContains   string
		expectedIPStartsWith string
		expectedIPEndsWith   string
	}{
		{
			name:       "exact IP match (default eq)",
			query:      "ip_address=192.168.1.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "exact IP match (explicit eq)",
			query:      "ip_address=eq:192.168.1.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "IP not equal",
			query:         "ip_address=not:10.0.0.1",
			expectedIPNot: "10.0.0.1",
		},
		{
			name:               "IP contains",
			query:              "ip_address=contains:168",
			expectedIPContains: "168",
		},
		{
			name:                 "IP starts with",
			query:                "ip_address=starts_with:192",
			expectedIPStartsWith: "192",
		},
		{
			name:               "IP ends with",
			query:              "ip_address=ends_with:.1",
			expectedIPEndsWith: ".1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedParams repository.AuditLogListParams
			mockService := &mocks.MockAuditService{
				ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
					capturedParams = params
					return &models.AuditLogListResponse{
						Items:      []models.AuditLog{},
						Total:      0,
						Page:       1,
						Limit:      20,
						TotalPages: 0,
					}, nil
				},
			}
			handler := NewAuditHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?"+tc.query, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
			}

			if tc.expectedIP != "" && capturedParams.IPAddress != tc.expectedIP {
				t.Errorf("Expected IPAddress '%s', got '%s'", tc.expectedIP, capturedParams.IPAddress)
			}
			if tc.expectedIPNot != "" && capturedParams.IPAddressNot != tc.expectedIPNot {
				t.Errorf("Expected IPAddressNot '%s', got '%s'", tc.expectedIPNot, capturedParams.IPAddressNot)
			}
			if tc.expectedIPContains != "" && capturedParams.IPAddressContains != tc.expectedIPContains {
				t.Errorf("Expected IPAddressContains '%s', got '%s'", tc.expectedIPContains, capturedParams.IPAddressContains)
			}
			if tc.expectedIPStartsWith != "" && capturedParams.IPAddressStartsWith != tc.expectedIPStartsWith {
				t.Errorf("Expected IPAddressStartsWith '%s', got '%s'", tc.expectedIPStartsWith, capturedParams.IPAddressStartsWith)
			}
			if tc.expectedIPEndsWith != "" && capturedParams.IPAddressEndsWith != tc.expectedIPEndsWith {
				t.Errorf("Expected IPAddressEndsWith '%s', got '%s'", tc.expectedIPEndsWith, capturedParams.IPAddressEndsWith)
			}
		})
	}
}

func TestAuditHandler_List_WithMultipleResourceTypes(t *testing.T) {
	var capturedParams repository.AuditLogListParams
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			capturedParams = params
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?resource_type=proxy,user,settings", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(capturedParams.ResourceTypes) != 3 {
		t.Errorf("Expected 3 resource types, got %d", len(capturedParams.ResourceTypes))
	}
	expectedTypes := []string{"proxy", "user", "settings"}
	for i, rt := range expectedTypes {
		if capturedParams.ResourceTypes[i] != rt {
			t.Errorf("Expected resource_type[%d] '%s', got '%s'", i, rt, capturedParams.ResourceTypes[i])
		}
	}
}

func TestAuditHandler_List_InvalidStatusNot(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	// New format: status=not:invalid
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?status=not:invalid", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuditHandler_List_InvalidPage(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?page=-1", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuditHandler_List_InvalidLimit(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=200", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuditHandler_List_InvalidStatus(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?status=invalid", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuditHandler_List_ServiceError(t *testing.T) {
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestAuditHandler_GetByID_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetAuditLogByIDFunc: func(id int) (*models.AuditLog, error) {
			return &models.AuditLog{
				ID:        1,
				Action:    "proxy.create",
				Status:    "success",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

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
}

func TestAuditHandler_GetByID_NotFound(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetAuditLogByIDFunc: func(id int) (*models.AuditLog, error) {
			return nil, errors.New("not found")
		},
	}
	handler := NewAuditHandler(mockService)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestAuditHandler_GetByID_InvalidID(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/invalid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestAuditHandler_GetStats_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetStatsFunc: func() (*models.AuditLogStats, error) {
			return &models.AuditLogStats{
				TotalLogs: 100,
				ByAction: map[string]int64{
					"proxy.create": 30,
					"proxy.update": 20,
					"auth.login":   50,
				},
				ByStatus: map[string]int64{
					"success": 90,
					"failure": 10,
				},
				ByResourceType: map[string]int64{
					"proxy": 50,
					"user":  50,
				},
				RecentActivity: []models.AuditLog{},
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

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
	if data["total_logs"].(float64) != 100 {
		t.Errorf("Expected total_logs 100, got %v", data["total_logs"])
	}
}

func TestAuditHandler_GetStats_ServiceError(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetStatsFunc: func() (*models.AuditLogStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// GetConfig Tests
// =============================================================================

func TestAuditHandler_GetConfig_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetConfigFunc: func() (*models.AuditConfig, error) {
			return &models.AuditConfig{
				ProxyCreate:        true,
				ProxyUpdate:        true,
				ProxyDelete:        true,
				ProxyEnable:        true,
				ProxyDisable:       true,
				AuthLogin:          true,
				AuthLogout:         true,
				AuthRegister:       true,
				AuthPasswordChange: true,
				AuthLoginFailed:    true,
				SettingsUpdate:     true,
				SyncStarted:        true,
				SyncCompleted:      true,
				SyncFailed:         true,
				SystemStartup:      true,
				CaddyReload:        true,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/config", nil)
	rec := httptest.NewRecorder()

	handler.GetConfig(rec, req)

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
	if !data["proxy_create"].(bool) {
		t.Error("Expected proxy_create to be true")
	}
}

func TestAuditHandler_GetConfig_ServiceError(t *testing.T) {
	mockService := &mocks.MockAuditService{
		GetConfigFunc: func() (*models.AuditConfig, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/config", nil)
	rec := httptest.NewRecorder()

	handler.GetConfig(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// UpdateConfig Tests
// =============================================================================

func TestAuditHandler_UpdateConfig_Success(t *testing.T) {
	var capturedConfig *models.AuditConfig
	mockService := &mocks.MockAuditService{
		SetConfigFunc: func(config *models.AuditConfig) error {
			capturedConfig = config
			return nil
		},
	}
	handler := NewAuditHandler(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"proxy_create":         true,
		"proxy_update":         true,
		"proxy_delete":         true,
		"proxy_enable":         true,
		"proxy_disable":        true,
		"auth_login":           false,
		"auth_logout":          false,
		"auth_register":        false,
		"auth_password_change": false,
		"auth_login_failed":    false,
		"settings_update":      true,
		"sync_started":         false,
		"sync_completed":       false,
		"sync_failed":          false,
		"system_startup":       true,
		"caddy_reload":         true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/audit-logs/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if capturedConfig.ProxyCreate != true {
		t.Error("Expected ProxyCreate to be true")
	}
	if capturedConfig.AuthLogin != false {
		t.Error("Expected AuthLogin to be false")
	}
	if capturedConfig.SettingsUpdate != true {
		t.Error("Expected SettingsUpdate to be true")
	}
	if capturedConfig.SyncStarted != false {
		t.Error("Expected SyncStarted to be false")
	}
	if capturedConfig.SystemStartup != true {
		t.Error("Expected SystemStartup to be true")
	}
}

func TestAuditHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/api/audit-logs/config", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAuditHandler_UpdateConfig_ServiceError(t *testing.T) {
	mockService := &mocks.MockAuditService{
		SetConfigFunc: func(config *models.AuditConfig) error {
			return errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService)

	body, _ := json.Marshal(map[string]interface{}{
		"proxy_create":         true,
		"proxy_update":         true,
		"proxy_delete":         true,
		"proxy_enable":         true,
		"proxy_disable":        true,
		"auth_login":           true,
		"auth_logout":          true,
		"auth_register":        true,
		"auth_password_change": true,
		"auth_login_failed":    true,
		"settings_update":      true,
		"sync_started":         true,
		"sync_completed":       true,
		"sync_failed":          true,
		"system_startup":       true,
		"caddy_reload":         true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/audit-logs/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateConfig(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// Export Tests
// =============================================================================

func TestAuditHandler_Export_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			resourceType := "proxy"
			resourceID := 1
			resourceName := "Test Proxy"
			userID := 1
			ipAddress := "192.168.1.1"
			userAgent := "Mozilla/5.0"

			return &models.AuditLogListResponse{
				Items: []models.AuditLog{
					{
						ID:           1,
						Action:       "proxy.create",
						Status:       "success",
						ResourceType: &resourceType,
						ResourceID:   &resourceID,
						ResourceName: &resourceName,
						UserID:       &userID,
						IPAddress:    &ipAddress,
						UserAgent:    &userAgent,
						CreatedAt:    time.Now(),
					},
				},
				Total:      1,
				Page:       1,
				Limit:      10000,
				TotalPages: 1,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/export", nil)
	rec := httptest.NewRecorder()

	handler.Export(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("Expected Content-Disposition header to be set")
	}

	// Check CSV contains header and data
	body := rec.Body.String()
	if body == "" {
		t.Error("Expected non-empty CSV body")
	}
}

func TestAuditHandler_Export_ServiceError(t *testing.T) {
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/export", nil)
	rec := httptest.NewRecorder()

	handler.Export(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// parseAuditListParams Tests
// =============================================================================

func TestParseAuditListParams_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)

	params, err := parseAuditListParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if params.Page != 1 {
		t.Errorf("Expected default page 1, got %d", params.Page)
	}
	if params.Limit != 20 {
		t.Errorf("Expected default limit 20, got %d", params.Limit)
	}
}

func TestParseAuditListParams_DateFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=2024-01-01&date_to=2024-12-31", nil)

	params, err := parseAuditListParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if params.DateFrom == nil {
		t.Error("Expected DateFrom to be set")
	}
	if params.DateTo == nil {
		t.Error("Expected DateTo to be set")
	}
}

func TestParseAuditListParams_DateFilters_RFC3339(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=2024-01-01T00:00:00Z&date_to=2024-12-31T23:59:59Z", nil)

	params, err := parseAuditListParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if params.DateFrom == nil {
		t.Error("Expected DateFrom to be set")
	}
	if params.DateTo == nil {
		t.Error("Expected DateTo to be set")
	}
}

func TestParseAuditListParams_InvalidDateFrom(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=invalid", nil)

	_, err := parseAuditListParams(req)
	if err == nil {
		t.Error("Expected error for invalid date_from")
	}
}

func TestParseAuditListParams_InvalidDateTo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_to=invalid", nil)

	_, err := parseAuditListParams(req)
	if err == nil {
		t.Error("Expected error for invalid date_to")
	}
}

func TestParseAuditListParams_UserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?user_id=5", nil)

	params, err := parseAuditListParams(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if params.UserID == nil || *params.UserID != 5 {
		t.Error("Expected UserID to be 5")
	}
}

func TestParseAuditListParams_InvalidUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?user_id=invalid", nil)

	_, err := parseAuditListParams(req)
	if err == nil {
		t.Error("Expected error for invalid user_id")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestStringOrEmpty(t *testing.T) {
	// Test nil
	result := stringOrEmpty(nil)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test non-nil
	str := "test"
	result = stringOrEmpty(&str)
	if result != "test" {
		t.Errorf("Expected 'test', got '%s'", result)
	}
}

func TestIntPtrToString(t *testing.T) {
	// Test nil
	result := intPtrToString(nil)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test non-nil
	num := 123
	result = intPtrToString(&num)
	if result != "123" {
		t.Errorf("Expected '123', got '%s'", result)
	}
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestAuditHandler_ResponseFormat(t *testing.T) {
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["success"]; !ok {
		t.Error("Expected 'success' field in response")
	}
	if _, ok := response["message"]; !ok {
		t.Error("Expected 'message' field in response")
	}
	if _, ok := response["data"]; !ok {
		t.Error("Expected 'data' field in response")
	}
}

// =============================================================================
// splitAndTrim Tests
// =============================================================================

func TestSplitAndTrim(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single value",
			input:    "proxy.create",
			expected: []string{"proxy.create"},
		},
		{
			name:     "multiple values",
			input:    "proxy.create,proxy.update,proxy.delete",
			expected: []string{"proxy.create", "proxy.update", "proxy.delete"},
		},
		{
			name:     "values with spaces",
			input:    "proxy.create, proxy.update , proxy.delete",
			expected: []string{"proxy.create", "proxy.update", "proxy.delete"},
		},
		{
			name:     "empty values filtered",
			input:    "proxy.create,,proxy.delete",
			expected: []string{"proxy.create", "proxy.delete"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only",
			input:    "   ,   ,   ",
			expected: []string{},
		},
		{
			name:     "single value with spaces",
			input:    "  proxy.create  ",
			expected: []string{"proxy.create"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitAndTrim(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d values, got %d", len(tc.expected), len(result))
				return
			}

			for i, v := range tc.expected {
				if result[i] != v {
					t.Errorf("Expected value[%d] '%s', got '%s'", i, v, result[i])
				}
			}
		})
	}
}

// =============================================================================
// GetEventGroups Tests
// =============================================================================

func TestAuditHandler_GetEventGroups_Success(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/event-groups", nil)
	rec := httptest.NewRecorder()

	handler.GetEventGroups(rec, req)

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

	data := response["data"].([]interface{})
	if len(data) == 0 {
		t.Error("Expected non-empty event groups")
	}

	// Verify structure of first group
	firstGroup := data[0].(map[string]interface{})
	if _, ok := firstGroup["key"]; !ok {
		t.Error("Expected 'key' field in event group")
	}
	if _, ok := firstGroup["label"]; !ok {
		t.Error("Expected 'label' field in event group")
	}
	if _, ok := firstGroup["description"]; !ok {
		t.Error("Expected 'description' field in event group")
	}
	if _, ok := firstGroup["events"]; !ok {
		t.Error("Expected 'events' field in event group")
	}
}

// =============================================================================
// parseFilterParam Tests
// =============================================================================

func TestParseFilterParam(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectedOperator string
		expectedValue    string
		expectedValues   []string
	}{
		{
			name:             "simple value defaults to eq",
			input:            "success",
			expectedOperator: "eq",
			expectedValue:    "success",
		},
		{
			name:             "explicit eq operator",
			input:            "eq:success",
			expectedOperator: "eq",
			expectedValue:    "success",
		},
		{
			name:             "not operator",
			input:            "not:failure",
			expectedOperator: "not",
			expectedValue:    "failure",
		},
		{
			name:             "contains operator",
			input:            "contains:168",
			expectedOperator: "contains",
			expectedValue:    "168",
		},
		{
			name:             "starts_with operator",
			input:            "starts_with:192",
			expectedOperator: "starts_with",
			expectedValue:    "192",
		},
		{
			name:             "ends_with operator",
			input:            "ends_with:.121",
			expectedOperator: "ends_with",
			expectedValue:    ".121",
		},
		{
			name:             "in operator with multiple values",
			input:            "in:proxy.create,proxy.update,proxy.delete",
			expectedOperator: "in",
			expectedValue:    "proxy.create,proxy.update,proxy.delete",
			expectedValues:   []string{"proxy.create", "proxy.update", "proxy.delete"},
		},
		{
			name:             "not_in operator with multiple values",
			input:            "not_in:auth.login,auth.logout",
			expectedOperator: "not_in",
			expectedValue:    "auth.login,auth.logout",
			expectedValues:   []string{"auth.login", "auth.logout"},
		},
		{
			name:             "URL with colon (not a valid operator) defaults to eq",
			input:            "http://example.com",
			expectedOperator: "eq",
			expectedValue:    "http://example.com",
		},
		{
			name:             "value with colon after unknown word defaults to eq",
			input:            "unknown:value",
			expectedOperator: "eq",
			expectedValue:    "unknown:value",
		},
		{
			name:             "empty string",
			input:            "",
			expectedOperator: "",
			expectedValue:    "",
		},
		{
			name:             "value containing colon after valid operator",
			input:            "contains:192:168",
			expectedOperator: "contains",
			expectedValue:    "192:168",
		},
		{
			name:             "IPv6 address defaults to eq",
			input:            "2001:db8::1",
			expectedOperator: "eq",
			expectedValue:    "2001:db8::1",
		},
		{
			name:             "contains with IPv6-like pattern",
			input:            "contains:db8:",
			expectedOperator: "contains",
			expectedValue:    "db8:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFilterParam(tc.input)

			if result.Operator != tc.expectedOperator {
				t.Errorf("Expected operator '%s', got '%s'", tc.expectedOperator, result.Operator)
			}
			if result.Value != tc.expectedValue {
				t.Errorf("Expected value '%s', got '%s'", tc.expectedValue, result.Value)
			}
			if tc.expectedValues != nil {
				if len(result.Values) != len(tc.expectedValues) {
					t.Errorf("Expected %d values, got %d", len(tc.expectedValues), len(result.Values))
				} else {
					for i, v := range tc.expectedValues {
						if result.Values[i] != v {
							t.Errorf("Expected values[%d] '%s', got '%s'", i, v, result.Values[i])
						}
					}
				}
			}
		})
	}
}

func TestParseFilterParam_InvalidOperatorForField(t *testing.T) {
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService)

	testCases := []struct {
		name  string
		query string
	}{
		{
			name:  "invalid operator for action",
			query: "action=contains:proxy",
		},
		{
			name:  "invalid operator for status",
			query: "status=in:success,failure",
		},
		{
			name:  "invalid operator for ip_address",
			query: "ip_address=in:192.168.1.1,10.0.0.1",
		},
		{
			name:  "user_id with non-eq operator",
			query: "user_id=not:5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?"+tc.query, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// =============================================================================
// Combined Filter Tests
// =============================================================================

func TestAuditHandler_List_CombinedFilters(t *testing.T) {
	var capturedParams repository.AuditLogListParams
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(params repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			capturedParams = params
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService)

	// Test combining multiple filter types with new format
	req := httptest.NewRequest(http.MethodGet,
		"/api/audit-logs?action=in:proxy.create,proxy.update&resource_type=proxy&status=success&ip_address=contains:192&search=test",
		nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify all filters are captured
	if len(capturedParams.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(capturedParams.Actions))
	}
	if len(capturedParams.ResourceTypes) != 1 {
		t.Errorf("Expected 1 resource type, got %d", len(capturedParams.ResourceTypes))
	}
	if capturedParams.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", capturedParams.Status)
	}
	if capturedParams.IPAddressContains != "192" {
		t.Errorf("Expected IP contains '192', got '%s'", capturedParams.IPAddressContains)
	}
	if capturedParams.Search != "test" {
		t.Errorf("Expected search 'test', got '%s'", capturedParams.Search)
	}
}
