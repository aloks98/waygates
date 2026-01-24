package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/goauth/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// Helper to create a request with user ID in context
func requestWithUserID(method, url string, body []byte, userID string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		ctx := middleware.SetUserID(req.Context(), userID)
		req = req.WithContext(ctx)
	}
	return req
}

// =============================================================================
// NewProxyHandler Tests
// =============================================================================

func TestNewProxyHandler(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockService, handler.service, "service should be set")
}

// =============================================================================
// ListProxies Tests
// =============================================================================

func TestListProxies_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(_ service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{
				Items: []models.Proxy{
					{ID: 1, Name: "Proxy 1", Hostname: "proxy1.example.com"},
					{ID: 2, Name: "Proxy 2", Hostname: "proxy2.example.com"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify response body
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items []models.Proxy `json:"items"`
			Total int64          `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, int64(2), resp.Data.Total)
	assert.Equal(t, "Proxy 1", resp.Data.Items[0].Name)
}

func TestListProxies_WithQueryParams(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=2&limit=10&search=test&type=reverse_proxy&status=active&sort=name&order=desc", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, capturedReq.Page)
	assert.Equal(t, 10, capturedReq.Limit)
	assert.Equal(t, "test", capturedReq.Search)
	assert.Equal(t, []string{"reverse_proxy"}, capturedReq.Types)
	assert.Equal(t, "active", capturedReq.Status)
}

// TestListProxies_ValidationErrors consolidates all validation error tests
func TestListProxies_ValidationErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		queryParams string
		wantStatus  int
	}{
		// Page validation
		{"Negative page", "page=-1", http.StatusBadRequest},
		{"Non-numeric page", "page=abc", http.StatusBadRequest},
		{"Float page", "page=1.5", http.StatusBadRequest},
		// Limit validation
		{"Negative limit", "limit=-1", http.StatusBadRequest},
		{"Too large limit", "limit=101", http.StatusBadRequest},
		{"Non-numeric limit", "limit=abc", http.StatusBadRequest},
		// Status validation
		{"Invalid status", "status=invalid", http.StatusBadRequest},
		// SSL validation
		{"Invalid ssl_enabled", "ssl_enabled=maybe", http.StatusBadRequest},
		// Type operator validation
		{"Invalid type operator", "type=contains:reverse_proxy", http.StatusBadRequest},
		// Status operator validation
		{"Invalid status operator", "status=in:active,inactive", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockService := &mocks.MockProxyService{}
			handler := NewProxyHandler(mockService, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/proxies?"+tc.queryParams, nil)
			rec := httptest.NewRecorder()

			handler.ListProxies(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestListProxies_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(_ service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// GetProxy Tests
// =============================================================================

func TestGetProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify response body
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Hostname string `json:"hostname"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Data.ID)
	assert.Equal(t, "Test Proxy", resp.Data.Name)
}

func TestGetProxy_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetProxy_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetProxy_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// CreateProxy Tests
// =============================================================================

func TestCreateProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	// Verify response body
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Data.ID)
}

