package caddyfile

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

func newTestBuilder() *Builder {
	return NewBuilder(zap.NewNop())
}

func TestBuildMainCaddyfile(t *testing.T) {
	builder := newTestBuilder()

	content := builder.BuildMainCaddyfile(MainCaddyfileOptions{Email: "admin@example.com"})

	// Check header
	if !strings.Contains(content, "Managed by Waygates") {
		t.Error("Expected header comment")
	}

	// Check email
	if !strings.Contains(content, "email admin@example.com") {
		t.Error("Expected email directive")
	}

	// Check global options
	if !strings.Contains(content, "acme_dns cloudflare") {
		t.Error("Expected acme_dns directive")
	}

	if !strings.Contains(content, "admin localhost:2019") {
		t.Error("Expected admin localhost:2019")
	}

	// Check imports
	if !strings.Contains(content, "import sites/*.conf") {
		t.Error("Expected sites import")
	}

	if !strings.Contains(content, "import catchall.conf") {
		t.Error("Expected catchall import")
	}
}

func TestBuildMainCaddyfile_NoEmail(t *testing.T) {
	builder := newTestBuilder()

	content := builder.BuildMainCaddyfile(MainCaddyfileOptions{})

	// Should not contain email directive
	if strings.Contains(content, "email ") {
		t.Error("Should not have email directive when email is empty")
	}
}

func TestBuildMainCaddyfile_DisableAutoHTTPS(t *testing.T) {
	builder := newTestBuilder()

	content := builder.BuildMainCaddyfile(MainCaddyfileOptions{DisableAutoHTTPS: true})

	// Should have auto_https off
	if !strings.Contains(content, "auto_https off") {
		t.Error("Expected auto_https off directive")
	}

	// Should not have acme_dns
	if strings.Contains(content, "acme_dns") {
		t.Error("Should not have acme_dns when auto_https is disabled")
	}
}

func TestBuildReverseProxy_Basic(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       1,
		Name:     "Test API",
		Hostname: "api.example.com",
		Type:     models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend", "port": float64(8080), "scheme": "http"},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check header
	if !strings.Contains(content, "# Proxy ID: 1") {
		t.Error("Expected proxy ID comment")
	}
	if !strings.Contains(content, "# Name: Test API") {
		t.Error("Expected name comment")
	}

	// Check site block
	if !strings.Contains(content, "api.example.com {") {
		t.Error("Expected hostname site block")
	}

	// Check reverse_proxy directive
	if !strings.Contains(content, "reverse_proxy backend:8080") {
		t.Error("Expected reverse_proxy directive with upstream")
	}

	// Check standard headers
	if !strings.Contains(content, "header_up X-Real-IP") {
		t.Error("Expected X-Real-IP header")
	}
	if !strings.Contains(content, "header_up X-Forwarded-For") {
		t.Error("Expected X-Forwarded-For header")
	}
}

func TestBuildReverseProxy_BlockExploits(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:            1,
		Name:          "Secure API",
		Hostname:      "secure.example.com",
		Type:          models.ProxyTypeReverseProxy,
		BlockExploits: true,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend", "port": float64(8080), "scheme": "http"},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check security snippet is imported
	if !strings.Contains(content, "import /etc/caddy/snippets/security.caddy") {
		t.Error("Expected security snippet import when BlockExploits is true")
	}
}

func TestBuildReverseProxy_NoBlockExploits(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:            1,
		Name:          "Open API",
		Hostname:      "open.example.com",
		Type:          models.ProxyTypeReverseProxy,
		BlockExploits: false,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend", "port": float64(8080), "scheme": "http"},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check security snippet is NOT imported
	if strings.Contains(content, "import /etc/caddy/snippets/security.caddy") {
		t.Error("Should not have security snippet import when BlockExploits is false")
	}
}

func TestBuildReverseProxy_MultipleUpstreams(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       2,
		Name:     "Load Balanced",
		Hostname: "lb.example.com",
		Type:     models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend1", "port": float64(8080), "scheme": "http"},
			map[string]interface{}{"host": "backend2", "port": float64(8080), "scheme": "http"},
			map[string]interface{}{"host": "backend3", "port": float64(8080), "scheme": "http"},
		},
		LoadBalancing: models.JSONField{
			"strategy": "round_robin",
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check all upstreams
	if !strings.Contains(content, "backend1:8080 backend2:8080 backend3:8080") {
		t.Error("Expected all upstreams in reverse_proxy directive")
	}

	// Check load balancing
	if !strings.Contains(content, "lb_policy round_robin") {
		t.Error("Expected load balancing policy")
	}
}

