package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ptr returns a pointer to v. Handy for building *bool fixtures for the
// tri-state SSLEnabled/SSLForced/BlockExploits/TLSInsecureSkipVerify fields.
func ptr[T any](v T) *T { return &v }

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
			handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// =============================================================================
// GetProxy effective / _source Tests (Task 6)
// =============================================================================

type effectiveViewJSON struct {
	SSLEnabled            bool `json:"ssl_enabled"`
	SSLForced             bool `json:"ssl_forced"`
	BlockExploits         bool `json:"block_exploits"`
	TLSInsecureSkipVerify bool `json:"tls_insecure_skip_verify"`
	Source                struct {
		SSLEnabled            string `json:"ssl_enabled"`
		SSLForced             string `json:"ssl_forced"`
		BlockExploits         string `json:"block_exploits"`
		TLSInsecureSkipVerify string `json:"tls_insecure_skip_verify"`
	} `json:"_source"`
}

type proxyDetailJSON struct {
	Success bool `json:"success"`
	Data    struct {
		SSLEnabled *bool             `json:"ssl_enabled"`
		Effective  effectiveViewJSON `json:"effective"`
	} `json:"data"`
}

// An ungrouped proxy with every tri-state field nil resolves to the system
// defaults, and every _source entry is "default".
func TestGetProxy_EffectiveView_UngroupedDefaults(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "svc", Hostname: "svc.example.com"}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp proxyDetailJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Nil(t, resp.Data.SSLEnabled, "raw column stays nil so the edit form can show 'inherit'")
	assert.True(t, resp.Data.Effective.SSLEnabled, "system default is true")
	assert.True(t, resp.Data.Effective.SSLForced)
	assert.True(t, resp.Data.Effective.BlockExploits)
	assert.False(t, resp.Data.Effective.TLSInsecureSkipVerify, "system default is false")
	assert.Equal(t, "default", resp.Data.Effective.Source.SSLEnabled)
	assert.Equal(t, "default", resp.Data.Effective.Source.SSLForced)
	assert.Equal(t, "default", resp.Data.Effective.Source.BlockExploits)
	assert.Equal(t, "default", resp.Data.Effective.Source.TLSInsecureSkipVerify)
}

