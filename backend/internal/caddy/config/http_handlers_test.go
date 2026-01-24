package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Upstream Tests
// =============================================================================

func TestNewUpstream(t *testing.T) {
	dial := "localhost:8080"
	upstream := NewUpstream(dial)

	require.NotNil(t, upstream)
	assert.Equal(t, dial, upstream.Dial)
}

func TestNewUpstreamFromHostPort(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "localhost with port",
			host:     "localhost",
			port:     8080,
			expected: "localhost:8080",
		},
		{
			name:     "IP with port",
			host:     "192.168.1.1",
			port:     3000,
			expected: "192.168.1.1:3000",
		},
		{
			name:     "hostname with port",
			host:     "backend.local",
			port:     443,
			expected: "backend.local:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := NewUpstreamFromHostPort(tt.host, tt.port)

			require.NotNil(t, upstream)
			assert.Equal(t, tt.expected, upstream.Dial)
		})
	}
}

func TestFormatDialAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "with port",
			host:     "localhost",
			port:     8080,
			expected: "localhost:8080",
		},
		{
			name:     "zero port returns just host",
			host:     "backend.local",
			port:     0,
			expected: "backend.local",
		},
		{
			name:     "standard HTTP port",
			host:     "example.com",
			port:     80,
			expected: "example.com:80",
		},
		{
			name:     "standard HTTPS port",
			host:     "secure.example.com",
			port:     443,
			expected: "secure.example.com:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDialAddress(tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"positive single digit", 5, "5"},
		{"positive multi digit", 123, "123"},
		{"standard port", 8080, "8080"},
		{"large number", 65535, "65535"},
		{"negative number", -42, "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := itoa(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// ReverseProxyHandler Tests
// =============================================================================

func TestReverseProxyHandler_WithLoadBalancing(t *testing.T) {
	handler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	result := handler.WithLoadBalancing("round_robin")

	assert.Same(t, handler, result)
	require.NotNil(t, handler.LoadBalancing)
	require.NotNil(t, handler.LoadBalancing.SelectionPolicy)
	assert.Equal(t, "round_robin", handler.LoadBalancing.SelectionPolicy.Policy)
}

func TestReverseProxyHandler_WithHealthChecks(t *testing.T) {
	handler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})

	result := handler.WithHealthChecks("/health", "30s", "5s")

	assert.Same(t, handler, result)
	require.NotNil(t, handler.HealthChecks)
	require.NotNil(t, handler.HealthChecks.Active)
	assert.Equal(t, "/health", handler.HealthChecks.Active.Path)
	assert.Equal(t, Duration("30s"), handler.HealthChecks.Active.Interval)
	assert.Equal(t, Duration("5s"), handler.HealthChecks.Active.Timeout)
}

func TestReverseProxyHandler_WithTLSTransport(t *testing.T) {
	t.Run("with insecure skip verify", func(t *testing.T) {
		handler := NewReverseProxyHandler(&Upstream{Dial: "localhost:443"})

		result := handler.WithTLSTransport(true)

		assert.Same(t, handler, result)
		require.NotNil(t, handler.Transport)
		assert.Equal(t, "http", handler.Transport.Protocol)
		require.NotNil(t, handler.Transport.TLS)
		assert.True(t, handler.Transport.TLS.InsecureSkipVerify)
	})

	t.Run("without insecure skip verify", func(t *testing.T) {
		handler := NewReverseProxyHandler(&Upstream{Dial: "localhost:443"})

		result := handler.WithTLSTransport(false)

		assert.Same(t, handler, result)
		require.NotNil(t, handler.Transport)
		require.NotNil(t, handler.Transport.TLS)
		assert.False(t, handler.Transport.TLS.InsecureSkipVerify)
	})
}

func TestReverseProxyHandler_WithHeaders(t *testing.T) {
	handler := NewReverseProxyHandler(&Upstream{Dial: "localhost:8080"})
	request := NewRequestHeaderOps()
	request.SetHeader("X-Custom", "value")

	response := NewRequestHeaderOps()
	response.SetHeader("X-Response", "value")

	result := handler.WithHeaders(request, response)

	assert.Same(t, handler, result)
	require.NotNil(t, handler.Headers)
	assert.Equal(t, request, handler.Headers.Request)
	assert.Equal(t, response, handler.Headers.Response)
}

// =============================================================================
// HeaderOps Tests
// =============================================================================

