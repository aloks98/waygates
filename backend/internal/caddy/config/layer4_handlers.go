// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// L4 Handler module names as constants.
const (
	L4HandlerProxy     = "proxy"
	L4HandlerTLS       = "tls"
	L4HandlerSubroute  = "subroute"
	L4HandlerEcho      = "echo"
	L4HandlerProxyProt = "proxy_protocol"
)

// L4 Load balancing policy constants.
const (
	L4LBRoundRobin = "round_robin"
	L4LBLeastConn  = "least_conn"
	L4LBRandom     = "random"
	L4LBFirst      = "first"
	L4LBIPHash     = "ip_hash"
)

// L4ProxyHandler configures the L4 proxy handler.
// See: https://github.com/mholt/caddy-l4#proxy
type L4ProxyHandler struct {
	Handler           string           `json:"handler"` // Must be "proxy"
	Upstreams         []*L4Upstream    `json:"upstreams,omitempty"`
	LoadBalancing     *L4LoadBalancing `json:"load_balancing,omitempty"`
	HealthChecks      *L4HealthChecks  `json:"health_checks,omitempty"`
	ProxyProtocol     string           `json:"proxy_protocol,omitempty"` // "v1" or "v2"
	DialTimeout       Duration         `json:"dial_timeout,omitempty"`
	ConnectTimeout    Duration         `json:"connect_timeout,omitempty"`
	IdleTimeout       Duration         `json:"idle_timeout,omitempty"`
	FailDuration      Duration         `json:"fail_duration,omitempty"`
	UnhealthyConnFail int              `json:"unhealthy_conn_fail,omitempty"`
	TLS               *L4UpstreamTLS   `json:"tls,omitempty"`
}

// L4Upstream represents an upstream server for L4 proxy.
type L4Upstream struct {
	Dial []string `json:"dial"`
}

// L4LoadBalancing configures load balancing for L4 proxy.
type L4LoadBalancing struct {
	SelectionPolicy *L4SelectionPolicy `json:"selection,omitempty"`
	TryDuration     Duration           `json:"try_duration,omitempty"`
	TryInterval     Duration           `json:"try_interval,omitempty"`
}

// L4SelectionPolicy configures the upstream selection policy for L4.
type L4SelectionPolicy struct {
	Policy string `json:"policy,omitempty"` // round_robin, least_conn, random, first, ip_hash
}

// L4HealthChecks configures health checking for L4 upstreams.
type L4HealthChecks struct {
	Active *L4ActiveHealthCheck `json:"active,omitempty"`
}

// L4ActiveHealthCheck configures active health checking for L4.
type L4ActiveHealthCheck struct {
	Port     int      `json:"port,omitempty"`
	Interval Duration `json:"interval,omitempty"`
	Timeout  Duration `json:"timeout,omitempty"`
}

// L4UpstreamTLS configures TLS for upstream connections in L4 proxy.
type L4UpstreamTLS struct {
	ServerName         string   `json:"server_name,omitempty"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"`
	RootCAPool         []string `json:"root_ca_pool,omitempty"`
}

// L4TLSHandler configures the TLS termination handler.
// See: https://github.com/mholt/caddy-l4#tls-1
type L4TLSHandler struct {
	Handler            string             `json:"handler"` // Must be "tls"
	ConnectionPolicies []*L4TLSConnPolicy `json:"connection_policies,omitempty"`
}

// L4TLSConnPolicy configures a TLS connection policy for L4.
type L4TLSConnPolicy struct {
	Match               *L4TLSMatch `json:"match,omitempty"`
	CertificateSelector interface{} `json:"certificate_selection,omitempty"`
	DefaultSNI          string      `json:"default_sni,omitempty"`
	Alpn                []string    `json:"alpn,omitempty"`
}

// L4TLSMatch defines matching criteria for L4 TLS connections.
type L4TLSMatch struct {
	SNI []string `json:"sni,omitempty"`
}

// L4SubrouteHandler configures the subroute handler for nested L4 routing.
type L4SubrouteHandler struct {
	Handler string     `json:"handler"` // Must be "subroute"
	Routes  []*L4Route `json:"routes,omitempty"`
}

// L4ProxyProtocolHandler configures the proxy protocol handler.
type L4ProxyProtocolHandler struct {
	Handler string   `json:"handler"`           // Must be "proxy_protocol"
	Version int      `json:"version,omitempty"` // 1 or 2
	Timeout Duration `json:"timeout,omitempty"`
	Allow   []string `json:"allow,omitempty"` // IP ranges to allow
}

// L4EchoHandler configures the echo handler (useful for testing).
type L4EchoHandler struct {
	Handler string `json:"handler"` // Must be "echo"
}

// NewL4ProxyHandler creates a new L4 proxy handler.
func NewL4ProxyHandler(upstreams ...*L4Upstream) *L4ProxyHandler {
	return &L4ProxyHandler{
		Handler:   L4HandlerProxy,
		Upstreams: upstreams,
	}
}

// NewL4Upstream creates a new L4 upstream configuration.
func NewL4Upstream(dial string) *L4Upstream {
	return &L4Upstream{
		Dial: []string{dial},
	}
}

// NewL4UpstreamFromHostPort creates an L4 upstream from host and port.
func NewL4UpstreamFromHostPort(host string, port int) *L4Upstream {
	return &L4Upstream{
		Dial: []string{formatDialAddress(host, port)},
	}
}

// WithLoadBalancing adds load balancing to an L4 proxy handler.
func (h *L4ProxyHandler) WithLoadBalancing(policy string) *L4ProxyHandler {
	h.LoadBalancing = &L4LoadBalancing{
		SelectionPolicy: &L4SelectionPolicy{
			Policy: policy,
		},
	}
	return h
}

