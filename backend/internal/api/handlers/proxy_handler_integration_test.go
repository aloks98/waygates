package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

func TestProxyHandler_ListProxies_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return &models.ProxyListResponse{
				Items: []models.Proxy{
					{ID: 1, Name: "Test Proxy", Hostname: "test.example.com", Type: models.ProxyTypeReverseProxy},
				},
				Total:      1,
				Page:       1,
				Limit:      20,
				TotalPages: 1,
			}, nil
		},
	}

	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

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

func TestProxyHandler_ListProxies_WithPagination(t *testing.T) {
	var capturedReq service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			capturedReq = req
			return &models.ProxyListResponse{Items: []models.Proxy{}, Total: 0}, nil
		},
	}

	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=2&limit=50&search=test&status=active", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedReq.Page != 2 {
		t.Errorf("Expected page 2, got %d", capturedReq.Page)
	}
	if capturedReq.Limit != 50 {
		t.Errorf("Expected limit 50, got %d", capturedReq.Limit)
	}
	if capturedReq.Search != "test" {
		t.Errorf("Expected search 'test', got '%s'", capturedReq.Search)
	}
	if capturedReq.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", capturedReq.Status)
	}
}

func TestProxyHandler_ListProxies_InvalidPage(t *testing.T) {
	handler := NewProxyHandler(&mocks.MockProxyService{})

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?page=-1", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_ListProxies_InvalidLimit(t *testing.T) {
	handler := NewProxyHandler(&mocks.MockProxyService{})

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?limit=101", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_ListProxies_InvalidStatus(t *testing.T) {
	handler := NewProxyHandler(&mocks.MockProxyService{})

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?status=invalid", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_ListProxies_ServiceError(t *testing.T) {
	mockService := &mocks.MockProxyService{
		ListProxiesFunc: func(req service.ListProxiesRequest) (*models.ProxyListResponse, error) {
			return nil, errors.New("database error")
		},
	}

	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	w := httptest.NewRecorder()

	handler.ListProxies(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestProxyHandler_GetProxy_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return &models.Proxy{
				ID:       id,
				Name:     "Test Proxy",
				Hostname: "test.example.com",
				Type:     models.ProxyTypeReverseProxy,
			}, nil
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_GetProxy_NotFound(t *testing.T) {
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(id int) (*models.Proxy, error) {
			return nil, service.ErrProxyNotFound
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestProxyHandler_GetProxy_InvalidID(t *testing.T) {
	handler := NewProxyHandler(&mocks.MockProxyService{})

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/invalid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_DeleteProxy_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		DeleteProxyFunc: func(id int) error {
			return nil
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_DeleteProxy_NotFound(t *testing.T) {
	mockService := &mocks.MockProxyService{
		DeleteProxyFunc: func(id int) error {
			return service.ErrProxyNotFound
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Delete("/api/proxies/{id}", handler.DeleteProxy)

	req := httptest.NewRequest(http.MethodDelete, "/api/proxies/999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestProxyHandler_EnableProxy_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(id int) error {
			return nil
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_EnableProxy_AlreadyEnabled(t *testing.T) {
	mockService := &mocks.MockProxyService{
		EnableProxyFunc: func(id int) error {
			return service.ErrProxyAlreadyEnabled
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/enable", handler.EnableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/enable", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_DisableProxy_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(id int) error {
			return nil
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_DisableProxy_AlreadyDisabled(t *testing.T) {
	mockService := &mocks.MockProxyService{
		DisableProxyFunc: func(id int) error {
			return service.ErrProxyAlreadyDisabled
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Post("/api/proxies/{id}/disable", handler.DisableProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/proxies/1/disable", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProxyHandler_GetStats_Success(t *testing.T) {
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
	w := httptest.NewRecorder()

	handler.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_GetStats_Error(t *testing.T) {
	mockService := &mocks.MockProxyService{
		GetStatsFunc: func() (*repository.ProxyStats, error) {
			return nil, errors.New("database error")
		},
	}

	handler := NewProxyHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil)
	w := httptest.NewRecorder()

	handler.GetStats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestProxyHandler_UpdateProxy_Success(t *testing.T) {
	mockService := &mocks.MockProxyService{
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return nil
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body := `{"name": "Updated Proxy", "hostname": "updated.example.com", "type": "reverse_proxy"}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxyHandler_UpdateProxy_NotFound(t *testing.T) {
	mockService := &mocks.MockProxyService{
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return service.ErrProxyNotFound
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body := `{"name": "Updated Proxy", "hostname": "updated.example.com", "type": "reverse_proxy"}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestProxyHandler_UpdateProxy_HostnameConflict(t *testing.T) {
	mockService := &mocks.MockProxyService{
		UpdateProxyFunc: func(id int, proxy *models.Proxy) error {
			return service.ErrHostnameConflict
		},
	}

	handler := NewProxyHandler(mockService)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body := `{"name": "Updated Proxy", "hostname": "existing.example.com", "type": "reverse_proxy"}`
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestProxyHandler_UpdateProxy_InvalidBody(t *testing.T) {
	handler := NewProxyHandler(&mocks.MockProxyService{})

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
