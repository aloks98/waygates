package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HTTP App and Server Tests
// =============================================================================

func TestNewHTTPApp(t *testing.T) {
	app := NewHTTPApp()
	require.NotNil(t, app)
	assert.NotNil(t, app.Servers)
	assert.Empty(t, app.Servers)
}

func TestNewHTTPServer(t *testing.T) {
	tests := []struct {
		name   string
		listen []string
		want   []string
	}{
		{
			name:   "single listen address",
			listen: []string{":443"},
			want:   []string{":443"},
		},
		{
			name:   "multiple listen addresses",
			listen: []string{":80", ":443"},
			want:   []string{":80", ":443"},
		},
		{
			name:   "no listen address",
			listen: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewHTTPServer(tt.listen...)
			require.NotNil(t, server)
			assert.Equal(t, tt.want, server.Listen)
			assert.NotNil(t, server.Routes)
			assert.Empty(t, server.Routes)
		})
	}
}

func TestNewHTTPRoute(t *testing.T) {
	route := NewHTTPRoute()
	require.NotNil(t, route)
	assert.NotNil(t, route.Match)
	assert.Empty(t, route.Match)
	assert.NotNil(t, route.Handle)
	assert.Empty(t, route.Handle)
}

func TestNewHTTPRouteWithMatch(t *testing.T) {
	hostMatcher := NewHostMatcher("example.com")
	pathMatcher := NewPathMatcher("/api/*")

	route := NewHTTPRouteWithMatch(hostMatcher, pathMatcher)
	require.NotNil(t, route)
	assert.Len(t, route.Match, 2)
	assert.NotNil(t, route.Handle)
	assert.Empty(t, route.Handle)
}

func TestHTTPApp_AddServer(t *testing.T) {
	t.Run("add to existing servers map", func(t *testing.T) {
		app := NewHTTPApp()
		server := NewHTTPServer(":443")

		result := app.AddServer("srv0", server)

		assert.Same(t, app, result)
		assert.Len(t, app.Servers, 1)
		assert.Equal(t, server, app.Servers["srv0"])
	})

	t.Run("add to nil servers map", func(t *testing.T) {
		app := &HTTPApp{Servers: nil}
		server := NewHTTPServer(":443")

		result := app.AddServer("srv0", server)

		assert.Same(t, app, result)
		assert.NotNil(t, app.Servers)
		assert.Len(t, app.Servers, 1)
	})

	t.Run("add multiple servers", func(t *testing.T) {
		app := NewHTTPApp()
		server1 := NewHTTPServer(":443")
		server2 := NewHTTPServer(":80")

		app.AddServer("https", server1).AddServer("http", server2)

		assert.Len(t, app.Servers, 2)
		assert.Equal(t, server1, app.Servers["https"])
		assert.Equal(t, server2, app.Servers["http"])
	})
}

func TestHTTPServer_AddRoute(t *testing.T) {
	server := NewHTTPServer(":443")
	route := NewHTTPRoute()

	result := server.AddRoute(route)

	assert.Same(t, server, result)
	assert.Len(t, server.Routes, 1)
	assert.Equal(t, route, server.Routes[0])
}

func TestHTTPServer_AddRoutes(t *testing.T) {
	server := NewHTTPServer(":443")
	route1 := NewHTTPRoute()
	route2 := NewHTTPRoute()

	result := server.AddRoutes(route1, route2)

	assert.Same(t, server, result)
	assert.Len(t, server.Routes, 2)
}

func TestHTTPServer_WithAutoHTTPS(t *testing.T) {
	server := NewHTTPServer(":443")
	config := &AutoHTTPSConfig{
		Disabled: false,
	}

	result := server.WithAutoHTTPS(config)

	assert.Same(t, server, result)
	assert.Equal(t, config, server.AutoHTTPS)
}

func TestHTTPServer_DisableAutoHTTPS(t *testing.T) {
	server := NewHTTPServer(":443")

	result := server.DisableAutoHTTPS()

	assert.Same(t, server, result)
	require.NotNil(t, server.AutoHTTPS)
	assert.True(t, server.AutoHTTPS.Disabled)
}

func TestHTTPRoute_AddMatch(t *testing.T) {
	route := NewHTTPRoute()
	hostMatcher := NewHostMatcher("example.com")
	pathMatcher := NewPathMatcher("/api/*")

	result := route.AddMatch(hostMatcher, pathMatcher)

	assert.Same(t, route, result)
	assert.Len(t, route.Match, 2)
}

