package caddy_test

import (
	"encoding/json"
	"testing"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/models"
)

func TestBuildReverseProxyConfig(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Test App",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "192.168.1.100",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		BlockExploits: true,
		CustomHeaders: map[string]interface{}{
			"X-Custom": "value",
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if route.ID != "proxy_1" {
		t.Errorf("Expected route ID 'proxy_1', got '%s'", route.ID)
	}

	if len(route.Match) == 0 || len(route.Match[0].Host) == 0 {
		t.Fatal("No hostname in match")
	}

	if route.Match[0].Host[0] != "app.example.com" {
		t.Errorf("Expected hostname 'app.example.com', got '%s'", route.Match[0].Host[0])
	}

	if len(route.Handle) == 0 {
		t.Fatal("No handlers in route")
	}

	handler := route.Handle[0]
	if handler.Handler != "reverse_proxy" {
		t.Errorf("Expected handler 'reverse_proxy', got '%s'", handler.Handler)
	}

	if len(handler.Upstreams) != 1 {
		t.Errorf("Expected 1 upstream, got %d", len(handler.Upstreams))
	}

	if handler.Upstreams[0].Dial != "192.168.1.100:8080" {
		t.Errorf("Expected upstream '192.168.1.100:8080', got '%s'", handler.Upstreams[0].Dial)
	}
}

func TestBuildRedirectConfig(t *testing.T) {
	proxy := &models.Proxy{
		ID:       2,
		Type:     models.ProxyTypeRedirect,
		Name:     "Redirect",
		Hostname: "old.example.com",
		RedirectConfig: map[string]interface{}{
			"target":         "https://new.example.com",
			"status_code":    float64(301),
			"preserve_path":  true,
			"preserve_query": false,
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build redirect config: %v", err)
	}

	if len(route.Handle) == 0 {
		t.Fatal("No handlers in route")
	}

	handler := route.Handle[0]
	if handler.Handler != "static_response" {
		t.Errorf("Expected handler 'static_response', got '%s'", handler.Handler)
	}

	if handler.StatusCode != 301 {
		t.Errorf("Expected status code 301, got %d", handler.StatusCode)
	}
}

func TestBuildStaticConfig(t *testing.T) {
	proxy := &models.Proxy{
		ID:       3,
		Type:     models.ProxyTypeStatic,
		Name:     "Static Site",
		Hostname: "static.example.com",
		StaticConfig: map[string]interface{}{
			"root_path":          "/var/www/html",
			"index_file":         "index.html",
			"browse":             false,
			"template_rendering": true,
			"try_files":          []interface{}{"index.html"},
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build static config: %v", err)
	}

	// Should have templates handler + file_server handler
	if len(route.Handle) < 2 {
		t.Errorf("Expected at least 2 handlers, got %d", len(route.Handle))
	}

	// Check for templates handler
	hasTemplates := false
	for _, h := range route.Handle {
		if h.Handler == "templates" {
			hasTemplates = true
		}
	}

	if !hasTemplates {
		t.Error("Expected templates handler")
	}
}

func TestParseReverseProxyRoute(t *testing.T) {
	route := &caddy.RouteConfig{
		ID: "proxy_1",
		Match: []caddy.MatchConfig{
			{Host: []string{"app.example.com"}},
		},
		Handle: []caddy.HandlerConfig{
			{
				Handler: "reverse_proxy",
				Upstreams: []caddy.UpstreamConfig{
					{Dial: "192.168.1.100:8080"},
				},
			},
		},
	}

	proxy, err := caddy.ParseRouteToProxy(route)
	if err != nil {
		t.Fatalf("Failed to parse route: %v", err)
	}

	if proxy.Type != models.ProxyTypeReverseProxy {
		t.Errorf("Expected type 'reverse_proxy', got '%s'", proxy.Type)
	}

	if proxy.Hostname != "app.example.com" {
		t.Errorf("Expected hostname 'app.example.com', got '%s'", proxy.Hostname)
	}

	upstreams, ok := proxy.Upstreams.([]interface{})
	if !ok || len(upstreams) != 1 {
		t.Fatalf("Expected 1 upstream, got %v", proxy.Upstreams)
	}

	upstream := upstreams[0].(map[string]interface{})
	if upstream["host"] != "192.168.1.100" {
		t.Errorf("Expected host '192.168.1.100', got '%v'", upstream["host"])
	}
}

func TestBuildCaddyfileSnippet(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Test App",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "backend",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		BlockExploits: true,
	}

	snippet := caddy.BuildCaddyfileSnippet(proxy)

	// Check that snippet contains expected elements
	if snippet == "" {
		t.Error("Snippet is empty")
	}

	// Should contain hostname
	if !contains(snippet, "app.example.com") {
		t.Error("Snippet does not contain hostname")
	}

	// Should contain security import
	if !contains(snippet, "import snippets/security") {
		t.Error("Snippet does not contain security import")
	}

	// Should contain reverse_proxy directive
	if !contains(snippet, "reverse_proxy") {
		t.Error("Snippet does not contain reverse_proxy directive")
	}
}

func TestSimplifyProxyForUI(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Test App",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host": "backend",
				"port": float64(8080),
			},
		},
		CreatedBy: 999, // This should not appear in simplified version
	}

	simplified := caddy.SimplifyProxyForUI(proxy)

	// Should have basic fields
	if simplified["id"] != 1 {
		t.Error("Missing or incorrect id")
	}

	if simplified["type"] != models.ProxyTypeReverseProxy {
		t.Error("Missing or incorrect type")
	}

	if simplified["hostname"] != "app.example.com" {
		t.Error("Missing or incorrect hostname")
	}

	// Should have upstreams
	if simplified["upstreams"] == nil {
		t.Error("Missing upstreams")
	}

	// Should NOT have internal fields
	if _, exists := simplified["created_by"]; exists {
		t.Error("Simplified version should not contain created_by field")
	}
}

