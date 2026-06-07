package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// Test Helpers
// =============================================================================

// newTestLogger creates a no-op logger for testing.
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// createTestProxy creates a test proxy with the given parameters.
func createTestProxy(id int, name, hostname, proxyType string, isActive, sslEnabled bool) models.Proxy {
	return models.Proxy{
		ID:         id,
		Name:       name,
		Hostname:   hostname,
		Type:       proxyType,
		IsActive:   isActive,
		SSLEnabled: sslEnabled,
	}
}

// createReverseProxy creates a reverse proxy with upstreams.
func createReverseProxy(id int, name, hostname string, upstreams []interface{}, isActive, sslEnabled bool) models.Proxy {
	proxy := createTestProxy(id, name, hostname, models.ProxyTypeReverseProxy, isActive, sslEnabled)
	proxy.Upstreams = upstreams
	return proxy
}

// createRedirectProxy creates a redirect proxy.
func createRedirectProxy(id int, name, hostname, target string, statusCode int, isActive, sslEnabled bool) models.Proxy {
	proxy := createTestProxy(id, name, hostname, models.ProxyTypeRedirect, isActive, sslEnabled)
	proxy.RedirectConfig = models.JSONField{
		"target":      target,
		"status_code": float64(statusCode),
	}
	return proxy
}

// createStaticProxy creates a static file server proxy.
func createStaticProxy(id int, name, hostname, rootPath string, isActive, sslEnabled bool) models.Proxy {
	proxy := createTestProxy(id, name, hostname, models.ProxyTypeStatic, isActive, sslEnabled)
	proxy.StaticConfig = models.JSONField{
		"root_path": rootPath,
	}
	return proxy
}

// createTestUpstream creates test upstream data.
func createTestUpstream(host string, port int, scheme string) map[string]interface{} {
	return map[string]interface{}{
		"host":   host,
		"port":   float64(port),
		"scheme": scheme,
	}
}

// createTestACLGroup creates a test ACL group.
func createTestACLGroup(id int, name, combinationMode string) models.ACLGroup {
	return models.ACLGroup{
		ID:              id,
		Name:            name,
		CombinationMode: combinationMode,
	}
}

// createTestIPRule creates a test IP rule.
func createTestIPRule(ruleType, cidr string, priority int) models.ACLIPRule {
	return models.ACLIPRule{
		RuleType: ruleType,
		CIDR:     cidr,
		Priority: priority,
	}
}

// createTestBasicAuthUser creates a test basic auth user.
func createTestBasicAuthUser(username, passwordHash string) models.ACLBasicAuthUser {
	return models.ACLBasicAuthUser{
		Username:     username,
		PasswordHash: passwordHash,
	}
}

// createTestExternalProvider creates a test external auth provider.
func createTestExternalProvider(providerType, name, verifyURL string) models.ACLExternalProvider {
	return models.ACLExternalProvider{
		ProviderType: providerType,
		Name:         name,
		VerifyURL:    verifyURL,
	}
}

// createTestWaygatesAuth creates a test Waygates auth config.
func createTestWaygatesAuth(enabled bool, providers []string) *models.ACLWaygatesAuth {
	return &models.ACLWaygatesAuth{
		Enabled:          enabled,
		AllowedProviders: providers,
	}
}

// =============================================================================
// Builder Tests
// =============================================================================

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name    string
		opts    []BuilderOption
		wantLog bool
	}{
		{
			name:    "default builder",
			opts:    nil,
			wantLog: false,
		},
		{
			name:    "with logger",
			opts:    []BuilderOption{WithLogger(newTestLogger())},
			wantLog: true,
		},
		{
			name:    "with nil logger uses nop logger",
			opts:    []BuilderOption{WithLogger(nil)},
			wantLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder(tt.opts...)
			require.NotNil(t, b)
			require.NotNil(t, b.logger)
			require.NotNil(t, b.httpBuilder)
			require.NotNil(t, b.tlsBuilder)
			require.NotNil(t, b.aclGroups)
			require.NotNil(t, b.aclAssigns)
		})
	}
}

func TestBuilder_Build_EmptyProxies(t *testing.T) {
	b := NewBuilder(WithLogger(newTestLogger()))

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have admin config
	assert.NotNil(t, config.Admin)
	assert.Equal(t, "localhost:2019", config.Admin.Listen)

	// Should have storage config
	assert.NotNil(t, config.Storage)
	assert.Equal(t, "file_system", config.Storage.Module)
	assert.Equal(t, "/data", config.Storage.Root)

	// Should have a catch-all route even with no proxies
	assert.NotNil(t, config.Apps)
	assert.NotNil(t, config.Apps.HTTP)
	assert.Contains(t, config.Apps.HTTP.Servers, DefaultServerName)
	assert.Len(t, config.Apps.HTTP.Servers[DefaultServerName].Routes, 1)
}

func TestBuilder_Build_WithActiveReverseProxy(t *testing.T) {
	upstreams := []interface{}{
		createTestUpstream("backend.local", 8080, "http"),
	}

	proxy := createReverseProxy(1, "test-proxy", "example.com", upstreams, true, true)

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies([]models.Proxy{proxy})

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have HTTP routes
	assert.NotNil(t, config.Apps.HTTP)
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)

	// Should have at least 2 routes (proxy + catch-all)
	assert.GreaterOrEqual(t, len(server.Routes), 2)

	// Should have TLS config for SSL-enabled proxy
	assert.NotNil(t, config.Apps.TLS)
	assert.Contains(t, config.Apps.TLS.Certificates.Automate, "example.com")
}

