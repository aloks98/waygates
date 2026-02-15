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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// NewL4ProxyHandler Tests
// =============================================================================

func TestNewL4ProxyHandler(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	require.NotNil(t, handler, "handler should be created")
	assert.Equal(t, mockService, handler.service, "service should be set")
}

// =============================================================================
// List L4 Proxies Tests
// =============================================================================

func TestL4ProxyHandler_List_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(_ *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			return &models.L4ProxyListResponse{
				Items: []models.L4Proxy{
					{ID: 1, Name: "TCP Proxy", ListenPort: 8080, Protocol: "tcp", IsActive: true},
					{ID: 2, Name: "UDP Proxy", ListenPort: 9090, Protocol: "udp", IsActive: false},
				},
				Total:      2,
				Page:       1,
				Limit:      20,
				TotalPages: 1,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items      []models.L4Proxy `json:"items"`
			Total      int64            `json:"total"`
			Page       int              `json:"page"`
			Limit      int              `json:"limit"`
			TotalPages int              `json:"total_pages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data.Items, 2)
	assert.Equal(t, int64(2), resp.Data.Total)
	assert.Equal(t, "TCP Proxy", resp.Data.Items[0].Name)
}

func TestL4ProxyHandler_List_WithPagination(t *testing.T) {
	t.Parallel()
	var capturedReq *service.ListL4ProxiesRequest
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(req *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			capturedReq = req
			return &models.L4ProxyListResponse{Items: []models.L4Proxy{}, Total: 0}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies?page=2&limit=10", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, 2, capturedReq.Page)
	assert.Equal(t, 10, capturedReq.Limit)
}

func TestL4ProxyHandler_List_WithFilters(t *testing.T) {
	t.Parallel()
	var capturedReq *service.ListL4ProxiesRequest
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(req *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			capturedReq = req
			return &models.L4ProxyListResponse{Items: []models.L4Proxy{}, Total: 0}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies?protocol=tcp&is_active=true&search=test&sort=name&order=desc", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, capturedReq)
	assert.Equal(t, "tcp", capturedReq.Protocol)
	require.NotNil(t, capturedReq.IsActive)
	assert.True(t, *capturedReq.IsActive)
	assert.Equal(t, "test", capturedReq.Search)
	assert.Equal(t, "name", capturedReq.Sort)
	assert.Equal(t, "desc", capturedReq.Order)
}

func TestL4ProxyHandler_List_ValidationErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		queryParams string
		wantStatus  int
	}{
		{"Negative page", "page=-1", http.StatusBadRequest},
		{"Non-numeric page", "page=abc", http.StatusBadRequest},
		{"Negative limit", "limit=-1", http.StatusBadRequest},
		{"Too large limit", "limit=101", http.StatusBadRequest},
		{"Non-numeric limit", "limit=abc", http.StatusBadRequest},
		{"Invalid protocol", "protocol=invalid", http.StatusBadRequest},
		{"Invalid is_active", "is_active=maybe", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockService := &mocks.MockL4ProxyService{}
			handler := NewL4ProxyHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies?"+tc.queryParams, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestL4ProxyHandler_List_IsActiveFalseFilter(t *testing.T) {
	t.Parallel()
	var capturedReq *service.ListL4ProxiesRequest
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(req *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			capturedReq = req
			return &models.L4ProxyListResponse{Items: []models.L4Proxy{}, Total: 0}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies?is_active=false", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, capturedReq)
	require.NotNil(t, capturedReq.IsActive)
	assert.False(t, *capturedReq.IsActive)
}

func TestL4ProxyHandler_List_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(_ *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Get L4 Proxy Tests
// =============================================================================

func TestL4ProxyHandler_Get_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{
				ID:         1,
				Name:       "Test L4 Proxy",
				ListenPort: 8080,
				Protocol:   "tcp",
				IsActive:   true,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/l4-proxies/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			ListenPort int    `json:"listen_port"`
			Protocol   string `json:"protocol"`
			IsActive   bool   `json:"is_active"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Data.ID)
	assert.Equal(t, "Test L4 Proxy", resp.Data.Name)
	assert.Equal(t, 8080, resp.Data.ListenPort)
	assert.Equal(t, "tcp", resp.Data.Protocol)
}