func TestRoundTrip(t *testing.T) {
	// Create original proxy
	original := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Round Trip Test",
		Hostname: "roundtrip.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "backend1",
				"port":   float64(8080),
				"scheme": "http",
			},
			map[string]interface{}{
				"host":   "backend2",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		LoadBalancing: map[string]interface{}{
			"strategy": "round_robin",
		},
	}

	// Convert to Caddy config
	route, err := caddy.BuildRouteConfig(original)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	// Convert back to Proxy
	parsed, err := caddy.ParseRouteToProxy(route)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify key fields match
	if parsed.Hostname != original.Hostname {
		t.Errorf("Hostname mismatch: expected '%s', got '%s'", original.Hostname, parsed.Hostname)
	}

	if parsed.Type != original.Type {
		t.Errorf("Type mismatch: expected '%s', got '%s'", original.Type, parsed.Type)
	}

	parsedUpstreams, _ := parsed.Upstreams.([]interface{})
	originalUpstreams, _ := original.Upstreams.([]interface{})
	if len(parsedUpstreams) != len(originalUpstreams) {
		t.Errorf("Upstream count mismatch: expected %d, got %d", len(originalUpstreams), len(parsedUpstreams))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestJSON(t *testing.T) {
	// Test that our config can be marshaled to JSON
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "JSON Test",
		Hostname: "json.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host": "backend",
				"port": float64(8080),
			},
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	_, err = json.Marshal(route)
	if err != nil {
		t.Fatalf("Failed to marshal config to JSON: %v", err)
	}
}