func TestBuilder_Build_WithSSLEnabledProxy_CollectsDomains(t *testing.T) {
	proxies := []models.Proxy{
		createReverseProxy(1, "proxy1", "example.com", []interface{}{createTestUpstream("backend1", 8080, "http")}, true, true),
		createReverseProxy(2, "proxy2", "api.example.com", []interface{}{createTestUpstream("backend2", 8080, "http")}, true, true),
		createReverseProxy(3, "proxy3", "internal.example.com", []interface{}{createTestUpstream("backend3", 8080, "http")}, true, false),
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies(proxies)

	config, err := b.Build()
	require.NoError(t, err)

	// Only SSL-enabled proxies should have domains in TLS config
	assert.NotNil(t, config.Apps.TLS)
	tlsDomains := config.Apps.TLS.Certificates.Automate
	assert.Contains(t, tlsDomains, "example.com")
	assert.Contains(t, tlsDomains, "api.example.com")
	assert.NotContains(t, tlsDomains, "internal.example.com")
}

func TestBuilder_Build_WithInactiveProxies_SkipsThem(t *testing.T) {
	proxies := []models.Proxy{
		createReverseProxy(1, "active-proxy", "active.example.com", []interface{}{createTestUpstream("backend", 8080, "http")}, true, true),
		createReverseProxy(2, "inactive-proxy", "inactive.example.com", []interface{}{createTestUpstream("backend", 8080, "http")}, false, true),
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies(proxies)

	config, err := b.Build()
	require.NoError(t, err)

	// TLS should only include active proxy domain
	assert.NotNil(t, config.Apps.TLS)
	tlsDomains := config.Apps.TLS.Certificates.Automate
	assert.Contains(t, tlsDomains, "active.example.com")
	assert.NotContains(t, tlsDomains, "inactive.example.com")
}

func TestBuilder_Build_WithACLAssignments(t *testing.T) {
	upstreams := []interface{}{
		createTestUpstream("backend.local", 8080, "http"),
	}

	proxy := createReverseProxy(1, "protected-proxy", "secure.example.com", upstreams, true, true)

	aclGroup := createTestACLGroup(1, "test-acl", models.ACLCombinationModeAny)
	aclGroup.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword"),
	}

	assignment := models.ProxyACLAssignment{
		ID:          1,
		ProxyID:     1,
		ACLGroupID:  1,
		PathPattern: "/*",
		Priority:    0,
		Enabled:     true,
	}

	aclBuilder := NewACLBuilder(newTestLogger())
	aclBuilder.SetWaygatesURLs("http://localhost:8080/verify", "http://localhost:8080/login")

	b := NewBuilder(WithLogger(newTestLogger()), WithACLBuilder(aclBuilder))
	b.SetHTTPProxies([]models.Proxy{proxy})
	b.SetACLGroups([]models.ACLGroup{aclGroup})
	b.SetACLAssignments([]models.ProxyACLAssignment{assignment})

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have routes with ACL
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)
	assert.Greater(t, len(server.Routes), 1)
}

func TestBuilder_Build_WithNotFoundRedirect(t *testing.T) {
	notFoundSettings := &models.NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://home.example.com",
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetNotFoundSettings(notFoundSettings)

	config, err := b.Build()
	require.NoError(t, err)

	// Should have catch-all redirect route
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)
	require.NotEmpty(t, server.Routes)

	// Last route should be the catch-all redirect
	lastRoute := server.Routes[len(server.Routes)-1]
	require.NotEmpty(t, lastRoute.Handle)

	// Check that handler has Location header for redirect
	handler := lastRoute.Handle[0]
	if headers, ok := handler["headers"].(map[string][]string); ok {
		assert.Contains(t, headers, "Location")
	}
}

func TestBuilder_Build_WithNotFoundDefault(t *testing.T) {
	notFoundSettings := &models.NotFoundSettings{
		Mode: "default",
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetNotFoundSettings(notFoundSettings)

	config, err := b.Build()
	require.NoError(t, err)

	// Should have catch-all 404 route
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)
	require.NotEmpty(t, server.Routes)

	lastRoute := server.Routes[len(server.Routes)-1]
	require.NotEmpty(t, lastRoute.Handle)

	// Check that handler returns 404
	handler := lastRoute.Handle[0]
	assert.Equal(t, 404, handler["status_code"])
}

func TestBuilder_BuildJSON(t *testing.T) {
	b := NewBuilder(WithLogger(newTestLogger()))

	jsonBytes, err := b.BuildJSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Should be valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	// Should have expected keys
	assert.Contains(t, result, "admin")
	assert.Contains(t, result, "storage")
	assert.Contains(t, result, "apps")
}

func TestBuilder_BuildCompactJSON(t *testing.T) {
	b := NewBuilder(WithLogger(newTestLogger()))

	jsonBytes, err := b.BuildCompactJSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Should be valid compact JSON (no newlines/indentation)
	assert.NotContains(t, string(jsonBytes), "\n")
	assert.NotContains(t, string(jsonBytes), "  ")
}

func TestBuilder_BuildSingleProxy(t *testing.T) {
	upstreams := []interface{}{
		createTestUpstream("backend.local", 8080, "http"),
	}
	proxy := createReverseProxy(1, "test-proxy", "example.com", upstreams, true, true)

	b := NewBuilder(WithLogger(newTestLogger()))

	config, err := b.BuildSingleProxy(&proxy)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have HTTP routes for the proxy
	assert.NotNil(t, config.Apps.HTTP)

	// Should have TLS for SSL-enabled proxy
	assert.NotNil(t, config.Apps.TLS)
}

func TestBuilder_BuildSingleProxy_UnknownType_ReturnsError(t *testing.T) {
	proxy := createTestProxy(1, "unknown-proxy", "example.com", "unknown_type", true, true)

	b := NewBuilder(WithLogger(newTestLogger()))

	config, err := b.BuildSingleProxy(&proxy)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "unknown proxy type")
}