func TestBuildReverseProxy_HealthChecks(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       3,
		Name:     "With Health Checks",
		Hostname: "hc.example.com",
		Type:     models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend", "port": float64(8080), "scheme": "http"},
		},
		LoadBalancing: models.JSONField{
			"strategy": "round_robin",
			"health_checks": map[string]interface{}{
				"enabled":  true,
				"path":     "/health",
				"interval": "30s",
				"timeout":  "10s",
			},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "health_uri /health") {
		t.Error("Expected health_uri directive")
	}
	if !strings.Contains(content, "health_interval 30s") {
		t.Error("Expected health_interval directive")
	}
	if !strings.Contains(content, "health_timeout 10s") {
		t.Error("Expected health_timeout directive")
	}
}

func TestBuildReverseProxy_HTTPSUpstream(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:                    4,
		Name:                  "HTTPS Backend",
		Hostname:              "secure.example.com",
		Type:                  models.ProxyTypeReverseProxy,
		TLSInsecureSkipVerify: true,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "secure-backend", "port": float64(443), "scheme": "https"},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "transport http {") {
		t.Error("Expected transport block")
	}
	if !strings.Contains(content, "tls") {
		t.Error("Expected tls in transport")
	}
	if !strings.Contains(content, "tls_insecure_skip_verify") {
		t.Error("Expected tls_insecure_skip_verify")
	}
}

func TestBuildReverseProxy_CustomHeaders(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       5,
		Name:     "Custom Headers",
		Hostname: "headers.example.com",
		Type:     models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{
			map[string]interface{}{"host": "backend", "port": float64(8080), "scheme": "http"},
		},
		CustomHeaders: models.JSONField{
			"X-Custom-Header": "custom-value",
			"X-API-Key":       "secret123",
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "header_up X-Custom-Header \"custom-value\"") {
		t.Error("Expected custom header")
	}
	if !strings.Contains(content, "header_up X-API-Key \"secret123\"") {
		t.Error("Expected API key header")
	}
}

func TestBuildReverseProxy_NoUpstreams(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:        6,
		Name:      "No Upstreams",
		Hostname:  "empty.example.com",
		Type:      models.ProxyTypeReverseProxy,
		Upstreams: []interface{}{},
	}

	_, err := builder.BuildProxyFile(proxy)
	if err == nil {
		t.Error("Expected error for reverse proxy without upstreams")
	}
}

func TestBuildStatic_Basic(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       10,
		Name:     "Static Files",
		Hostname: "static.example.com",
		Type:     models.ProxyTypeStatic,
		StaticConfig: models.JSONField{
			"root_path":  "/var/www/static",
			"index_file": "index.html",
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "static.example.com {") {
		t.Error("Expected hostname site block")
	}
	if !strings.Contains(content, "root * /var/www/static") {
		t.Error("Expected root directive")
	}
	if !strings.Contains(content, "file_server") {
		t.Error("Expected file_server directive")
	}
	if !strings.Contains(content, "index index.html") {
		t.Error("Expected index file")
	}
}

func TestBuildStatic_SPA(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       11,
		Name:     "SPA App",
		Hostname: "app.example.com",
		Type:     models.ProxyTypeStatic,
		StaticConfig: models.JSONField{
			"root_path":  "/var/www/spa",
			"index_file": "index.html",
			"try_files":  []interface{}{"/index.html"},
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "try_files {path} /index.html") {
		t.Error("Expected try_files directive for SPA")
	}
}

func TestBuildStatic_Templates(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       12,
		Name:     "Template Site",
		Hostname: "templates.example.com",
		Type:     models.ProxyTypeStatic,
		StaticConfig: models.JSONField{
			"root_path":          "/var/www/templates",
			"template_rendering": true,
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "templates") {
		t.Error("Expected templates directive")
	}
}

func TestBuildStatic_NoConfig(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       13,
		Name:     "No Config",
		Hostname: "noconfig.example.com",
		Type:     models.ProxyTypeStatic,
	}

	_, err := builder.BuildProxyFile(proxy)
	if err == nil {
		t.Error("Expected error for static proxy without config")
	}
}

func TestBuildRedirect_Basic(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       20,
		Name:     "Basic Redirect",
		Hostname: "old.example.com",
		Type:     models.ProxyTypeRedirect,
		RedirectConfig: models.JSONField{
			"target":      "https://new.example.com",
			"status_code": float64(301),
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "old.example.com {") {
		t.Error("Expected hostname site block")
	}
	if !strings.Contains(content, "redir https://new.example.com permanent") {
		t.Error("Expected redirect directive with permanent status")
	}
}

func TestBuildRedirect_PreservePath(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       21,
		Name:     "Preserve Path",
		Hostname: "path.example.com",
		Type:     models.ProxyTypeRedirect,
		RedirectConfig: models.JSONField{
			"target":        "https://new.example.com",
			"status_code":   float64(302),
			"preserve_path": true,
		},
	}

	content, err := builder.BuildProxyFile(proxy)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(content, "redir https://new.example.com{uri} temporary") {
		t.Error("Expected redirect with {uri} placeholder and temporary status")
	}
}