// TestRedirectConfig_RoundTrip tests that redirect configs survive build/parse cycle
func TestRedirectConfig_RoundTrip(t *testing.T) {
	original := &models.Proxy{
		ID:       100,
		Type:     models.ProxyTypeRedirect,
		Name:     "Redirect Test",
		Hostname: "old.example.com",
		RedirectConfig: map[string]interface{}{
			"target":         "https://new.example.com",
			"status_code":    float64(301),
			"preserve_path":  true,
			"preserve_query": true,
		},
	}

	// Build Caddy config
	route, err := caddy.BuildRouteConfig(original)
	if err != nil {
		t.Fatalf("Failed to build redirect config: %v", err)
	}

	// Verify the handler has StaticHeaders (not nested Headers)
	if len(route.Handle) == 0 {
		t.Fatal("No handlers in route")
	}

	handler := route.Handle[0]
	if handler.Handler != "static_response" {
		t.Errorf("Expected handler 'static_response', got '%s'", handler.Handler)
	}

	// Verify the status code
	if handler.StatusCode != 301 {
		t.Errorf("Expected status code 301, got %d", handler.StatusCode)
	}

	// Verify StaticHeaders contains Location
	if handler.StaticHeaders == nil {
		t.Fatal("StaticHeaders is nil")
	}

	locations, ok := handler.StaticHeaders["Location"]
	if !ok || len(locations) == 0 {
		t.Fatal("No Location header in StaticHeaders")
	}

	expectedLocation := "https://new.example.com{http.request.uri.path}?{http.request.uri.query}"
	if locations[0] != expectedLocation {
		t.Errorf("Expected Location '%s', got '%s'", expectedLocation, locations[0])
	}

	// Parse back to proxy
	parsed, err := caddy.ParseRouteToProxy(route)
	if err != nil {
		t.Fatalf("Failed to parse redirect config: %v", err)
	}

	if parsed.Type != models.ProxyTypeRedirect {
		t.Errorf("Expected type 'redirect', got '%s'", parsed.Type)
	}

	// Verify redirect config was parsed correctly
	if parsed.RedirectConfig == nil {
		t.Fatal("RedirectConfig is nil after parsing")
	}

	target, _ := parsed.RedirectConfig["target"].(string)
	if target != "https://new.example.com" {
		t.Errorf("Expected target 'https://new.example.com', got '%s'", target)
	}

	preservePath, _ := parsed.RedirectConfig["preserve_path"].(bool)
	if !preservePath {
		t.Error("Expected preserve_path to be true")
	}

	preserveQuery, _ := parsed.RedirectConfig["preserve_query"].(bool)
	if !preserveQuery {
		t.Error("Expected preserve_query to be true")
	}
}

// TestLoadBalancing_RoundTrip tests that load balancing configs survive build/parse cycle
func TestLoadBalancing_RoundTrip(t *testing.T) {
	testCases := []struct {
		name     string
		strategy string
	}{
		{"round_robin", "round_robin"},
		{"least_conn", "least_conn"},
		{"ip_hash", "ip_hash"},
		{"random", "random"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &models.Proxy{
				ID:       200,
				Type:     models.ProxyTypeReverseProxy,
				Name:     "LB Test",
				Hostname: "lb.example.com",
				Upstreams: []interface{}{
					map[string]interface{}{
						"host":   "backend1",
						"port":   float64(8080),
						"scheme": "http",
					},
					map[string]interface{}{
						"host":   "backend2",
						"port":   float64(8080),
						"scheme": "http",
					},
				},
				LoadBalancing: map[string]interface{}{
					"strategy": tc.strategy,
				},
			}

			// Build Caddy config
			route, err := caddy.BuildRouteConfig(original)
			if err != nil {
				t.Fatalf("Failed to build config: %v", err)
			}

			// Verify the handler has load balancing config
			if len(route.Handle) == 0 {
				t.Fatal("No handlers in route")
			}

			handler := route.Handle[0]
			if handler.LoadBalancing == nil {
				t.Fatal("LoadBalancing is nil")
			}

			// SelectionPolicy should be a map with "policy" key
			selPolicy, ok := handler.LoadBalancing.SelectionPolicy.(map[string]interface{})
			if !ok {
				t.Fatalf("SelectionPolicy is not a map, got %T", handler.LoadBalancing.SelectionPolicy)
			}

			policy, ok := selPolicy["policy"].(string)
			if !ok || policy != tc.strategy {
				t.Errorf("Expected policy '%s', got '%v'", tc.strategy, selPolicy["policy"])
			}

			// Parse back to proxy
			parsed, err := caddy.ParseRouteToProxy(route)
			if err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			// Verify load balancing was parsed correctly
			if parsed.LoadBalancing == nil {
				t.Fatal("LoadBalancing is nil after parsing")
			}

			parsedStrategy, _ := parsed.LoadBalancing["strategy"].(string)
			if parsedStrategy != tc.strategy {
				t.Errorf("Expected strategy '%s', got '%s'", tc.strategy, parsedStrategy)
			}
		})
	}
}