func TestBuilder_SetSettings(t *testing.T) {
	settings := &Settings{
		AdminEmail:        "admin@example.com",
		ACMEProvider:      ACMEProviderHTTP,
		WaygatesVerifyURL: "http://localhost:8080/verify",
		WaygatesLoginURL:  "http://localhost:8080/login",
	}

	aclBuilder := NewACLBuilder(newTestLogger())
	b := NewBuilder(WithLogger(newTestLogger()), WithACLBuilder(aclBuilder))
	b.SetSettings(settings)

	// Verify settings were applied to sub-builders
	assert.NotNil(t, b.tlsBuilder.settings)
	assert.Equal(t, settings.WaygatesVerifyURL, aclBuilder.waygatesVerifyURL)
	assert.Equal(t, settings.WaygatesLoginURL, aclBuilder.waygatesLoginURL)
}

func TestBuilder_SetACLGroups(t *testing.T) {
	groups := []models.ACLGroup{
		createTestACLGroup(1, "group1", models.ACLCombinationModeAny),
		createTestACLGroup(2, "group2", models.ACLCombinationModeAll),
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetACLGroups(groups)

	assert.Len(t, b.aclGroups, 2)
	assert.NotNil(t, b.aclGroups[1])
	assert.NotNil(t, b.aclGroups[2])
	assert.Equal(t, "group1", b.aclGroups[1].Name)
	assert.Equal(t, "group2", b.aclGroups[2].Name)
}

func TestBuilder_SetACLAssignments_OnlyEnabled(t *testing.T) {
	assignments := []models.ProxyACLAssignment{
		{ID: 1, ProxyID: 1, ACLGroupID: 1, Enabled: true},
		{ID: 2, ProxyID: 1, ACLGroupID: 2, Enabled: false},
		{ID: 3, ProxyID: 2, ACLGroupID: 1, Enabled: true},
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetACLAssignments(assignments)

	// Only enabled assignments should be stored
	assert.Len(t, b.aclAssigns[1], 1)
	assert.Len(t, b.aclAssigns[2], 1)
}

// =============================================================================
// HTTPBuilder Tests
// =============================================================================

func TestNewHTTPBuilder(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "with logger",
			logger: newTestLogger(),
		},
		{
			name:   "with nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewHTTPBuilder(tt.logger)
			require.NotNil(t, b)
			require.NotNil(t, b.logger)
		})
	}
}

