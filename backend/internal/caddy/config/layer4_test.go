package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// =============================================================================
// L4 Listen Address Tests
// =============================================================================

func TestL4ListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		host     string
		port     int
		expected string
	}{
		{
			name:     "TCP with empty host",
			protocol: L4ProtocolTCP,
			host:     "",
			port:     8080,
			expected: "tcp/:8080",
		},
		{
			name:     "TCP with localhost",
			protocol: L4ProtocolTCP,
			host:     "localhost",
			port:     3306,
			expected: "tcp/localhost:3306",
		},
		{
			name:     "TCP with IP address",
			protocol: L4ProtocolTCP,
			host:     "0.0.0.0",
			port:     443,
			expected: "tcp/0.0.0.0:443",
		},
		{
			name:     "UDP with empty host",
			protocol: L4ProtocolUDP,
			host:     "",
			port:     53,
			expected: "udp/:53",
		},
		{
			name:     "UDP with IP address",
			protocol: L4ProtocolUDP,
			host:     "192.168.1.1",
			port:     1194,
			expected: "udp/192.168.1.1:1194",
		},
		{
			name:     "standard SSH port",
			protocol: L4ProtocolTCP,
			host:     "",
			port:     22,
			expected: "tcp/:22",
		},
		{
			name:     "high port number",
			protocol: L4ProtocolTCP,
			host:     "",
			port:     65535,
			expected: "tcp/:65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := L4ListenAddress(tt.protocol, tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestL4TCPListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "empty host",
			host:     "",
			port:     8080,
			expected: "tcp/:8080",
		},
		{
			name:     "with host",
			host:     "127.0.0.1",
			port:     3000,
			expected: "tcp/127.0.0.1:3000",
		},
		{
			name:     "standard MySQL port",
			host:     "",
			port:     3306,
			expected: "tcp/:3306",
		},
		{
			name:     "standard PostgreSQL port",
			host:     "",
			port:     5432,
			expected: "tcp/:5432",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := L4TCPListenAddress(tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestL4UDPListenAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "empty host",
			host:     "",
			port:     53,
			expected: "udp/:53",
		},
		{
			name:     "with host",
			host:     "0.0.0.0",
			port:     1194,
			expected: "udp/0.0.0.0:1194",
		},
		{
			name:     "VPN port",
			host:     "",
			port:     51820,
			expected: "udp/:51820",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := L4UDPListenAddress(tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Layer4 App Tests
// =============================================================================

func TestNewLayer4App(t *testing.T) {
	app := NewLayer4App()

	require.NotNil(t, app)
	assert.NotNil(t, app.Servers)
	assert.Empty(t, app.Servers)
}

func TestLayer4App_AddServer(t *testing.T) {
	t.Run("add to existing servers map", func(t *testing.T) {
		app := NewLayer4App()
		server := NewL4Server("tcp/:8080")

		result := app.AddServer("srv0", server)

		assert.Same(t, app, result)
		assert.Len(t, app.Servers, 1)
		assert.Equal(t, server, app.Servers["srv0"])
	})

	t.Run("add to nil servers map", func(t *testing.T) {
		app := &Layer4App{Servers: nil}
		server := NewL4Server("tcp/:443")

		result := app.AddServer("srv0", server)

		assert.Same(t, app, result)
		require.NotNil(t, app.Servers)
		assert.Len(t, app.Servers, 1)
	})

	t.Run("add multiple servers", func(t *testing.T) {
		app := NewLayer4App()
		server1 := NewL4Server("tcp/:443")
		server2 := NewL4Server("tcp/:8080")

		app.AddServer("https", server1).AddServer("alt", server2)

		assert.Len(t, app.Servers, 2)
		assert.Equal(t, server1, app.Servers["https"])
		assert.Equal(t, server2, app.Servers["alt"])
	})
}

// =============================================================================
// L4 Server Tests
// =============================================================================

func TestNewL4Server(t *testing.T) {
	tests := []struct {
		name   string
		listen []string
		want   []string
	}{
		{
			name:   "single listen address",
			listen: []string{"tcp/:443"},
			want:   []string{"tcp/:443"},
		},
		{
			name:   "multiple listen addresses",
			listen: []string{"tcp/:80", "tcp/:443"},
			want:   []string{"tcp/:80", "tcp/:443"},
		},
		{
			name:   "no listen address",
			listen: nil,
			want:   nil,
		},
		{
			name:   "UDP listen address",
			listen: []string{"udp/:53"},
			want:   []string{"udp/:53"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewL4Server(tt.listen...)

			require.NotNil(t, server)
			assert.Equal(t, tt.want, server.Listen)
			assert.NotNil(t, server.Routes)
			assert.Empty(t, server.Routes)
		})
	}
}

func TestL4Server_AddRoute(t *testing.T) {
	server := NewL4Server("tcp/:8080")
	route := NewL4Route()

	result := server.AddRoute(route)

	assert.Same(t, server, result)
	assert.Len(t, server.Routes, 1)
	assert.Equal(t, route, server.Routes[0])
}

func TestL4Server_AddRoutes(t *testing.T) {
	server := NewL4Server("tcp/:8080")
	route1 := NewL4Route()
	route2 := NewL4Route()

	result := server.AddRoutes(route1, route2)

	assert.Same(t, server, result)
	assert.Len(t, server.Routes, 2)
	assert.Equal(t, route1, server.Routes[0])
	assert.Equal(t, route2, server.Routes[1])
}

// =============================================================================
// L4 Route Tests
// =============================================================================

func TestNewL4Route(t *testing.T) {
	route := NewL4Route()

	require.NotNil(t, route)
	assert.NotNil(t, route.Match)
	assert.Empty(t, route.Match)
	assert.NotNil(t, route.Handle)
	assert.Empty(t, route.Handle)
}

func TestL4Route_AddMatch(t *testing.T) {
	route := NewL4Route()
	tlsMatcher := NewL4TLSMatcher([]string{"example.com"})
	ipMatcher := NewL4RemoteIPMatcher([]string{"10.0.0.0/8"})

	result := route.AddMatch(tlsMatcher, ipMatcher)

	assert.Same(t, route, result)
	assert.Len(t, route.Match, 2)
}

func TestL4Route_AddHandler(t *testing.T) {
	route := NewL4Route()
	handler := ToL4Handler(NewL4ProxyHandler(NewL4Upstream("localhost:8080")))

	result := route.AddHandler(handler)

	assert.Same(t, route, result)
	assert.Len(t, route.Handle, 1)
}

// =============================================================================
// L4 Matcher Tests
// =============================================================================

func TestNewL4TLSMatcher(t *testing.T) {
	t.Run("with SNI hostnames", func(t *testing.T) {
		sni := []string{"example.com", "api.example.com"}
		matcher := NewL4TLSMatcher(sni)

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherTLS)

		tlsMatcher := matcher[L4MatcherTLS].(*L4MatchTLS)
		assert.Equal(t, sni, tlsMatcher.SNI)
	})

	t.Run("with empty SNI", func(t *testing.T) {
		matcher := NewL4TLSMatcher([]string{})

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherTLS)

		tlsMatcher := matcher[L4MatcherTLS].(*L4MatchTLS)
		assert.Empty(t, tlsMatcher.SNI)
	})
}

func TestNewL4TLSMatcherAny(t *testing.T) {
	matcher := NewL4TLSMatcherAny()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherTLS)

	tlsMatcher := matcher[L4MatcherTLS].(*L4MatchTLS)
	assert.Nil(t, tlsMatcher.SNI)
}

func TestNewL4SSHMatcher(t *testing.T) {
	matcher := NewL4SSHMatcher()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherSSH)
}