func TestBuildRedirect_StatusCodes(t *testing.T) {
	builder := newTestBuilder()

	testCases := []struct {
		code     float64
		expected string
	}{
		{301, "permanent"},
		{302, "temporary"},
		{303, "303"},
		{307, "307"},
		{308, "308"},
	}

	for _, tc := range testCases {
		proxy := &models.Proxy{
			ID:       30,
			Name:     "Status Test",
			Hostname: "status.example.com",
			Type:     models.ProxyTypeRedirect,
			RedirectConfig: models.JSONField{
				"target":      "https://target.example.com",
				"status_code": tc.code,
			},
		}

		content, err := builder.BuildProxyFile(proxy)
		if err != nil {
			t.Fatalf("Unexpected error for status %v: %v", tc.code, err)
		}

		if !strings.Contains(content, tc.expected) {
			t.Errorf("Expected status keyword '%s' for code %v, got: %s", tc.expected, tc.code, content)
		}
	}
}

func TestBuildRedirect_NoConfig(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       40,
		Name:     "No Config",
		Hostname: "noconfig.example.com",
		Type:     models.ProxyTypeRedirect,
	}

	_, err := builder.BuildProxyFile(proxy)
	if err == nil {
		t.Error("Expected error for redirect proxy without config")
	}
}

func TestBuildCatchAll_Default(t *testing.T) {
	builder := newTestBuilder()

	settings := &models.NotFoundSettings{
		Mode: "default",
	}

	content := builder.BuildCatchAllFile(settings)

	// Catch-all uses port 80 only - specific domains handle their own HTTPS
	if !strings.Contains(content, ":80 {") {
		t.Error("Expected :80 catch-all block")
	}
	if !strings.Contains(content, "respond \"Not Found\" 404") {
		t.Error("Expected 404 respond directive")
	}
}

func TestBuildCatchAll_Redirect(t *testing.T) {
	builder := newTestBuilder()

	settings := &models.NotFoundSettings{
		Mode:        "redirect",
		RedirectURL: "https://example.com/404-page",
	}

	content := builder.BuildCatchAllFile(settings)

	// Catch-all uses port 80 only
	if !strings.Contains(content, ":80 {") {
		t.Error("Expected :80 catch-all block")
	}
	if !strings.Contains(content, "redir https://example.com/404-page 302") {
		t.Error("Expected redirect directive")
	}
	if strings.Contains(content, "respond") {
		t.Error("Should not have respond directive in redirect mode")
	}
}

func TestGetProxyFilename(t *testing.T) {
	builder := newTestBuilder()

	testCases := []struct {
		proxy    *models.Proxy
		expected string
	}{
		{
			proxy:    &models.Proxy{ID: 1, Hostname: "api.example.com"},
			expected: "1_api_example_com.conf",
		},
		{
			proxy:    &models.Proxy{ID: 42, Hostname: "my-app.example.com"},
			expected: "42_my-app_example_com.conf",
		},
		{
			proxy:    &models.Proxy{ID: 100, Hostname: "test"},
			expected: "100_test.conf",
		},
	}

	for _, tc := range testCases {
		result := builder.GetProxyFilename(tc.proxy)
		if result != tc.expected {
			t.Errorf("GetProxyFilename(%v) = %s, expected %s", tc.proxy.Hostname, result, tc.expected)
		}
	}
}

func TestGetDisabledFilename(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{ID: 5, Hostname: "disabled.example.com"}
	expected := "5_disabled_example_com.conf.disabled"

	result := builder.GetDisabledFilename(proxy)
	if result != expected {
		t.Errorf("GetDisabledFilename() = %s, expected %s", result, expected)
	}
}

func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"api.example.com", "api_example_com"},
		{"my-app.test.com", "my-app_test_com"},
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"special!@#chars", "special_chars"},
		{"multiple...dots", "multiple_dots"},
		{"_leading_", "leading"},
	}

	for _, tc := range testCases {
		result := sanitizeFilename(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeFilename(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestBuildProxyFile_UnknownType(t *testing.T) {
	builder := newTestBuilder()

	proxy := &models.Proxy{
		ID:       999,
		Name:     "Unknown",
		Hostname: "unknown.example.com",
		Type:     "invalid_type",
	}

	_, err := builder.BuildProxyFile(proxy)
	if err == nil {
		t.Error("Expected error for unknown proxy type")
	}
}

func TestBuildProxyFile_NilProxy(t *testing.T) {
	builder := newTestBuilder()

	_, err := builder.BuildProxyFile(nil)
	if err == nil {
		t.Error("Expected error for nil proxy")
	}
}