func TestHTTPRoute_AddHandler(t *testing.T) {
	route := NewHTTPRoute()
	handler := HTTPHandler{"handler": "static_response", "status_code": 200}

	result := route.AddHandler(handler)

	assert.Same(t, route, result)
	assert.Len(t, route.Handle, 1)
	assert.Equal(t, handler, route.Handle[0])
}

func TestHTTPRoute_SetTerminal(t *testing.T) {
	tests := []struct {
		name     string
		terminal bool
	}{
		{"set terminal true", true},
		{"set terminal false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := NewHTTPRoute()

			result := route.SetTerminal(tt.terminal)

			assert.Same(t, route, result)
			assert.Equal(t, tt.terminal, route.Terminal)
		})
	}
}

// =============================================================================
// Route Factory Tests
// =============================================================================

func TestNewReverseProxyRoute(t *testing.T) {
	t.Run("with hosts", func(t *testing.T) {
		hosts := []string{"example.com", "api.example.com"}
		upstreams := []*Upstream{{Dial: "localhost:8080"}}

		route := NewReverseProxyRoute(hosts, upstreams)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)

		// Check handler type
		handler := route.Handle[0]
		assert.Equal(t, HandlerReverseProxy, handler["handler"])
	})

	t.Run("without hosts", func(t *testing.T) {
		upstreams := []*Upstream{{Dial: "localhost:8080"}}

		route := NewReverseProxyRoute(nil, upstreams)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})
}

func TestNewRedirectRoute(t *testing.T) {
	t.Run("with hosts", func(t *testing.T) {
		hosts := []string{"old.example.com"}
		targetURL := "https://new.example.com"
		statusCode := 301

		route := NewRedirectRoute(hosts, targetURL, statusCode)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)

		handler := route.Handle[0]
		assert.Equal(t, HandlerStaticResponse, handler["handler"])
		assert.Equal(t, statusCode, handler["status_code"])
	})

	t.Run("without hosts", func(t *testing.T) {
		route := NewRedirectRoute(nil, "https://example.com", 302)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})
}

func TestNewStaticFileRoute(t *testing.T) {
	t.Run("with hosts and index names", func(t *testing.T) {
		hosts := []string{"docs.example.com"}
		rootPath := "/var/www/docs"
		indexNames := []string{"index.html", "default.html"}

		route := NewStaticFileRoute(hosts, rootPath, indexNames)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)

		handler := route.Handle[0]
		assert.Equal(t, HandlerFileServer, handler["handler"])
	})

	t.Run("without index names", func(t *testing.T) {
		route := NewStaticFileRoute([]string{"static.example.com"}, "/var/www/static", nil)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)
	})

	t.Run("without hosts", func(t *testing.T) {
		route := NewStaticFileRoute(nil, "/var/www", []string{"index.html"})

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})
}

func TestNewErrorRoute(t *testing.T) {
	t.Run("with hosts", func(t *testing.T) {
		hosts := []string{"error.example.com"}
		statusCode := 503
		body := "Service Unavailable"

		route := NewErrorRoute(hosts, statusCode, body)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)

		handler := route.Handle[0]
		assert.Equal(t, HandlerStaticResponse, handler["handler"])
		assert.Equal(t, statusCode, handler["status_code"])
		assert.Equal(t, body, handler["body"])
	})

	t.Run("without hosts", func(t *testing.T) {
		route := NewErrorRoute(nil, 500, "Internal Server Error")

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})
}

func TestNewCatchAllRoute(t *testing.T) {
	route := NewCatchAllRoute()

	require.NotNil(t, route)
	assert.Empty(t, route.Match)
	assert.Len(t, route.Handle, 1)

	handler := route.Handle[0]
	assert.Equal(t, HandlerStaticResponse, handler["handler"])
	assert.Equal(t, 404, handler["status_code"])
	assert.Equal(t, "Not Found", handler["body"])
}

func TestNewCatchAllRedirectRoute(t *testing.T) {
	targetURL := "https://home.example.com"

	route := NewCatchAllRedirectRoute(targetURL)

	require.NotNil(t, route)
	assert.Empty(t, route.Match)
	assert.Len(t, route.Handle, 1)

	handler := route.Handle[0]
	assert.Equal(t, HandlerStaticResponse, handler["handler"])
	assert.Equal(t, 302, handler["status_code"])
}

// =============================================================================
// Server Builder Tests
// =============================================================================

func TestBuildDefaultServer(t *testing.T) {
	route := NewHTTPRoute()
	routes := []*HTTPRoute{route}

	server := BuildDefaultServer(routes)

	require.NotNil(t, server)
	assert.Equal(t, []string{":443"}, server.Listen)
	assert.Len(t, server.Routes, 1)
}