func TestNewL4PostgresMatcher(t *testing.T) {
	matcher := NewL4PostgresMatcher()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherPostgres)
}

func TestNewL4HTTPMatcher(t *testing.T) {
	t.Run("with hosts", func(t *testing.T) {
		hosts := []string{"example.com", "api.example.com"}
		matcher := NewL4HTTPMatcher(hosts...)

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherHTTP)

		httpMatcher := matcher[L4MatcherHTTP].(*L4MatchHTTP)
		assert.Equal(t, hosts, httpMatcher.Host)
	})

	t.Run("without hosts", func(t *testing.T) {
		matcher := NewL4HTTPMatcher()

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherHTTP)

		httpMatcher := matcher[L4MatcherHTTP].(*L4MatchHTTP)
		assert.Nil(t, httpMatcher.Host)
	})
}

func TestNewL4RDPMatcher(t *testing.T) {
	matcher := NewL4RDPMatcher()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherRDP)
}

func TestNewL4SOCKS5Matcher(t *testing.T) {
	matcher := NewL4SOCKS5Matcher()

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherSocks5)
}

func TestNewL4RemoteIPMatcher(t *testing.T) {
	t.Run("with IP ranges", func(t *testing.T) {
		ranges := []string{"10.0.0.0/8", "192.168.0.0/16"}
		matcher := NewL4RemoteIPMatcher(ranges)

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherRemoteIP)

		ipMatcher := matcher[L4MatcherRemoteIP].(*L4MatchRemoteIP)
		assert.Equal(t, ranges, ipMatcher.Ranges)
	})

	t.Run("with single IP", func(t *testing.T) {
		ranges := []string{"192.168.1.100/32"}
		matcher := NewL4RemoteIPMatcher(ranges)

		require.NotNil(t, matcher)
		ipMatcher := matcher[L4MatcherRemoteIP].(*L4MatchRemoteIP)
		assert.Equal(t, ranges, ipMatcher.Ranges)
	})
}

func TestNewL4RegexpMatcher(t *testing.T) {
	t.Run("with pattern and count", func(t *testing.T) {
		pattern := "^SSH-.*"
		count := 10
		matcher := NewL4RegexpMatcher(pattern, count)

		require.NotNil(t, matcher)
		assert.Contains(t, matcher, L4MatcherRegexp)

		regexpMatcher := matcher[L4MatcherRegexp].(*L4MatchRegexp)
		assert.Equal(t, pattern, regexpMatcher.Pattern)
		assert.Equal(t, count, regexpMatcher.Count)
	})

	t.Run("with zero count", func(t *testing.T) {
		pattern := ".*HTTP.*"
		matcher := NewL4RegexpMatcher(pattern, 0)

		require.NotNil(t, matcher)
		regexpMatcher := matcher[L4MatcherRegexp].(*L4MatchRegexp)
		assert.Equal(t, pattern, regexpMatcher.Pattern)
		assert.Equal(t, 0, regexpMatcher.Count)
	})
}

func TestNewL4NotMatcher(t *testing.T) {
	sshMatcher := NewL4SSHMatcher()
	rdpMatcher := NewL4RDPMatcher()

	matcher := NewL4NotMatcher(sshMatcher, rdpMatcher)

	require.NotNil(t, matcher)
	assert.Contains(t, matcher, L4MatcherNot)

	notMatchers := matcher[L4MatcherNot].(L4MatchNot)
	assert.Len(t, notMatchers, 2)
}

func TestCombineL4Matchers(t *testing.T) {
	t.Run("combine two matchers", func(t *testing.T) {
		tlsMatcher := NewL4TLSMatcher([]string{"example.com"})
		ipMatcher := NewL4RemoteIPMatcher([]string{"10.0.0.0/8"})

		combined := CombineL4Matchers(tlsMatcher, ipMatcher)

		require.NotNil(t, combined)
		assert.Contains(t, combined, L4MatcherTLS)
		assert.Contains(t, combined, L4MatcherRemoteIP)
	})

	t.Run("combine with duplicate keys (first wins)", func(t *testing.T) {
		matcher1 := NewL4TLSMatcher([]string{"first.com"})
		matcher2 := NewL4TLSMatcher([]string{"second.com"})

		combined := CombineL4Matchers(matcher1, matcher2)

		tlsMatcher := combined[L4MatcherTLS].(*L4MatchTLS)
		assert.Equal(t, []string{"first.com"}, tlsMatcher.SNI)
	})

	t.Run("combine empty matchers", func(t *testing.T) {
		combined := CombineL4Matchers()

		require.NotNil(t, combined)
		assert.Empty(t, combined)
	})
}

func TestAddRemoteIPToL4Matcher(t *testing.T) {
	tlsMatcher := NewL4TLSMatcher([]string{"example.com"})
	ranges := []string{"10.0.0.0/8", "172.16.0.0/12"}

	result := AddRemoteIPToL4Matcher(tlsMatcher, ranges)

	// Result should be the same map as the input (modified in place)
	assert.Equal(t, tlsMatcher, result)
	assert.Contains(t, result, L4MatcherTLS)
	assert.Contains(t, result, L4MatcherRemoteIP)

	ipMatcher := result[L4MatcherRemoteIP].(*L4MatchRemoteIP)
	assert.Equal(t, ranges, ipMatcher.Ranges)
}

// =============================================================================
// MapMatcherTypeToL4Matcher Tests
// =============================================================================