// WithProxyProtocol adds proxy protocol to an L4 proxy handler.
func (h *L4ProxyHandler) WithProxyProtocol(version string) *L4ProxyHandler {
	h.ProxyProtocol = version
	return h
}

// WithUpstreamTLS adds TLS configuration for upstream connections.
func (h *L4ProxyHandler) WithUpstreamTLS(serverName string, insecureSkipVerify bool) *L4ProxyHandler {
	h.TLS = &L4UpstreamTLS{
		ServerName:         serverName,
		InsecureSkipVerify: insecureSkipVerify,
	}
	return h
}

// WithTimeouts adds timeout configuration to an L4 proxy handler.
func (h *L4ProxyHandler) WithTimeouts(dial, connect, idle Duration) *L4ProxyHandler {
	if dial != "" {
		h.DialTimeout = dial
	}
	if connect != "" {
		h.ConnectTimeout = connect
	}
	if idle != "" {
		h.IdleTimeout = idle
	}
	return h
}

// WithHealthChecks adds health checks to an L4 proxy handler.
func (h *L4ProxyHandler) WithHealthChecks(port int, interval, timeout Duration) *L4ProxyHandler {
	h.HealthChecks = &L4HealthChecks{
		Active: &L4ActiveHealthCheck{
			Port:     port,
			Interval: interval,
			Timeout:  timeout,
		},
	}
	return h
}

// NewL4TLSHandler creates a new TLS termination handler.
func NewL4TLSHandler() *L4TLSHandler {
	return &L4TLSHandler{
		Handler: L4HandlerTLS,
	}
}

// WithConnectionPolicies adds connection policies to a TLS handler.
func (h *L4TLSHandler) WithConnectionPolicies(policies ...*L4TLSConnPolicy) *L4TLSHandler {
	h.ConnectionPolicies = policies
	return h
}

// NewL4TLSConnPolicy creates a new TLS connection policy.
func NewL4TLSConnPolicy(sni []string) *L4TLSConnPolicy {
	policy := &L4TLSConnPolicy{}
	if len(sni) > 0 {
		policy.Match = &L4TLSMatch{SNI: sni}
	}
	return policy
}

// NewL4SubrouteHandler creates a new L4 subroute handler.
func NewL4SubrouteHandler(routes ...*L4Route) *L4SubrouteHandler {
	return &L4SubrouteHandler{
		Handler: L4HandlerSubroute,
		Routes:  routes,
	}
}

// NewL4ProxyProtocolHandler creates a new proxy protocol handler.
func NewL4ProxyProtocolHandler(version int) *L4ProxyProtocolHandler {
	return &L4ProxyProtocolHandler{
		Handler: L4HandlerProxyProt,
		Version: version,
	}
}

// NewL4EchoHandler creates a new echo handler.
func NewL4EchoHandler() *L4EchoHandler {
	return &L4EchoHandler{
		Handler: L4HandlerEcho,
	}
}

// ToL4Handler converts a typed L4 handler to the generic L4Handler map.
func ToL4Handler(handler interface{}) L4Handler {
	switch h := handler.(type) {
	case *L4ProxyHandler:
		result := L4Handler{
			"handler":   h.Handler,
			"upstreams": h.Upstreams,
		}
		if h.LoadBalancing != nil {
			result["load_balancing"] = h.LoadBalancing
		}
		if h.HealthChecks != nil {
			result["health_checks"] = h.HealthChecks
		}
		if h.ProxyProtocol != "" {
			result["proxy_protocol"] = h.ProxyProtocol
		}
		if h.DialTimeout != "" {
			result["dial_timeout"] = h.DialTimeout
		}
		if h.ConnectTimeout != "" {
			result["connect_timeout"] = h.ConnectTimeout
		}
		if h.IdleTimeout != "" {
			result["idle_timeout"] = h.IdleTimeout
		}
		if h.FailDuration != "" {
			result["fail_duration"] = h.FailDuration
		}
		if h.UnhealthyConnFail > 0 {
			result["unhealthy_conn_fail"] = h.UnhealthyConnFail
		}
		if h.TLS != nil {
			result["tls"] = h.TLS
		}
		return result
	case *L4TLSHandler:
		result := L4Handler{
			"handler": h.Handler,
		}
		if len(h.ConnectionPolicies) > 0 {
			result["connection_policies"] = h.ConnectionPolicies
		}
		return result
	case *L4SubrouteHandler:
		return L4Handler{
			"handler": h.Handler,
			"routes":  h.Routes,
		}
	case *L4ProxyProtocolHandler:
		result := L4Handler{
			"handler": h.Handler,
		}
		if h.Version > 0 {
			result["version"] = h.Version
		}
		if h.Timeout != "" {
			result["timeout"] = h.Timeout
		}
		if len(h.Allow) > 0 {
			result["allow"] = h.Allow
		}
		return result
	case *L4EchoHandler:
		return L4Handler{
			"handler": h.Handler,
		}
	default:
		return nil
	}
}

// MapLBPolicyToL4 maps model load balancing policy to L4 policy.
func MapLBPolicyToL4(policy string) string {
	switch policy {
	case "round_robin":
		return L4LBRoundRobin
	case "least_conn":
		return L4LBLeastConn
	case "random":
		return L4LBRandom
	case "first":
		return L4LBFirst
	case "ip_hash":
		return L4LBIPHash
	default:
		return L4LBRoundRobin
	}
}
