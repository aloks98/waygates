package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityRoutes(t *testing.T) {
	routes := SecurityRoutes()

	// Should return 6 security routes
	assert.Len(t, routes, 6, "Expected 6 security routes")

	// All routes should be terminal
	for i, route := range routes {
		assert.True(t, route.Terminal, "Route %d should be terminal", i)
	}

	// All routes should have handlers returning 403
	for i, route := range routes {
		require.Len(t, route.Handle, 1, "Route %d should have 1 handler", i)
		assert.Equal(t, "static_response", route.Handle[0]["handler"])
		assert.Equal(t, 403, route.Handle[0]["status_code"])
	}
}

func TestSecurityRoutes_SQLInjection(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[0] // SQL injection route

	require.Len(t, route.Match, 1)
	pathRegexp, ok := route.Match[0]["path_regexp"].(map[string]string)
	require.True(t, ok, "path_regexp should be a map[string]string")

	assert.Equal(t, "sql_injection", pathRegexp["name"])
	assert.Contains(t, pathRegexp["pattern"], "union.*select")
	assert.Contains(t, pathRegexp["pattern"], "drop.*table")

	assert.Equal(t, "Forbidden - SQL injection detected", route.Handle[0]["body"])
}

func TestSecurityRoutes_FileInjection(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[1] // File injection route

	require.Len(t, route.Match, 1)
	pathRegexp, ok := route.Match[0]["path_regexp"].(map[string]string)
	require.True(t, ok)

	assert.Equal(t, "file_injection", pathRegexp["name"])
	assert.Contains(t, pathRegexp["pattern"], `\.\.`)

	assert.Equal(t, "Forbidden - Path traversal detected", route.Handle[0]["body"])
}

func TestSecurityRoutes_XSS(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[2] // XSS route

	require.Len(t, route.Match, 1)
	pathRegexp, ok := route.Match[0]["path_regexp"].(map[string]string)
	require.True(t, ok)

	assert.Equal(t, "xss_attack", pathRegexp["name"])
	assert.Contains(t, pathRegexp["pattern"], "<script")
	assert.Contains(t, pathRegexp["pattern"], "javascript:")

	assert.Equal(t, "Forbidden - XSS detected", route.Handle[0]["body"])
}

func TestSecurityRoutes_PHPGlobals(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[3] // PHP globals route

	require.Len(t, route.Match, 1)
	pathRegexp, ok := route.Match[0]["path_regexp"].(map[string]string)
	require.True(t, ok)

	assert.Equal(t, "php_globals", pathRegexp["name"])
	assert.Contains(t, pathRegexp["pattern"], "globals")
	assert.Contains(t, pathRegexp["pattern"], "_request")

	assert.Equal(t, "Forbidden - PHP globals injection detected", route.Handle[0]["body"])
}

func TestSecurityRoutes_BadUserAgents(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[4] // Bad user agents route

	require.Len(t, route.Match, 1)
	headerRegexp, ok := route.Match[0]["header_regexp"].(map[string]interface{})
	require.True(t, ok, "header_regexp should exist")

	userAgent, ok := headerRegexp["User-Agent"].(map[string]string)
	require.True(t, ok, "User-Agent should be a map[string]string")

	assert.Equal(t, "bad_user_agent", userAgent["name"])
	assert.Contains(t, userAgent["pattern"], "sqlmap")
	assert.Contains(t, userAgent["pattern"], "nikto")

	assert.Equal(t, "Forbidden - Blocked user agent", route.Handle[0]["body"])
}

func TestSecurityRoutes_VulnScanners(t *testing.T) {
	routes := SecurityRoutes()
	route := routes[5] // Vuln scanners route

	require.Len(t, route.Match, 1)
	pathRegexp, ok := route.Match[0]["path_regexp"].(map[string]string)
	require.True(t, ok)

	assert.Equal(t, "vuln_scanners", pathRegexp["name"])
	assert.Contains(t, pathRegexp["pattern"], "wp-admin")
	assert.Contains(t, pathRegexp["pattern"], "phpmyadmin")
	assert.Contains(t, pathRegexp["pattern"], ".env")

	assert.Equal(t, "Forbidden - Access denied", route.Handle[0]["body"])
}

func TestSecurityRoutesForHost(t *testing.T) {
	hostname := "example.com"
	routes := SecurityRoutesForHost(hostname)

	assert.Len(t, routes, 6, "Expected 6 security routes")

	// All routes should have the host matcher
	for i, route := range routes {
		require.Len(t, route.Match, 1, "Route %d should have 1 matcher set", i)

		hosts, ok := route.Match[0]["host"].([]string)
		require.True(t, ok, "Route %d should have host matcher", i)
		assert.Equal(t, []string{hostname}, hosts, "Route %d should match hostname", i)
	}
}

func TestSecurityRoutes_ValidJSON(t *testing.T) {
	routes := SecurityRoutes()

	// Ensure all routes can be marshaled to valid JSON
	for i, route := range routes {
		data, err := json.Marshal(route)
		require.NoError(t, err, "Route %d should marshal to JSON", i)
		assert.NotEmpty(t, data, "Route %d JSON should not be empty", i)
	}
}

func TestSecurityRoutesForHost_ValidJSON(t *testing.T) {
	routes := SecurityRoutesForHost("test.example.com")

	// Ensure all routes can be marshaled to valid JSON
	for i, route := range routes {
		data, err := json.Marshal(route)
		require.NoError(t, err, "Route %d should marshal to JSON", i)

		// Verify the JSON contains the host
		assert.Contains(t, string(data), "test.example.com")
	}
}
