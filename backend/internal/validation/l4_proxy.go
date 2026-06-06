package validation

// CreateL4ProxyRequestDTO represents the request body for creating an L4 proxy
type CreateL4ProxyRequestDTO struct {
	Name        string                    `json:"name" validate:"required,min=1,max=255"`
	Description *string                   `json:"description,omitempty" validate:"omitempty,max=1000"`
	ListenPort  int                       `json:"listen_port" validate:"required,min=1,max=65535"`
	Protocol    string                    `json:"protocol" validate:"required,oneof=tcp udp"`
	IsActive    *bool                     `json:"is_active,omitempty"` // Defaults to true if not provided
	Routes      []CreateL4RouteRequestDTO `json:"routes,omitempty" validate:"dive"`
}

// UpdateL4ProxyRequestDTO represents the request body for updating an L4 proxy
type UpdateL4ProxyRequestDTO struct {
	Name        string                    `json:"name" validate:"required,min=1,max=255"`
	Description *string                   `json:"description,omitempty" validate:"omitempty,max=1000"`
	ListenPort  int                       `json:"listen_port" validate:"required,min=1,max=65535"`
	Protocol    string                    `json:"protocol" validate:"required,oneof=tcp udp"`
	IsActive    bool                      `json:"is_active"`
	Routes      []CreateL4RouteRequestDTO `json:"routes,omitempty" validate:"dive"`
}

// CreateL4RouteRequestDTO represents a route configuration in L4 proxy requests
type CreateL4RouteRequestDTO struct {
	Priority    int    `json:"priority" validate:"gte=0"`
	MatcherType string `json:"matcher_type" validate:"required,oneof=any tls ssh postgres http rdp socks5 remote_ip regexp"`
	// SNIHostnames are only meaningful for the "tls" matcher (SNI is a TLS
	// concept). They are required when MatcherType == "tls" but are NOT
	// restricted for other matcher types here: setting them on, e.g., the
	// "http" matcher is accepted and then silently ignored at config-build
	// time (the L4 http matcher matches any connection; HTTP host routing is
	// L7). See MapMatcherTypeToL4Matcher in caddy/config/layer4_matchers.go.
	SNIHostnames         []string               `json:"sni_hostnames,omitempty" validate:"omitempty,dive,min=1"`
	AllowedIPRanges      []string               `json:"allowed_ip_ranges,omitempty" validate:"omitempty,dive,cidr"`
	RegexPattern         *string                `json:"regex_pattern,omitempty" validate:"omitempty,min=1,max=500"`
	Upstreams            []L4UpstreamRequestDTO `json:"upstreams" validate:"required,min=1,dive"`
	LoadBalancingPolicy  string                 `json:"load_balancing_policy" validate:"required,oneof=round_robin least_conn random first ip_hash"`
	TLSTerminate         bool                   `json:"tls_terminate"`
	TLSPassthrough       bool                   `json:"tls_passthrough"`
	ProxyProtocolVersion *string                `json:"proxy_protocol_version,omitempty" validate:"omitempty,oneof=v1 v2"`
}

// L4UpstreamRequestDTO represents an upstream server in L4 route requests
type L4UpstreamRequestDTO struct {
	Host   string `json:"host" validate:"required,min=1,max=255"`
	Port   int    `json:"port" validate:"required,min=1,max=65535"`
	Weight *int   `json:"weight,omitempty" validate:"omitempty,min=0,max=100"`
}