func TestHTTPBuilder_BuildReverseProxyRoutes(t *testing.T) {
	tests := []struct {
		name       string
		proxy      models.Proxy
		wantErr    bool
		errContain string
	}{
		{
			name: "valid reverse proxy",
			proxy: createReverseProxy(1, "test", "example.com",
				[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true),
			wantErr: false,
		},
		{
			name: "multiple upstreams",
			proxy: createReverseProxy(1, "test", "example.com",
				[]interface{}{
					createTestUpstream("backend1", 8080, "http"),
					createTestUpstream("backend2", 8081, "http"),
				}, true, true),
			wantErr: false,
		},
		{
			name:       "nil upstreams",
			proxy:      createTestProxy(1, "test", "example.com", models.ProxyTypeReverseProxy, true, true),
			wantErr:    true,
			errContain: "requires at least one upstream",
		},
		{
			name: "empty upstreams array",
			proxy: createReverseProxy(1, "test", "example.com",
				[]interface{}{}, true, true),
			wantErr:    true,
			errContain: "requires at least one upstream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewHTTPBuilder(newTestLogger())

			routes, err := b.BuildReverseProxyRoutes(&tt.proxy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, routes)
			assert.True(t, routes[0].Terminal)

			// Check host matcher
			require.NotEmpty(t, routes[0].Match)
			hostMatcher := routes[0].Match[0]["host"]
			assert.Contains(t, hostMatcher, tt.proxy.Hostname)
		})
	}
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithCustomHeaders(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.CustomHeaders = models.CustomHeaders{
		Request: map[string]string{"X-Custom-Header": "custom-value", "X-Another": "another-value"},
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	handler := routes[0].Handle[0]
	require.NotNil(t, handler["headers"])
	hc := handler["headers"].(*HeadersConfig)
	require.NotNil(t, hc.Request)
	assert.Equal(t, []string{"custom-value"}, hc.Request.Set["X-Custom-Header"])
	assert.Equal(t, []string{"another-value"}, hc.Request.Set["X-Another"])
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithResponseHeaders(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.CustomHeaders = models.CustomHeaders{
		Request:  map[string]string{"X-Req": "r"},
		Response: map[string]string{"X-Res": "s"},
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	hc := routes[0].Handle[0]["headers"].(*HeadersConfig)
	require.NotNil(t, hc.Request)
	assert.Equal(t, []string{"r"}, hc.Request.Set["X-Req"])
	require.NotNil(t, hc.Response)
	assert.Equal(t, []string{"s"}, hc.Response.Set["X-Res"])
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithLoadBalancing(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{
			createTestUpstream("backend1", 8080, "http"),
			createTestUpstream("backend2", 8081, "http"),
		}, true, true)
	proxy.LoadBalancing = models.JSONField{
		"strategy": "round_robin",
		"health_checks": map[string]interface{}{
			"enabled":  true,
			"path":     "/health",
			"interval": "30s",
			"timeout":  "5s",
		},
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Check that handler has load balancing config
	handler := routes[0].Handle[0]
	assert.NotNil(t, handler["load_balancing"])
	assert.NotNil(t, handler["health_checks"])
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithHTTPSUpstream(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 443, "https")}, true, true)

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Should have transport config for HTTPS
	handler := routes[0].Handle[0]
	assert.NotNil(t, handler["transport"])
}

func TestHTTPBuilder_BuildReverseProxyRoutes_WithTLSInsecureSkipVerify(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.TLSInsecureSkipVerify = true

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutes(&proxy)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Should have transport config with insecure skip verify
	handler := routes[0].Handle[0]
	assert.NotNil(t, handler["transport"])
}

func TestHTTPBuilder_BuildReverseProxyRoutesWithACL(t *testing.T) {
	proxy := createReverseProxy(1, "test", "example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)

	aclGroup := createTestACLGroup(1, "test-acl", models.ACLCombinationModeAny)
	aclGroup.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword"),
	}

	assignments := []models.ProxyACLAssignment{
		{ID: 1, ProxyID: 1, ACLGroupID: 1, PathPattern: "/*", Priority: 0, Enabled: true},
	}

	aclGroups := map[int64]*models.ACLGroup{
		1: &aclGroup,
	}

	aclBuilder := NewACLBuilder(newTestLogger())

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildReverseProxyRoutesWithACL(&proxy, assignments, aclGroups, aclBuilder)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Should have multiple routes (ACL routes + fallback)
	assert.Greater(t, len(routes), 1)
}

func TestHTTPBuilder_BuildRedirectRoutes(t *testing.T) {
	tests := []struct {
		name       string
		proxy      models.Proxy
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid redirect",
			proxy:   createRedirectProxy(1, "redirect", "old.example.com", "https://new.example.com", 301, true, true),
			wantErr: false,
		},
		{
			name: "redirect with preserve path",
			proxy: func() models.Proxy {
				p := createRedirectProxy(1, "redirect", "old.example.com", "https://new.example.com", 302, true, true)
				p.RedirectConfig["preserve_path"] = true
				return p
			}(),
			wantErr: false,
		},
		{
			name: "redirect with preserve query",
			proxy: func() models.Proxy {
				p := createRedirectProxy(1, "redirect", "old.example.com", "https://new.example.com", 302, true, true)
				p.RedirectConfig["preserve_query"] = true
				return p
			}(),
			wantErr: false,
		},
		{
			name: "missing redirect config",
			proxy: func() models.Proxy {
				p := createTestProxy(1, "redirect", "old.example.com", models.ProxyTypeRedirect, true, true)
				p.RedirectConfig = nil
				return p
			}(),
			wantErr:    true,
			errContain: "redirect config is required",
		},
		{
			name: "empty redirect config",
			proxy: func() models.Proxy {
				p := createTestProxy(1, "redirect", "old.example.com", models.ProxyTypeRedirect, true, true)
				p.RedirectConfig = models.JSONField{}
				return p
			}(),
			wantErr:    true,
			errContain: "redirect config is required",
		},
		{
			name: "missing target",
			proxy: func() models.Proxy {
				p := createTestProxy(1, "redirect", "old.example.com", models.ProxyTypeRedirect, true, true)
				p.RedirectConfig = models.JSONField{
					"status_code": float64(301),
				}
				return p
			}(),
			wantErr:    true,
			errContain: "redirect target is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewHTTPBuilder(newTestLogger())

			routes, err := b.BuildRedirectRoutes(&tt.proxy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, routes)

			// Check handler type
			handler := routes[0].Handle[0]
			assert.Equal(t, HandlerStaticResponse, handler["handler"])
			assert.NotNil(t, handler["headers"])
		})
	}
}

func TestHTTPBuilder_BuildRedirectRoutes_DefaultStatusCode(t *testing.T) {
	proxy := createTestProxy(1, "redirect", "old.example.com", models.ProxyTypeRedirect, true, true)
	proxy.RedirectConfig = models.JSONField{
		"target": "https://new.example.com",
		// No status_code - should default to 302
	}

	b := NewHTTPBuilder(newTestLogger())
	routes, err := b.BuildRedirectRoutes(&proxy)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	handler := routes[0].Handle[0]
	assert.Equal(t, 302, handler["status_code"])
}

func TestHTTPBuilder_BuildStaticRoutes(t *testing.T) {
	tests := []struct {
		name       string
		proxy      models.Proxy
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid static",
			proxy:   createStaticProxy(1, "static", "static.example.com", "/var/www/html", true, true),
			wantErr: false,
		},
		{
			name: "static with index file",
			proxy: func() models.Proxy {
				p := createStaticProxy(1, "static", "static.example.com", "/var/www/html", true, true)
				p.StaticConfig["index_file"] = "index.html"
				return p
			}(),
			wantErr: false,
		},
		{
			name: "static with browse",
			proxy: func() models.Proxy {
				p := createStaticProxy(1, "static", "static.example.com", "/var/www/html", true, true)
				p.StaticConfig["browse"] = true
				return p
			}(),
			wantErr: false,
		},
		{
			name: "static with try_files (SPA)",
			proxy: func() models.Proxy {
				p := createStaticProxy(1, "static", "static.example.com", "/var/www/html", true, true)
				p.StaticConfig["try_files"] = []interface{}{"{path}", "/index.html"}
				return p
			}(),
			wantErr: false,
		},
		{
			name: "missing static config",
			proxy: func() models.Proxy {
				p := createTestProxy(1, "static", "static.example.com", models.ProxyTypeStatic, true, true)
				p.StaticConfig = nil
				return p
			}(),
			wantErr:    true,
			errContain: "static config is required",
		},
		{
			name: "missing root_path",
			proxy: func() models.Proxy {
				p := createTestProxy(1, "static", "static.example.com", models.ProxyTypeStatic, true, true)
				p.StaticConfig = models.JSONField{
					"browse": true,
				}
				return p
			}(),
			wantErr:    true,
			errContain: "root_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewHTTPBuilder(newTestLogger())

			routes, err := b.BuildStaticRoutes(&tt.proxy)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, routes)
		})
	}
}