func TestNewRequestHeaderOps(t *testing.T) {
	ops := NewRequestHeaderOps()

	require.NotNil(t, ops)
	assert.NotNil(t, ops.Set)
	assert.NotNil(t, ops.Add)
	assert.Empty(t, ops.Set)
	assert.Empty(t, ops.Add)
}

func TestHeaderOps_SetHeader(t *testing.T) {
	t.Run("set header with existing map", func(t *testing.T) {
		ops := NewRequestHeaderOps()

		result := ops.SetHeader("X-Custom", "value1", "value2")

		assert.Same(t, ops, result)
		assert.Equal(t, []string{"value1", "value2"}, ops.Set["X-Custom"])
	})

	t.Run("set header with nil map", func(t *testing.T) {
		ops := &HeaderOps{}

		result := ops.SetHeader("X-Custom", "value")

		assert.Same(t, ops, result)
		require.NotNil(t, ops.Set)
		assert.Equal(t, []string{"value"}, ops.Set["X-Custom"])
	})
}

func TestHeaderOps_AddHeader(t *testing.T) {
	t.Run("add header with existing map", func(t *testing.T) {
		ops := NewRequestHeaderOps()

		result := ops.AddHeader("X-Custom", "value1", "value2")

		assert.Same(t, ops, result)
		assert.Equal(t, []string{"value1", "value2"}, ops.Add["X-Custom"])
	})

	t.Run("add header with nil map", func(t *testing.T) {
		ops := &HeaderOps{}

		result := ops.AddHeader("X-Custom", "value")

		assert.Same(t, ops, result)
		require.NotNil(t, ops.Add)
		assert.Equal(t, []string{"value"}, ops.Add["X-Custom"])
	})
}

func TestHeaderOps_DeleteHeader(t *testing.T) {
	ops := NewRequestHeaderOps()

	result := ops.DeleteHeader("X-Remove-Me", "X-Also-Remove")

	assert.Same(t, ops, result)
	assert.Len(t, ops.Delete, 2)
	assert.Contains(t, ops.Delete, "X-Remove-Me")
	assert.Contains(t, ops.Delete, "X-Also-Remove")
}

// =============================================================================
// Handler Factory Tests
// =============================================================================

func TestNewHeadersHandler(t *testing.T) {
	request := NewRequestHeaderOps()
	request.SetHeader("X-Request", "value")

	response := NewRequestHeaderOps()
	response.SetHeader("X-Response", "value")

	handler := NewHeadersHandler(request, response)

	require.NotNil(t, handler)
	assert.Equal(t, HandlerHeaders, handler.Handler)
	assert.Equal(t, request, handler.Request)
	assert.Equal(t, response, handler.Response)
}

func TestNewSubrouteHandler(t *testing.T) {
	route1 := NewHTTPRoute()
	route2 := NewHTTPRoute()

	handler := NewSubrouteHandler(route1, route2)

	require.NotNil(t, handler)
	assert.Equal(t, HandlerSubroute, handler.Handler)
	assert.Len(t, handler.Routes, 2)
	assert.Equal(t, route1, handler.Routes[0])
	assert.Equal(t, route2, handler.Routes[1])
}

// =============================================================================
// Matchers Tests
// =============================================================================

func TestNewPathREMatcher(t *testing.T) {
	name := "image_files"
	pattern := "\\.(jpg|png|gif)$"

	matcher := NewPathREMatcher(name, pattern)

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, "path_regexp")
}

func TestNewHeaderMatcher(t *testing.T) {
	headers := map[string][]string{
		"X-Custom-Header": {"value1", "value2"},
	}

	matcher := NewHeaderMatcher(headers)

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, "header")
}

func TestNewMethodMatcher(t *testing.T) {
	methods := []string{"GET", "POST"}

	matcher := NewMethodMatcher(methods...)

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, "method")
}

func TestNewHostPathMatcher(t *testing.T) {
	host := "example.com"
	paths := []string{"/api/*", "/v1/*"}

	matcher := NewHostPathMatcher(host, paths...)

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, "host")
	assert.Contains(t, matcher, "path")
}

func TestAddHostToMatcher(t *testing.T) {
	matcher := MatcherSet{}

	AddHostToMatcher(matcher, "example.com", "api.example.com")

	assert.Contains(t, matcher, "host")
	hosts := matcher["host"].(MatchHost)
	assert.Contains(t, hosts, "example.com")
	assert.Contains(t, hosts, "api.example.com")
}

func TestNewStaticAssetMatcher(t *testing.T) {
	matcher := NewStaticAssetMatcher()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, "path")
}