func TestMapMatcherTypeToL4Matcher(t *testing.T) {
	tests := []struct {
		name         string
		matcherType  string
		sniHostnames []string
		ipRanges     []string
		regexPattern string
		expectNil    bool
		expectKeys   []string
	}{
		{
			name:        "any without IP ranges",
			matcherType: "any",
			expectNil:   true,
		},
		{
			name:        "any with IP ranges",
			matcherType: "any",
			ipRanges:    []string{"10.0.0.0/8"},
			expectKeys:  []string{L4MatcherRemoteIP},
		},
		{
			name:         "tls with SNI",
			matcherType:  "tls",
			sniHostnames: []string{"example.com"},
			expectKeys:   []string{L4MatcherTLS},
		},
		{
			name:         "tls with SNI and IP ranges",
			matcherType:  "tls",
			sniHostnames: []string{"example.com"},
			ipRanges:     []string{"192.168.0.0/16"},
			expectKeys:   []string{L4MatcherTLS, L4MatcherRemoteIP},
		},
		{
			name:        "ssh",
			matcherType: "ssh",
			expectKeys:  []string{L4MatcherSSH},
		},
		{
			name:        "ssh with IP ranges",
			matcherType: "ssh",
			ipRanges:    []string{"10.0.0.0/8"},
			expectKeys:  []string{L4MatcherSSH, L4MatcherRemoteIP},
		},
		{
			name:        "postgres",
			matcherType: "postgres",
			expectKeys:  []string{L4MatcherPostgres},
		},
		{
			name:        "postgres with IP ranges",
			matcherType: "postgres",
			ipRanges:    []string{"172.16.0.0/12"},
			expectKeys:  []string{L4MatcherPostgres, L4MatcherRemoteIP},
		},
		{
			name:        "http",
			matcherType: "http",
			expectKeys:  []string{L4MatcherHTTP},
		},
		{
			name:        "http with IP ranges",
			matcherType: "http",
			ipRanges:    []string{"10.0.0.0/8"},
			expectKeys:  []string{L4MatcherHTTP, L4MatcherRemoteIP},
		},
		{
			name:        "rdp",
			matcherType: "rdp",
			expectKeys:  []string{L4MatcherRDP},
		},
		{
			name:        "rdp with IP ranges",
			matcherType: "rdp",
			ipRanges:    []string{"192.168.1.0/24"},
			expectKeys:  []string{L4MatcherRDP, L4MatcherRemoteIP},
		},
		{
			name:        "socks5",
			matcherType: "socks5",
			expectKeys:  []string{L4MatcherSocks5},
		},
		{
			name:        "socks5 with IP ranges",
			matcherType: "socks5",
			ipRanges:    []string{"10.0.0.0/8"},
			expectKeys:  []string{L4MatcherSocks5, L4MatcherRemoteIP},
		},
		{
			name:        "remote_ip",
			matcherType: "remote_ip",
			ipRanges:    []string{"10.0.0.0/8", "192.168.0.0/16"},
			expectKeys:  []string{L4MatcherRemoteIP},
		},
		{
			name:         "regexp",
			matcherType:  "regexp",
			regexPattern: "^SSH-.*",
			expectKeys:   []string{L4MatcherRegexp},
		},
		{
			name:         "regexp with IP ranges",
			matcherType:  "regexp",
			regexPattern: ".*HTTP.*",
			ipRanges:     []string{"10.0.0.0/8"},
			expectKeys:   []string{L4MatcherRegexp, L4MatcherRemoteIP},
		},
		{
			name:        "unknown matcher type",
			matcherType: "unknown",
			expectNil:   true,
		},
		{
			name:        "empty matcher type",
			matcherType: "",
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapMatcherTypeToL4Matcher(tt.matcherType, tt.sniHostnames, tt.ipRanges, tt.regexPattern)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				for _, key := range tt.expectKeys {
					assert.Contains(t, result, key, "Expected key %s to be present", key)
				}
			}
		})
	}
}

// =============================================================================
// L4 Handler Tests
// =============================================================================

func TestNewL4ProxyHandler(t *testing.T) {
	t.Run("with single upstream", func(t *testing.T) {
		upstream := NewL4Upstream("localhost:8080")
		handler := NewL4ProxyHandler(upstream)

		require.NotNil(t, handler)
		assert.Equal(t, L4HandlerProxy, handler.Handler)
		assert.Len(t, handler.Upstreams, 1)
		assert.Equal(t, []string{"localhost:8080"}, handler.Upstreams[0].Dial)
	})

	t.Run("with multiple upstreams", func(t *testing.T) {
		upstream1 := NewL4Upstream("backend1:8080")
		upstream2 := NewL4Upstream("backend2:8080")
		handler := NewL4ProxyHandler(upstream1, upstream2)

		require.NotNil(t, handler)
		assert.Len(t, handler.Upstreams, 2)
	})

	t.Run("with no upstreams", func(t *testing.T) {
		handler := NewL4ProxyHandler()

		require.NotNil(t, handler)
		assert.Equal(t, L4HandlerProxy, handler.Handler)
		assert.Empty(t, handler.Upstreams)
	})
}

func TestNewL4Upstream(t *testing.T) {
	dial := "backend.local:3000"
	upstream := NewL4Upstream(dial)

	require.NotNil(t, upstream)
	assert.Equal(t, []string{dial}, upstream.Dial)
}

func TestNewL4UpstreamFromHostPort(t *testing.T) {
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
			host:     "192.168.1.100",
			port:     3306,
			expected: "192.168.1.100:3306",
		},
		{
			name:     "hostname with port",
			host:     "backend.internal",
			port:     443,
			expected: "backend.internal:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := NewL4UpstreamFromHostPort(tt.host, tt.port)

			require.NotNil(t, upstream)
			assert.Equal(t, []string{tt.expected}, upstream.Dial)
		})
	}
}

func TestL4ProxyHandler_WithLoadBalancing(t *testing.T) {
	tests := []struct {
		name   string
		policy string
	}{
		{"round_robin", L4LBRoundRobin},
		{"least_conn", L4LBLeastConn},
		{"random", L4LBRandom},
		{"first", L4LBFirst},
		{"ip_hash", L4LBIPHash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

			result := handler.WithLoadBalancing(tt.policy)

			assert.Same(t, handler, result)
			require.NotNil(t, handler.LoadBalancing)
			require.NotNil(t, handler.LoadBalancing.SelectionPolicy)
			assert.Equal(t, tt.policy, handler.LoadBalancing.SelectionPolicy.Policy)
		})
	}
}

func TestL4ProxyHandler_WithProxyProtocol(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"v1", "v1"},
		{"v2", "v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

			result := handler.WithProxyProtocol(tt.version)

			assert.Same(t, handler, result)
			assert.Equal(t, tt.version, handler.ProxyProtocol)
		})
	}
}

func TestL4ProxyHandler_WithUpstreamTLS(t *testing.T) {
	t.Run("with insecure skip verify", func(t *testing.T) {
		handler := NewL4ProxyHandler(NewL4Upstream("secure-backend:443"))

		result := handler.WithUpstreamTLS("secure-backend", true)

		assert.Same(t, handler, result)
		require.NotNil(t, handler.TLS)
		assert.Equal(t, "secure-backend", handler.TLS.ServerName)
		assert.True(t, handler.TLS.InsecureSkipVerify)
	})

	t.Run("without insecure skip verify", func(t *testing.T) {
		handler := NewL4ProxyHandler(NewL4Upstream("secure-backend:443"))

		result := handler.WithUpstreamTLS("secure-backend", false)

		assert.Same(t, handler, result)
		require.NotNil(t, handler.TLS)
		assert.False(t, handler.TLS.InsecureSkipVerify)
	})
}

