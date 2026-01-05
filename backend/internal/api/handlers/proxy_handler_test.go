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
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
	if handler.service != mockService {
		t.Error("Expected service to be set")
	}
}

// =============================================================================
// ListProxies Tests
// =============================================================================

func TestListProxies_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListProxies_WithQueryParams(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if capturedReq.Page != 2 {
		t.Errorf("Expected page 2, got %d", capturedReq.Page)
	}
	if capturedReq.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", capturedReq.Limit)
	}
	if capturedReq.Search != "test" {
		t.Errorf("Expected search 'test', got '%s'", capturedReq.Search)
	}
	if len(capturedReq.Types) != 1 || capturedReq.Types[0] != "reverse_proxy" {
		t.Errorf("Expected types ['reverse_proxy'], got %v", capturedReq.Types)
	}
	if capturedReq.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", capturedReq.Status)
	}
}

func TestListProxies_InvalidPage(t *testing.T) {
	testCases := []struct {
		name  string
		page  string
		valid bool
	}{
		{"Negative page", "-1", false},
		{"Non-numeric page", "abc", false},
		{"Float page", "1.5", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mocks.MockProxyService{}
			handler := NewProxyHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/proxies?page="+tc.page, nil)
			rec := httptest.NewRecorder()

			handler.ListProxies(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestListProxies_InvalidLimit(t *testing.T) {
	testCases := []struct {
		name  string
		limit string
	}{
		{"Negative limit", "-1"},
		{"Too large limit", "101"},
		{"Non-numeric limit", "abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mocks.MockProxyService{}
			handler := NewProxyHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/proxies?limit="+tc.limit, nil)
			rec := httptest.NewRecorder()

			handler.ListProxies(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestListProxies_InvalidStatus(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?status=invalid", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListProxies_ServiceError(t *testing.T) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// GetProxy Tests
// =============================================================================

func TestGetProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetProxy_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestGetProxy_ServiceError(t *testing.T) {
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// CreateProxy Tests
// =============================================================================

func TestCreateProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestCreateProxy_MissingUserID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateProxy_InvalidUserID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "not-a-number")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateProxy_InvalidJSON(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	req := requestWithUserID(http.MethodPost, "/api/proxies", []byte(`{invalid json}`), "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateProxy_HostnameConflict(t *testing.T) {
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

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestCreateProxy_CaddyError(t *testing.T) {
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

	if rec.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
}

func TestCreateProxy_ValidationError(t *testing.T) {
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

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// =============================================================================
// UpdateProxy Tests
// =============================================================================

func TestUpdateProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateProxy_InvalidJSON(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateProxy_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUpdateProxy_HostnameConflict(t *testing.T) {
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

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestUpdateProxy_CaddyError(t *testing.T) {
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

	if rec.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
}

func TestUpdateProxy_ValidationError(t *testing.T) {
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

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// =============================================================================
// DeleteProxy Tests
// =============================================================================

func TestDeleteProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDeleteProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestDeleteProxy_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDeleteProxy_ServiceError(t *testing.T) {
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// EnableProxy Tests
// =============================================================================

func TestEnableProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestEnableProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEnableProxy_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEnableProxy_AlreadyEnabled(t *testing.T) {
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

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEnableProxy_CaddyError(t *testing.T) {
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

	if rec.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
}

func TestEnableProxy_ServiceError(t *testing.T) {
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// DisableProxy Tests
// =============================================================================

func TestDisableProxy_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDisableProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/abc/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestDisableProxy_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDisableProxy_AlreadyDisabled(t *testing.T) {
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

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestDisableProxy_ServiceError(t *testing.T) {
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestGetStats_Success(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetStats_ServiceError(t *testing.T) {
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// ListProxies Advanced Filter Tests
// =============================================================================

func TestListProxies_TypeFilterWithInOperator(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if len(capturedReq.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(capturedReq.Types))
	}
	if capturedReq.Types[0] != "reverse_proxy" || capturedReq.Types[1] != "redirect" {
		t.Errorf("Expected types ['reverse_proxy', 'redirect'], got %v", capturedReq.Types)
	}
}

func TestListProxies_TypeFilterWithNotInOperator(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if len(capturedReq.TypesExclude) != 1 || capturedReq.TypesExclude[0] != "static" {
		t.Errorf("Expected types_exclude ['static'], got %v", capturedReq.TypesExclude)
	}
}

func TestListProxies_StatusFilterWithNotOperator(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if capturedReq.StatusNot != "inactive" {
		t.Errorf("Expected status_not 'inactive', got '%s'", capturedReq.StatusNot)
	}
}

func TestListProxies_SSLEnabledFilter(t *testing.T) {
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

			if rec.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
			if tc.expectedStatus == http.StatusOK {
				if tc.expectedValue == nil && capturedReq.SSLEnabled != nil {
					t.Errorf("Expected nil SSLEnabled, got %v", *capturedReq.SSLEnabled)
				}
				if tc.expectedValue != nil {
					if capturedReq.SSLEnabled == nil {
						t.Errorf("Expected SSLEnabled %v, got nil", *tc.expectedValue)
					} else if *capturedReq.SSLEnabled != *tc.expectedValue {
						t.Errorf("Expected SSLEnabled %v, got %v", *tc.expectedValue, *capturedReq.SSLEnabled)
					}
				}
			}
		})
	}
}

func TestListProxies_TargetFilter(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if capturedReq.Target != "localhost:8080" {
		t.Errorf("Expected target 'localhost:8080', got '%s'", capturedReq.Target)
	}
}

func TestListProxies_CombinedFilters(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if len(capturedReq.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(capturedReq.Types))
	}
	if capturedReq.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", capturedReq.Status)
	}
	if capturedReq.SSLEnabled == nil || *capturedReq.SSLEnabled != true {
		t.Errorf("Expected SSLEnabled true, got %v", capturedReq.SSLEnabled)
	}
	if capturedReq.Target != "backend" {
		t.Errorf("Expected target 'backend', got '%s'", capturedReq.Target)
	}
}

// Helper function for bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// getClientIP Tests
// =============================================================================

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")

	ip := getClientIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("Expected '192.168.1.100', got '%s'", ip)
	}
}

func TestGetClientIP_XForwardedForSingle(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.50")

	ip := getClientIP(req)

	if ip != "192.168.1.50" {
		t.Errorf("Expected '192.168.1.50', got '%s'", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.50")

	ip := getClientIP(req)

	if ip != "10.0.0.50" {
		t.Errorf("Expected '10.0.0.50', got '%s'", ip)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"

	ip := getClientIP(req)

	if ip != "127.0.0.1" {
		t.Errorf("Expected '127.0.0.1', got '%s'", ip)
	}
}

func TestGetClientIP_RemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1"

	ip := getClientIP(req)

	if ip != "127.0.0.1" {
		t.Errorf("Expected '127.0.0.1', got '%s'", ip)
	}
}

func TestGetClientIP_XForwardedForPriority(t *testing.T) {
	// X-Forwarded-For should take priority over X-Real-IP
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.RemoteAddr = "127.0.0.1:8080"

	ip := getClientIP(req)

	if ip != "192.168.1.1" {
		t.Errorf("Expected '192.168.1.1', got '%s'", ip)
	}
}

// =============================================================================
// ListProxies Invalid Operator Tests
// =============================================================================

func TestListProxies_InvalidTypeOperator(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=contains:reverse_proxy", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListProxies_InvalidStatusOperator(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?status=in:active,inactive", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListProxies_TypeNotOperator(t *testing.T) {
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if len(capturedReq.TypesExclude) != 1 || capturedReq.TypesExclude[0] != "static" {
		t.Errorf("Expected types_exclude ['static'], got %v", capturedReq.TypesExclude)
	}
}

// =============================================================================
// UpdateProxy SSL Handling Tests
// =============================================================================

func TestUpdateProxy_WithoutSSLEnabled_FetchExisting(t *testing.T) {
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", SSLEnabled: true}, nil
		},
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			// Verify ssl_enabled is preserved from existing proxy
			if !proxy.SSLEnabled {
				t.Error("Expected SSLEnabled to be preserved as true")
			}
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	// Don't include ssl_enabled in the request
	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Updated Proxy",
		"hostname": "updated.example.com",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateProxy_WithoutSSLEnabled_NotFound(t *testing.T) {
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

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUpdateProxy_WithoutSSLEnabled_GetError(t *testing.T) {
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// =============================================================================
// CreateProxy SSLEnabled Default Tests
// =============================================================================

func TestCreateProxy_SSLEnabledExplicitlyFalse(t *testing.T) {
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

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if capturedProxy.SSLEnabled {
		t.Error("Expected SSLEnabled to be false when explicitly set")
	}
}

func TestCreateProxy_SSLEnabledDefault(t *testing.T) {
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, userID int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil)

	// Don't include ssl_enabled in request
	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !capturedProxy.SSLEnabled {
		t.Error("Expected SSLEnabled to default to true when not specified")
	}
}

// =============================================================================
// Audit Logging Tests
// =============================================================================

func TestCreateProxy_WithAuditService(t *testing.T) {
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
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
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

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !auditCalled {
		t.Error("Expected audit service to be called")
	}
}

func TestUpdateProxy_WithAuditService(t *testing.T) {
	auditCalled := false
	mockService := &mocks.MockProxyService{
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return nil
		},
	}
	mockAuditService := &mocks.MockAuditService{
		LogProxyUpdateFunc: func(ctx context.Context, userID int, proxy *models.Proxy, changes map[string]interface{}, ip, userAgent string) error {
			auditCalled = true
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !auditCalled {
		t.Error("Expected audit service to be called")
	}
}

func TestDeleteProxy_WithAuditService(t *testing.T) {
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
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
			if proxyName != "Test Proxy" {
				t.Errorf("Expected proxyName 'Test Proxy', got '%s'", proxyName)
			}
			if hostname != "test.example.com" {
				t.Errorf("Expected hostname 'test.example.com', got '%s'", hostname)
			}
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := requestWithUserID(http.MethodDelete, "/api/proxies/1", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !auditCalled {
		t.Error("Expected audit service to be called")
	}
}

func TestDeleteProxy_WithAuditService_GetProxyError(t *testing.T) {
	// Even if GetProxyByID fails for audit, delete should still succeed
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestEnableProxy_WithAuditService(t *testing.T) {
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
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/enable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !auditCalled {
		t.Error("Expected audit service to be called")
	}
}

func TestEnableProxy_WithAuditService_GetProxyError(t *testing.T) {
	// Even if GetProxyByID fails for audit, enable should still succeed
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDisableProxy_WithAuditService(t *testing.T) {
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
			if userID != 123 {
				t.Errorf("Expected userID 123, got %d", userID)
			}
			return nil
		},
	}
	handler := NewProxyHandler(mockService, mockAuditService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := requestWithUserID(http.MethodPost, "/api/proxies/1/disable", nil, "123")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !auditCalled {
		t.Error("Expected audit service to be called")
	}
}

func TestDisableProxy_WithAuditService_GetProxyError(t *testing.T) {
	// Even if GetProxyByID fails for audit, disable should still succeed
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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestUpdateProxy_WithoutUserID_NoAudit(t *testing.T) {
	// When user ID is not valid (0), audit should not be called
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
	// No user ID in context - userID will be 0
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if auditCalled {
		t.Error("Expected audit service NOT to be called when userID is 0")
	}
}

func TestDeleteProxy_WithoutUserID_NoAudit(t *testing.T) {
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

	// No user ID in context
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if auditCalled {
		t.Error("Expected audit service NOT to be called when userID is 0")
	}
}

func TestEnableProxy_WithoutUserID_NoAudit(t *testing.T) {
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

	// No user ID in context
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if auditCalled {
		t.Error("Expected audit service NOT to be called when userID is 0")
	}
}

func TestDisableProxy_WithoutUserID_NoAudit(t *testing.T) {
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

	// No user ID in context
	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if auditCalled {
		t.Error("Expected audit service NOT to be called when userID is 0")
	}
}
