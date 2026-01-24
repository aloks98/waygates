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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewAuditHandler Tests
// =============================================================================

func TestNewAuditHandler(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockService, handler.auditService, "audit service should be set")
}

// =============================================================================
// List Tests
// =============================================================================

func TestAuditHandler_List_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items []models.AuditLog `json:"items"`
			Total int64             `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.Len(t, response.Data.Items, 2, "expected 2 items")
}

func TestAuditHandler_List_WithFilters(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?page=2&limit=10&action=proxy.create&status=success&search=test", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, 2, capturedParams.Page)
	assert.Equal(t, 10, capturedParams.Limit)
	require.Len(t, capturedParams.Actions, 1)
	assert.Equal(t, "proxy.create", capturedParams.Actions[0])
	assert.Equal(t, "success", capturedParams.Status)
	assert.Equal(t, "test", capturedParams.Search)
}

func TestAuditHandler_List_WithMultipleActions(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?action=proxy.create,proxy.update,proxy.delete", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, capturedParams.Actions, 3)
	expectedActions := []string{"proxy.create", "proxy.update", "proxy.delete"}
	for i, action := range expectedActions {
		assert.Equal(t, action, capturedParams.Actions[i])
	}
}

func TestAuditHandler_List_WithExclusionFilters(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	// New format: field=operator:value
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?action=not_in:proxy.delete,auth.logout&resource_type=not:system&status=not:failure", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, capturedParams.ActionsExclude, 2)
	assert.Equal(t, "proxy.delete", capturedParams.ActionsExclude[0])
	require.Len(t, capturedParams.ResourceTypesExclude, 1)
	assert.Equal(t, "system", capturedParams.ResourceTypesExclude[0])
	assert.Equal(t, "failure", capturedParams.StatusExclude)
}

func TestAuditHandler_List_WithIPAddressFilters(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
			handler := NewAuditHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?"+tc.query, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)

			if tc.expectedIP != "" {
				assert.Equal(t, tc.expectedIP, capturedParams.IPAddress)
			}
			if tc.expectedIPNot != "" {
				assert.Equal(t, tc.expectedIPNot, capturedParams.IPAddressNot)
			}
			if tc.expectedIPContains != "" {
				assert.Equal(t, tc.expectedIPContains, capturedParams.IPAddressContains)
			}
			if tc.expectedIPStartsWith != "" {
				assert.Equal(t, tc.expectedIPStartsWith, capturedParams.IPAddressStartsWith)
			}
			if tc.expectedIPEndsWith != "" {
				assert.Equal(t, tc.expectedIPEndsWith, capturedParams.IPAddressEndsWith)
			}
		})
	}
}

func TestAuditHandler_List_WithMultipleResourceTypes(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?resource_type=proxy,user,settings", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, capturedParams.ResourceTypes, 3)
	expectedTypes := []string{"proxy", "user", "settings"}
	for i, rt := range expectedTypes {
		assert.Equal(t, rt, capturedParams.ResourceTypes[i])
	}
}

func TestAuditHandler_List_InvalidStatusNot(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	// New format: status=not:invalid
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?status=not:invalid", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_InvalidPage(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?page=-1", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_InvalidLimit(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?limit=200", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_InvalidStatus(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?status=invalid", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_List_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestAuditHandler_GetByID_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		GetAuditLogByIDFunc: func(_ int) (*models.AuditLog, error) {
			return &models.AuditLog{
				ID:        1,
				Action:    "proxy.create",
				Status:    "success",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	handler := NewAuditHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
}

func TestAuditHandler_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		GetAuditLogByIDFunc: func(_ int) (*models.AuditLog, error) {
			return nil, errors.New("not found")
		},
	}
	handler := NewAuditHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAuditHandler_GetByID_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/audit-logs/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/invalid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestAuditHandler_GetStats_Success(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TotalLogs int64 `json:"total_logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.Equal(t, int64(100), response.Data.TotalLogs)
}