func TestL4ProxyHandler_WithTimeouts(t *testing.T) {
	t.Run("all timeouts set", func(t *testing.T) {
		handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

		result := handler.WithTimeouts("10s", "5s", "30s")

		assert.Same(t, handler, result)
		assert.Equal(t, Duration("10s"), handler.DialTimeout)
		assert.Equal(t, Duration("5s"), handler.ConnectTimeout)
		assert.Equal(t, Duration("30s"), handler.IdleTimeout)
	})

	t.Run("partial timeouts", func(t *testing.T) {
		handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

		result := handler.WithTimeouts("10s", "", "")

		assert.Same(t, handler, result)
		assert.Equal(t, Duration("10s"), handler.DialTimeout)
		assert.Equal(t, Duration(""), handler.ConnectTimeout)
		assert.Equal(t, Duration(""), handler.IdleTimeout)
	})

	t.Run("no timeouts", func(t *testing.T) {
		handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

		result := handler.WithTimeouts("", "", "")

		assert.Same(t, handler, result)
		assert.Equal(t, Duration(""), handler.DialTimeout)
	})
}

func TestL4ProxyHandler_WithHealthChecks(t *testing.T) {
	handler := NewL4ProxyHandler(NewL4Upstream("localhost:8080"))

	result := handler.WithHealthChecks(8081, "30s", "5s")

	assert.Same(t, handler, result)
	require.NotNil(t, handler.HealthChecks)
	require.NotNil(t, handler.HealthChecks.Active)
	assert.Equal(t, 8081, handler.HealthChecks.Active.Port)
	assert.Equal(t, Duration("30s"), handler.HealthChecks.Active.Interval)
	assert.Equal(t, Duration("5s"), handler.HealthChecks.Active.Timeout)
}

func TestNewL4TLSHandler(t *testing.T) {
	handler := NewL4TLSHandler()

	require.NotNil(t, handler)
	assert.Equal(t, L4HandlerTLS, handler.Handler)
	assert.Nil(t, handler.ConnectionPolicies)
}

func TestL4TLSHandler_WithConnectionPolicies(t *testing.T) {
	handler := NewL4TLSHandler()
	policy1 := NewL4TLSConnPolicy([]string{"example.com"})
	policy2 := NewL4TLSConnPolicy([]string{"api.example.com"})

	result := handler.WithConnectionPolicies(policy1, policy2)

	assert.Same(t, handler, result)
	assert.Len(t, handler.ConnectionPolicies, 2)
}

func TestNewL4TLSConnPolicy(t *testing.T) {
	t.Run("with SNI", func(t *testing.T) {
		sni := []string{"example.com", "www.example.com"}
		policy := NewL4TLSConnPolicy(sni)

		require.NotNil(t, policy)
		require.NotNil(t, policy.Match)
		assert.Equal(t, sni, policy.Match.SNI)
	})

	t.Run("without SNI", func(t *testing.T) {
		policy := NewL4TLSConnPolicy(nil)

		require.NotNil(t, policy)
		assert.Nil(t, policy.Match)
	})

	t.Run("with empty SNI", func(t *testing.T) {
		policy := NewL4TLSConnPolicy([]string{})

		require.NotNil(t, policy)
		assert.Nil(t, policy.Match)
	})
}

func TestNewL4SubrouteHandler(t *testing.T) {
	route1 := NewL4Route()
	route2 := NewL4Route()

	handler := NewL4SubrouteHandler(route1, route2)

	require.NotNil(t, handler)
	assert.Equal(t, L4HandlerSubroute, handler.Handler)
	assert.Len(t, handler.Routes, 2)
}

func TestNewL4ProxyProtocolHandler(t *testing.T) {
	tests := []struct {
		name    string
		version int
	}{
		{"version 1", 1},
		{"version 2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewL4ProxyProtocolHandler(tt.version)

			require.NotNil(t, handler)
			assert.Equal(t, L4HandlerProxyProt, handler.Handler)
			assert.Equal(t, tt.version, handler.Version)
		})
	}
}

func TestNewL4EchoHandler(t *testing.T) {
	handler := NewL4EchoHandler()

	require.NotNil(t, handler)
	assert.Equal(t, L4HandlerEcho, handler.Handler)
}

// =============================================================================
// ToL4Handler Conversion Tests
// =============================================================================

func TestToL4Handler_ProxyHandler(t *testing.T) {
	t.Run("basic proxy handler", func(t *testing.T) {
		proxyHandler := NewL4ProxyHandler(
			NewL4Upstream("backend1:8080"),
			NewL4Upstream("backend2:8080"),
		)

		result := ToL4Handler(proxyHandler)

		require.NotNil(t, result)
		assert.Equal(t, L4HandlerProxy, result["handler"])
		assert.NotNil(t, result["upstreams"])
	})

	t.Run("proxy handler with all options", func(t *testing.T) {
		proxyHandler := NewL4ProxyHandler(NewL4Upstream("backend:8080"))
		proxyHandler.WithLoadBalancing(L4LBRoundRobin).
			WithProxyProtocol("v2").
			WithUpstreamTLS("backend", true).
			WithTimeouts("10s", "5s", "30s").
			WithHealthChecks(8081, "30s", "5s")
		proxyHandler.FailDuration = "10s"
		proxyHandler.UnhealthyConnFail = 3

		result := ToL4Handler(proxyHandler)

		require.NotNil(t, result)
		assert.Equal(t, L4HandlerProxy, result["handler"])
		assert.NotNil(t, result["load_balancing"])
		assert.Equal(t, "v2", result["proxy_protocol"])
		assert.NotNil(t, result["tls"])
		assert.Equal(t, Duration("10s"), result["dial_timeout"])
		assert.Equal(t, Duration("5s"), result["connect_timeout"])
		assert.Equal(t, Duration("30s"), result["idle_timeout"])
		assert.Equal(t, Duration("10s"), result["fail_duration"])
		assert.Equal(t, 3, result["unhealthy_conn_fail"])
		assert.NotNil(t, result["health_checks"])
	})

	t.Run("proxy handler without optional fields", func(t *testing.T) {
		proxyHandler := NewL4ProxyHandler(NewL4Upstream("backend:8080"))

		result := ToL4Handler(proxyHandler)

		require.NotNil(t, result)
		assert.NotContains(t, result, "load_balancing")
		assert.NotContains(t, result, "proxy_protocol")
		assert.NotContains(t, result, "tls")
		assert.NotContains(t, result, "dial_timeout")
		assert.NotContains(t, result, "health_checks")
	})
}

