package caddy_test

import (
	"encoding/json"
	"testing"

	"github.com/aloks98/homelab-proxy/backend/internal/caddy"
	"github.com/aloks98/homelab-proxy/backend/internal/models"
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