func TestHTTPBuilder_MapLBStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"round_robin", "round_robin"},
		{"least_conn", "least_conn"},
		{"random", "random"},
		{"first", "first"},
		{"ip_hash", "ip_hash"},
		{"uri_hash", "uri_hash"},
		{"header", "header"},
		{"unknown", "round_robin"},
		{"", "round_robin"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapLBStrategy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// TLSBuilder Tests
// =============================================================================

func TestNewTLSBuilder(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "with logger",
			logger: newTestLogger(),
		},
		{
			name:   "with nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewTLSBuilder(tt.logger)
			require.NotNil(t, b)
			require.NotNil(t, b.logger)
		})
	}
}

func TestTLSBuilder_Build_NoDomains(t *testing.T) {
	b := NewTLSBuilder(newTestLogger())

	tlsApp, err := b.Build(nil)
	require.NoError(t, err)
	assert.Nil(t, tlsApp)

	tlsApp, err = b.Build([]string{})
	require.NoError(t, err)
	assert.Nil(t, tlsApp)
}

func TestTLSBuilder_Build_BasicDomains(t *testing.T) {
	b := NewTLSBuilder(newTestLogger())

	domains := []string{"example.com", "api.example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	assert.NotNil(t, tlsApp.Certificates)
	assert.ElementsMatch(t, domains, tlsApp.Certificates.Automate)
}

func TestTLSBuilder_Build_WithHTTPProvider(t *testing.T) {
	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderHTTP,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)
	require.NotEmpty(t, tlsApp.Automation.Policies)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	assert.Equal(t, "acme", issuer.Module)
	assert.Equal(t, "admin@example.com", issuer.Email)
}

func TestTLSBuilder_Build_WithOffProvider(t *testing.T) {
	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderOff,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	// With "off" provider, no automation should be configured
	assert.Nil(t, tlsApp.Automation)
}

func TestTLSBuilder_Build_WithCloudflareProvider(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "test-cf-token")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderCloudflare,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)
	require.NotEmpty(t, tlsApp.Automation.Policies)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	assert.NotNil(t, issuer.Challenges)
	assert.NotNil(t, issuer.Challenges.DNS)

	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderCloudflare)
	require.True(t, ok)
	assert.Equal(t, "cloudflare", provider.Name)
	assert.Equal(t, "test-cf-token", provider.APIToken)
}

func TestTLSBuilder_Build_WithCloudflareProvider_MissingCredentials(t *testing.T) {
	// Set to empty to simulate missing credentials
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderCloudflare,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	// Should not have automation when credentials are missing
	assert.Nil(t, tlsApp.Automation)
}

func TestTLSBuilder_Build_WithRoute53Provider(t *testing.T) {
	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderRoute53,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	assert.NotNil(t, issuer.Challenges)
	assert.NotNil(t, issuer.Challenges.DNS)

	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderRoute53)
	require.True(t, ok)
	assert.Equal(t, "route53", provider.Name)
}

func TestTLSBuilder_Build_WithDuckDNSProvider(t *testing.T) {
	t.Setenv("DUCKDNS_API_TOKEN", "test-duckdns-token")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderDuckDNS,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.duckdns.org"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderDuckDNS)
	require.True(t, ok)
	assert.Equal(t, "duckdns", provider.Name)
	assert.Equal(t, "test-duckdns-token", provider.APIToken)
}

func TestTLSBuilder_Build_WithDigitalOceanProvider(t *testing.T) {
	t.Setenv("DO_AUTH_TOKEN", "test-do-token")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderDigitalOcean,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderDigitalOcean)
	require.True(t, ok)
	assert.Equal(t, "digitalocean", provider.Name)
	assert.Equal(t, "test-do-token", provider.AuthToken)
}

func TestTLSBuilder_Build_WithHetznerProvider(t *testing.T) {
	t.Setenv("HETZNER_API_TOKEN", "test-hetzner-token")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderHetzner,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderHetzner)
	require.True(t, ok)
	assert.Equal(t, "hetzner", provider.Name)
	assert.Equal(t, "test-hetzner-token", provider.APIToken)
}

func TestTLSBuilder_Build_WithPorkbunProvider(t *testing.T) {
	t.Setenv("PORKBUN_API_KEY", "test-porkbun-key")
	t.Setenv("PORKBUN_API_SECRET_KEY", "test-porkbun-secret")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderPorkbun,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderPorkbun)
	require.True(t, ok)
	assert.Equal(t, "porkbun", provider.Name)
	assert.Equal(t, "test-porkbun-key", provider.APIKey)
	assert.Equal(t, "test-porkbun-secret", provider.APISecretKey)
}

func TestTLSBuilder_Build_WithVultrProvider(t *testing.T) {
	t.Setenv("VULTR_API_KEY", "test-vultr-key")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderVultr,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderVultr)
	require.True(t, ok)
	assert.Equal(t, "vultr", provider.Name)
	assert.Equal(t, "test-vultr-key", provider.APIKey)
}

func TestTLSBuilder_Build_WithNamecheapProvider(t *testing.T) {
	t.Setenv("NAMECHEAP_API_KEY", "test-namecheap-key")
	t.Setenv("NAMECHEAP_API_USER", "test-namecheap-user")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderNamecheap,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderNamecheap)
	require.True(t, ok)
	assert.Equal(t, "namecheap", provider.Name)
	assert.Equal(t, "test-namecheap-key", provider.APIKey)
	assert.Equal(t, "test-namecheap-user", provider.User)
}

func TestTLSBuilder_Build_WithOVHProvider(t *testing.T) {
	t.Setenv("OVH_ENDPOINT", "ovh-eu")
	t.Setenv("OVH_APPLICATION_KEY", "test-ovh-app-key")
	t.Setenv("OVH_APPLICATION_SECRET", "test-ovh-app-secret")
	t.Setenv("OVH_CONSUMER_KEY", "test-ovh-consumer-key")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderOVH,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderOVH)
	require.True(t, ok)
	assert.Equal(t, "ovh", provider.Name)
	assert.Equal(t, "ovh-eu", provider.Endpoint)
}