func TestToL4Handler_TLSHandler(t *testing.T) {
	t.Run("basic TLS handler", func(t *testing.T) {
		tlsHandler := NewL4TLSHandler()

		result := ToL4Handler(tlsHandler)

		require.NotNil(t, result)
		assert.Equal(t, L4HandlerTLS, result["handler"])
		assert.NotContains(t, result, "connection_policies")
	})

	t.Run("TLS handler with connection policies", func(t *testing.T) {
		tlsHandler := NewL4TLSHandler()
		tlsHandler.WithConnectionPolicies(
			NewL4TLSConnPolicy([]string{"example.com"}),
		)

		result := ToL4Handler(tlsHandler)

		require.NotNil(t, result)
		assert.Equal(t, L4HandlerTLS, result["handler"])
		assert.Contains(t, result, "connection_policies")
	})
}

func TestToL4Handler_SubrouteHandler(t *testing.T) {
	route := NewL4Route()
	subrouteHandler := NewL4SubrouteHandler(route)

	result := ToL4Handler(subrouteHandler)

	require.NotNil(t, result)
	assert.Equal(t, L4HandlerSubroute, result["handler"])
	assert.Contains(t, result, "routes")
}

func TestToL4Handler_ProxyProtocolHandler(t *testing.T) {
	t.Run("basic proxy protocol handler", func(t *testing.T) {
		handler := NewL4ProxyProtocolHandler(2)

		result := ToL4Handler(handler)

		require.NotNil(t, result)
		assert.Equal(t, L4HandlerProxyProt, result["handler"])
		assert.Equal(t, 2, result["version"])
	})

	t.Run("proxy protocol handler with all options", func(t *testing.T) {
		handler := &L4ProxyProtocolHandler{
			Handler: L4HandlerProxyProt,
			Version: 2,
			Timeout: "5s",
			Allow:   []string{"10.0.0.0/8"},
		}

		result := ToL4Handler(handler)

		require.NotNil(t, result)
		assert.Equal(t, Duration("5s"), result["timeout"])
		assert.Contains(t, result, "allow")
	})

	t.Run("proxy protocol handler with zero version", func(t *testing.T) {
		handler := &L4ProxyProtocolHandler{
			Handler: L4HandlerProxyProt,
			Version: 0,
		}

		result := ToL4Handler(handler)

		require.NotNil(t, result)
		assert.NotContains(t, result, "version")
	})
}

func TestToL4Handler_EchoHandler(t *testing.T) {
	echoHandler := NewL4EchoHandler()

	result := ToL4Handler(echoHandler)

	require.NotNil(t, result)
	assert.Equal(t, L4HandlerEcho, result["handler"])
}

func TestToL4Handler_UnknownType(t *testing.T) {
	result := ToL4Handler("invalid")

	assert.Nil(t, result)
}

func TestToL4Handler_NilHandler(t *testing.T) {
	result := ToL4Handler(nil)

	assert.Nil(t, result)
}

// =============================================================================
// Route Factory Tests
// =============================================================================

func TestNewL4ProxyRoute(t *testing.T) {
	t.Run("with single upstream", func(t *testing.T) {
		upstreams := []*L4Upstream{NewL4Upstream("backend:8080")}

		route := NewL4ProxyRoute(upstreams)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})

	t.Run("with multiple upstreams", func(t *testing.T) {
		upstreams := []*L4Upstream{
			NewL4Upstream("backend1:8080"),
			NewL4Upstream("backend2:8080"),
		}

		route := NewL4ProxyRoute(upstreams)

		require.NotNil(t, route)
		assert.Len(t, route.Handle, 1)

		handler := route.Handle[0]
		assert.Equal(t, L4HandlerProxy, handler["handler"])
	})
}

func TestNewL4TLSProxyRoute(t *testing.T) {
	t.Run("with SNI hostnames", func(t *testing.T) {
		sniHostnames := []string{"example.com", "api.example.com"}
		upstreams := []*L4Upstream{NewL4Upstream("backend:443")}

		route := NewL4TLSProxyRoute(sniHostnames, upstreams)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 1)

		// Check matcher
		assert.Contains(t, route.Match[0], L4MatcherTLS)
	})

	t.Run("without SNI hostnames", func(t *testing.T) {
		upstreams := []*L4Upstream{NewL4Upstream("backend:443")}

		route := NewL4TLSProxyRoute(nil, upstreams)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 1)
	})

	t.Run("with empty SNI hostnames", func(t *testing.T) {
		upstreams := []*L4Upstream{NewL4Upstream("backend:443")}

		route := NewL4TLSProxyRoute([]string{}, upstreams)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
	})
}

func TestNewL4TLSTerminateRoute(t *testing.T) {
	t.Run("with SNI hostnames", func(t *testing.T) {
		sniHostnames := []string{"secure.example.com"}
		upstreams := []*L4Upstream{NewL4Upstream("backend:8080")}

		route := NewL4TLSTerminateRoute(sniHostnames, upstreams)

		require.NotNil(t, route)
		assert.Len(t, route.Match, 1)
		assert.Len(t, route.Handle, 2)

		// First handler should be TLS
		assert.Equal(t, L4HandlerTLS, route.Handle[0]["handler"])
		// Second handler should be proxy
		assert.Equal(t, L4HandlerProxy, route.Handle[1]["handler"])
	})

	t.Run("without SNI hostnames", func(t *testing.T) {
		upstreams := []*L4Upstream{NewL4Upstream("backend:8080")}

		route := NewL4TLSTerminateRoute(nil, upstreams)

		require.NotNil(t, route)
		assert.Empty(t, route.Match)
		assert.Len(t, route.Handle, 2)
	})
}

func TestGroupL4RoutesByPort(t *testing.T) {
	t.Run("group routes by different ports", func(t *testing.T) {
		route1 := NewL4Route()
		route2 := NewL4Route()
		route3 := NewL4Route()

		routes := map[int][]*L4Route{
			443:  {route1},
			8080: {route2, route3},
		}

		servers := GroupL4RoutesByPort(routes)

		require.NotNil(t, servers)
		assert.Len(t, servers, 2)

		assert.Len(t, servers[443].Routes, 1)
		assert.Len(t, servers[8080].Routes, 2)
	})

	t.Run("empty routes map", func(t *testing.T) {
		routes := map[int][]*L4Route{}

		servers := GroupL4RoutesByPort(routes)

		require.NotNil(t, servers)
		assert.Empty(t, servers)
	})
}

// =============================================================================
// MapLBPolicyToL4 Tests
// =============================================================================