func TestL4ProxyHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/l4-proxies/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Get_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return nil, service.ErrL4ProxyNotFound
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/l4-proxies/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestL4ProxyHandler_Get_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/l4-proxies/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Create L4 Proxy Tests
// =============================================================================

func TestL4ProxyHandler_Create_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(req *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{
				ID:         1,
				Name:       req.Name,
				ListenPort: req.ListenPort,
				Protocol:   req.Protocol,
				IsActive:   req.IsActive,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "New L4 Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
		"routes": []map[string]interface{}{
			{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams": []map[string]interface{}{
					{"host": "192.168.1.100", "port": 8080},
				},
			},
		},
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			ListenPort int    `json:"listen_port"`
			Protocol   string `json:"protocol"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Data.ID)
	assert.Equal(t, "New L4 Proxy", resp.Data.Name)
}

func TestL4ProxyHandler_Create_DefaultIsActive(t *testing.T) {
	t.Parallel()
	var capturedReq *service.CreateL4ProxyRequest
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(req *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			capturedReq = req
			return &models.L4Proxy{ID: 1, Name: req.Name, IsActive: req.IsActive}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, capturedReq)
	assert.True(t, capturedReq.IsActive, "IsActive should default to true")
}

func TestL4ProxyHandler_Create_MissingUserID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestL4ProxyHandler_Create_InvalidUserID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "not-a-number")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestL4ProxyHandler_Create_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", []byte(`{invalid json}`), "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Create_ValidationError_MissingName(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Create_ValidationError_InvalidPort(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 70000, // Invalid port
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Create_ValidationError_InvalidProtocol(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "invalid",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Create_PortConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(_ *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			return nil, service.ErrL4ProxyPortConflict
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestL4ProxyHandler_Create_ServiceValidationError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(_ *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			return nil, errors.New("validation: invalid configuration")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Update L4 Proxy Tests
// =============================================================================

func TestL4ProxyHandler_Update_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		UpdateFunc: func(id int, req *service.UpdateL4ProxyRequest) (*models.L4Proxy, error) {
			return &models.L4Proxy{
				ID:         id,
				Name:       req.Name,
				ListenPort: req.ListenPort,
				Protocol:   req.Protocol,
				IsActive:   req.IsActive,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Updated L4 Proxy",
		"listen_port": 9090,
		"protocol":    "udp",
		"is_active":   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			ListenPort int    `json:"listen_port"`
			Protocol   string `json:"protocol"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "Updated L4 Proxy", resp.Data.Name)
	assert.Equal(t, 9090, resp.Data.ListenPort)
}