// TestBuildRouteConfig_UnsupportedType tests that unsupported types return error
func TestBuildRouteConfig_UnsupportedType(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     "unsupported_type",
		Name:     "Test",
		Hostname: "test.example.com",
	}

	_, err := caddy.BuildRouteConfig(proxy)
	if err == nil {
		t.Error("Expected error for unsupported proxy type")
	}
}

// TestBuildReverseProxy_NoUpstreams tests that reverse proxy without upstreams returns error
func TestBuildReverseProxy_NoUpstreams(t *testing.T) {
	proxy := &models.Proxy{
		ID:        1,
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Test",
		Hostname:  "test.example.com",
		Upstreams: nil,
	}

	_, err := caddy.BuildRouteConfig(proxy)
	if err == nil {
		t.Error("Expected error for reverse proxy without upstreams")
	}
}

// TestBuildReverseProxy_EmptyUpstreams tests that reverse proxy with empty upstreams returns error
func TestBuildReverseProxy_EmptyUpstreams(t *testing.T) {
	proxy := &models.Proxy{
		ID:        1,
		Type:      models.ProxyTypeReverseProxy,
		Name:      "Test",
		Hostname:  "test.example.com",
		Upstreams: []interface{}{},
	}

	_, err := caddy.BuildRouteConfig(proxy)
	if err == nil {
		t.Error("Expected error for reverse proxy with empty upstreams")
	}
}

// TestBuildReverseProxy_HTTPSUpstream tests that HTTPS upstreams get proper TLS transport
func TestBuildReverseProxy_HTTPSUpstream(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "HTTPS Backend",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "secure-backend",
				"port":   float64(443),
				"scheme": "https",
			},
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	handler := route.Handle[0]
	if handler.Transport == nil {
		t.Fatal("Expected transport config for HTTPS upstream")
	}

	if handler.Transport.Protocol != "http" {
		t.Errorf("Expected transport protocol 'http', got '%s'", handler.Transport.Protocol)
	}

	if handler.Transport.TLS == nil {
		t.Error("Expected TLS config for HTTPS upstream")
	}
}

// TestBuildReverseProxy_TLSInsecureSkipVerify tests TLS insecure skip verify setting
func TestBuildReverseProxy_TLSInsecureSkipVerify(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "Insecure Backend",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "backend",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		TLSInsecureSkipVerify: true,
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	handler := route.Handle[0]
	if handler.Transport == nil {
		t.Fatal("Expected transport config for TLS insecure skip verify")
	}

	if handler.Transport.TLS == nil {
		t.Fatal("Expected TLS config")
	}

	if !handler.Transport.TLS.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}

// TestBuildReverseProxy_HealthChecks tests health check configuration
func TestBuildReverseProxy_HealthChecks(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeReverseProxy,
		Name:     "HA Backend",
		Hostname: "app.example.com",
		Upstreams: []interface{}{
			map[string]interface{}{
				"host":   "backend1",
				"port":   float64(8080),
				"scheme": "http",
			},
			map[string]interface{}{
				"host":   "backend2",
				"port":   float64(8080),
				"scheme": "http",
			},
		},
		LoadBalancing: map[string]interface{}{
			"strategy": "round_robin",
			"health_checks": map[string]interface{}{
				"enabled":             true,
				"path":                "/health",
				"interval":            "30s",
				"timeout":             "5s",
				"unhealthy_threshold": float64(3),
				"healthy_threshold":   float64(2),
			},
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	handler := route.Handle[0]
	if handler.HealthChecks == nil {
		t.Fatal("Expected health checks config")
	}

	if handler.HealthChecks.Active == nil {
		t.Fatal("Expected active health checks")
	}

	if handler.HealthChecks.Active.Path != "/health" {
		t.Errorf("Expected path '/health', got '%s'", handler.HealthChecks.Active.Path)
	}

	if handler.HealthChecks.Active.Interval != "30s" {
		t.Errorf("Expected interval '30s', got '%s'", handler.HealthChecks.Active.Interval)
	}

	if handler.HealthChecks.Active.Fails != 3 {
		t.Errorf("Expected fails 3, got %d", handler.HealthChecks.Active.Fails)
	}

	if handler.HealthChecks.Active.Passes != 2 {
		t.Errorf("Expected passes 2, got %d", handler.HealthChecks.Active.Passes)
	}
}

// TestBuildRedirect_MissingTarget tests that redirect without target returns error
func TestBuildRedirect_MissingTarget(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeRedirect,
		Name:     "Redirect",
		Hostname: "old.example.com",
		RedirectConfig: map[string]interface{}{
			"status_code": float64(301),
		},
	}

	_, err := caddy.BuildRouteConfig(proxy)
	if err == nil {
		t.Error("Expected error for redirect without target")
	}
}