func TestBuildHTTPOnlyServer(t *testing.T) {
	route := NewHTTPRoute()
	routes := []*HTTPRoute{route}

	server := BuildHTTPOnlyServer(routes)

	require.NotNil(t, server)
	assert.Equal(t, []string{":80"}, server.Listen)
	assert.Len(t, server.Routes, 1)
	require.NotNil(t, server.AutoHTTPS)
	assert.True(t, server.AutoHTTPS.Disabled)
}

// =============================================================================
// Utility Function Tests
// =============================================================================

func TestListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "empty host",
			host:     "",
			port:     443,
			expected: ":443",
		},
		{
			name:     "with host",
			host:     "0.0.0.0",
			port:     8080,
			expected: "0.0.0.0:8080",
		},
		{
			name:     "localhost",
			host:     "localhost",
			port:     80,
			expected: "localhost:80",
		},
		{
			name:     "ipv4 address",
			host:     "192.168.1.1",
			port:     3000,
			expected: "192.168.1.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ListenAddress(tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGroupRoutesByHost(t *testing.T) {
	t.Run("routes with different hosts", func(t *testing.T) {
		route1 := NewHTTPRoute()
		route1.AddMatch(NewHostMatcher("example.com"))

		route2 := NewHTTPRoute()
		route2.AddMatch(NewHostMatcher("api.example.com"))

		route3 := NewHTTPRoute() // No host matcher

		routes := []*HTTPRoute{route1, route2, route3}

		grouped := GroupRoutesByHost(routes)

		assert.Len(t, grouped["example.com"], 1)
		assert.Len(t, grouped["api.example.com"], 1)
		assert.Len(t, grouped[""], 1)
	})

	t.Run("route with multiple hosts", func(t *testing.T) {
		route := NewHTTPRoute()
		route.AddMatch(NewHostMatcher("example.com", "www.example.com"))

		grouped := GroupRoutesByHost([]*HTTPRoute{route})

		assert.Len(t, grouped["example.com"], 1)
		assert.Len(t, grouped["www.example.com"], 1)
	})

	t.Run("empty routes", func(t *testing.T) {
		grouped := GroupRoutesByHost([]*HTTPRoute{})
		assert.Empty(t, grouped)
	})
}

func TestExtractHosts(t *testing.T) {
	t.Run("with MatchHost type", func(t *testing.T) {
		route := NewHTTPRoute()
		// Add host matcher as MatchHost type
		route.Match = append(route.Match, MatcherSet{
			"host": MatchHost{"example.com", "api.example.com"},
		})

		hosts := extractHosts(route)

		assert.Len(t, hosts, 2)
		assert.Contains(t, hosts, "example.com")
		assert.Contains(t, hosts, "api.example.com")
	})

	t.Run("with string slice type", func(t *testing.T) {
		route := NewHTTPRoute()
		// Add host matcher as []string type
		route.Match = append(route.Match, MatcherSet{
			"host": []string{"static.example.com"},
		})

		hosts := extractHosts(route)

		assert.Len(t, hosts, 1)
		assert.Contains(t, hosts, "static.example.com")
	})

	t.Run("no host matcher", func(t *testing.T) {
		route := NewHTTPRoute()
		route.Match = append(route.Match, MatcherSet{
			"path": []string{"/api/*"},
		})

		hosts := extractHosts(route)

		assert.Empty(t, hosts)
	})

	t.Run("empty match", func(t *testing.T) {
		route := NewHTTPRoute()
		hosts := extractHosts(route)
		assert.Empty(t, hosts)
	})
}

func TestCollectDomainsFromRoutes(t *testing.T) {
	t.Run("collect unique domains", func(t *testing.T) {
		route1 := NewHTTPRoute()
		route1.AddMatch(NewHostMatcher("example.com"))

		route2 := NewHTTPRoute()
		route2.AddMatch(NewHostMatcher("api.example.com"))

		route3 := NewHTTPRoute()
		route3.AddMatch(NewHostMatcher("example.com")) // Duplicate

		routes := []*HTTPRoute{route1, route2, route3}

		domains := CollectDomainsFromRoutes(routes)

		assert.Len(t, domains, 2)
		assert.Contains(t, domains, "example.com")
		assert.Contains(t, domains, "api.example.com")
	})

	t.Run("empty routes", func(t *testing.T) {
		domains := CollectDomainsFromRoutes([]*HTTPRoute{})
		assert.Empty(t, domains)
	})

	t.Run("routes without hosts", func(t *testing.T) {
		route := NewHTTPRoute()
		route.AddMatch(NewPathMatcher("/api/*"))

		domains := CollectDomainsFromRoutes([]*HTTPRoute{route})
		assert.Empty(t, domains)
	})
}
