package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/goauth/middleware"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/service/mocks"
)

// =============================================================================
// Test Server Helpers
// =============================================================================

// testServer wraps handler setup for cleaner tests
type testServer struct {
	t            *testing.T
	router       *chi.Mux
	proxyHandler *ProxyHandler
	auditHandler *AuditHandler
}

// testServerOption configures a test server
type testServerOption func(*testServer)

// newTestServer creates a new test server with the given options
func newTestServer(t *testing.T, opts ...testServerOption) *testServer {
	t.Helper()

	ts := &testServer{
		t:      t,
		router: chi.NewRouter(),
	}

	for _, opt := range opts {
		opt(ts)
	}

	return ts
}

// withProxyHandler configures the test server with a proxy handler
func withProxyHandler(svc service.ProxyServiceInterface, auditSvc service.AuditServiceInterface) testServerOption {
	return func(ts *testServer) {
		ts.proxyHandler = NewProxyHandler(svc, auditSvc)
		ts.router.Route("/api/proxies", func(r chi.Router) {
			r.Get("/", ts.proxyHandler.ListProxies)
			r.Get("/stats", ts.proxyHandler.GetStats)
			r.Post("/", ts.proxyHandler.CreateProxy)
			r.Get("/{id}", ts.proxyHandler.GetProxy)
			r.Put("/{id}", ts.proxyHandler.UpdateProxy)
			r.Delete("/{id}", ts.proxyHandler.DeleteProxy)
			r.Post("/{id}/enable", ts.proxyHandler.EnableProxy)
			r.Post("/{id}/disable", ts.proxyHandler.DisableProxy)
		})
	}
}

// withAuditHandler configures the test server with an audit handler
func withAuditHandler(svc service.AuditServiceInterface) testServerOption {
	return func(ts *testServer) {
		ts.auditHandler = NewAuditHandler(svc)
		ts.router.Route("/api/audit", func(r chi.Router) {
			r.Get("/", ts.auditHandler.List)
			r.Get("/export", ts.auditHandler.Export)
			r.Get("/config", ts.auditHandler.GetConfig)
			r.Put("/config", ts.auditHandler.UpdateConfig)
		})
	}
}

// request creates a new HTTP request and returns the response
func (ts *testServer) request(method, path string, body interface{}, userID string) *httptest.ResponseRecorder {
	ts.t.Helper()

	var req *http.Request
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(ts.t, err)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if userID != "" {
		ctx := middleware.SetUserID(req.Context(), userID)
		req = req.WithContext(ctx)
	}

	rec := httptest.NewRecorder()
	ts.router.ServeHTTP(rec, req)
	return rec
}

// get performs a GET request
func (ts *testServer) get(path string) *httptest.ResponseRecorder {
	return ts.request(http.MethodGet, path, nil, "")
}

// getWithUser performs a GET request with user ID
func (ts *testServer) getWithUser(path, userID string) *httptest.ResponseRecorder {
	return ts.request(http.MethodGet, path, nil, userID)
}

// post performs a POST request
func (ts *testServer) post(path string, body interface{}, userID string) *httptest.ResponseRecorder {
	return ts.request(http.MethodPost, path, body, userID)
}

// put performs a PUT request
func (ts *testServer) put(path string, body interface{}, userID string) *httptest.ResponseRecorder {
	return ts.request(http.MethodPut, path, body, userID)
}

// delete performs a DELETE request
func (ts *testServer) delete(path string, userID string) *httptest.ResponseRecorder {
	return ts.request(http.MethodDelete, path, nil, userID)
}

// =============================================================================
// Response Helpers
// =============================================================================

// apiResponse represents the standard API response structure
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *apiError       `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// parseResponse parses the API response from a recorder
func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")
	return resp
}

// parseResponseData parses the data field from an API response into the given type
func parseResponseData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	resp := parseResponse(t, rec)
	var data T
	if resp.Data != nil {
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "failed to parse response data")
	}
	return data
}

// =============================================================================
// Mock Helpers
// =============================================================================

// mockProxyServiceBuilder helps build mock proxy services with fluent API
type mockProxyServiceBuilder struct {
	mock *mocks.MockProxyService
}

// newMockProxyService creates a new mock proxy service builder
func newMockProxyService() *mockProxyServiceBuilder {
	return &mockProxyServiceBuilder{
		mock: &mocks.MockProxyService{},
	}
}

// withListProxies sets the ListProxies function
func (b *mockProxyServiceBuilder) withListProxies(fn func(req service.ListProxiesRequest) (*models.ProxyListResponse, error)) *mockProxyServiceBuilder {
	b.mock.ListProxiesFunc = fn
	return b
}

// withGetProxyByID sets the GetProxyByID function
func (b *mockProxyServiceBuilder) withGetProxyByID(fn func(id int) (*models.Proxy, error)) *mockProxyServiceBuilder {
	b.mock.GetProxyByIDFunc = fn
	return b
}

// build returns the configured mock
func (b *mockProxyServiceBuilder) build() *mocks.MockProxyService {
	return b.mock
}

// =============================================================================
// Assertion Helpers
// =============================================================================

// assertStatusCode asserts the HTTP status code
func assertStatusCode(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	require.Equal(t, expected, rec.Code, "unexpected status code")
}

// assertSuccess asserts a successful response
func assertSuccess(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	resp := parseResponse(t, rec)
	require.True(t, resp.Success, "expected success response")
}

// assertError asserts an error response with the given message
func assertError(t *testing.T, rec *httptest.ResponseRecorder, expectedMsg string) {
	t.Helper()
	resp := parseResponse(t, rec)
	require.False(t, resp.Success, "expected error response")
	if expectedMsg != "" && resp.Error != nil {
		require.Contains(t, resp.Error.Message, expectedMsg)
	}
}

// =============================================================================
// Test Data Fixtures
// =============================================================================

// testProxy returns a standard test proxy configuration
func testProxy() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Test Proxy",
		"hostname":    "test.example.com",
		"type":        "reverse_proxy",
		"ssl_enabled": true,
	}
}

// testProxyWithSSL returns a test proxy with SSL disabled
func testProxyWithoutSSL() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Test Proxy",
		"hostname":    "test.example.com",
		"type":        "reverse_proxy",
		"ssl_enabled": false,
	}
}

// testProxyMinimal returns a minimal test proxy
func testProxyMinimal() map[string]interface{} {
	return map[string]interface{}{
		"name":     "Test Proxy",
		"hostname": "test.example.com",
	}
}
