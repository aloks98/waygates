package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// L4ProxyResponse represents the API response structure for L4 proxy operations
type L4ProxyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		ListenPort  int    `json:"listen_port"`
		Protocol    string `json:"protocol"`
		IsActive    bool   `json:"is_active"`
		Routes      []struct {
			ID                  int `json:"id"`
			Priority            int `json:"priority"`
			MatcherType         string
			LoadBalancingPolicy string `json:"load_balancing_policy"`
			Upstreams           []struct {
				Host   string `json:"host"`
				Port   int    `json:"port"`
				Weight *int   `json:"weight,omitempty"`
			} `json:"upstreams"`
		} `json:"routes,omitempty"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// L4ProxyListResponse represents the API response for listing L4 proxies
type L4ProxyListResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    struct {
		Items []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			ListenPort  int    `json:"listen_port"`
			Protocol    string `json:"protocol"`
			IsActive    bool   `json:"is_active"`
		} `json:"items"`
		Total      int64 `json:"total"`
		Page       int   `json:"page"`
		Limit      int   `json:"limit"`
		TotalPages int   `json:"total_pages"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// L4ProxyStatsResponse represents the API response for L4 proxy stats
type L4ProxyStatsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    struct {
		TotalProxies   int64 `json:"total_proxies"`
		ActiveProxies  int64 `json:"active_proxies"`
		TCPProxies     int64 `json:"tcp_proxies"`
		UDPProxies     int64 `json:"udp_proxies"`
		TotalRoutes    int64 `json:"total_routes"`
		TotalUpstreams int64 `json:"total_upstreams"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

// =============================================================================
// L4 Proxy Lifecycle Integration Test
// =============================================================================

// TestIntegration_L4ProxyLifecycle tests the full L4 proxy lifecycle
// NOTE: Subtests in this function share state and MUST run sequentially.
// Do NOT add t.Parallel() to any subtest as they depend on the order of execution.
func TestIntegration_L4ProxyLifecycle(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// State shared between sequential subtests
	type l4ProxyState struct {
		id         int
		name       string
		listenPort int
		protocol   string
	}
	state := &l4ProxyState{
		name:       "Test L4 Proxy",
		listenPort: 9000,
		protocol:   "tcp",
	}

	// Test 1: List L4 proxies (initially empty)
	t.Run("ListL4Proxies_EmptyInitially", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(0), response.Data.Total)
		assert.Len(t, response.Data.Items, 0)
	})

	// Test 2: Get stats (initially zero)
	t.Run("GetStats_InitiallyZero", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies/stats", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyStatsResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(0), response.Data.TotalProxies)
		assert.Equal(t, int64(0), response.Data.ActiveProxies)
	})

	// Test 3: Create an L4 proxy with routes
	t.Run("CreateL4Proxy_Success", func(t *testing.T) {
		proxy := map[string]interface{}{
			"name":        state.name,
			"description": "Integration test L4 proxy",
			"listen_port": state.listenPort,
			"protocol":    state.protocol,
			"is_active":   true,
			"routes": []map[string]interface{}{
				{
					"priority":              0,
					"matcher_type":          "any",
					"load_balancing_policy": "round_robin",
					"tls_terminate":         false,
					"tls_passthrough":       true,
					"upstreams": []map[string]interface{}{
						{"host": "backend1.local", "port": 8080},
						{"host": "backend2.local", "port": 8080, "weight": 2},
					},
				},
			},
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		require.Equalf(t, http.StatusCreated, resp.StatusCode, "Response body: %s", string(body))

		var response L4ProxyResponse
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.NotZero(t, response.Data.ID)
		assert.Equal(t, state.name, response.Data.Name)
		assert.Equal(t, state.listenPort, response.Data.ListenPort)
		assert.Equal(t, state.protocol, response.Data.Protocol)
		assert.True(t, response.Data.IsActive)
		assert.NotEmpty(t, response.Data.Routes)

		state.id = response.Data.ID
		t.Logf("Created L4 proxy with ID: %d", state.id)
	})

	// Test 4: List L4 proxies (should have one)
	t.Run("ListL4Proxies_HasOneProxy", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(1), response.Data.Total)
		require.Len(t, response.Data.Items, 1)
		assert.Equal(t, state.name, response.Data.Items[0].Name)
	})

	// Test 5: Get L4 proxy by ID
	t.Run("GetL4Proxy_Success", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/l4-proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, state.id, response.Data.ID)
		assert.Equal(t, state.name, response.Data.Name)
		assert.Equal(t, state.listenPort, response.Data.ListenPort)
		assert.Equal(t, state.protocol, response.Data.Protocol)
	})

	// Test 6: Update L4 proxy
	t.Run("UpdateL4Proxy_Success", func(t *testing.T) {
		updatedName := "Updated L4 Proxy"
		updateData := map[string]interface{}{
			"name":        updatedName,
			"description": "Updated description",
			"listen_port": 9001, // Change port
			"protocol":    "tcp",
			"is_active":   true,
			"routes": []map[string]interface{}{
				{
					"priority":              0,
					"matcher_type":          "any",
					"load_balancing_policy": "least_conn",
					"tls_terminate":         false,
					"tls_passthrough":       true,
					"upstreams": []map[string]interface{}{
						{"host": "backend3.local", "port": 8081},
					},
				},
			},
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, fmt.Sprintf("/api/l4-proxies/%d", state.id), updateData)
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		require.Equalf(t, http.StatusOK, resp.StatusCode, "Response body: %s", string(body))

		var response L4ProxyResponse
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, updatedName, response.Data.Name)
		assert.Equal(t, 9001, response.Data.ListenPort)

		// Update state for later tests
		state.name = updatedName
		state.listenPort = 9001
	})

	// Test 7: Toggle active status (disable)
	t.Run("ToggleActive_Disable", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/l4-proxies/%d/toggle", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		require.Equalf(t, http.StatusOK, resp.StatusCode, "Response body: %s", string(body))

		var response L4ProxyResponse
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.False(t, response.Data.IsActive, "Proxy should be disabled after toggle")
	})

	// Test 8: Toggle active status (enable)
	t.Run("ToggleActive_Enable", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/l4-proxies/%d/toggle", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		require.Equalf(t, http.StatusOK, resp.StatusCode, "Response body: %s", string(body))

		var response L4ProxyResponse
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.True(t, response.Data.IsActive, "Proxy should be enabled after second toggle")
	})

	// Test 9: Get stats after creating proxy
	t.Run("GetStats_HasProxies", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies/stats", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyStatsResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(1), response.Data.TotalProxies)
		assert.Equal(t, int64(1), response.Data.ActiveProxies)
		assert.Equal(t, int64(1), response.Data.TCPProxies)
		assert.Equal(t, int64(0), response.Data.UDPProxies)
	})

	// Test 10: Delete L4 proxy
	t.Run("DeleteL4Proxy_Success", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/l4-proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		require.Equalf(t, http.StatusOK, resp.StatusCode, "Response body: %s", string(body))

		// Verify deletion
		resp = env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/l4-proxies/%d", state.id), nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// L4 Proxy Validation Tests
// =============================================================================

// TestIntegration_L4ProxyValidation tests validation errors for L4 proxy creation
func TestIntegration_L4ProxyValidation(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	testCases := []struct {
		name           string
		proxy          map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "Missing name",
			proxy: map[string]interface{}{
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy without name",
		},
		{
			name: "Invalid protocol",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "invalid",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy with invalid protocol",
		},
		{
			name: "Invalid port - too low",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 0,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy with port 0",
		},
		{
			name: "Invalid port - too high",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 70000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy with port > 65535",
		},
		{
			name: "Invalid matcher type",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "invalid_matcher",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy with invalid matcher type",
		},
		{
			name: "Invalid load balancing policy",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "invalid_policy",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 proxy with invalid load balancing policy",
		},
		{
			name: "Route without upstreams",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject L4 route without upstreams",
		},
		{
			name: "Upstream with missing host",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"port": 8080}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject upstream without host",
		},
		{
			name: "Upstream with invalid port",
			proxy: map[string]interface{}{
				"name":        "Test Proxy",
				"listen_port": 9000,
				"protocol":    "tcp",
				"routes": []map[string]interface{}{
					{
						"matcher_type":          "any",
						"load_balancing_policy": "round_robin",
						"upstreams":             []map[string]interface{}{{"host": "backend", "port": 0}},
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject upstream with invalid port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", tc.proxy)
			defer func() { _ = resp.Body.Close() }()

			body, _ := io.ReadAll(resp.Body)
			require.Equalf(t, tc.expectedStatus, resp.StatusCode, "%s: %s", tc.description, string(body))
			t.Logf("Correctly rejected: %s", tc.description)
		})
	}
}

// =============================================================================
// L4 Proxy Port Conflict Test
// =============================================================================

// TestIntegration_L4ProxyPortConflict tests that duplicate port/protocol combinations are rejected
func TestIntegration_L4ProxyPortConflict(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	listenPort := 9500

	// Create first L4 proxy
	proxy := map[string]interface{}{
		"name":        "First L4 Proxy",
		"listen_port": listenPort,
		"protocol":    "tcp",
		"routes": []map[string]interface{}{
			{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]interface{}{{"host": "backend1", "port": 8080}},
			},
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "First proxy creation failed: %s", string(body))

	// Try to create second proxy with same port and protocol
	proxy["name"] = "Second L4 Proxy"
	resp = env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
	defer func() { _ = resp.Body.Close() }()

	body, _ = io.ReadAll(resp.Body)
	require.Equalf(t, http.StatusConflict, resp.StatusCode, "Expected 409 Conflict for duplicate port/protocol: %s", string(body))
	t.Log("Correctly rejected duplicate port/protocol combination")

	// Verify same port with different protocol is allowed (UDP instead of TCP)
	proxy["name"] = "UDP L4 Proxy"
	proxy["protocol"] = "udp"
	resp = env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "UDP proxy creation should succeed: %s", string(body))
	t.Log("Successfully created UDP proxy on same port")
}

// =============================================================================
// L4 Proxy Not Found Tests
// =============================================================================

// TestIntegration_L4ProxyNotFound tests 404 responses for non-existent proxies
func TestIntegration_L4ProxyNotFound(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	nonExistentID := 99999

	t.Run("GetL4Proxy_NotFound", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, fmt.Sprintf("/api/l4-proxies/%d", nonExistentID), nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("UpdateL4Proxy_NotFound", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":        "Non Existent",
			"listen_port": 9000,
			"protocol":    "tcp",
			"is_active":   true,
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, fmt.Sprintf("/api/l4-proxies/%d", nonExistentID), updateData)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DeleteL4Proxy_NotFound", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/api/l4-proxies/%d", nonExistentID), nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ToggleL4Proxy_NotFound", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/l4-proxies/%d/toggle", nonExistentID), nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// =============================================================================
// L4 Proxy Filter Tests
// =============================================================================

// TestIntegration_L4ProxyFilters tests list filtering capabilities
func TestIntegration_L4ProxyFilters(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create multiple L4 proxies with different attributes
	// All proxies start as active, then we toggle one to inactive
	proxies := []map[string]interface{}{
		{
			"name":        "TCP Proxy Active",
			"listen_port": 9100,
			"protocol":    "tcp",
			"routes": []map[string]interface{}{
				{
					"matcher_type":          "any",
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
				},
			},
		},
		{
			"name":        "UDP Proxy Active",
			"listen_port": 9101,
			"protocol":    "udp",
			"routes": []map[string]interface{}{
				{
					"matcher_type":          "any",
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
				},
			},
		},
		{
			"name":        "TCP Proxy ToBeInactive",
			"listen_port": 9102,
			"protocol":    "tcp",
			"routes": []map[string]interface{}{
				{
					"matcher_type":          "any",
					"load_balancing_policy": "round_robin",
					"upstreams":             []map[string]interface{}{{"host": "backend", "port": 8080}},
				},
			},
		},
	}

	// Create all proxies and collect their IDs
	proxyIDs := make([]int, 0, len(proxies))
	for _, proxy := range proxies {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.Equalf(t, http.StatusCreated, resp.StatusCode, "Failed to create proxy %s: %s", proxy["name"], string(body))

		var response L4ProxyResponse
		require.NoError(t, json.Unmarshal(body, &response))
		proxyIDs = append(proxyIDs, response.Data.ID)
	}

	// Toggle the third proxy to inactive
	resp := env.MakeAuthenticatedRequest(t, http.MethodPatch, fmt.Sprintf("/api/l4-proxies/%d/toggle", proxyIDs[2]), nil)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, http.StatusOK, resp.StatusCode, "Failed to toggle proxy: %s", string(body))

	t.Run("FilterByProtocol_TCP", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?protocol=tcp", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(2), response.Data.Total)
		for _, item := range response.Data.Items {
			assert.Equal(t, "tcp", item.Protocol)
		}
	})

	t.Run("FilterByProtocol_UDP", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?protocol=udp", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(1), response.Data.Total)
		require.NotEmpty(t, response.Data.Items, "Expected at least one UDP proxy")
		assert.Equal(t, "udp", response.Data.Items[0].Protocol)
	})

	t.Run("FilterByIsActive_True", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?is_active=true", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(2), response.Data.Total)
		for _, item := range response.Data.Items {
			assert.True(t, item.IsActive)
		}
	})

	t.Run("FilterByIsActive_False", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?is_active=false", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(1), response.Data.Total)
		require.NotEmpty(t, response.Data.Items, "Expected at least one inactive proxy")
		assert.False(t, response.Data.Items[0].IsActive)
	})

	t.Run("CombinedFilters", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?protocol=tcp&is_active=true", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(1), response.Data.Total)
		require.NotEmpty(t, response.Data.Items, "Expected at least one active TCP proxy")
		assert.Equal(t, "tcp", response.Data.Items[0].Protocol)
		assert.True(t, response.Data.Items[0].IsActive)
	})

	t.Run("Pagination", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?page=1&limit=2", nil)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response L4ProxyListResponse
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, json.Unmarshal(body, &response))

		assert.True(t, response.Success)
		assert.Equal(t, int64(3), response.Data.Total)
		assert.Len(t, response.Data.Items, 2)
	})

	t.Run("InvalidProtocol", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?protocol=invalid", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InvalidIsActive", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?is_active=maybe", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InvalidPage", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?page=-1", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InvalidLimit", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies?limit=101", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// L4 Proxy Update Port Conflict Test
// =============================================================================

// TestIntegration_L4ProxyUpdatePortConflict tests port conflict during update
func TestIntegration_L4ProxyUpdatePortConflict(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	// Create first L4 proxy
	proxy1 := map[string]interface{}{
		"name":        "First L4 Proxy",
		"listen_port": 9600,
		"protocol":    "tcp",
		"routes": []map[string]interface{}{
			{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]interface{}{{"host": "backend1", "port": 8080}},
			},
		},
	}

	resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy1)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "First proxy creation failed: %s", string(body))

	// Create second L4 proxy on different port
	proxy2 := map[string]interface{}{
		"name":        "Second L4 Proxy",
		"listen_port": 9601,
		"protocol":    "tcp",
		"routes": []map[string]interface{}{
			{
				"matcher_type":          "any",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]interface{}{{"host": "backend2", "port": 8080}},
			},
		},
	}

	resp = env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy2)
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equalf(t, http.StatusCreated, resp.StatusCode, "Second proxy creation failed: %s", string(body))

	var response L4ProxyResponse
	require.NoError(t, json.Unmarshal(body, &response))
	proxy2ID := response.Data.ID

	// Try to update second proxy to use first proxy's port
	updateData := map[string]interface{}{
		"name":        "Second L4 Proxy",
		"listen_port": 9600, // This port is already in use by first proxy
		"protocol":    "tcp",
		"is_active":   true,
	}

	resp = env.MakeAuthenticatedRequest(t, http.MethodPut, fmt.Sprintf("/api/l4-proxies/%d", proxy2ID), updateData)
	defer func() { _ = resp.Body.Close() }()

	body, _ = io.ReadAll(resp.Body)
	require.Equalf(t, http.StatusConflict, resp.StatusCode, "Expected 409 Conflict when updating to existing port: %s", string(body))
	t.Log("Correctly rejected update to conflicting port")
}

// =============================================================================
// L4 Proxy Route Types Test
// =============================================================================

// TestIntegration_L4ProxyRouteTypes tests various route matcher types
func TestIntegration_L4ProxyRouteTypes(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	testCases := []struct {
		name  string
		route map[string]interface{}
	}{
		{
			name: "TLS Matcher with SNI",
			route: map[string]interface{}{
				"matcher_type":          "tls",
				"sni_hostnames":         []string{"example.com", "*.example.com"},
				"load_balancing_policy": "round_robin",
				"tls_passthrough":       true,
				"tls_terminate":         false,
				"upstreams":             []map[string]interface{}{{"host": "backend", "port": 443}},
			},
		},
		{
			name: "SSH Matcher",
			route: map[string]interface{}{
				"matcher_type":          "ssh",
				"load_balancing_policy": "least_conn",
				"upstreams":             []map[string]interface{}{{"host": "ssh-server", "port": 22}},
			},
		},
		{
			name: "Postgres Matcher",
			route: map[string]interface{}{
				"matcher_type":          "postgres",
				"load_balancing_policy": "round_robin",
				"upstreams":             []map[string]interface{}{{"host": "pg-server", "port": 5432}},
			},
		},
		{
			name: "Remote IP Matcher",
			route: map[string]interface{}{
				"matcher_type":          "remote_ip",
				"allowed_ip_ranges":     []string{"10.0.0.0/8", "192.168.0.0/16"},
				"load_balancing_policy": "ip_hash",
				"upstreams":             []map[string]interface{}{{"host": "internal", "port": 8080}},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proxy := map[string]interface{}{
				"name":        fmt.Sprintf("Route Type Test %d", i),
				"listen_port": 9700 + i,
				"protocol":    "tcp",
				"routes":      []map[string]interface{}{tc.route},
			}

			resp := env.MakeAuthenticatedRequest(t, http.MethodPost, "/api/l4-proxies", proxy)
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			require.Equalf(t, http.StatusCreated, resp.StatusCode, "Failed to create proxy with %s: %s", tc.name, string(body))
			t.Logf("Successfully created L4 proxy with %s", tc.name)
		})
	}
}

// =============================================================================
// L4 Proxy Invalid ID Test
// =============================================================================

// TestIntegration_L4ProxyInvalidID tests invalid ID parameter handling
func TestIntegration_L4ProxyInvalidID(t *testing.T) {
	env := SetupContainerEnvironment(t)
	defer env.Cleanup(t)

	env.RegisterAndLogin(t)

	t.Run("GetL4Proxy_InvalidID", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodGet, "/api/l4-proxies/invalid", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("UpdateL4Proxy_InvalidID", func(t *testing.T) {
		updateData := map[string]interface{}{
			"name":        "Test",
			"listen_port": 9000,
			"protocol":    "tcp",
			"is_active":   true,
		}

		resp := env.MakeAuthenticatedRequest(t, http.MethodPut, "/api/l4-proxies/abc", updateData)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("DeleteL4Proxy_InvalidID", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodDelete, "/api/l4-proxies/xyz", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("ToggleL4Proxy_InvalidID", func(t *testing.T) {
		resp := env.MakeAuthenticatedRequest(t, http.MethodPatch, "/api/l4-proxies/notanid/toggle", nil)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