// TestBuildRedirect_DefaultStatusCode tests default redirect status code
func TestBuildRedirect_DefaultStatusCode(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeRedirect,
		Name:     "Redirect",
		Hostname: "old.example.com",
		RedirectConfig: map[string]interface{}{
			"target": "https://new.example.com",
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	handler := route.Handle[0]
	if handler.StatusCode != 302 {
		t.Errorf("Expected default status code 302, got %d", handler.StatusCode)
	}
}

// TestBuildStatic_MissingRootPath tests that static without root_path returns error
func TestBuildStatic_MissingRootPath(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeStatic,
		Name:     "Static",
		Hostname: "static.example.com",
		StaticConfig: map[string]interface{}{
			"index_file": "index.html",
		},
	}

	_, err := caddy.BuildRouteConfig(proxy)
	if err == nil {
		t.Error("Expected error for static without root_path")
	}
}

// TestBuildStatic_DefaultIndexFile tests default index file
func TestBuildStatic_DefaultIndexFile(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeStatic,
		Name:     "Static",
		Hostname: "static.example.com",
		StaticConfig: map[string]interface{}{
			"root_path": "/var/www/html",
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	// Find file_server handler
	var fileHandler *caddy.HandlerConfig
	for i := range route.Handle {
		if route.Handle[i].Handler == "file_server" {
			fileHandler = &route.Handle[i]
			break
		}
	}

	if fileHandler == nil {
		t.Fatal("No file_server handler found")
	}

	if len(fileHandler.IndexNames) == 0 || fileHandler.IndexNames[0] != "index.html" {
		t.Errorf("Expected default index file 'index.html', got %v", fileHandler.IndexNames)
	}
}

// TestBuildStatic_CatchAllPage tests catch all page configuration
func TestBuildStatic_CatchAllPage(t *testing.T) {
	proxy := &models.Proxy{
		ID:       1,
		Type:     models.ProxyTypeStatic,
		Name:     "SPA",
		Hostname: "spa.example.com",
		StaticConfig: map[string]interface{}{
			"root_path":      "/var/www/app",
			"catch_all_page": "/index.html",
		},
	}

	route, err := caddy.BuildRouteConfig(proxy)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	// Should have rewrite handler before file_server
	hasRewrite := false
	for _, h := range route.Handle {
		if h.Handler == "rewrite" && h.URI == "/index.html" {
			hasRewrite = true
			break
		}
	}

	if !hasRewrite {
		t.Error("Expected rewrite handler for catch_all_page")
	}
}

// TestGetSecuritySnippetPath tests security snippet path generation
func TestGetSecuritySnippetPath(t *testing.T) {
	testCases := []struct {
		name          string
		proxy         *models.Proxy
		expectedPath  string
	}{
		{
			name: "Reverse proxy with block exploits",
			proxy: &models.Proxy{
				Type:          models.ProxyTypeReverseProxy,
				BlockExploits: true,
			},
			expectedPath: "snippets/security",
		},
		{
			name: "Reverse proxy without block exploits",
			proxy: &models.Proxy{
				Type:          models.ProxyTypeReverseProxy,
				BlockExploits: false,
			},
			expectedPath: "",
		},
		{
			name: "Redirect with block exploits (should be empty)",
			proxy: &models.Proxy{
				Type:          models.ProxyTypeRedirect,
				BlockExploits: true,
			},
			expectedPath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := caddy.GetSecuritySnippetPath(tc.proxy)
			if path != tc.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tc.expectedPath, path)
			}
		})
	}
}

// TestRoutesPathConstant tests that RoutesPath constant is defined correctly
func TestRoutesPathConstant(t *testing.T) {
	expected := "/config/apps/http/servers/srv0/routes"
	if caddy.RoutesPath != expected {
		t.Errorf("Expected RoutesPath '%s', got '%s'", expected, caddy.RoutesPath)
	}
}