// A grouped proxy that leaves ssl_enabled nil inherits the group's value and
// reports _source "group"; a field it sets explicitly reports _source
// "proxy" and overrides the group's opinion — this is the exact divergence
// the resolver exists to prevent, made visible to the UI.
func TestGetProxy_EffectiveView_InheritsFromGroupAndCanOverride(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{
				ID: 1, Name: "svc", Hostname: "svc.acme.in",
				GroupID:       ptr(7),
				SSLEnabled:    nil,        // inherit
				BlockExploits: ptr(false), // explicit override
			}, nil
		},
	}
	mockGroupService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(id int) (*models.ProxyGroup, error) {
			require.Equal(t, 7, id)
			return &models.ProxyGroup{
				ID:            id,
				SSLEnabled:    ptr(false),
				BlockExploits: ptr(true),
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, mockGroupService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp proxyDetailJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.False(t, resp.Data.Effective.SSLEnabled, "inherited from the group's false")
	assert.Equal(t, "group", resp.Data.Effective.Source.SSLEnabled)
	assert.False(t, resp.Data.Effective.BlockExploits, "the proxy's explicit false wins over the group's true")
	assert.Equal(t, "proxy", resp.Data.Effective.Source.BlockExploits)
}

// A group lookup failure surfaces as 500, not a panic or a silently-wrong
// "ungrouped" response.
func TestGetProxy_EffectiveView_GroupLookupError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, GroupID: ptr(7)}, nil
		},
	}
	mockGroupService := &mocks.MockProxyGroupService{
		GetGroupByIDFunc: func(int) (*models.ProxyGroup, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, mockGroupService, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/proxies/{id}", handler.GetProxy)
	req := httptest.NewRequest(http.MethodGet, "/api/proxies/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestSourceOf pins sourceOf's proxy -> group -> default precedence directly,
// independent of the GetProxy handler tests above (which exercise it only
// indirectly via the JSON response). sourceOf re-encodes the exact
// proxy -> group -> default precedence proxygroup.Resolve's resolveBool
// implements (internal/proxygroup/resolve.go) so the UI can report
// provenance; if the two ever diverge, GetProxy would report a _source that
// doesn't match what was actually served.
func TestSourceOf(t *testing.T) {
	t.Parallel()

	proxyVal := ptr(true)
	groupVal := ptr(false)

	assert.Equal(t, "proxy", sourceOf(proxyVal, groupVal), "the proxy's own value wins over the group's")
	assert.Equal(t, "proxy", sourceOf(proxyVal, nil), "the proxy's own value wins even with no group present")
	assert.Equal(t, "group", sourceOf(nil, groupVal), "no proxy opinion falls through to the group's value")
	assert.Equal(t, "default", sourceOf(nil, nil), "neither proxy nor group set: falls through to the system default")
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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "Test Proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// A group_id referencing a non-existent proxy group must be reported as 404,
// not the generic 400 the catch-all would otherwise produce.
func TestCreateProxy_GroupNotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(_ *models.Proxy, _ int) error {
			return service.ErrGroupNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"group_id": 99,
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

// A genuine DB failure while loading the proxy's group must map to a 500 with
// a generic message — the raw underlying error text (which could contain
// connection details) must never be echoed back to the API caller.
func TestCreateProxy_GroupLookupFailed(t *testing.T) {
	t.Parallel()
	rawErr := errors.New("dial tcp 10.0.0.5:5432: connection refused")
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(_ *models.Proxy, _ int) error {
			return fmt.Errorf("%w (group %d): %w", service.ErrGroupLookupFailed, 5, rawErr)
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"group_id": 5,
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "connection refused", "raw DB error text must not be echoed to the client")
	assert.NotContains(t, rec.Body.String(), "10.0.0.5", "raw DB error text must not be echoed to the client")
}

// =============================================================================
// ImportProxies Tests
// =============================================================================

func TestImportProxies_DryRunMixedItems(t *testing.T) {
	t.Parallel()
	var capturedInputs []service.ImportInput
	var capturedDryRun bool
	mockService := &mocks.MockProxyService{
		ImportProxiesFunc: func(inputs []service.ImportInput, dryRun bool, _ int) service.ImportReport {
			capturedInputs = inputs
			capturedDryRun = dryRun
			// Mirror the real service classification for the two items.
			report := service.ImportReport{
				Summary: service.ImportSummary{Total: len(inputs), Importable: 1, Invalid: 1},
				Items:   make([]service.ImportItemResult, 0, len(inputs)),
			}
			for i, in := range inputs {
				if in.Proxy == nil {
					report.Items = append(report.Items, service.ImportItemResult{
						Index:  i,
						Status: service.ImportStatusInvalid,
						Reason: in.DecodeError,
					})
					continue
				}
				report.Items = append(report.Items, service.ImportItemResult{
					Index:    i,
					Name:     in.Proxy.Name,
					Hostname: in.Proxy.Hostname,
					Status:   service.ImportStatusValid,
				})
			}
			return report
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body := []byte(`{"dry_run":true,"proxies":[{"name":"Valid","hostname":"valid.example.com","type":"reverse"},"not-an-object"]}`)
	req := requestWithUserID(http.MethodPost, "/api/proxies/import", body, "123")
	rec := httptest.NewRecorder()

	handler.ImportProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, capturedDryRun, "dry_run should be passed through")
	require.Len(t, capturedInputs, 2)
	assert.NotNil(t, capturedInputs[0].Proxy, "first item should decode")
	assert.Nil(t, capturedInputs[0].Proxy.SSLEnabled, "ssl_enabled is nil (inherit) when omitted")
	assert.Nil(t, capturedInputs[0].Proxy.BlockExploits, "block_exploits is nil (inherit) when omitted")
	assert.Nil(t, capturedInputs[0].Proxy.SSLForced, "ssl_forced is nil (inherit) when omitted")
	assert.True(t, capturedInputs[0].Proxy.IsActive)
	assert.Nil(t, capturedInputs[1].Proxy, "second item should fail to decode")
	assert.NotEmpty(t, capturedInputs[1].DecodeError)

	var resp struct {
		Success bool                 `json:"success"`
		Data    service.ImportReport `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	require.Len(t, resp.Data.Items, 2)
	assert.Equal(t, service.ImportStatusValid, resp.Data.Items[0].Status)
	assert.Equal(t, service.ImportStatusInvalid, resp.Data.Items[1].Status)
}

func TestImportProxies_PreservesIsActive(t *testing.T) {
	t.Parallel()
	var capturedInputs []service.ImportInput
	mockService := &mocks.MockProxyService{
		ImportProxiesFunc: func(inputs []service.ImportInput, _ bool, _ int) service.ImportReport {
			capturedInputs = inputs
			return service.ImportReport{
				Summary: service.ImportSummary{Total: len(inputs), Importable: len(inputs)},
				Items:   make([]service.ImportItemResult, 0),
			}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	// Item 0: is_active:false must be preserved (exported inactive proxy).
	// Item 1: is_active omitted must default to active (matches CreateProxy).
	body := []byte(`{"dry_run":true,"proxies":[` +
		`{"name":"Inactive","hostname":"inactive.example.com","type":"reverse","is_active":false},` +
		`{"name":"Default","hostname":"default.example.com","type":"reverse"}` +
		`]}`)
	req := requestWithUserID(http.MethodPost, "/api/proxies/import", body, "123")
	rec := httptest.NewRecorder()

	handler.ImportProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, capturedInputs, 2)
	require.NotNil(t, capturedInputs[0].Proxy)
	assert.False(t, capturedInputs[0].Proxy.IsActive, "is_active:false must be preserved on import")
	require.NotNil(t, capturedInputs[1].Proxy)
	assert.True(t, capturedInputs[1].Proxy.IsActive, "omitted is_active defaults to active")
}

// TestImportProxies_GroupFieldsPassThrough confirms that group_id and
// hostname_label decode straight through importProxyRequest's shared
// createProxyRequest (the same struct CreateProxy uses): via the embedded
// models.Proxy, a plain *int/*string has no "omitted vs explicit zero value"
// ambiguity, so no wrapper field or extra plumbing is needed here — unlike
// the tri-state *bool settings, which createProxyRequest wraps explicitly.
func TestImportProxies_GroupFieldsPassThrough(t *testing.T) {
	t.Parallel()
	var capturedInputs []service.ImportInput
	mockService := &mocks.MockProxyService{
		ImportProxiesFunc: func(inputs []service.ImportInput, _ bool, _ int) service.ImportReport {
			capturedInputs = inputs
			return service.ImportReport{
				Summary: service.ImportSummary{Total: len(inputs), Importable: len(inputs)},
				Items:   make([]service.ImportItemResult, 0),
			}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body := []byte(`{"dry_run":true,"proxies":[` +
		`{"name":"Grouped","type":"reverse_proxy","group_id":3,"hostname_label":"abc","ssl_forced":false}` +
		`]}`)
	req := requestWithUserID(http.MethodPost, "/api/proxies/import", body, "123")
	rec := httptest.NewRecorder()

	handler.ImportProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, capturedInputs, 1)
	require.NotNil(t, capturedInputs[0].Proxy)
	require.NotNil(t, capturedInputs[0].Proxy.GroupID)
	assert.Equal(t, 3, *capturedInputs[0].Proxy.GroupID)
	require.NotNil(t, capturedInputs[0].Proxy.HostnameLabel)
	assert.Equal(t, "abc", *capturedInputs[0].Proxy.HostnameLabel)
	require.NotNil(t, capturedInputs[0].Proxy.SSLForced, "explicit ssl_forced:false must not be coerced to nil")
	assert.False(t, *capturedInputs[0].Proxy.SSLForced)
	assert.Nil(t, capturedInputs[0].Proxy.SSLEnabled, "omitted ssl_enabled must stay nil (inherit), not be defaulted")
}

func TestImportProxies_EmptyProxies(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ImportProxiesFunc: func(_ []service.ImportInput, _ bool, _ int) service.ImportReport {
			t.Fatal("service should not be called for empty proxies")
			return service.ImportReport{}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body := []byte(`{"dry_run":false,"proxies":[]}`)
	req := requestWithUserID(http.MethodPost, "/api/proxies/import", body, "123")
	rec := httptest.NewRecorder()

	handler.ImportProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportProxies_TooManyProxies(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ImportProxiesFunc: func(_ []service.ImportInput, _ bool, _ int) service.ImportReport {
			t.Fatal("service should not be called when over the cap")
			return service.ImportReport{}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	items := make([]json.RawMessage, maxImportProxies+1)
	for i := range items {
		items[i] = json.RawMessage(`{"name":"p","hostname":"p.example.com"}`)
	}
	body, _ := json.Marshal(importProxyRequest{Proxies: items})
	req := requestWithUserID(http.MethodPost, "/api/proxies/import", body, "123")
	rec := httptest.NewRecorder()

	handler.ImportProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// UpdateProxy Tests
// =============================================================================

func TestUpdateProxy_Success(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Old Name", Hostname: "old.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "old.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return service.ErrHostnameConflict
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return service.NewCaddyError("caddy reload failed")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return errors.New("validation: invalid hostname format")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "invalid", "ssl_enabled": true})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// A group_id referencing a non-existent proxy group must be reported as 404,
// not the generic 400 the catch-all would otherwise produce.
func TestUpdateProxy_GroupNotFound(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return service.ErrGroupNotFound
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "test.example.com", "ssl_enabled": true, "group_id": 99})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

// A genuine DB failure while loading the proxy's group must map to a 500 with
// a generic message — the raw underlying error text must never be echoed back
// to the API caller.
func TestUpdateProxy_GroupLookupFailed(t *testing.T) {
	t.Parallel()
	rawErr := errors.New("dial tcp 10.0.0.5:5432: connection refused")
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, _ *models.Proxy) error {
			return fmt.Errorf("%w (group %d): %w", service.ErrGroupLookupFailed, 5, rawErr)
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{"hostname": "test.example.com", "ssl_enabled": true, "group_id": 5})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "connection refused", "raw DB error text must not be echoed to the client")
	assert.NotContains(t, rec.Body.String(), "10.0.0.5", "raw DB error text must not be echoed to the client")
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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
			handler := NewProxyHandler(mockService, nil, nil, nil)

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

// TestListProxies_GroupFilter covers the four supported `group` query forms:
// eq:<id>, in:<ids>, not:<id>, and eq:none (ungrouped).
func TestListProxies_GroupFilter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		param          string
		expectedStatus int
		check          func(t *testing.T, req service.ListProxiesRequest)
	}{
		{
			name: "eq sets GroupID", param: "eq:3", expectedStatus: http.StatusOK,
			check: func(t *testing.T, req service.ListProxiesRequest) {
				require.NotNil(t, req.GroupID)
				assert.Equal(t, 3, *req.GroupID)
				assert.False(t, req.Ungrouped)
			},
		},
		{
			name: "bare value defaults to eq", param: "3", expectedStatus: http.StatusOK,
			check: func(t *testing.T, req service.ListProxiesRequest) {
				require.NotNil(t, req.GroupID)
				assert.Equal(t, 3, *req.GroupID)
			},
		},
		{
			name: "eq:none sets Ungrouped", param: "eq:none", expectedStatus: http.StatusOK,
			check: func(t *testing.T, req service.ListProxiesRequest) {
				assert.True(t, req.Ungrouped)
				assert.Nil(t, req.GroupID)
			},
		},
		{
			name: "in sets GroupIDIn", param: "in:1,2", expectedStatus: http.StatusOK,
			check: func(t *testing.T, req service.ListProxiesRequest) {
				assert.Equal(t, []int{1, 2}, req.GroupIDIn)
			},
		},
		{
			name: "not sets GroupIDNot", param: "not:3", expectedStatus: http.StatusOK,
			check: func(t *testing.T, req service.ListProxiesRequest) {
				require.NotNil(t, req.GroupIDNot)
				assert.Equal(t, 3, *req.GroupIDNot)
			},
		},
		{
			name: "eq non-integer is a bad request", param: "eq:abc", expectedStatus: http.StatusBadRequest,
		},
		{
			name: "in with a non-integer member is a bad request", param: "in:1,abc", expectedStatus: http.StatusBadRequest,
		},
		{
			name: "not non-integer is a bad request", param: "not:abc", expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unsupported operator is a bad request", param: "contains:foo", expectedStatus: http.StatusBadRequest,
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
			handler := NewProxyHandler(mockService, nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/proxies?group="+tc.param, nil)
			rec := httptest.NewRecorder()

			handler.ListProxies(rec, req)

			require.Equal(t, tc.expectedStatus, rec.Code)
			if tc.check != nil {
				tc.check(t, capturedReq)
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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies?type=contains:reverse_proxy", nil)
	rec := httptest.NewRecorder()

	handler.ListProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_InvalidStatusOperator(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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

// TestUpdateProxy_WithoutSSLEnabled_ResolvesToNil documents the current update
// semantics: SSLEnabled is tri-state, so an omitted field now resolves to nil
// (inherit) rather than preserving the existing row's value. This is a
// deliberate behavior change from before *bool (see handlers/proxy.go
// UpdateProxy); Task 6 revisits whether update should keep-existing instead.
func TestUpdateProxy_WithoutSSLEnabled_ResolvesToNil(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", SSLEnabled: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, proxy *models.Proxy) error {
			assert.Nil(t, proxy.SSLEnabled, "SSLEnabled is nil (inherit) when omitted from the update body")
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	require.NotNil(t, capturedProxy.SSLEnabled)
	assert.False(t, *capturedProxy.SSLEnabled, "SSLEnabled should be false when explicitly set")
}

// TestCreateProxy_SSLEnabledOmitted_ResolvesToNil documents that CreateProxy
// no longer defaults ssl_enabled to true when the field is omitted: nil now
// means "inherit from the proxy's group, or the system default if ungrouped".
// proxygroup.Resolve (a later task) is where that default is actually applied.
func TestCreateProxy_SSLEnabledOmitted_ResolvesToNil(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Nil(t, capturedProxy.SSLEnabled, "SSLEnabled is nil (inherit) when omitted")
}

func TestCreateProxy_BlockExploitsExplicitlyFalse(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":           "Test Proxy",
		"hostname":       "test.example.com",
		"type":           "reverse_proxy",
		"block_exploits": false,
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, capturedProxy.BlockExploits)
	assert.False(t, *capturedProxy.BlockExploits, "BlockExploits should be false when explicitly set")
}

// TestCreateProxy_BlockExploitsOmitted_ResolvesToNil documents that
// CreateProxy no longer defaults block_exploits to true when the field is
// omitted: nil now means "inherit from the proxy's group, or the system
// default if ungrouped".
func TestCreateProxy_BlockExploitsOmitted_ResolvesToNil(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Nil(t, capturedProxy.BlockExploits, "BlockExploits is nil (inherit) when omitted")
}

// =============================================================================
// CreateProxy / UpdateProxy SSLForced tests (Task 6): ssl_forced gains a wire
// field since a group can now set it and a proxy must be able to override it.
// =============================================================================

func TestCreateProxy_SSLForcedExplicitlyFalse(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "Test Proxy",
		"hostname":   "test.example.com",
		"type":       "reverse_proxy",
		"ssl_forced": false,
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, capturedProxy.SSLForced)
	assert.False(t, *capturedProxy.SSLForced, "SSLForced should be false when explicitly set")
}

// TestCreateProxy_SSLForcedOmitted_ResolvesToNil documents that ssl_forced,
// like the other three tri-state settings, is nil (inherit) when omitted —
// CreateProxy no longer hardcodes it to nil regardless of the request body;
// it now passes whatever the client actually sent.
func TestCreateProxy_SSLForcedOmitted_ResolvesToNil(t *testing.T) {
	t.Parallel()
	var capturedProxy *models.Proxy
	mockService := &mocks.MockProxyService{
		CreateProxyFunc: func(proxy *models.Proxy, _ int) error {
			capturedProxy = proxy
			proxy.ID = 1
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
		"type":     "reverse_proxy",
	})
	req := requestWithUserID(http.MethodPost, "/api/proxies", body, "123")
	rec := httptest.NewRecorder()

	handler.CreateProxy(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Nil(t, capturedProxy.SSLForced, "SSLForced is nil (inherit) when omitted")
}

// TestUpdateProxy_SSLForcedOmitted_ResolvesToNil is the update-path analogue
// of TestUpdateProxy_WithoutSSLEnabled_ResolvesToNil: ssl_forced is no longer
// force-preserved from the existing row (that was the bug — it could never be
// changed via the API at all). An omitted field now means inherit.
func TestUpdateProxy_SSLForcedOmitted_ResolvesToNil(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing", SSLForced: ptr(true)}, nil
		},
		UpdateProxyFunc: func(_ int, proxy *models.Proxy) error {
			assert.Nil(t, proxy.SSLForced, "SSLForced is nil (inherit) when omitted from the update body")
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

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

func TestUpdateProxy_SSLForcedExplicitlyFalse(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		GetProxyByIDFunc: func(_ int) (*models.Proxy, error) {
			return &models.Proxy{ID: 1, Name: "Existing"}, nil
		},
		UpdateProxyFunc: func(_ int, proxy *models.Proxy) error {
			require.NotNil(t, proxy.SSLForced)
			assert.False(t, *proxy.SSLForced)
			return nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	r := chi.NewRouter()
	r.Put("/api/proxies/{id}", handler.UpdateProxy)

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "Updated Proxy",
		"hostname":   "updated.example.com",
		"ssl_forced": false,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
			return &models.Proxy{ID: 1, Name: "Old Name", Hostname: "old.example.com", SSLEnabled: ptr(false)}, nil
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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
			return &models.Proxy{ID: 1, Name: "Existing", Hostname: "test.example.com", SSLEnabled: ptr(true)}, nil
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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
		SSLEnabled: ptr(true),
		IsActive:   true,
	}
	updated := &models.Proxy{
		ID:         1,
		Name:       "Test Proxy",
		Hostname:   "test.example.com",
		Type:       "reverse_proxy",
		SSLEnabled: ptr(true),
		IsActive:   true,
	}

	changes := buildProxyChanges(old, updated)

	assert.Nil(t, changes, "should return nil when no changes")
}

func TestBuildProxyChanges_NewlyTrackedScalarFields(t *testing.T) {
	t.Parallel()
	descOld := "old desc"
	descNew := "new desc"
	old := &models.Proxy{
		Description:           &descOld,
		BlockExploits:         ptr(true),
		TLSInsecureSkipVerify: ptr(false),
	}
	updated := &models.Proxy{
		Description:           &descNew,
		BlockExploits:         ptr(false),
		TLSInsecureSkipVerify: ptr(true),
	}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "description")
	require.Contains(t, changes, "block_exploits")
	require.Contains(t, changes, "tls_insecure_skip_verify")
	be := changes["block_exploits"].(map[string]interface{})
	assert.Equal(t, old.BlockExploits, be["old"])
	assert.Equal(t, updated.BlockExploits, be["new"])
}

func TestBuildProxyChanges_CustomHeadersChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{
		CustomHeaders: models.CustomHeaders{Request: map[string]string{"X-A": "1"}},
	}
	updated := &models.Proxy{
		CustomHeaders: models.CustomHeaders{Request: map[string]string{"X-A": "2"}},
	}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "custom_headers")
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
	old := &models.Proxy{SSLEnabled: ptr(true)}
	updated := &models.Proxy{SSLEnabled: ptr(false)}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "ssl_enabled")
	sslChange := changes["ssl_enabled"].(map[string]interface{})
	assert.Equal(t, old.SSLEnabled, sslChange["old"])
	assert.Equal(t, updated.SSLEnabled, sslChange["new"])
}

// TestBuildProxyChanges_InheritToExplicitFalseIsTracked is the audit-defect
// regression test: derefBool(nil) == derefBool(false) == false, so a
// transition between "inherit" (nil) and an explicit "off" (false) used to
// produce no diff at all — a security-relevant change (a proxy silently
// stopped inheriting its group's SSL/exploit-blocking posture) with nothing
// in the audit log to show it. ptrChanged must catch both directions, and the
// stored old/new values must be the raw *bool (so the JSON is `null` vs
// `false`, not `false` vs `false`).
func TestBuildProxyChanges_InheritToExplicitFalseIsTracked(t *testing.T) {
	t.Parallel()

	t.Run("nil to false", func(t *testing.T) {
		old := &models.Proxy{SSLEnabled: nil, BlockExploits: nil, SSLForced: nil, TLSInsecureSkipVerify: nil}
		updated := &models.Proxy{SSLEnabled: ptr(false), BlockExploits: ptr(false), SSLForced: ptr(false), TLSInsecureSkipVerify: ptr(false)}

		changes := buildProxyChanges(old, updated)

		require.NotNil(t, changes)
		for _, field := range []string{"ssl_enabled", "block_exploits", "ssl_forced", "tls_insecure_skip_verify"} {
			require.Contains(t, changes, field, "%s: inherit -> explicit false must be tracked", field)
			c := changes[field].(map[string]interface{})
			assert.Nil(t, c["old"], "%s: old must serialize as null, not false", field)
			require.NotNil(t, c["new"])
			assert.False(t, *c["new"].(*bool))
		}
	})

	t.Run("false to nil", func(t *testing.T) {
		old := &models.Proxy{SSLEnabled: ptr(false)}
		updated := &models.Proxy{SSLEnabled: nil}

		changes := buildProxyChanges(old, updated)

		require.NotNil(t, changes)
		require.Contains(t, changes, "ssl_enabled")
		c := changes["ssl_enabled"].(map[string]interface{})
		require.NotNil(t, c["old"])
		assert.False(t, *c["old"].(*bool))
		assert.Nil(t, c["new"], "new must serialize as null (inherit), not false")
	})

	t.Run("nil to nil is not a change", func(t *testing.T) {
		old := &models.Proxy{SSLEnabled: nil}
		updated := &models.Proxy{SSLEnabled: nil}

		changes := buildProxyChanges(old, updated)

		assert.Nil(t, changes)
	})
}

// buildProxyChanges also tracks group_id / hostname_label moves — reassigning
// or detaching a proxy from its group is exactly the kind of change an audit
// trail exists for.
func TestBuildProxyChanges_GroupMembershipChange(t *testing.T) {
	t.Parallel()
	old := &models.Proxy{GroupID: ptr(3), HostnameLabel: ptr("abc")}
	updated := &models.Proxy{GroupID: nil, HostnameLabel: nil}

	changes := buildProxyChanges(old, updated)

	require.NotNil(t, changes)
	require.Contains(t, changes, "group_id")
	require.Contains(t, changes, "hostname_label")
	gc := changes["group_id"].(map[string]interface{})
	assert.Equal(t, old.GroupID, gc["old"])
	assert.Nil(t, gc["new"])
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
		SSLEnabled: ptr(true),
	}
	updated := &models.Proxy{
		Name:       "New Name",
		Hostname:   "new.example.com",
		SSLEnabled: ptr(false),
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
				SSLEnabled: ptr(false),
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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	require.IsType(t, (*bool)(nil), sslChange["old"])
	require.IsType(t, (*bool)(nil), sslChange["new"])
	assert.False(t, *sslChange["old"].(*bool))
	assert.True(t, *sslChange["new"].(*bool))
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
				SSLEnabled: ptr(true),
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
	handler := NewProxyHandler(mockService, nil, mockAuditService, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

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
	handler := NewProxyHandler(mockService, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/stats", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetStats(rec, req)

	// The handler should still work even with canceled context
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusInternalServerError)
}

// =============================================================================
// Bulk Operation Tests
// =============================================================================

func TestBulkEnableProxies_Success(t *testing.T) {
	t.Parallel()
	var capturedIDs []int
	var capturedEnable bool
	mockService := &mocks.MockProxyService{
		BulkSetActiveFunc: func(ids []int, enable bool) service.BulkResult {
			capturedIDs = ids
			capturedEnable = enable
			return service.BulkResult{
				Requested: 3,
				Succeeded: 2,
				Failed:    1,
				Results: []service.BulkItemResult{
					{ID: 1, Status: "ok"},
					{ID: 2, Status: "error", Error: "proxy not found"},
					{ID: 3, Status: "ok"},
				},
			}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{"ids": []int{1, 2, 3}})
	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/enable", body, "123")
	rec := httptest.NewRecorder()

	handler.BulkEnableProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []int{1, 2, 3}, capturedIDs)
	assert.True(t, capturedEnable, "enable should be true for BulkEnableProxies")

	var resp struct {
		Success bool               `json:"success"`
		Data    service.BulkResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 3, resp.Data.Requested)
	assert.Equal(t, 2, resp.Data.Succeeded)
	assert.Equal(t, 1, resp.Data.Failed)
	require.Len(t, resp.Data.Results, 3)
	assert.Equal(t, "error", resp.Data.Results[1].Status)
}

func TestBulkDisableProxies_PassesEnableFalse(t *testing.T) {
	t.Parallel()
	var capturedEnable bool
	mockService := &mocks.MockProxyService{
		BulkSetActiveFunc: func(ids []int, enable bool) service.BulkResult {
			capturedEnable = enable
			return service.BulkResult{Requested: len(ids), Succeeded: len(ids)}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{"ids": []int{7}})
	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/disable", body, "123")
	rec := httptest.NewRecorder()

	handler.BulkDisableProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, capturedEnable, "enable should be false for BulkDisableProxies")
}

func TestBulkDeleteProxies_Success(t *testing.T) {
	t.Parallel()
	var capturedIDs []int
	mockService := &mocks.MockProxyService{
		BulkDeleteFunc: func(ids []int) service.BulkResult {
			capturedIDs = ids
			return service.BulkResult{
				Requested: 2,
				Succeeded: 2,
				Results: []service.BulkItemResult{
					{ID: 10, Status: "ok"},
					{ID: 11, Status: "ok"},
				},
			}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{"ids": []int{10, 11}})
	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/delete", body, "123")
	rec := httptest.NewRecorder()

	handler.BulkDeleteProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []int{10, 11}, capturedIDs)

	var resp struct {
		Success bool               `json:"success"`
		Data    service.BulkResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.Data.Succeeded)
}

func TestBulkProxies_EmptyIDs(t *testing.T) {
	t.Parallel()
	called := false
	mockService := &mocks.MockProxyService{
		BulkSetActiveFunc: func(_ []int, _ bool) service.BulkResult {
			called = true
			return service.BulkResult{}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{"ids": []int{}})
	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/enable", body, "123")
	rec := httptest.NewRecorder()

	handler.BulkEnableProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called, "service should not be called for empty ids")
}

func TestBulkProxies_TooManyIDs(t *testing.T) {
	t.Parallel()
	called := false
	mockService := &mocks.MockProxyService{
		BulkDeleteFunc: func(_ []int) service.BulkResult {
			called = true
			return service.BulkResult{}
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	ids := make([]int, 1001)
	for i := range ids {
		ids[i] = i + 1
	}
	body, _ := json.Marshal(map[string]interface{}{"ids": ids})
	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/delete", body, "123")
	rec := httptest.NewRecorder()

	handler.BulkDeleteProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called, "service should not be called when over the limit")
}

func TestBulkProxies_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := requestWithUserID(http.MethodPost, "/api/proxies/bulk/enable", []byte(`{invalid}`), "123")
	rec := httptest.NewRecorder()

	handler.BulkEnableProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// =============================================================================
// ExportProxies Tests
// =============================================================================

func TestExportProxies_WithIDs(t *testing.T) {
	t.Parallel()
	var capturedIDs []int
	var capturedFilters service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ExportProxiesFunc: func(ids []int, filters service.ListProxiesRequest) ([]service.ProxyExport, error) {
			capturedIDs = ids
			capturedFilters = filters
			return []service.ProxyExport{
				{Type: "reverse_proxy", Name: "p1", Hostname: "one.example.com", SSLEnabled: ptr(true), IsActive: true},
				{Type: "redirect", Name: "p3", Hostname: "three.example.com"},
			}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/export?ids=1,2,3", nil)
	rec := httptest.NewRecorder()

	handler.ExportProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []int{1, 2, 3}, capturedIDs)
	assert.Empty(t, capturedFilters.Types)

	var resp struct {
		Success bool                  `json:"success"`
		Data    []service.ProxyExport `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	require.Len(t, resp.Data, 2)
	assert.Equal(t, "p1", resp.Data[0].Name)
	assert.True(t, resp.Data[0].IsActive)
}

func TestExportProxies_WithFilters(t *testing.T) {
	t.Parallel()
	var capturedIDs []int
	var capturedFilters service.ListProxiesRequest
	mockService := &mocks.MockProxyService{
		ExportProxiesFunc: func(ids []int, filters service.ListProxiesRequest) ([]service.ProxyExport, error) {
			capturedIDs = ids
			capturedFilters = filters
			return []service.ProxyExport{}, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/export?type=in:reverse_proxy,redirect&status=active&ssl_enabled=true&search=foo", nil)
	rec := httptest.NewRecorder()

	handler.ExportProxies(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, capturedIDs, "no ids means export all matching filters")
	assert.Equal(t, []string{"reverse_proxy", "redirect"}, capturedFilters.Types)
	assert.Equal(t, "active", capturedFilters.Status)
	require.NotNil(t, capturedFilters.SSLEnabled)
	assert.True(t, *capturedFilters.SSLEnabled)
	assert.Equal(t, "foo", capturedFilters.Search)
}

func TestExportProxies_InvalidIDs(t *testing.T) {
	t.Parallel()
	called := false
	mockService := &mocks.MockProxyService{
		ExportProxiesFunc: func(_ []int, _ service.ListProxiesRequest) ([]service.ProxyExport, error) {
			called = true
			return nil, nil
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/export?ids=1,abc,3", nil)
	rec := httptest.NewRecorder()

	handler.ExportProxies(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called, "service should not be called for malformed ids")
}

func TestExportProxies_ServiceError(t *testing.T) {
	t.Parallel()
	mockService := &mocks.MockProxyService{
		ExportProxiesFunc: func(_ []int, _ service.ListProxiesRequest) ([]service.ProxyExport, error) {
			return nil, errors.New("database error")
		},
	}
	handler := NewProxyHandler(mockService, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/export", nil)
	rec := httptest.NewRecorder()

	handler.ExportProxies(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