func TestTLSBuilder_Build_WithAzureProvider(t *testing.T) {
	t.Setenv("AZURE_TENANT_ID", "test-tenant")
	t.Setenv("AZURE_CLIENT_ID", "test-client")
	t.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "test-subscription")
	t.Setenv("AZURE_RESOURCE_GROUP", "test-rg")

	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: ACMEProviderAzure,
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	provider, ok := issuer.Challenges.DNS.Provider.(*DNSProviderAzure)
	require.True(t, ok)
	assert.Equal(t, "azure", provider.Name)
	assert.Equal(t, "test-tenant", provider.TenantID)
	assert.Equal(t, "test-client", provider.ClientID)
}

func TestTLSBuilder_Build_WithUnknownProvider_FallsBackToHTTP(t *testing.T) {
	settings := &Settings{
		AdminEmail:   "admin@example.com",
		ACMEProvider: "unknown_provider",
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	domains := []string{"example.com"}
	tlsApp, err := b.Build(domains)

	require.NoError(t, err)
	require.NotNil(t, tlsApp)
	require.NotNil(t, tlsApp.Automation)

	issuer := tlsApp.Automation.Policies[0].Issuers[0]
	assert.Equal(t, "acme", issuer.Module)
	assert.Nil(t, issuer.Challenges) // No DNS challenge for fallback
}

func TestTLSBuilder_GetCredential_FromSettings(t *testing.T) {
	settings := &Settings{
		DNSCredentials: map[string]string{
			"TEST_KEY": "from-settings",
		},
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	// Should prefer settings over env
	value := b.getCredential("TEST_KEY")
	assert.Equal(t, "from-settings", value)
}

func TestTLSBuilder_GetCredential_FromEnv(t *testing.T) {
	t.Setenv("TEST_CREDENTIAL_KEY", "from-env")

	settings := &Settings{
		DNSCredentials: nil, // No settings credentials
	}

	b := NewTLSBuilder(newTestLogger())
	b.SetSettings(settings)

	value := b.getCredential("TEST_CREDENTIAL_KEY")
	assert.Equal(t, "from-env", value)
}

// =============================================================================
// ACLBuilder Tests
// =============================================================================

func TestNewACLBuilder(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "with logger",
			logger: newTestLogger(),
		},
		{
			name:   "with nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewACLBuilder(tt.logger)
			require.NotNil(t, b)
			require.NotNil(t, b.logger)
		})
	}
}

func TestACLBuilder_SetWaygatesURLs(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	b.SetWaygatesURLs("http://localhost:8080/verify", "http://localhost:8080/login")

	assert.Equal(t, "http://localhost:8080/verify", b.waygatesVerifyURL)
	assert.Equal(t, "http://localhost:8080/login", b.waygatesLoginURL)
}

func TestACLBuilder_BuildACLRoutes_NilGroup(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	routes, err := b.BuildACLRoutes("example.com", "/*", nil, upstreamHandler)

	require.NoError(t, err)
	assert.Nil(t, routes)
}

func TestACLBuilder_BuildACLRoutes_NoAuthConfigured(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	// Group with no auth methods
	group := createTestACLGroup(1, "empty-acl", models.ACLCombinationModeAny)

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	assert.Nil(t, routes)
}

func TestACLBuilder_BuildACLRoutes_AnyMode_WithIPRules(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "ip-acl", models.ACLCombinationModeAny)
	group.IPRules = []models.ACLIPRule{
		createTestIPRule(models.ACLIPRuleTypeDeny, "10.0.0.0/8", 1),
		createTestIPRule(models.ACLIPRuleTypeBypass, "192.168.1.0/24", 2),
		createTestIPRule(models.ACLIPRuleTypeAllow, "172.16.0.0/12", 3),
	}

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Should have routes for deny, bypass, allow, static assets
	assert.GreaterOrEqual(t, len(routes), 4)
}

func TestACLBuilder_BuildACLRoutes_AnyMode_WithBasicAuth(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "basic-auth-acl", models.ACLCombinationModeAny)
	group.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword1"),
		createTestBasicAuthUser("user", "$2a$12$hashedpassword2"),
	}

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Check that one route has authentication handler
	foundAuth := false
	for _, route := range routes {
		for _, handler := range route.Handle {
			if handler["handler"] == HandlerAuthentication {
				foundAuth = true
				break
			}
		}
	}
	assert.True(t, foundAuth, "Should have authentication handler")
}

func TestACLBuilder_BuildACLRoutes_AnyMode_WithWaygatesAuth(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	b.SetWaygatesURLs("http://localhost:8080/verify", "http://localhost:8080/login")
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "waygates-acl", models.ACLCombinationModeAny)
	group.WaygatesAuth = createTestWaygatesAuth(true, []string{"google", "github"})

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)
}

func TestACLBuilder_BuildACLRoutes_WaygatesAuth_EmptyVerifyURL_FailsClosed(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	// Intentionally leave the Waygates URLs unset (empty verify + login URLs).
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "waygates-empty-url-acl", models.ACLCombinationModeAny)
	group.WaygatesAuth = createTestWaygatesAuth(true, nil)

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	// Serialize the generated routes to JSON so we can inspect the actual config
	// that Caddy would receive.
	data, err := json.Marshal(routes)
	require.NoError(t, err)
	rendered := string(data)

	// The forward-auth handler must fail closed: a static 503 deny response is
	// emitted with a clear message about the missing verify URL.
	assert.Contains(t, rendered, "static_response")
	assert.Contains(t, rendered, "ACL_WAYGATES_VERIFY_URL is empty")

	// Verify structurally that no reverse_proxy upstream dials a blank address.
	// Such an upstream is invalid and Caddy would reject it, breaking the entire
	// config reload. (The JSON omits empty dials via `omitempty`, so the typed
	// route tree must be inspected directly.)
	assertNoEmptyDialUpstream(t, routes)
}