func TestMapLBPolicyToL4(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected string
	}{
		{"round_robin", "round_robin", L4LBRoundRobin},
		{"least_conn", "least_conn", L4LBLeastConn},
		{"random", "random", L4LBRandom},
		{"first", "first", L4LBFirst},
		{"ip_hash", "ip_hash", L4LBIPHash},
		{"unknown defaults to round_robin", "unknown", L4LBRoundRobin},
		{"empty defaults to round_robin", "", L4LBRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapLBPolicyToL4(tt.policy)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Protocol Constants Tests
// =============================================================================

func TestL4ProtocolConstants(t *testing.T) {
	assert.Equal(t, "tcp", L4ProtocolTCP)
	assert.Equal(t, "udp", L4ProtocolUDP)
}

func TestL4MatcherConstants(t *testing.T) {
	assert.Equal(t, "tls", L4MatcherTLS)
	assert.Equal(t, "ssh", L4MatcherSSH)
	assert.Equal(t, "postgres", L4MatcherPostgres)
	assert.Equal(t, "http", L4MatcherHTTP)
	assert.Equal(t, "rdp", L4MatcherRDP)
	assert.Equal(t, "socks5", L4MatcherSocks5)
	assert.Equal(t, "remote_ip", L4MatcherRemoteIP)
	assert.Equal(t, "regexp", L4MatcherRegexp)
	assert.Equal(t, "not", L4MatcherNot)
}

func TestL4HandlerConstants(t *testing.T) {
	assert.Equal(t, "proxy", L4HandlerProxy)
	assert.Equal(t, "tls", L4HandlerTLS)
	assert.Equal(t, "subroute", L4HandlerSubroute)
	assert.Equal(t, "echo", L4HandlerEcho)
	assert.Equal(t, "proxy_protocol", L4HandlerProxyProt)
}

func TestL4LBPolicyConstants(t *testing.T) {
	assert.Equal(t, "round_robin", L4LBRoundRobin)
	assert.Equal(t, "least_conn", L4LBLeastConn)
	assert.Equal(t, "random", L4LBRandom)
	assert.Equal(t, "first", L4LBFirst)
	assert.Equal(t, "ip_hash", L4LBIPHash)
}

// =============================================================================
// L4Builder Tests
// =============================================================================

func TestNewL4Builder(t *testing.T) {
	t.Run("with logger", func(t *testing.T) {
		logger := zap.NewNop()
		builder := NewL4Builder(logger)

		require.NotNil(t, builder)
		assert.NotNil(t, builder.logger)
	})

	t.Run("with nil logger", func(t *testing.T) {
		builder := NewL4Builder(nil)

		require.NotNil(t, builder)
		assert.NotNil(t, builder.logger)
	})
}

func TestL4Builder_BuildL4Config(t *testing.T) {
	builder := NewL4Builder(nil)

	t.Run("empty proxies", func(t *testing.T) {
		app, err := builder.BuildL4Config([]models.L4Proxy{})

		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Empty(t, app.Servers)
	})

	t.Run("single active proxy with routes", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "mysql",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 3306,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:                  1,
						MatcherType:         models.L4MatcherAny,
						Upstreams:           models.L4UpstreamSlice{{Host: "mysql-server", Port: 3306}},
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)

		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Len(t, app.Servers, 1)

		// Check server name format
		serverName := "mysql_tcp_3306"
		server, exists := app.Servers[serverName]
		assert.True(t, exists)
		assert.Len(t, server.Routes, 1)
		assert.Equal(t, []string{"tcp/:3306"}, server.Listen)
	})

	t.Run("skip inactive proxies", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "inactive",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 8080,
				IsActive:   false,
				Routes: []models.L4Route{
					{
						ID:                  1,
						MatcherType:         models.L4MatcherAny,
						Upstreams:           models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)

		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Empty(t, app.Servers)
	})

	t.Run("skip proxies with no routes", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "empty",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 8080,
				IsActive:   true,
				Routes:     []models.L4Route{},
			},
		}

		app, err := builder.BuildL4Config(proxies)

		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Empty(t, app.Servers)
	})

	t.Run("multiple proxies on different ports", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "mysql",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 3306,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:                  1,
						MatcherType:         models.L4MatcherAny,
						Upstreams:           models.L4UpstreamSlice{{Host: "mysql-server", Port: 3306}},
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					},
				},
			},
			{
				ID:         2,
				Name:       "postgres",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 5432,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:                  2,
						MatcherType:         models.L4MatcherPostgres,
						Upstreams:           models.L4UpstreamSlice{{Host: "pg-server", Port: 5432}},
						LoadBalancingPolicy: models.L4LoadBalancingLeastConn,
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)

		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Len(t, app.Servers, 2)
	})

	t.Run("UDP proxy", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "dns",
				Protocol:   models.L4ProtocolUDP,
				ListenPort: 53,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:                  1,
						MatcherType:         models.L4MatcherAny,
						Upstreams:           models.L4UpstreamSlice{{Host: "dns-server", Port: 53}},
						LoadBalancingPolicy: models.L4LoadBalancingRoundRobin,
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)

		require.NoError(t, err)
		require.NotNil(t, app)

		server := app.Servers["dns_udp_53"]
		require.NotNil(t, server)
		assert.Equal(t, []string{"udp/:53"}, server.Listen)
	})
}

func TestL4Builder_BuildL4Config_TLSPassthrough(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "tls-proxy",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 443,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:             1,
					MatcherType:    models.L4MatcherTLS,
					SNIHostnames:   []string{"example.com", "api.example.com"},
					Upstreams:      models.L4UpstreamSlice{{Host: "backend", Port: 443}},
					TLSPassthrough: true,
					TLSTerminate:   false,
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Len(t, app.Servers, 1)

	server := app.Servers["tls-proxy_tcp_443"]
	require.NotNil(t, server)
	assert.Len(t, server.Routes, 1)

	// Route should have TLS matcher and proxy handler (no TLS termination handler)
	route := server.Routes[0]
	assert.Len(t, route.Match, 1)
	assert.Contains(t, route.Match[0], L4MatcherTLS)
	assert.Len(t, route.Handle, 1)
	assert.Equal(t, L4HandlerProxy, route.Handle[0]["handler"])
}

func TestL4Builder_BuildL4Config_TLSTerminate(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "tls-terminate",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 443,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:             1,
					MatcherType:    models.L4MatcherTLS,
					SNIHostnames:   []string{"secure.example.com"},
					Upstreams:      models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
					TLSPassthrough: false,
					TLSTerminate:   true,
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["tls-terminate_tcp_443"]
	require.NotNil(t, server)

	// Route should have TLS matcher, TLS handler, and proxy handler
	route := server.Routes[0]
	assert.Len(t, route.Match, 1)
	assert.Len(t, route.Handle, 2)
	assert.Equal(t, L4HandlerTLS, route.Handle[0]["handler"])
	assert.Equal(t, L4HandlerProxy, route.Handle[1]["handler"])
}

