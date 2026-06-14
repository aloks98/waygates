package models

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// L4Proxy Validation Tests
// =============================================================================

func TestL4Proxy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		proxy       L4Proxy
		expectedErr error
	}{
		{
			name: "success - valid TCP proxy",
			proxy: L4Proxy{
				Name:       "Test TCP Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 8080,
			},
			expectedErr: nil,
		},
		{
			name: "success - valid UDP proxy",
			proxy: L4Proxy{
				Name:       "Test UDP Proxy",
				Protocol:   L4ProtocolUDP,
				ListenPort: 53,
			},
			expectedErr: nil,
		},
		{
			name: "success - valid proxy with routes",
			proxy: L4Proxy{
				Name:       "Proxy with Routes",
				Protocol:   L4ProtocolTCP,
				ListenPort: 443,
				Routes: []L4Route{
					{
						MatcherType:         L4MatcherAny,
						LoadBalancingPolicy: L4LoadBalancingRoundRobin,
						Upstreams: L4UpstreamSlice{
							{Host: "backend.local", Port: 8080},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid proxy with multiple routes",
			proxy: L4Proxy{
				Name:       "Multi-route Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 443,
				Routes: []L4Route{
					{
						MatcherType:         L4MatcherTLS,
						LoadBalancingPolicy: L4LoadBalancingRoundRobin,
						SNIHostnames:        pq.StringArray{"example.com"},
						Upstreams: L4UpstreamSlice{
							{Host: "backend1.local", Port: 8080},
						},
					},
					{
						MatcherType:         L4MatcherAny,
						LoadBalancingPolicy: L4LoadBalancingLeastConn,
						Upstreams: L4UpstreamSlice{
							{Host: "backend2.local", Port: 8081},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - port at lower boundary (1)",
			proxy: L4Proxy{
				Name:       "Min Port Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 1,
			},
			expectedErr: nil,
		},
		{
			name: "success - port at upper boundary (65535)",
			proxy: L4Proxy{
				Name:       "Max Port Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 65535,
			},
			expectedErr: nil,
		},
		{
			name: "success - name at max length (255)",
			proxy: L4Proxy{
				Name:       strings.Repeat("a", 255),
				Protocol:   L4ProtocolTCP,
				ListenPort: 8080,
			},
			expectedErr: nil,
		},
		{
			name: "error - empty name",
			proxy: L4Proxy{
				Name:       "",
				Protocol:   L4ProtocolTCP,
				ListenPort: 8080,
			},
			expectedErr: ErrL4ProxyNameRequired,
		},
		{
			name: "error - name too long (256 chars)",
			proxy: L4Proxy{
				Name:       strings.Repeat("a", 256),
				Protocol:   L4ProtocolTCP,
				ListenPort: 8080,
			},
			expectedErr: ErrL4ProxyNameTooLong,
		},
		{
			name: "error - invalid protocol",
			proxy: L4Proxy{
				Name:       "Invalid Protocol Proxy",
				Protocol:   "sctp",
				ListenPort: 8080,
			},
			expectedErr: ErrL4ProxyInvalidProtocol,
		},
		{
			name: "error - empty protocol",
			proxy: L4Proxy{
				Name:       "No Protocol Proxy",
				Protocol:   "",
				ListenPort: 8080,
			},
			expectedErr: ErrL4ProxyInvalidProtocol,
		},
		{
			name: "error - port below range (0)",
			proxy: L4Proxy{
				Name:       "Zero Port Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 0,
			},
			expectedErr: ErrL4ProxyInvalidPort,
		},
		{
			name: "error - port below range (negative)",
			proxy: L4Proxy{
				Name:       "Negative Port Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: -1,
			},
			expectedErr: ErrL4ProxyInvalidPort,
		},
		{
			name: "error - port above range (65536)",
			proxy: L4Proxy{
				Name:       "Port Too High Proxy",
				Protocol:   L4ProtocolTCP,
				ListenPort: 65536,
			},
			expectedErr: ErrL4ProxyInvalidPort,
		},
		{
			name: "error - proxy with invalid route",
			proxy: L4Proxy{
				Name:       "Proxy with Invalid Route",
				Protocol:   L4ProtocolTCP,
				ListenPort: 8080,
				Routes: []L4Route{
					{
						MatcherType:         "invalid_matcher",
						LoadBalancingPolicy: L4LoadBalancingRoundRobin,
						Upstreams: L4UpstreamSlice{
							{Host: "backend.local", Port: 8080},
						},
					},
				},
			},
			expectedErr: ErrL4RouteInvalidMatcher,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.proxy.Validate()

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

// =============================================================================
// L4Route Validation Tests
// =============================================================================

func TestL4Route_Validate(t *testing.T) {
	t.Parallel()

	regexPattern := "^hello.*"
	emptyRegex := ""

	tests := []struct {
		name        string
		route       L4Route
		expectedErr error
	}{
		{
			name: "success - valid route with any matcher",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with TLS matcher",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        pq.StringArray{"example.com", "*.example.com"},
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 443},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with SSH matcher",
			route: L4Route{
				MatcherType:         L4MatcherSSH,
				LoadBalancingPolicy: L4LoadBalancingFirst,
				Upstreams: L4UpstreamSlice{
					{Host: "ssh-server.local", Port: 22},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with postgres matcher",
			route: L4Route{
				MatcherType:         L4MatcherPostgres,
				LoadBalancingPolicy: L4LoadBalancingLeastConn,
				Upstreams: L4UpstreamSlice{
					{Host: "db.local", Port: 5432},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with HTTP matcher",
			route: L4Route{
				MatcherType:         L4MatcherHTTP,
				LoadBalancingPolicy: L4LoadBalancingRandom,
				Upstreams: L4UpstreamSlice{
					{Host: "web.local", Port: 80},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with RDP matcher",
			route: L4Route{
				MatcherType:         L4MatcherRDP,
				LoadBalancingPolicy: L4LoadBalancingIPHash,
				Upstreams: L4UpstreamSlice{
					{Host: "rdp.local", Port: 3389},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with SOCKS5 matcher",
			route: L4Route{
				MatcherType:         L4MatcherSocks5,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "proxy.local", Port: 1080},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with remote_ip matcher",
			route: L4Route{
				MatcherType:         L4MatcherRemoteIP,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				AllowedIPRanges:     pq.StringArray{"10.0.0.0/8", "192.168.1.0/24"},
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with regexp matcher",
			route: L4Route{
				MatcherType:         L4MatcherRegexp,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				RegexPattern:        &regexPattern,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - valid route with multiple upstreams",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "backend1.local", Port: 8080},
					{Host: "backend2.local", Port: 8081},
					{Host: "backend3.local", Port: 8082},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - route with TLS terminate (passthrough off)",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        pq.StringArray{"example.com"},
				TLSTerminate:        true,
				TLSPassthrough:      false,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: nil,
		},
		{
			name: "success - route with TLS passthrough (terminate off)",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        pq.StringArray{"example.com"},
				TLSTerminate:        false,
				TLSPassthrough:      true,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 443},
				},
			},
			expectedErr: nil,
		},
		{
			name: "error - invalid matcher type",
			route: L4Route{
				MatcherType:         "invalid_matcher",
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteInvalidMatcher,
		},
		{
			name: "error - empty matcher type",
			route: L4Route{
				MatcherType:         "",
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteInvalidMatcher,
		},
		{
			name: "error - invalid load balancing policy",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: "weighted",
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteInvalidLBPolicy,
		},
		{
			name: "error - empty load balancing policy",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: "",
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteInvalidLBPolicy,
		},
		{
			name: "error - empty upstreams slice",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams:           L4UpstreamSlice{},
			},
			expectedErr: ErrL4RouteUpstreamsRequired,
		},
		{
			name: "error - nil upstreams",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams:           nil,
			},
			expectedErr: ErrL4RouteUpstreamsRequired,
		},
		{
			name: "error - TLS terminate and passthrough both enabled",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        pq.StringArray{"example.com"},
				TLSTerminate:        true,
				TLSPassthrough:      true,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteTLSConflict,
		},
		{
			name: "error - regexp matcher without pattern",
			route: L4Route{
				MatcherType:         L4MatcherRegexp,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				RegexPattern:        nil,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteRegexRequired,
		},
		{
			name: "error - regexp matcher with empty pattern",
			route: L4Route{
				MatcherType:         L4MatcherRegexp,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				RegexPattern:        &emptyRegex,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			},
			expectedErr: ErrL4RouteRegexRequired,
		},
		{
			name: "error - TLS matcher without SNI hostnames",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        nil,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 443},
				},
			},
			expectedErr: ErrL4RouteSNIRequired,
		},
		{
			name: "error - TLS matcher with empty SNI hostnames",
			route: L4Route{
				MatcherType:         L4MatcherTLS,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				SNIHostnames:        pq.StringArray{},
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 443},
				},
			},
			expectedErr: ErrL4RouteSNIRequired,
		},
		{
			name: "error - route with invalid upstream",
			route: L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: L4LoadBalancingRoundRobin,
				Upstreams: L4UpstreamSlice{
					{Host: "", Port: 8080}, // empty host
				},
			},
			expectedErr: ErrL4UpstreamHostRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.route.Validate()

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

// =============================================================================
// L4Upstream Validation Tests
// =============================================================================

func TestL4Upstream_Validate(t *testing.T) {
	t.Parallel()

	weight := 10

	tests := []struct {
		name        string
		upstream    L4Upstream
		expectedErr error
	}{
		{
			name: "success - valid upstream",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: 8080,
			},
			expectedErr: nil,
		},
		{
			name: "success - valid upstream with IP address",
			upstream: L4Upstream{
				Host: "192.168.1.100",
				Port: 8080,
			},
			expectedErr: nil,
		},
		{
			name: "success - valid upstream with IPv6",
			upstream: L4Upstream{
				Host: "::1",
				Port: 8080,
			},
			expectedErr: nil,
		},
		{
			name: "success - valid upstream with weight",
			upstream: L4Upstream{
				Host:   "backend.local",
				Port:   8080,
				Weight: &weight,
			},
			expectedErr: nil,
		},
		{
			name: "success - port at lower boundary (1)",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: 1,
			},
			expectedErr: nil,
		},
		{
			name: "success - port at upper boundary (65535)",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: 65535,
			},
			expectedErr: nil,
		},
		{
			name: "error - empty host",
			upstream: L4Upstream{
				Host: "",
				Port: 8080,
			},
			expectedErr: ErrL4UpstreamHostRequired,
		},
		{
			name: "error - port below range (0)",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: 0,
			},
			expectedErr: ErrL4UpstreamInvalidPort,
		},
		{
			name: "error - port below range (negative)",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: -1,
			},
			expectedErr: ErrL4UpstreamInvalidPort,
		},
		{
			name: "error - port above range (65536)",
			upstream: L4Upstream{
				Host: "backend.local",
				Port: 65536,
			},
			expectedErr: ErrL4UpstreamInvalidPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.upstream.Validate()

			if tt.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

// =============================================================================
// L4UpstreamSlice JSONB Tests
// =============================================================================

func TestL4UpstreamSlice_Value(t *testing.T) {
	t.Parallel()

	weight := 10

	tests := []struct {
		name     string
		slice    L4UpstreamSlice
		expected string
	}{
		{
			name:     "nil slice returns empty array",
			slice:    nil,
			expected: "[]",
		},
		{
			name:     "empty slice returns empty array",
			slice:    L4UpstreamSlice{},
			expected: "[]",
		},
		{
			name: "single upstream",
			slice: L4UpstreamSlice{
				{Host: "backend.local", Port: 8080},
			},
			expected: `[{"host":"backend.local","port":8080}]`,
		},
		{
			name: "multiple upstreams",
			slice: L4UpstreamSlice{
				{Host: "backend1.local", Port: 8080},
				{Host: "backend2.local", Port: 8081},
			},
			expected: `[{"host":"backend1.local","port":8080},{"host":"backend2.local","port":8081}]`,
		},
		{
			name: "upstream with weight",
			slice: L4UpstreamSlice{
				{Host: "backend.local", Port: 8080, Weight: &weight},
			},
			expected: `[{"host":"backend.local","port":8080,"weight":10}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, err := tt.slice.Value()
			require.NoError(t, err)

			if tt.slice == nil {
				assert.Equal(t, tt.expected, val)
			} else {
				byteVal, ok := val.([]byte)
				require.True(t, ok, "expected []byte type")
				assert.JSONEq(t, tt.expected, string(byteVal))
			}
		})
	}
}

func TestL4UpstreamSlice_Scan(t *testing.T) {
	t.Parallel()

	weight := 10

	tests := []struct {
		name     string
		input    interface{}
		expected L4UpstreamSlice
		wantErr  bool
	}{
		{
			name:     "scan from nil",
			input:    nil,
			expected: L4UpstreamSlice{},
			wantErr:  false,
		},
		{
			name:     "scan from empty byte slice",
			input:    []byte{},
			expected: L4UpstreamSlice{},
			wantErr:  false,
		},
		{
			name:  "scan from []byte with single upstream",
			input: []byte(`[{"host":"backend.local","port":8080}]`),
			expected: L4UpstreamSlice{
				{Host: "backend.local", Port: 8080},
			},
			wantErr: false,
		},
		{
			name:  "scan from []byte with multiple upstreams",
			input: []byte(`[{"host":"backend1.local","port":8080},{"host":"backend2.local","port":8081}]`),
			expected: L4UpstreamSlice{
				{Host: "backend1.local", Port: 8080},
				{Host: "backend2.local", Port: 8081},
			},
			wantErr: false,
		},
		{
			name:  "scan from string",
			input: `[{"host":"backend.local","port":8080}]`,
			expected: L4UpstreamSlice{
				{Host: "backend.local", Port: 8080},
			},
			wantErr: false,
		},
		{
			name:  "scan from []byte with weight",
			input: []byte(`[{"host":"backend.local","port":8080,"weight":10}]`),
			expected: L4UpstreamSlice{
				{Host: "backend.local", Port: 8080, Weight: &weight},
			},
			wantErr: false,
		},
		{
			name:     "scan from empty JSON array",
			input:    []byte(`[]`),
			expected: L4UpstreamSlice{},
			wantErr:  false,
		},
		{
			name:     "scan from unknown type",
			input:    12345,
			expected: L4UpstreamSlice{},
			wantErr:  false,
		},
		{
			name:     "scan from invalid JSON",
			input:    []byte(`not valid json`),
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var u L4UpstreamSlice
			err := u.Scan(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, u)
			}
		})
	}
}

func TestL4UpstreamSlice_RoundTrip(t *testing.T) {
	t.Parallel()

	weight1 := 5
	weight2 := 10

	original := L4UpstreamSlice{
		{Host: "backend1.local", Port: 8080, Weight: &weight1},
		{Host: "backend2.local", Port: 8081, Weight: &weight2},
		{Host: "backend3.local", Port: 8082}, // no weight
	}

	// Marshal to value
	val, err := original.Value()
	require.NoError(t, err)

	// Scan back
	var scanned L4UpstreamSlice
	err = scanned.Scan(val)
	require.NoError(t, err)

	assert.Equal(t, original, scanned)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestIsValidL4Protocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		protocol string
		expected bool
	}{
		{"valid TCP", L4ProtocolTCP, true},
		{"valid UDP", L4ProtocolUDP, true},
		{"lowercase tcp", "tcp", true},
		{"lowercase udp", "udp", true},
		{"invalid - empty", "", false},
		{"invalid - sctp", "sctp", false},
		{"invalid - uppercase TCP", "TCP", false},
		{"invalid - uppercase UDP", "UDP", false},
		{"invalid - mixed case", "Tcp", false},
		{"invalid - random string", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsValidL4Protocol(tt.protocol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidL4MatcherType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		matcherType string
		expected    bool
	}{
		{"valid - any", L4MatcherAny, true},
		{"valid - tls", L4MatcherTLS, true},
		{"valid - ssh", L4MatcherSSH, true},
		{"valid - postgres", L4MatcherPostgres, true},
		{"valid - http", L4MatcherHTTP, true},
		{"valid - rdp", L4MatcherRDP, true},
		{"valid - socks5", L4MatcherSocks5, true},
		{"valid - remote_ip", L4MatcherRemoteIP, true},
		{"valid - regexp", L4MatcherRegexp, true},
		{"invalid - empty", "", false},
		{"invalid - unknown type", "unknown", false},
		{"invalid - uppercase ANY", "ANY", false},
		{"invalid - uppercase TLS", "TLS", false},
		{"invalid - mysql", "mysql", false},
		{"invalid - grpc", "grpc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsValidL4MatcherType(tt.matcherType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidL4LoadBalancingPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		policy   string
		expected bool
	}{
		{"valid - round_robin", L4LoadBalancingRoundRobin, true},
		{"valid - least_conn", L4LoadBalancingLeastConn, true},
		{"valid - random", L4LoadBalancingRandom, true},
		{"valid - first", L4LoadBalancingFirst, true},
		{"valid - ip_hash", L4LoadBalancingIPHash, true},
		{"invalid - empty", "", false},
		{"invalid - unknown policy", "unknown", false},
		{"invalid - uppercase ROUND_ROBIN", "ROUND_ROBIN", false},
		{"invalid - weighted", "weighted", false},
		{"invalid - least_time", "least_time", false},
		{"invalid - hash", "hash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsValidL4LoadBalancingPolicy(tt.policy)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Table Name Tests
// =============================================================================

func TestL4Proxy_TableName(t *testing.T) {
	t.Parallel()
	proxy := L4Proxy{}
	assert.Equal(t, "l4_proxies", proxy.TableName())
}

func TestL4Route_TableName(t *testing.T) {
	t.Parallel()
	route := L4Route{}
	assert.Equal(t, "l4_routes", route.TableName())
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestL4ProtocolConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tcp", L4ProtocolTCP)
	assert.Equal(t, "udp", L4ProtocolUDP)
}

func TestL4MatcherTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "any", L4MatcherAny)
	assert.Equal(t, "tls", L4MatcherTLS)
	assert.Equal(t, "ssh", L4MatcherSSH)
	assert.Equal(t, "postgres", L4MatcherPostgres)
	assert.Equal(t, "http", L4MatcherHTTP)
	assert.Equal(t, "rdp", L4MatcherRDP)
	assert.Equal(t, "socks5", L4MatcherSocks5)
	assert.Equal(t, "remote_ip", L4MatcherRemoteIP)
	assert.Equal(t, "regexp", L4MatcherRegexp)
}

func TestL4LoadBalancingPolicyConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "round_robin", L4LoadBalancingRoundRobin)
	assert.Equal(t, "least_conn", L4LoadBalancingLeastConn)
	assert.Equal(t, "random", L4LoadBalancingRandom)
	assert.Equal(t, "first", L4LoadBalancingFirst)
	assert.Equal(t, "ip_hash", L4LoadBalancingIPHash)
}

// =============================================================================
// Response Types Tests
// =============================================================================

func TestL4ProxyListResponse(t *testing.T) {
	t.Parallel()

	resp := L4ProxyListResponse{
		Items: []L4Proxy{
			{ID: 1, Name: "Proxy 1"},
			{ID: 2, Name: "Proxy 2"},
		},
		Total:      100,
		Page:       2,
		Limit:      10,
		TotalPages: 10,
	}

	assert.Len(t, resp.Items, 2)
	assert.Equal(t, int64(100), resp.Total)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 10, resp.TotalPages)
}

func TestL4ProxyStats(t *testing.T) {
	t.Parallel()

	stats := L4ProxyStats{
		TotalProxies:   50,
		ActiveProxies:  45,
		TCPProxies:     40,
		UDPProxies:     10,
		TotalRoutes:    100,
		TotalUpstreams: 200,
	}

	assert.Equal(t, int64(50), stats.TotalProxies)
	assert.Equal(t, int64(45), stats.ActiveProxies)
	assert.Equal(t, int64(40), stats.TCPProxies)
	assert.Equal(t, int64(10), stats.UDPProxies)
	assert.Equal(t, int64(100), stats.TotalRoutes)
	assert.Equal(t, int64(200), stats.TotalUpstreams)
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestL4Proxy_JSONSerialization(t *testing.T) {
	t.Parallel()

	description := "Test proxy description"
	proxy := L4Proxy{
		ID:          1,
		Name:        "Test Proxy",
		Description: &description,
		ListenPort:  8080,
		Protocol:    L4ProtocolTCP,
		IsActive:    true,
	}

	// Marshal to JSON
	data, err := json.Marshal(proxy)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled L4Proxy
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, proxy.ID, unmarshaled.ID)
	assert.Equal(t, proxy.Name, unmarshaled.Name)
	assert.Equal(t, proxy.ListenPort, unmarshaled.ListenPort)
	assert.Equal(t, proxy.Protocol, unmarshaled.Protocol)
	assert.Equal(t, proxy.IsActive, unmarshaled.IsActive)
	require.NotNil(t, unmarshaled.Description)
	assert.Equal(t, *proxy.Description, *unmarshaled.Description)
}

func TestL4Route_JSONSerialization(t *testing.T) {
	t.Parallel()

	regexPattern := "^test.*"
	route := L4Route{
		ID:                  1,
		L4ProxyID:           1,
		Priority:            10,
		MatcherType:         L4MatcherTLS,
		SNIHostnames:        pq.StringArray{"example.com", "*.example.com"},
		RegexPattern:        &regexPattern,
		LoadBalancingPolicy: L4LoadBalancingRoundRobin,
		Upstreams: L4UpstreamSlice{
			{Host: "backend.local", Port: 8080},
		},
		TLSTerminate:   false,
		TLSPassthrough: true,
	}

	// Marshal to JSON
	data, err := json.Marshal(route)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled L4Route
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, route.ID, unmarshaled.ID)
	assert.Equal(t, route.L4ProxyID, unmarshaled.L4ProxyID)
	assert.Equal(t, route.MatcherType, unmarshaled.MatcherType)
	assert.Equal(t, route.SNIHostnames, unmarshaled.SNIHostnames)
	assert.Equal(t, route.LoadBalancingPolicy, unmarshaled.LoadBalancingPolicy)
	assert.Equal(t, route.Upstreams, unmarshaled.Upstreams)
}

func TestL4Upstream_JSONSerialization(t *testing.T) {
	t.Parallel()

	weight := 10
	upstream := L4Upstream{
		Host:   "backend.local",
		Port:   8080,
		Weight: &weight,
	}

	// Marshal to JSON
	data, err := json.Marshal(upstream)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled L4Upstream
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, upstream.Host, unmarshaled.Host)
	assert.Equal(t, upstream.Port, unmarshaled.Port)
	require.NotNil(t, unmarshaled.Weight)
	assert.Equal(t, *upstream.Weight, *unmarshaled.Weight)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestL4Proxy_Validate_WithDescription(t *testing.T) {
	t.Parallel()

	description := "This is a test proxy with a description"
	proxy := L4Proxy{
		Name:        "Proxy with Description",
		Description: &description,
		Protocol:    L4ProtocolTCP,
		ListenPort:  8080,
	}

	err := proxy.Validate()
	require.NoError(t, err)
}

func TestL4Route_Validate_RemoteIPMatcher(t *testing.T) {
	t.Parallel()

	// Remote IP matcher doesn't require AllowedIPRanges per the current implementation
	// but we can still test it with IP ranges
	route := L4Route{
		MatcherType:         L4MatcherRemoteIP,
		LoadBalancingPolicy: L4LoadBalancingRoundRobin,
		AllowedIPRanges:     pq.StringArray{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Upstreams: L4UpstreamSlice{
			{Host: "backend.local", Port: 8080},
		},
	}

	err := route.Validate()
	require.NoError(t, err)
}

func TestL4Route_Validate_AllLoadBalancingPolicies(t *testing.T) {
	t.Parallel()

	policies := []string{
		L4LoadBalancingRoundRobin,
		L4LoadBalancingLeastConn,
		L4LoadBalancingRandom,
		L4LoadBalancingFirst,
		L4LoadBalancingIPHash,
	}

	for _, policy := range policies {
		t.Run(policy, func(t *testing.T) {
			t.Parallel()
			route := L4Route{
				MatcherType:         L4MatcherAny,
				LoadBalancingPolicy: policy,
				Upstreams: L4UpstreamSlice{
					{Host: "backend.local", Port: 8080},
				},
			}

			err := route.Validate()
			require.NoError(t, err)
		})
	}
}

func TestL4Route_Validate_AllMatcherTypes(t *testing.T) {
	t.Parallel()

	regexPattern := "^test.*"

	matcherConfigs := map[string]L4Route{
		L4MatcherAny: {
			MatcherType:         L4MatcherAny,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 8080}},
		},
		L4MatcherTLS: {
			MatcherType:         L4MatcherTLS,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			SNIHostnames:        pq.StringArray{"example.com"},
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 443}},
		},
		L4MatcherSSH: {
			MatcherType:         L4MatcherSSH,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 22}},
		},
		L4MatcherPostgres: {
			MatcherType:         L4MatcherPostgres,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 5432}},
		},
		L4MatcherHTTP: {
			MatcherType:         L4MatcherHTTP,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 80}},
		},
		L4MatcherRDP: {
			MatcherType:         L4MatcherRDP,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 3389}},
		},
		L4MatcherSocks5: {
			MatcherType:         L4MatcherSocks5,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 1080}},
		},
		L4MatcherRemoteIP: {
			MatcherType:         L4MatcherRemoteIP,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			AllowedIPRanges:     pq.StringArray{"10.0.0.0/8"},
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 8080}},
		},
		L4MatcherRegexp: {
			MatcherType:         L4MatcherRegexp,
			LoadBalancingPolicy: L4LoadBalancingRoundRobin,
			RegexPattern:        &regexPattern,
			Upstreams:           L4UpstreamSlice{{Host: "backend.local", Port: 8080}},
		},
	}

	for matcherType, route := range matcherConfigs {
		t.Run(matcherType, func(t *testing.T) {
			t.Parallel()
			err := route.Validate()
			require.NoError(t, err, "matcher type %s should be valid", matcherType)
		})
	}
}

// =============================================================================
// Validation Error Tests
// =============================================================================

func TestL4ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrL4ProxyNameRequired", ErrL4ProxyNameRequired, "L4 proxy name is required"},
		{"ErrL4ProxyNameTooLong", ErrL4ProxyNameTooLong, "L4 proxy name must be at most 255 characters"},
		{"ErrL4ProxyInvalidProtocol", ErrL4ProxyInvalidProtocol, "L4 proxy protocol must be 'tcp' or 'udp'"},
		{"ErrL4ProxyInvalidPort", ErrL4ProxyInvalidPort, "L4 proxy port must be between 1 and 65535"},
		{"ErrL4RouteInvalidMatcher", ErrL4RouteInvalidMatcher, "invalid L4 route matcher type"},
		{"ErrL4RouteInvalidLBPolicy", ErrL4RouteInvalidLBPolicy, "invalid L4 route load balancing policy"},
		{"ErrL4RouteUpstreamsRequired", ErrL4RouteUpstreamsRequired, "at least one upstream is required for L4 route"},
		{"ErrL4UpstreamHostRequired", ErrL4UpstreamHostRequired, "upstream host is required"},
		{"ErrL4UpstreamInvalidPort", ErrL4UpstreamInvalidPort, "upstream port must be between 1 and 65535"},
		{"ErrL4RouteTLSConflict", ErrL4RouteTLSConflict, "TLS terminate and TLS passthrough cannot both be enabled"},
		{"ErrL4RouteRegexRequired", ErrL4RouteRegexRequired, "regex pattern is required when matcher type is 'regexp'"},
		{"ErrL4RouteSNIRequired", ErrL4RouteSNIRequired, "SNI hostnames are required when matcher type is 'tls'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}