// assertNoEmptyDialUpstream walks the generated routes and fails if any
// reverse_proxy upstream has an empty Dial field.
func assertNoEmptyDialUpstream(t *testing.T, routes []*HTTPRoute) {
	t.Helper()
	for _, route := range routes {
		for _, handler := range route.Handle {
			upstreams, ok := handler["upstreams"]
			if !ok {
				continue
			}
			ups, ok := upstreams.([]*Upstream)
			if !ok {
				continue
			}
			for _, u := range ups {
				assert.NotEmpty(t, u.Dial, "found reverse_proxy upstream with empty Dial address")
			}
		}
	}
}

func TestACLBuilder_BuildACLRoutes_AnyMode_WithExternalProvider(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "external-acl", models.ACLCombinationModeAny)
	group.ExternalProviders = []models.ACLExternalProvider{
		createTestExternalProvider(models.ACLProviderTypeAuthelia, "authelia", "http://authelia.local/api/verify"),
	}

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)
}

func TestACLBuilder_BuildACLRoutes_AllMode(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "all-mode-acl", models.ACLCombinationModeAll)
	group.IPRules = []models.ACLIPRule{
		createTestIPRule(models.ACLIPRuleTypeAllow, "192.168.1.0/24", 1),
	}
	group.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword"),
	}

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)
}

func TestACLBuilder_BuildACLRoutes_IPBypassMode(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "ip-bypass-acl", models.ACLCombinationModeIPBypass)
	group.IPRules = []models.ACLIPRule{
		createTestIPRule(models.ACLIPRuleTypeBypass, "192.168.1.0/24", 1),
	}
	group.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword"),
	}

	routes, err := b.BuildACLRoutes("example.com", "/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)
}

func TestACLBuilder_BuildACLRoutes_WithPathPattern(t *testing.T) {
	b := NewACLBuilder(newTestLogger())
	upstreamHandler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	group := createTestACLGroup(1, "path-acl", models.ACLCombinationModeAny)
	group.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("admin", "$2a$12$hashedpassword"),
	}

	routes, err := b.BuildACLRoutes("example.com", "/api/*", &group, upstreamHandler)

	require.NoError(t, err)
	require.NotEmpty(t, routes)
}

func TestCategorizeIPRules(t *testing.T) {
	rules := []models.ACLIPRule{
		createTestIPRule(models.ACLIPRuleTypeBypass, "192.168.1.0/24", 3),
		createTestIPRule(models.ACLIPRuleTypeDeny, "10.0.0.0/8", 1),
		createTestIPRule(models.ACLIPRuleTypeAllow, "172.16.0.0/12", 2),
		createTestIPRule(models.ACLIPRuleTypeBypass, "192.168.2.0/24", 4),
	}

	bypass, allow, deny := categorizeIPRules(rules)

	// Should be sorted by priority and categorized
	assert.Len(t, deny, 1)
	assert.Contains(t, deny, "10.0.0.0/8")

	assert.Len(t, allow, 1)
	assert.Contains(t, allow, "172.16.0.0/12")

	assert.Len(t, bypass, 2)
	assert.Contains(t, bypass, "192.168.1.0/24")
	assert.Contains(t, bypass, "192.168.2.0/24")
}

func TestGetDefaultWaygatesHeaders(t *testing.T) {
	headers := GetDefaultWaygatesHeaders()

	assert.Contains(t, headers, "X-Auth-User")
	assert.Contains(t, headers, "X-Auth-User-ID")
	assert.Contains(t, headers, "X-Auth-User-Email")
}

func TestGetProviderDefaultHeaders(t *testing.T) {
	tests := []struct {
		providerType    string
		expectedHeaders []string
	}{
		{
			providerType:    models.ACLProviderTypeAuthelia,
			expectedHeaders: []string{"Remote-User", "Remote-Groups", "Remote-Name", "Remote-Email"},
		},
		{
			providerType:    models.ACLProviderTypeAuthentik,
			expectedHeaders: []string{"X-authentik-username", "X-authentik-groups", "X-authentik-email"},
		},
		{
			providerType:    "unknown",
			expectedHeaders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			headers := GetProviderDefaultHeaders(tt.providerType)
			if tt.expectedHeaders == nil {
				assert.Nil(t, headers)
			} else {
				for _, expected := range tt.expectedHeaders {
					assert.Contains(t, headers, expected)
				}
			}
		})
	}
}

func TestExtractDialAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "http URL with port",
			input:    "http://waygates:8080",
			expected: "waygates:8080",
		},
		{
			name:     "https URL with port",
			input:    "https://auth.example.com:443",
			expected: "auth.example.com:443",
		},
		{
			name:     "http URL without port",
			input:    "http://waygates",
			expected: "waygates:80",
		},
		{
			name:     "https URL without port",
			input:    "https://secure.example.com",
			expected: "secure.example.com:443",
		},
		{
			name:     "URL with path",
			input:    "http://localhost:8080/verify",
			expected: "localhost:8080",
		},
		{
			name:     "already host:port format",
			input:    "waygates:8080",
			expected: "waygates:8080",
		},
		{
			name:     "IP address with port",
			input:    "http://192.168.1.100:9000",
			expected: "192.168.1.100:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDialAddress(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestBuilder_FullIntegration_ReverseProxyWithACL(t *testing.T) {
	// Create a complete configuration with reverse proxy and ACL
	upstreams := []interface{}{
		createTestUpstream("backend1.local", 8080, "http"),
		createTestUpstream("backend2.local", 8081, "http"),
	}

	proxy := createReverseProxy(1, "api-proxy", "api.example.com", upstreams, true, true)
	proxy.LoadBalancing = models.JSONField{
		"strategy": "round_robin",
	}
	proxy.CustomHeaders = models.CustomHeaders{
		Request: map[string]string{"X-API-Version": "v1"},
	}

	aclGroup := createTestACLGroup(1, "api-acl", models.ACLCombinationModeAny)
	aclGroup.IPRules = []models.ACLIPRule{
		createTestIPRule(models.ACLIPRuleTypeDeny, "10.0.0.0/8", 1),
		createTestIPRule(models.ACLIPRuleTypeBypass, "192.168.1.0/24", 2),
	}
	aclGroup.BasicAuthUsers = []models.ACLBasicAuthUser{
		createTestBasicAuthUser("api-user", "$2a$12$hashedpassword"),
	}

	assignment := models.ProxyACLAssignment{
		ID:          1,
		ProxyID:     1,
		ACLGroupID:  1,
		PathPattern: "/api/*",
		Priority:    0,
		Enabled:     true,
	}

	notFoundSettings := &models.NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://home.example.com",
	}

	settings := &Settings{
		AdminEmail:        "admin@example.com",
		ACMEProvider:      ACMEProviderHTTP,
		WaygatesVerifyURL: "http://localhost:8080/verify",
		WaygatesLoginURL:  "http://localhost:8080/login",
	}

	aclBuilder := NewACLBuilder(newTestLogger())

	b := NewBuilder(WithLogger(newTestLogger()), WithACLBuilder(aclBuilder))
	b.SetSettings(settings)
	b.SetHTTPProxies([]models.Proxy{proxy})
	b.SetACLGroups([]models.ACLGroup{aclGroup})
	b.SetACLAssignments([]models.ProxyACLAssignment{assignment})
	b.SetNotFoundSettings(notFoundSettings)

	// Build the configuration
	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify admin config
	assert.Equal(t, "localhost:2019", config.Admin.Listen)

	// Verify storage config
	assert.Equal(t, "file_system", config.Storage.Module)

	// Verify HTTP app
	assert.NotNil(t, config.Apps.HTTP)
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)
	assert.NotEmpty(t, server.Routes)

	// Verify TLS app
	assert.NotNil(t, config.Apps.TLS)
	assert.Contains(t, config.Apps.TLS.Certificates.Automate, "api.example.com")

	// Build JSON and verify it's valid
	jsonBytes, err := b.BuildJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)
	assert.Contains(t, result, "admin")
	assert.Contains(t, result, "storage")
	assert.Contains(t, result, "apps")
}

func TestBuilder_FullIntegration_MultipleProxyTypes(t *testing.T) {
	proxies := []models.Proxy{
		createReverseProxy(1, "api", "api.example.com",
			[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true),
		createRedirectProxy(2, "old-site", "old.example.com",
			"https://new.example.com", 301, true, true),
		createStaticProxy(3, "docs", "docs.example.com",
			"/var/www/docs", true, true),
	}

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies(proxies)

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have routes for all active proxies
	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)
	assert.GreaterOrEqual(t, len(server.Routes), 4) // 3 proxies + catch-all

	// TLS should include all SSL-enabled domains
	tlsDomains := config.Apps.TLS.Certificates.Automate
	assert.Contains(t, tlsDomains, "api.example.com")
	assert.Contains(t, tlsDomains, "old.example.com")
	assert.Contains(t, tlsDomains, "docs.example.com")
}

func TestBuilder_FullIntegration_SecurityRoutes(t *testing.T) {
	// Create a proxy with BlockExploits enabled
	proxy := createReverseProxy(1, "secure-api", "secure.example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.BlockExploits = true

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies([]models.Proxy{proxy})

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)

	// Should have security routes (6) + proxy route (1) + catch-all (1) = 8 routes
	assert.GreaterOrEqual(t, len(server.Routes), 8, "Expected at least 8 routes (6 security + 1 proxy + 1 catch-all)")

	// Verify security routes are present (they should be first)
	// Check first route is a security route (SQL injection)
	firstRoute := server.Routes[0]
	require.Len(t, firstRoute.Match, 1, "First route should have 1 matcher")

	// Check that the route has host matching for our hostname
	hostMatcher, hasHost := firstRoute.Match[0]["host"]
	if hasHost {
		hosts, ok := hostMatcher.([]string)
		require.True(t, ok)
		assert.Contains(t, hosts, "secure.example.com")
	}

	// Verify security route has 403 response
	require.Len(t, firstRoute.Handle, 1)
	assert.Equal(t, "static_response", firstRoute.Handle[0]["handler"])
	assert.Equal(t, 403, firstRoute.Handle[0]["status_code"])
}

func TestBuilder_FullIntegration_NoSecurityRoutesWhenDisabled(t *testing.T) {
	// Create a proxy with BlockExploits disabled
	proxy := createReverseProxy(1, "api", "api.example.com",
		[]interface{}{createTestUpstream("backend", 8080, "http")}, true, true)
	proxy.BlockExploits = false

	b := NewBuilder(WithLogger(newTestLogger()))
	b.SetHTTPProxies([]models.Proxy{proxy})

	config, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, config)

	server := config.Apps.HTTP.Servers[DefaultServerName]
	require.NotNil(t, server)

	// Should have only proxy route (1) + catch-all (1) = 2 routes
	assert.Equal(t, 2, len(server.Routes), "Expected 2 routes (1 proxy + 1 catch-all)")

	// First route should be the proxy route, not a security route
	firstRoute := server.Routes[0]
	if len(firstRoute.Handle) > 0 {
		// Security routes return static_response with 403
		// Proxy routes use reverse_proxy or subroute handler
		handler := firstRoute.Handle[0]["handler"]
		assert.NotEqual(t, "static_response", handler, "First route should not be a security route")
	}
}