func TestL4Builder_BuildL4Config_MultipleUpstreams(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "loadbalanced",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 8080,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:          1,
					MatcherType: models.L4MatcherAny,
					Upstreams: models.L4UpstreamSlice{
						{Host: "backend1", Port: 8080},
						{Host: "backend2", Port: 8080},
						{Host: "backend3", Port: 8080},
					},
					LoadBalancingPolicy: models.L4LoadBalancingLeastConn,
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["loadbalanced_tcp_8080"]
	require.NotNil(t, server)

	route := server.Routes[0]
	handler := route.Handle[0]

	// Check upstreams
	upstreams := handler["upstreams"].([]*L4Upstream)
	assert.Len(t, upstreams, 3)
	assert.Equal(t, []string{"backend1:8080"}, upstreams[0].Dial)
	assert.Equal(t, []string{"backend2:8080"}, upstreams[1].Dial)
	assert.Equal(t, []string{"backend3:8080"}, upstreams[2].Dial)

	// Check load balancing
	lb := handler["load_balancing"].(*L4LoadBalancing)
	require.NotNil(t, lb)
	assert.Equal(t, L4LBLeastConn, lb.SelectionPolicy.Policy)
}

func TestL4Builder_BuildL4Config_ProxyProtocol(t *testing.T) {
	builder := NewL4Builder(nil)
	v2 := "v2"

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "proxy-protocol",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 8080,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:                   1,
					MatcherType:          models.L4MatcherAny,
					Upstreams:            models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
					ProxyProtocolVersion: &v2,
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["proxy-protocol_tcp_8080"]
	route := server.Routes[0]
	handler := route.Handle[0]

	assert.Equal(t, "v2", handler["proxy_protocol"])
}

func TestL4Builder_BuildL4Config_RoutePriority(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "priority-test",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 443,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:          3,
					Priority:    20,
					MatcherType: models.L4MatcherAny,
					Upstreams:   models.L4UpstreamSlice{{Host: "default", Port: 443}},
				},
				{
					ID:           1,
					Priority:     10,
					MatcherType:  models.L4MatcherTLS,
					SNIHostnames: []string{"api.example.com"},
					Upstreams:    models.L4UpstreamSlice{{Host: "api", Port: 443}},
				},
				{
					ID:           2,
					Priority:     5,
					MatcherType:  models.L4MatcherTLS,
					SNIHostnames: []string{"example.com"},
					Upstreams:    models.L4UpstreamSlice{{Host: "main", Port: 443}},
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["priority-test_tcp_443"]
	require.NotNil(t, server)
	assert.Len(t, server.Routes, 3)

	// Routes should be sorted by priority (lowest first)
	// Priority 5, then 10, then 20
	// First route (priority 5) should route to "main"
	upstreams1 := server.Routes[0].Handle[0]["upstreams"].([]*L4Upstream)
	assert.Equal(t, []string{"main:443"}, upstreams1[0].Dial)

	// Second route (priority 10) should route to "api"
	upstreams2 := server.Routes[1].Handle[0]["upstreams"].([]*L4Upstream)
	assert.Equal(t, []string{"api:443"}, upstreams2[0].Dial)

	// Third route (priority 20) should route to "default"
	upstreams3 := server.Routes[2].Handle[0]["upstreams"].([]*L4Upstream)
	assert.Equal(t, []string{"default:443"}, upstreams3[0].Dial)
}

func TestL4Builder_BuildL4Config_AllMatcherTypes(t *testing.T) {
	builder := NewL4Builder(nil)
	regexPattern := "^SSH-.*"

	tests := []struct {
		name        string
		matcherType string
		expectKey   string
	}{
		{"any matcher", models.L4MatcherAny, ""},
		{"ssh matcher", models.L4MatcherSSH, L4MatcherSSH},
		{"postgres matcher", models.L4MatcherPostgres, L4MatcherPostgres},
		{"http matcher", models.L4MatcherHTTP, L4MatcherHTTP},
		{"rdp matcher", models.L4MatcherRDP, L4MatcherRDP},
		{"socks5 matcher", models.L4MatcherSocks5, L4MatcherSocks5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxies := []models.L4Proxy{
				{
					ID:         1,
					Name:       "test",
					Protocol:   models.L4ProtocolTCP,
					ListenPort: 8080,
					IsActive:   true,
					Routes: []models.L4Route{
						{
							ID:          1,
							MatcherType: tt.matcherType,
							Upstreams:   models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
						},
					},
				},
			}

			app, err := builder.BuildL4Config(proxies)

			require.NoError(t, err)
			require.NotNil(t, app)
			server := app.Servers["test_tcp_8080"]
			require.NotNil(t, server)

			route := server.Routes[0]
			if tt.expectKey == "" {
				assert.Empty(t, route.Match)
			} else {
				require.Len(t, route.Match, 1)
				assert.Contains(t, route.Match[0], tt.expectKey)
			}
		})
	}

	// Test TLS matcher separately (requires SNI hostnames)
	t.Run("tls matcher", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "tls-test",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 443,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:           1,
						MatcherType:  models.L4MatcherTLS,
						SNIHostnames: []string{"example.com"},
						Upstreams:    models.L4UpstreamSlice{{Host: "backend", Port: 443}},
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)
		require.NoError(t, err)

		server := app.Servers["tls-test_tcp_443"]
		route := server.Routes[0]
		require.Len(t, route.Match, 1)
		assert.Contains(t, route.Match[0], L4MatcherTLS)
	})

	// Test regexp matcher separately (requires pattern)
	t.Run("regexp matcher", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "regexp-test",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 22,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:           1,
						MatcherType:  models.L4MatcherRegexp,
						RegexPattern: &regexPattern,
						Upstreams:    models.L4UpstreamSlice{{Host: "ssh-server", Port: 22}},
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)
		require.NoError(t, err)

		server := app.Servers["regexp-test_tcp_22"]
		route := server.Routes[0]
		require.Len(t, route.Match, 1)
		assert.Contains(t, route.Match[0], L4MatcherRegexp)
	})

	// Test remote_ip matcher
	t.Run("remote_ip matcher", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "ip-test",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 8080,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:              1,
						MatcherType:     models.L4MatcherRemoteIP,
						AllowedIPRanges: []string{"10.0.0.0/8"},
						Upstreams:       models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
					},
				},
			},
		}

		app, err := builder.BuildL4Config(proxies)
		require.NoError(t, err)

		server := app.Servers["ip-test_tcp_8080"]
		route := server.Routes[0]
		require.Len(t, route.Match, 1)
		assert.Contains(t, route.Match[0], L4MatcherRemoteIP)
	})
}

func TestL4Builder_BuildL4Config_MatcherWithIPRanges(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "ssh-restricted",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 22,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:              1,
					MatcherType:     models.L4MatcherSSH,
					AllowedIPRanges: []string{"10.0.0.0/8", "192.168.0.0/16"},
					Upstreams:       models.L4UpstreamSlice{{Host: "ssh-server", Port: 22}},
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["ssh-restricted_tcp_22"]
	route := server.Routes[0]

	// Matcher should contain both SSH and remote_ip matchers
	require.Len(t, route.Match, 1)
	assert.Contains(t, route.Match[0], L4MatcherSSH)
	assert.Contains(t, route.Match[0], L4MatcherRemoteIP)
}

