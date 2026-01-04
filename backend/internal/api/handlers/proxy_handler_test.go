package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aloks98/goauth/middleware"
	"github.com/go-chi/chi/v5"

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=2&limit=10&search=test&type=reverse&status=active&sort=name&order=desc", nil)
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
	if capturedReq.Type != "reverse" {
		t.Errorf("Expected type 'reverse', got '%s'", capturedReq.Type)
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
			handler := NewProxyHandler(mockService)

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
			handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateProxy_InvalidID(t *testing.T) {
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "existing.example.com"})
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
	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
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
	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "invalid"})
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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

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
	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