func TestCreateProxy_MissingUserID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateProxy_InvalidUserID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "not-a-number")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateProxy_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	req := requestWithUserID(http.MethodPost, "/api/proxies", []byte(`{invalid json}`), "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateProxy_HostnameConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(_ *models.Proxy, _ int) error {
			return service.ErrHostnameConflict
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "existing.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateProxy_CaddyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(_ *models.Proxy, _ int) error {
			return service.NewCaddyError("caddy validation failed")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestCreateProxy_ValidationError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(_ *models.Proxy, _ int) error {
			return errors.New("validation: hostname is required")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Test Proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// UpdateProxy Tests
// =============================================================================

func TestUpdateProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Old Name", Hostname: "old.example.com", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Updated Proxy",
		"hostname":    "updated.example.com",
		"ssl_enabled": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify response body
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestUpdateProxy_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateProxy_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateProxy_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateProxy_HostnameConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "old.example.com", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return service.ErrHostnameConflict
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "existing.example.com", "ssl_enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestUpdateProxy_CaddyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return service.NewCaddyError("caddy reload failed")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test", "ssl_enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestUpdateProxy_ValidationError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return errors.New("validation: invalid hostname format")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "invalid", "ssl_enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// DeleteProxy Tests
// =============================================================================

func TestDeleteProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DeleteProxyFunc: func(_ int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteProxy_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteProxy_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DeleteProxyFunc: func(_ int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteProxy_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DeleteProxyFunc: func(_ int) error {
			return errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// EnableProxy Tests
// =============================================================================

func TestEnableProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestEnableProxy_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnableProxy_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/999/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestEnableProxy_AlreadyEnabled(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return service.ErrProxyAlreadyEnabled
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnableProxy_CaddyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return service.NewCaddyError("caddy error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestEnableProxy_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return errors.New("unknown error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// DisableProxy Tests
// =============================================================================

func TestDisableProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDisableProxy_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDisableProxy_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/999/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDisableProxy_AlreadyDisabled(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return service.ErrProxyAlreadyDisabled
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDisableProxy_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return errors.New("unknown error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestGetStats_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return &repository.ProxyStats{
				Total:    10,
				Active:   8,
				Inactive: 2,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify response body
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Total    int64 `json:"total"`
			Active   int64 `json:"active"`
			Inactive int64 `json:"inactive"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, int64(10), resp.Data.Total)
	assert.Equal(t, int64(8), resp.Data.Active)
	assert.Equal(t, int64(2), resp.Data.Inactive)
}

func TestGetStats_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// ListProxies Advanced Filter Tests
// =============================================================================

func TestListProxies_TypeFilterWithInOperator(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=in:reverse_proxy,redirect", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, capturedReq.Types, 2)
	assert.Equal(t, "reverse_proxy", capturedReq.Types[0])
	assert.Equal(t, "redirect", capturedReq.Types[1])
}

func TestListProxies_TypeFilterWithNotInOperator(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=not_in:static", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, capturedReq.TypesExclude, 1)
	assert.Equal(t, "static", capturedReq.TypesExclude[0])
}

func TestListProxies_StatusFilterWithNotOperator(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?status=not:inactive", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "inactive", capturedReq.StatusNot)
}

func TestListProxies_SSLEnabledFilter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		param          string
		expectedStatus int
		expectedValue  *bool
	}{
		{
			name:           "SSL enabled true",
			param:          "true",
			expectedStatus: http.StatusOK,
			expectedValue:  boolPtr(true),
		},
		{
			name:           "SSL enabled false",
			param:          "false",
			expectedStatus: http.StatusOK,
			expectedValue:  boolPtr(false),
		},
		{
			name:           "Invalid value",
			param:          "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedValue:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedReq service.ListProxiesRequest
			mockService := &mocks.MockProxyService{
				ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
					capturedReq = req
					return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
				},
			}
			handler := NewProxyHandler(mockService, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/proxies?ssl_enabled="+tc.param, nil)
			rec := httptest.NewRecorder()

			handler.ListProxies(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)
			if tc.expectedStatus == http.StatusOK {
				if tc.expectedValue == nil {
					assert.Nil(t, capturedReq.SSLEnabled)
				} else {
					require.NotNil(t, capturedReq.SSLEnabled)
					assert.Equal(t, *tc.expectedValue, *capturedReq.SSLEnabled)
				}
			}
		})
	}
}

func TestListProxies_TargetFilter(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?target=localhost:8080", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "localhost:8080", capturedReq.Target)
}

func TestListProxies_CombinedFilters(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=in:reverse_proxy,redirect&status=active&ssl_enabled=true&target=backend", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, capturedReq.Types, 2)
	assert.Equal(t, "active", capturedReq.Status)
	require.NotNil(t, capturedReq.SSLEnabled)
	assert.True(t, *capturedReq.SSLEnabled)
	assert.Equal(t, "backend", capturedReq.Target)
}

// Helper function for bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// getClientIP Tests
// =============================================================================

func TestGetClientIP_XForwardedFor(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")

	ip := getClientIP(req)

	assert.Equal(t, "192.168.1.100", ip)
}

func TestGetClientIP_XForwardedForSingle(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.50")

	ip := getClientIP(req)

	assert.Equal(t, "192.168.1.50", ip)
}

func TestGetClientIP_XRealIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.50")

	ip := getClientIP(req)

	assert.Equal(t, "10.0.0.50", ip)
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"

	ip := getClientIP(req)

	assert.Equal(t, "127.0.0.1", ip)
}

func TestGetClientIP_RemoteAddrWithoutPort(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1"

	ip := getClientIP(req)

	assert.Equal(t, "127.0.0.1", ip)
}

func TestGetClientIP_XForwardedForPriority(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.RemoteAddr = "127.0.0.1:8080"

	ip := getClientIP(req)

	assert.Equal(t, "192.168.1.1", ip, "X-Forwarded-For should take priority")
}

// =============================================================================
// ListProxies Invalid Operator Tests
// =============================================================================

func TestListProxies_InvalidTypeOperator(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=contains:reverse_proxy", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_InvalidStatusOperator(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?status=in:active,inactive", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_TypeNotOperator(t *testing.T) {
	t.Parallel()
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=not:static", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, capturedReq.TypesExclude, 1)
	assert.Equal(t, "static", capturedReq.TypesExclude[0])
}

// =============================================================================
// UpdateProxy SSL Handling Tests
// =============================================================================

func TestUpdateProxy_WithoutSSLEnabled_FetchExisting(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, proxy *models.Proxy) error {
			assert.True(t, proxy.SSLEnabled, "SSLEnabled should be preserved from existing proxy")
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Updated Proxy",
		"hostname": "updated.example.com",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateProxy_WithoutSSLEnabled_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Updated Proxy",
		"hostname": "updated.example.com",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateProxy_WithoutSSLEnabled_GetError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Updated Proxy",
		"hostname": "updated.example.com",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// CreateProxy SSLEnabled Default Tests
// =============================================================================

func TestCreateProxy_SSLEnabledExplicitlyFalse(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"hostname":    "test.example.com",
		"type":        "reverse_proxy",
		"ssl_enabled": false,
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.False(t, capturedProxy.SSLEnabled, "SSLEnabled should be false when explicitly set")
}

func TestCreateProxy_SSLEnabledDefault(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, capturedProxy.SSLEnabled, "SSLEnabled should default to true")
}

// =============================================================================
// Audit Logging Tests
// =============================================================================

func TestCreateProxy_WithAuditService(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			proxy.ID = 1
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyCreateFunc: func(_ context.Context, userID int, _ *models.Proxy, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, auditCalled, "audit service should be called")
}

func TestUpdateProxy_WithAuditService(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Old Name", Hostname: "old.example.com", SSLEnabled: false}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(_ context.Context, userID int, _ *models.Proxy, changes map[string]interface{}, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			// Verify changes were captured
			assert.NotNil(t, changes, "changes map should not be nil when fields changed")
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Updated Proxy",
		"hostname":    "updated.example.com",
		"ssl_enabled": true,
	})
	req := requestWithUserID(http.MethodPut, "/api/proxies/1", body, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled, "audit service should be called")
}

func TestDeleteProxy_WithAuditService(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
		DeleteProxyFunc: func(_ int) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyDeleteFunc: func(_ context.Context, userID int, _ int, proxyName, hostname string, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			assert.Equal(t, "Test Proxy", proxyName)
			assert.Equal(t, "test.example.com", hostname)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := requestWithUserID(http.MethodDelete, "/api/proxies/1", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled, "audit service should be called")
}

func TestDeleteProxy_WithAuditService_GetProxyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
		DeleteProxyFunc: func(_ int) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := requestWithUserID(http.MethodDelete, "/api/proxies/1", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "delete should succeed even if audit lookup fails")
}

func TestEnableProxy_WithAuditService(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyEnableFunc: func(_ context.Context, userID int, _ *models.Proxy, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/enable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled, "audit service should be called")
}

func TestEnableProxy_WithAuditService_GetProxyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/enable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "enable should succeed even if audit lookup fails")
}

func TestDisableProxy_WithAuditService(t *testing.T) {
	t.Parallel()
	auditCalled := false
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyDisableFunc: func(_ context.Context, userID int, _ *models.Proxy, _, _ string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/disable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, auditCalled, "audit service should be called")
}

func TestDisableProxy_WithAuditService_GetProxyError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/disable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "disable should succeed even if audit lookup fails")
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestUpdateProxy_WithoutUserID_NoAudit(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(_ context.Context, _ int, _ *models.Proxy, _ map[string]interface{}, _, _ string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Updated Proxy",
		"hostname":    "updated.example.com",
		"ssl_enabled": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditCalled, "audit should not be called when userID is 0")
}

func TestDeleteProxy_WithoutUserID_NoAudit(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
		DeleteProxyFunc: func(_ int) error {
			return nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyDeleteFunc: func(_ context.Context, _ int, _ int, _, _ string, _, _ string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditCalled, "audit should not be called when userID is 0")
}

func TestEnableProxy_WithoutUserID_NoAudit(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy"}, nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyEnableFunc: func(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditCalled, "audit should not be called when userID is 0")
}

func TestDisableProxy_WithoutUserID_NoAudit(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(_ int) error {
			return nil
		},
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy"}, nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyDisableFunc: func(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditCalled, "audit should not be called when userID is 0")
}

// =============================================================================
// Change Tracking Tests
// =============================================================================

func TestBuildProxyChanges_NoChanges(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{
		ID:         1,
		Name:       "Test Proxy",
		Hostname:   "test.example.com",
		Type:       "reverse_proxy",
		SSLEnabled: true,
		IsActive:   true,
	}
	updated := &models.Proxy{
		ID:         1,
		Name:       "Test Proxy",
		Hostname:   "test.example.com",
		Type:       "reverse_proxy",
		SSLEnabled: true,
		IsActive:   true,
	}

	changes := buildProxyChanges(old, updated)

	assert.Nil(t, changes, "should return nil when no changes")
}

func TestBuildProxyChanges_HostnameChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{Hostname: "old.example.com"}
	updated := &models.Proxy{Hostname: "new.example.com"}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "hostname")
	hostnameChange := changes["hostname"].(map[string]interface{})
	assert.Equal(t, "old.example.com", hostnameChange["old"])
	assert.Equal(t, "new.example.com", hostnameChange["new"])
}

func TestBuildProxyChanges_NameChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{Name: "Old Name"}
	updated := &models.Proxy{Name: "New Name"}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "name")
	nameChange := changes["name"].(map[string]interface{})
	assert.Equal(t, "Old Name", nameChange["old"])
	assert.Equal(t, "New Name", nameChange["new"])
}

func TestBuildProxyChanges_TypeChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{Type: "reverse_proxy"}
	updated := &models.Proxy{Type: "redirect"}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "type")
	typeChange := changes["type"].(map[string]interface{})
	assert.Equal(t, "reverse_proxy", typeChange["old"])
	assert.Equal(t, "redirect", typeChange["new"])
}

func TestBuildProxyChanges_SSLEnabledChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{SSLEnabled: true}
	updated := &models.Proxy{SSLEnabled: false}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "ssl_enabled")
	sslChange := changes["ssl_enabled"].(map[string]interface{})
	assert.Equal(t, true, sslChange["old"])
	assert.Equal(t, false, sslChange["new"])
}

func TestBuildProxyChanges_IsActiveChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{IsActive: true}
	updated := &models.Proxy{IsActive: false}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "is_active")
	activeChange := changes["is_active"].(map[string]interface{})
	assert.Equal(t, true, activeChange["old"])
	assert.Equal(t, false, activeChange["new"])
}

func TestBuildProxyChanges_UpstreamsChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{Upstreams: []interface{}{"http://localhost:8080"}}
	updated := &models.Proxy{Upstreams: []interface{}{"http://localhost:9090"}}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "upstreams")
}

func TestBuildProxyChanges_RedirectChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{RedirectConfig: models.JSONField{"url": "https://old.example.com"}}
	updated := &models.Proxy{RedirectConfig: models.JSONField{"url": "https://new.example.com"}}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "redirect")
}

func TestBuildProxyChanges_MultipleChanges(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{
		Name:       "Old Name",
		Hostname:   "old.example.com",
		SSLEnabled: true,
	}
	updated := &models.Proxy{
		Name:       "New Name",
		Hostname:   "new.example.com",
		SSLEnabled: false,
	}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	assert.Len(t, changes, 3, "should have 3 changes")
	assert.Contains(t, changes, "name")
	assert.Contains(t, changes, "hostname")
	assert.Contains(t, changes, "ssl_enabled")
}

func TestUpdateProxy_WithAuditService_ChangesTracked(t *testing.T) {
	t.Parallel()
	var capturedChanges map[string]interface{}
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{
				ID:         1,
				Name:       "Old Name",
				Hostname:   "old.example.com",
				Type:       "reverse_proxy",
				SSLEnabled: false,
				IsActive:   true,
			}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(_ context.Context, _ int, _ *models.Proxy, changes map[string]interface{}, _, _ string) error {
			capturedChanges = changes
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "New Name",
		"hostname":    "new.example.com",
		"type":        "reverse_proxy",
		"ssl_enabled": true,
	})
	req := requestWithUserID(http.MethodPut, "/api/proxies/1", body, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, capturedChanges, "changes should be captured")

	// Verify specific changes
	assert.Contains(t, capturedChanges, "name")
	assert.Contains(t, capturedChanges, "hostname")
	assert.Contains(t, capturedChanges, "ssl_enabled")

	// Verify name change structure
	nameChange := capturedChanges["name"].(map[string]interface{})
	assert.Equal(t, "Old Name", nameChange["old"])
	assert.Equal(t, "New Name", nameChange["new"])

	// Verify hostname change structure
	hostnameChange := capturedChanges["hostname"].(map[string]interface{})
	assert.Equal(t, "old.example.com", hostnameChange["old"])
	assert.Equal(t, "new.example.com", hostnameChange["new"])

	// Verify ssl_enabled change structure
	sslChange := capturedChanges["ssl_enabled"].(map[string]interface{})
	assert.Equal(t, false, sslChange["old"])
	assert.Equal(t, true, sslChange["new"])
}

func TestUpdateProxy_WithAuditService_NoChanges(t *testing.T) {
	t.Parallel()
	var capturedChanges map[string]interface{}
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{
				ID:         1,
				Name:       "Same Name",
				Hostname:   "same.example.com",
				Type:       "reverse_proxy",
				SSLEnabled: true,
			}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(_ context.Context, _ int, _ *models.Proxy, changes map[string]interface{}, _, _ string) error {
			capturedChanges = changes
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Same Name",
		"hostname":    "same.example.com",
		"type":        "reverse_proxy",
		"ssl_enabled": true,
	})
	req := requestWithUserID(http.MethodPut, "/api/proxies/1", body, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, capturedChanges, "changes should be nil when nothing changed")
}

func TestJsonEqual_NilValues(t *testing.T) {
	t.Parallel()
	assert.True(t, jsonEqual(nil, nil), "nil == nil should be true")
	assert.False(t, jsonEqual(nil, "value"), "nil != value should be false")
	assert.False(t, jsonEqual("value", nil), "value != nil should be false")
}

func TestJsonEqual_SameValues(t *testing.T) {
	t.Parallel()
	assert.True(t, jsonEqual("test", "test"), "same strings should be equal")
	assert.True(t, jsonEqual(123, 123), "same numbers should be equal")
	assert.True(t, jsonEqual(true, true), "same booleans should be equal")
}

func TestJsonEqual_DifferentValues(t *testing.T) {
	t.Parallel()
	assert.False(t, jsonEqual("test1", "test2"), "different strings should not be equal")
	assert.False(t, jsonEqual(123, 456), "different numbers should not be equal")
	assert.False(t, jsonEqual(true, false), "different booleans should not be equal")
}

func TestJsonEqual_Slices(t *testing.T) {
	t.Parallel()
	slice1 := []interface{}{"a", "b"}
	slice2 := []interface{}{"a", "b"}
	slice3 := []interface{}{"a", "c"}

	assert.True(t, jsonEqual(slice1, slice2), "same slices should be equal")
	assert.False(t, jsonEqual(slice1, slice3), "different slices should not be equal")
}

func TestJsonEqual_Maps(t *testing.T) {
	t.Parallel()
	map1 := map[string]interface{}{"key": "value"}
	map2 := map[string]interface{}{"key": "value"}
	map3 := map[string]interface{}{"key": "different"}

	assert.True(t, jsonEqual(map1, map2), "same maps should be equal")
	assert.False(t, jsonEqual(map1, map3), "different maps should not be equal")
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkListProxies(b *testing.B) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(_ service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{
				Items: []models.Proxy{
					{ID: 1, Name: "Proxy 1", Hostname: "proxy1.example.com"},
					{ID: 2, Name: "Proxy 2", Hostname: "proxy2.example.com"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
		rec := httptest.NewRecorder()
		handler.ListProxies(rec, req)
	}
}

func BenchmarkListProxies_WithFilters(b *testing.B) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(_ service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=1&limit=10&type=in:reverse_proxy,redirect&status=active&ssl_enabled=true", nil)
		rec := httptest.NewRecorder()
		handler.ListProxies(rec, req)
	}
}

func BenchmarkGetProxy(b *testing.B) {
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkCreateProxy(b *testing.B) {
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
		rec := httptest.NewRecorder()
		handler.CreateProxy(rec, req)
	}
}

func BenchmarkGetStats(b *testing.B) {
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return &repository.ProxyStats{
				Total:    100,
				Active:   80,
				Inactive: 20,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
		rec := httptest.NewRecorder()
		handler.GetStats(rec, req)
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestListProxies_ContextCancellation(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(_ service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	// The handler should still work even with canceled context
	// as the mock doesn't check context - this tests that the handler doesn't panic
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusInternalServerError)
}

func TestGetProxy_ContextCancellation(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	// The handler should still work even with canceled context
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusInternalServerError)
}

func TestCreateProxy_ContextCancellation(t *testing.T) {
	t.Parallel()
	createCalled := false
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			createCalled = true
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/proxies", bytes.NewBuffer(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	ctx2 := middleware.SetUserID(req.Context(), "123")
	req = req.WithContext(ctx2)

	rec := httptest.NewRecorder()
	handler.CreateProxy(rec, req)

	// The handler should still process the request
	// This tests that the handler doesn't panic on canceled context
	require.True(t, createCalled || rec.Code != http.StatusCreated)
}

func TestGetStats_ContextCancellation(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return &repository.ProxyStats{Total: 10, Active: 8, Inactive: 2}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	// The handler should still work even with canceled context
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusInternalServerError)
}