func TestL4Builder_BuildL4Routes(t *testing.T) {
	builder := NewL4Builder(nil)

	t.Run("inactive proxy returns nil", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:       1,
			Name:     "inactive",
			IsActive: false,
			Routes: []models.L4Route{
				{
					ID:        1,
					Upstreams: models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
				},
			},
		}

		routes, err := builder.BuildL4Routes(proxy)

		require.NoError(t, err)
		assert.Nil(t, routes)
	})

	t.Run("proxy with no routes returns nil", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:       1,
			Name:     "empty",
			IsActive: true,
			Routes:   []models.L4Route{},
		}

		routes, err := builder.BuildL4Routes(proxy)

		require.NoError(t, err)
		assert.Nil(t, routes)
	})

	t.Run("builds routes successfully", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:       1,
			Name:     "test",
			IsActive: true,
			Routes: []models.L4Route{
				{
					ID:        1,
					Priority:  10,
					Upstreams: models.L4UpstreamSlice{{Host: "backend1", Port: 8080}},
				},
				{
					ID:        2,
					Priority:  5,
					Upstreams: models.L4UpstreamSlice{{Host: "backend2", Port: 8080}},
				},
			},
		}

		routes, err := builder.BuildL4Routes(proxy)

		require.NoError(t, err)
		assert.Len(t, routes, 2)

		// Routes should be sorted by priority
		upstreams1 := routes[0].Handle[0]["upstreams"].([]*L4Upstream)
		assert.Equal(t, []string{"backend2:8080"}, upstreams1[0].Dial)
	})
}

func TestL4Builder_BuildL4Server(t *testing.T) {
	builder := NewL4Builder(nil)

	t.Run("inactive proxy returns nil", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:       1,
			Name:     "inactive",
			IsActive: false,
		}

		server, err := builder.BuildL4Server(proxy)

		require.NoError(t, err)
		assert.Nil(t, server)
	})

	t.Run("proxy with no routes returns nil", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:       1,
			Name:     "empty",
			IsActive: true,
			Routes:   []models.L4Route{},
		}

		server, err := builder.BuildL4Server(proxy)

		require.NoError(t, err)
		assert.Nil(t, server)
	})

	t.Run("builds server successfully", func(t *testing.T) {
		proxy := &models.L4Proxy{
			ID:         1,
			Name:       "test",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 8080,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:        1,
					Upstreams: models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
				},
			},
		}

		server, err := builder.BuildL4Server(proxy)

		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Equal(t, []string{"tcp/:8080"}, server.Listen)
		assert.Len(t, server.Routes, 1)
	})
}

func TestL4Builder_MergeL4Servers(t *testing.T) {
	builder := NewL4Builder(nil)

	t.Run("empty proxies", func(t *testing.T) {
		servers, err := builder.MergeL4Servers([]models.L4Proxy{})

		require.NoError(t, err)
		assert.Empty(t, servers)
	})

	t.Run("skip inactive proxies", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "inactive",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 8080,
				IsActive:   false,
				Routes: []models.L4Route{
					{
						ID:        1,
						Upstreams: models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
					},
				},
			},
		}

		servers, err := builder.MergeL4Servers(proxies)

		require.NoError(t, err)
		assert.Empty(t, servers)
	})

	t.Run("merge proxies on same port", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "proxy1",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 443,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:           1,
						MatcherType:  models.L4MatcherTLS,
						SNIHostnames: []string{"example.com"},
						Upstreams:    models.L4UpstreamSlice{{Host: "backend1", Port: 443}},
					},
				},
			},
			{
				ID:         2,
				Name:       "proxy2",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 443,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:           2,
						MatcherType:  models.L4MatcherTLS,
						SNIHostnames: []string{"api.example.com"},
						Upstreams:    models.L4UpstreamSlice{{Host: "backend2", Port: 443}},
					},
				},
			},
		}

		servers, err := builder.MergeL4Servers(proxies)

		require.NoError(t, err)
		assert.Len(t, servers, 1)

		server := servers["tcp:443"]
		require.NotNil(t, server)
		assert.Len(t, server.Routes, 2)
		assert.Equal(t, []string{"tcp/:443"}, server.Listen)
	})

	t.Run("separate servers for different ports", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "mysql",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 3306,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:        1,
						Upstreams: models.L4UpstreamSlice{{Host: "mysql", Port: 3306}},
					},
				},
			},
			{
				ID:         2,
				Name:       "postgres",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 5432,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:        2,
						Upstreams: models.L4UpstreamSlice{{Host: "postgres", Port: 5432}},
					},
				},
			},
		}

		servers, err := builder.MergeL4Servers(proxies)

		require.NoError(t, err)
		assert.Len(t, servers, 2)
		assert.Contains(t, servers, "tcp:3306")
		assert.Contains(t, servers, "tcp:5432")
	})

	t.Run("separate servers for different protocols", func(t *testing.T) {
		proxies := []models.L4Proxy{
			{
				ID:         1,
				Name:       "tcp-service",
				Protocol:   models.L4ProtocolTCP,
				ListenPort: 8080,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:        1,
						Upstreams: models.L4UpstreamSlice{{Host: "tcp-backend", Port: 8080}},
					},
				},
			},
			{
				ID:         2,
				Name:       "udp-service",
				Protocol:   models.L4ProtocolUDP,
				ListenPort: 8080,
				IsActive:   true,
				Routes: []models.L4Route{
					{
						ID:        2,
						Upstreams: models.L4UpstreamSlice{{Host: "udp-backend", Port: 8080}},
					},
				},
			},
		}

		servers, err := builder.MergeL4Servers(proxies)

		require.NoError(t, err)
		assert.Len(t, servers, 2)
		assert.Contains(t, servers, "tcp:8080")
		assert.Contains(t, servers, "udp:8080")

		assert.Equal(t, []string{"tcp/:8080"}, servers["tcp:8080"].Listen)
		assert.Equal(t, []string{"udp/:8080"}, servers["udp:8080"].Listen)
	})
}

func TestL4Builder_BuildL4Config_RouteWithNoUpstreams(t *testing.T) {
	builder := NewL4Builder(nil)

	proxies := []models.L4Proxy{
		{
			ID:         1,
			Name:       "test",
			Protocol:   models.L4ProtocolTCP,
			ListenPort: 8080,
			IsActive:   true,
			Routes: []models.L4Route{
				{
					ID:        1,
					Upstreams: models.L4UpstreamSlice{}, // Empty upstreams
				},
				{
					ID:        2,
					Upstreams: models.L4UpstreamSlice{{Host: "backend", Port: 8080}},
				},
			},
		},
	}

	app, err := builder.BuildL4Config(proxies)

	require.NoError(t, err)
	require.NotNil(t, app)

	server := app.Servers["test_tcp_8080"]
	require.NotNil(t, server)

	// Only the route with upstreams should be included
	assert.Len(t, server.Routes, 1)
}
