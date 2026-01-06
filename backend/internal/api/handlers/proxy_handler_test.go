package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/goauth/middleware"

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
	handler := NewProxyHandler(mockService, nil)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockService, handler.service, "service should be set")
}

// =============================================================================
// ListProxies Tests
// =============================================================================

func TestListProxies_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{
				Items: []models.Proxy{
					{ID: 1, Name: "Proxy 1", Hostname: "proxy1.example.com"},
					{ID: 2, Name: "Proxy 2", Hostname: "proxy2.example.com"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockService := &mocks.MockProxyService{}
			handler := NewProxyHandler(mockService, nil)

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
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

	req := requestWithUserID(http.MethodPost, "/api/proxies", []byte(`{invalid json}`), "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateProxy_HostnameConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			return service.ErrHostnameConflict
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			return service.NewCaddyError("caddy validation failed")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			return errors.New("validation: hostname is required")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return service.ErrHostnameConflict
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return service.NewCaddyError("caddy reload failed")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return errors.New("validation: invalid hostname format")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		DeleteProxyFunc: func(id int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		DeleteProxyFunc: func(id int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		DeleteProxyFunc: func(id int) error {
			return errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		EnableProxyFunc: func(id int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		EnableProxyFunc: func(id int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		EnableProxyFunc: func(id int) error {
			return service.ErrProxyAlreadyEnabled
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		EnableProxyFunc: func(id int) error {
			return service.NewCaddyError("caddy error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		EnableProxyFunc: func(id int) error {
			return errors.New("unknown error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		DisableProxyFunc: func(id int) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		DisableProxyFunc: func(id int) error {
			return service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		DisableProxyFunc: func(id int) error {
			return service.ErrProxyAlreadyDisabled
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		DisableProxyFunc: func(id int) error {
			return errors.New("unknown error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
			handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=contains:reverse_proxy", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_InvalidStatusOperator(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			assert.True(t, proxy.SSLEnabled, "SSLEnabled should be preserved from existing proxy")
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			proxy.ID = 1
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyCreateFunc: func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
		DeleteProxyFunc: func(id int) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyDeleteFunc: func(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			assert.Equal(t, "Test Proxy", proxyName)
			assert.Equal(t, "test.example.com", hostname)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
		DeleteProxyFunc: func(id int) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		EnableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyEnableFunc: func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		EnableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		DisableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyDisableFunc: func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
			auditCalled = true
			assert.Equal(t, 123, userID)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		DisableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, errors.New("proxy not found for audit")
		},
	}
	mockAuditService := &mocks.MockAuditService{}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
		DeleteProxyFunc: func(id int) error {
			return nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyDeleteFunc: func(ctx context.Context, userID int, proxyID int, proxyName, hostname string, ip, userAgent string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		EnableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy"}, nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyEnableFunc: func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

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
		DisableProxyFunc: func(id int) error {
			return nil
		},
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy"}, nil
		},
	}
	auditCalled := false
	mockAuditService := &mocks.MockAuditService{
		LogProxyDisableFunc: func(ctx context.Context, userID int, proxy *models.Proxy, ip, userAgent string) error {
			auditCalled = true
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, auditCalled, "audit should not be called when userID is 0")
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkListProxies(b *testing.B) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{
				Items: []models.Proxy{
					{ID: 1, Name: "Proxy 1", Hostname: "proxy1.example.com"},
					{ID: 2, Name: "Proxy 2", Hostname: "proxy2.example.com"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
		rec := httptest.NewRecorder()
		handler.ListProxies(rec, req)
	}
}

func BenchmarkListProxies_WithFilters(b *testing.B) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=1&limit=10&type=in:reverse_proxy,redirect&status=active&ssl_enabled=true", nil)
		rec := httptest.NewRecorder()
		handler.ListProxies(rec, req)
	}
}

func BenchmarkGetProxy(b *testing.B) {
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test Proxy", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

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
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Test", Hostname: "test.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			createCalled = true
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

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
	handler := NewProxyHandler(mockService, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	// The handler should still work even with canceled context
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusInternalServerError)
}