func TestL4ProxyHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test",
		"listen_port": 8080,
		"protocol":    "tcp",
		"is_active":   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Update_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/1", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Update_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		UpdateFunc: func(_ int, _ *service.UpdateL4ProxyRequest) (*models.L4Proxy, error) {
			return nil, service.ErrL4ProxyNotFound
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test",
		"listen_port": 8080,
		"protocol":    "tcp",
		"is_active":   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestL4ProxyHandler_Update_PortConflict(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		UpdateFunc: func(_ int, _ *service.UpdateL4ProxyRequest) (*models.L4Proxy, error) {
			return nil, service.ErrL4ProxyPortConflict
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test",
		"listen_port": 8080,
		"protocol":    "tcp",
		"is_active":   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestL4ProxyHandler_Update_ValidationError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	// Missing required fields
	body, _ := json.Marshal(map[string]interface{}{
		"name": "Test",
		// missing listen_port and protocol
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Update_ServiceValidationError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		UpdateFunc: func(_ int, _ *service.UpdateL4ProxyRequest) (*models.L4Proxy, error) {
			return nil, errors.New("validation: invalid configuration")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Put("/api/l4-proxies/{id}", handler.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test",
		"listen_port": 8080,
		"protocol":    "tcp",
		"is_active":   true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/l4-proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// Delete L4 Proxy Tests
// =============================================================================

func TestL4ProxyHandler_Delete_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		DeleteFunc: func(_ int) error {
			return nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Delete("/api/l4-proxies/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/l4-proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestL4ProxyHandler_Delete_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Delete("/api/l4-proxies/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/l4-proxies/abc", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_Delete_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		DeleteFunc: func(_ int) error {
			return service.ErrL4ProxyNotFound
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Delete("/api/l4-proxies/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/l4-proxies/999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestL4ProxyHandler_Delete_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		DeleteFunc: func(_ int) error {
			return errors.New("database error")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Delete("/api/l4-proxies/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/l4-proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Toggle Active Tests
// =============================================================================

func TestL4ProxyHandler_ToggleActive_Success_Enable(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ToggleActiveFunc: func(_ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{
				ID:       1,
				Name:     "Test Proxy",
				IsActive: true,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Patch("/api/l4-proxies/{id}/toggle", handler.ToggleActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/l4-proxies/1/toggle", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID       int  `json:"id"`
			IsActive bool `json:"is_active"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.True(t, resp.Data.IsActive)
	assert.Contains(t, resp.Message, "enabled")
}

func TestL4ProxyHandler_ToggleActive_Success_Disable(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ToggleActiveFunc: func(_ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{
				ID:       1,
				Name:     "Test Proxy",
				IsActive: false,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Patch("/api/l4-proxies/{id}/toggle", handler.ToggleActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/l4-proxies/1/toggle", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID       int  `json:"id"`
			IsActive bool `json:"is_active"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.False(t, resp.Data.IsActive)
	assert.Contains(t, resp.Message, "disabled")
}

func TestL4ProxyHandler_ToggleActive_InvalidID(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Patch("/api/l4-proxies/{id}/toggle", handler.ToggleActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/l4-proxies/abc/toggle", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestL4ProxyHandler_ToggleActive_NotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ToggleActiveFunc: func(_ int) (*models.L4Proxy, error) {
			return nil, service.ErrL4ProxyNotFound
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Patch("/api/l4-proxies/{id}/toggle", handler.ToggleActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/l4-proxies/999/toggle", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestL4ProxyHandler_ToggleActive_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		ToggleActiveFunc: func(_ int) (*models.L4Proxy, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Patch("/api/l4-proxies/{id}/toggle", handler.ToggleActive)

	req := httptest.NewRequest(http.MethodPatch, "/api/l4-proxies/1/toggle", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Get Stats Tests
// =============================================================================

func TestL4ProxyHandler_GetStats_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		GetStatsFunc: func() (*models.L4ProxyStats, error) {
			return &models.L4ProxyStats{
				TotalProxies:   10,
				ActiveProxies:  8,
				TCPProxies:     6,
				UDPProxies:     4,
				TotalRoutes:    15,
				TotalUpstreams: 30,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalProxies   int64 `json:"total_proxies"`
			ActiveProxies  int64 `json:"active_proxies"`
			TCPProxies     int64 `json:"tcp_proxies"`
			UDPProxies     int64 `json:"udp_proxies"`
			TotalRoutes    int64 `json:"total_routes"`
			TotalUpstreams int64 `json:"total_upstreams"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, int64(10), resp.Data.TotalProxies)
	assert.Equal(t, int64(8), resp.Data.ActiveProxies)
	assert.Equal(t, int64(6), resp.Data.TCPProxies)
	assert.Equal(t, int64(4), resp.Data.UDPProxies)
	assert.Equal(t, int64(15), resp.Data.TotalRoutes)
	assert.Equal(t, int64(30), resp.Data.TotalUpstreams)
}

func TestL4ProxyHandler_GetStats_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockL4ProxyService{
		GetStatsFunc: func() (*models.L4ProxyStats, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/stats", nil)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// Create L4 Proxy with Routes Tests
// =============================================================================

func TestL4ProxyHandler_Create_WithRoutes(t *testing.T) {
	t.Parallel()
	var capturedReq *service.CreateL4ProxyRequest
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(req *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			capturedReq = req
			return &models.L4Proxy{
				ID:         1,
				Name:       req.Name,
				ListenPort: req.ListenPort,
				Protocol:   req.Protocol,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Proxy with Routes",
		"listen_port": 8443,
		"protocol":    "tcp",
		"routes": []map[string]interface{}{
			{
				"priority":              10,
				"matcher_type":          "tls",
				"sni_hostnames":         []string{"example.com", "*.example.com"},
				"load_balancing_policy": "round_robin",
				"tls_passthrough":       true,
				"upstreams": []map[string]interface{}{
					{"host": "192.168.1.100", "port": 443, "weight": 50},
					{"host": "192.168.1.101", "port": 443, "weight": 50},
				},
			},
		},
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, capturedReq)
	assert.Len(t, capturedReq.Routes, 1)
	assert.Equal(t, "tls", capturedReq.Routes[0].MatcherType)
	assert.Equal(t, "round_robin", capturedReq.Routes[0].LoadBalancingPolicy)
	assert.Len(t, capturedReq.Routes[0].Upstreams, 2)
}

// =============================================================================
// DTO Conversion Tests
// =============================================================================

func TestDtoToCreateL4ProxyRequest_Conversion(t *testing.T) {
	t.Parallel()
	var capturedReq *service.CreateL4ProxyRequest
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(req *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			capturedReq = req
			return &models.L4Proxy{ID: 1}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	description := "Test description"
	proxyProtocol := "v2"

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Conversion Test",
		"description": description,
		"listen_port": 5432,
		"protocol":    "tcp",
		"is_active":   false,
		"routes": []map[string]interface{}{
			{
				"priority":               5,
				"matcher_type":           "postgres",
				"allowed_ip_ranges":      []string{"10.0.0.0/8"},
				"load_balancing_policy":  "least_conn",
				"proxy_protocol_version": proxyProtocol,
				"upstreams": []map[string]interface{}{
					{"host": "db.internal", "port": 5432},
				},
			},
		},
	})
	req := requestWithUserID(http.MethodPost, "/api/l4-proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, capturedReq)

	assert.Equal(t, "Conversion Test", capturedReq.Name)
	assert.Equal(t, &description, capturedReq.Description)
	assert.Equal(t, 5432, capturedReq.ListenPort)
	assert.Equal(t, "tcp", capturedReq.Protocol)
	assert.False(t, capturedReq.IsActive)

	require.Len(t, capturedReq.Routes, 1)
	route := capturedReq.Routes[0]
	assert.Equal(t, 5, route.Priority)
	assert.Equal(t, "postgres", route.MatcherType)
	assert.Equal(t, []string{"10.0.0.0/8"}, route.AllowedIPRanges)
	assert.Equal(t, "least_conn", route.LoadBalancingPolicy)
	assert.Equal(t, &proxyProtocol, route.ProxyProtocolVersion)

	require.Len(t, route.Upstreams, 1)
	assert.Equal(t, "db.internal", route.Upstreams[0].Host)
	assert.Equal(t, 5432, route.Upstreams[0].Port)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkL4ProxyHandler_List(b *testing.B) {
	mockService := &mocks.MockL4ProxyService{
		ListFunc: func(_ *service.ListL4ProxiesRequest) (*models.L4ProxyListResponse, error) {
			return &models.L4ProxyListResponse{
				Items: []models.L4Proxy{
					{ID: 1, Name: "TCP Proxy", ListenPort: 8080, Protocol: "tcp"},
					{ID: 2, Name: "UDP Proxy", ListenPort: 9090, Protocol: "udp"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies", nil)
		rec := httptest.NewRecorder()
		handler.List(rec, req)
	}
}

func BenchmarkL4ProxyHandler_Get(b *testing.B) {
	mockService := &mocks.MockL4ProxyService{
		GetByIDFunc: func(_ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{ID: 1, Name: "Test Proxy", ListenPort: 8080, Protocol: "tcp"}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	r := chi.NewRouter()
	r.Get("/api/l4-proxies/{id}", handler.Get)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/1", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkL4ProxyHandler_Create(b *testing.B) {
	mockService := &mocks.MockL4ProxyService{
		CreateFunc: func(req *service.CreateL4ProxyRequest, _ int) (*models.L4Proxy, error) {
			return &models.L4Proxy{ID: 1, Name: req.Name}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Test Proxy",
		"listen_port": 8080,
		"protocol":    "tcp",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/l4-proxies", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.SetUserID(req.Context(), "123")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.Create(rec, req)
	}
}

func BenchmarkL4ProxyHandler_GetStats(b *testing.B) {
	mockService := &mocks.MockL4ProxyService{
		GetStatsFunc: func() (*models.L4ProxyStats, error) {
			return &models.L4ProxyStats{
				TotalProxies:   100,
				ActiveProxies:  80,
				TCPProxies:     60,
				UDPProxies:     40,
				TotalRoutes:    150,
				TotalUpstreams: 300,
			}, nil
		},
	}
	handler := NewL4ProxyHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/l4-proxies/stats", nil)
		rec := httptest.NewRecorder()
		handler.GetStats(rec, req)
	}
}