func TestAuditHandler_GetStats_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		GetStatsFunc: func() (*models.AuditLogStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// GetConfig Tests
// =============================================================================

func TestAuditHandler_GetConfig_Success(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/config", nil)
	rec := httptest.NewRecorder()

	handler.GetConfig(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ProxyCreate bool `json:"proxy_create"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.True(t, response.Data.ProxyCreate, "expected proxy_create to be true")
}

func TestAuditHandler_GetConfig_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		GetConfigFunc: func() (*models.AuditConfig, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/config", nil)
	rec := httptest.NewRecorder()

	handler.GetConfig(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// UpdateConfig Tests
// =============================================================================

func TestAuditHandler_UpdateConfig_Success(t *testing.T) {
	t.Parallel()
	var capturedConfig *models.AuditConfig
	mockService := &mocks.MockAuditService{
		SetConfigFunc: func(config *models.AuditConfig) error {
			capturedConfig = config
			return nil
		},
	}
	handler := NewAuditHandler(mockService, nil)

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

	require.Equal(t, http.StatusOK, rec.Code)

	require.NotNil(t, capturedConfig)
	assert.True(t, capturedConfig.ProxyCreate)
	assert.False(t, capturedConfig.AuthLogin)
	assert.True(t, capturedConfig.SettingsUpdate)
	assert.False(t, capturedConfig.SyncStarted)
	assert.True(t, capturedConfig.SystemStartup)
}

func TestAuditHandler_UpdateConfig_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPut, "/api/audit-logs/config", bytes.NewBufferString("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateConfig(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAuditHandler_UpdateConfig_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		SetConfigFunc: func(_ *models.AuditConfig) error {
			return errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService, nil)

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

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Export Tests
// =============================================================================

func TestAuditHandler_Export_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
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
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/export", nil)
	rec := httptest.NewRecorder()

	handler.Export(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
	assert.NotEmpty(t, rec.Header().Get("Content-Disposition"))

	// Check CSV contains header and data
	assert.NotEmpty(t, rec.Body.String(), "expected non-empty CSV body")
}

func TestAuditHandler_Export_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/export", nil)
	rec := httptest.NewRecorder()

	handler.Export(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// parseAuditListParams Tests
// =============================================================================

func TestParseAuditListParams_Defaults(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)

	params, err := parseAuditListParams(req)
	require.NoError(t, err)

	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 20, params.Limit)
}

func TestParseAuditListParams_DateFilters(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=2024-01-01&date_to=2024-12-31", nil)

	params, err := parseAuditListParams(req)
	require.NoError(t, err)

	assert.NotNil(t, params.DateFrom, "expected DateFrom to be set")
	assert.NotNil(t, params.DateTo, "expected DateTo to be set")
}

func TestParseAuditListParams_DateFilters_RFC3339(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=2024-01-01T00:00:00Z&date_to=2024-12-31T23:59:59Z", nil)

	params, err := parseAuditListParams(req)
	require.NoError(t, err)

	assert.NotNil(t, params.DateFrom, "expected DateFrom to be set")
	assert.NotNil(t, params.DateTo, "expected DateTo to be set")
}

func TestParseAuditListParams_InvalidDateFrom(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_from=invalid", nil)

	_, err := parseAuditListParams(req)
	assert.Error(t, err, "expected error for invalid date_from")
}

func TestParseAuditListParams_InvalidDateTo(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?date_to=invalid", nil)

	_, err := parseAuditListParams(req)
	assert.Error(t, err, "expected error for invalid date_to")
}

func TestParseAuditListParams_UserID(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?user_id=5", nil)

	params, err := parseAuditListParams(req)
	require.NoError(t, err)

	require.NotNil(t, params.UserID)
	assert.Equal(t, 5, *params.UserID)
}

func TestParseAuditListParams_InvalidUserID(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?user_id=invalid", nil)

	_, err := parseAuditListParams(req)
	assert.Error(t, err, "expected error for invalid user_id")
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestStringOrEmpty(t *testing.T) {
	t.Parallel()
	// Test nil
	result := stringOrEmpty(nil)
	assert.Equal(t, "", result)

	// Test non-nil
	str := "test"
	result = stringOrEmpty(&str)
	assert.Equal(t, "test", result)
}

func TestIntPtrToString(t *testing.T) {
	t.Parallel()
	// Test nil
	result := intPtrToString(nil)
	assert.Equal(t, "", result)

	// Test non-nil
	num := 123
	result = intPtrToString(&num)
	assert.Equal(t, "123", result)
}

// =============================================================================
// Response Format Tests
// =============================================================================

func TestAuditHandler_ResponseFormat(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{
		ListAuditLogsFunc: func(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
			return &models.AuditLogListResponse{
				Items:      []models.AuditLog{},
				Total:      0,
				Page:       1,
				Limit:      20,
				TotalPages: 0,
			}, nil
		},
	}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	var response struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected 'success' field")
	assert.NotEmpty(t, response.Message, "expected 'message' field")
	assert.NotNil(t, response.Data, "expected 'data' field")
}

// =============================================================================
// splitAndTrim Tests
// =============================================================================

func TestSplitAndTrim(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := splitAndTrim(tc.input)

			require.Len(t, result, len(tc.expected))

			for i, v := range tc.expected {
				assert.Equal(t, v, result[i])
			}
		})
	}
}

// =============================================================================
// GetEventGroups Tests
// =============================================================================

func TestAuditHandler_GetEventGroups_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/audit-logs/event-groups", nil)
	rec := httptest.NewRecorder()

	handler.GetEventGroups(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			Key         string `json:"key"`
			Label       string `json:"label"`
			Description string `json:"description"`
			Events      []struct {
				Key   string `json:"key"`
				Label string `json:"label"`
			} `json:"events"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), "failed to parse response")

	assert.True(t, response.Success, "expected success to be true")
	assert.NotEmpty(t, response.Data, "expected non-empty event groups")

	// Verify structure of first group
	firstGroup := response.Data[0]
	assert.NotEmpty(t, firstGroup.Key, "expected 'key' field in event group")
	assert.NotEmpty(t, firstGroup.Label, "expected 'label' field in event group")
}

// =============================================================================
// parseFilterParam Tests
// =============================================================================

func TestParseFilterParam(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := parseFilterParam(tc.input)

			assert.Equal(t, tc.expectedOperator, result.Operator)
			assert.Equal(t, tc.expectedValue, result.Value)
			if tc.expectedValues != nil {
				require.Len(t, result.Values, len(tc.expectedValues))
				for i, v := range tc.expectedValues {
					assert.Equal(t, v, result.Values[i])
				}
			}
		})
	}
}

func TestParseFilterParam_InvalidOperatorForField(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockAuditService{}
	handler := NewAuditHandler(mockService, nil)

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
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/audit-logs?"+tc.query, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

// =============================================================================
// Combined Filter Tests
// =============================================================================

func TestAuditHandler_List_CombinedFilters(t *testing.T) {
	t.Parallel()
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
	handler := NewAuditHandler(mockService, nil)

	// Test combining multiple filter types with new format
	req := httptest.NewRequest(http.MethodGet,
		"/api/audit-logs?action=in:proxy.create,proxy.update&resource_type=proxy&status=success&ip_address=contains:192&search=test",
		nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify all filters are captured
	assert.Len(t, capturedParams.Actions, 2)
	assert.Len(t, capturedParams.ResourceTypes, 1)
	assert.Equal(t, "success", capturedParams.Status)
	assert.Equal(t, "192", capturedParams.IPAddressContains)
	assert.Equal(t, "test", capturedParams.Search)
}
