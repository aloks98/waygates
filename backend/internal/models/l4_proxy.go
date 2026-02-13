package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// L4 Protocol constants
const (
	L4ProtocolTCP = "tcp"
	L4ProtocolUDP = "udp"
)

// L4 Matcher type constants
const (
	L4MatcherAny      = "any"
	L4MatcherTLS      = "tls"
	L4MatcherSSH      = "ssh"
	L4MatcherPostgres = "postgres"
	L4MatcherHTTP     = "http"
	L4MatcherRDP      = "rdp"
	L4MatcherSocks5   = "socks5"
	L4MatcherRemoteIP = "remote_ip"
	L4MatcherRegexp   = "regexp"
)

// L4 Load balancing policy constants
const (
	L4LoadBalancingRoundRobin = "round_robin"
	L4LoadBalancingLeastConn  = "least_conn"
	L4LoadBalancingRandom     = "random"
	L4LoadBalancingFirst      = "first"
	L4LoadBalancingIPHash     = "ip_hash"
)

// L4Proxy represents a Layer 4 (TCP/UDP) proxy configuration
type L4Proxy struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description *string   `json:"description,omitempty" gorm:"type:text"`
	ListenPort  int       `json:"listen_port" gorm:"not null"`
	Protocol    string    `json:"protocol" gorm:"type:varchar(10);not null;default:tcp"`
	IsActive    bool      `json:"is_active" gorm:"default:true;not null"`
	CreatedBy   int       `json:"-" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relations
	Routes  []L4Route `json:"routes,omitempty" gorm:"foreignKey:L4ProxyID;constraint:OnDelete:CASCADE"`
	Creator *User     `json:"created_by,omitempty" gorm:"foreignKey:CreatedBy"`
}

// TableName specifies the table name for GORM
func (*L4Proxy) TableName() string {
	return "l4_proxies"
}

// L4Route represents a routing rule for an L4 proxy
type L4Route struct {
	ID                   int             `json:"id" gorm:"primaryKey;autoIncrement"`
	L4ProxyID            int             `json:"l4_proxy_id" gorm:"not null"`
	Priority             int             `json:"priority" gorm:"not null;default:0"`
	MatcherType          string          `json:"matcher_type" gorm:"type:varchar(50);not null;default:any"`
	SNIHostnames         pq.StringArray  `json:"sni_hostnames,omitempty" gorm:"type:text[]"`
	AllowedIPRanges      pq.StringArray  `json:"allowed_ip_ranges,omitempty" gorm:"type:text[]"`
	RegexPattern         *string         `json:"regex_pattern,omitempty" gorm:"type:varchar(500)"`
	Upstreams            L4UpstreamSlice `json:"upstreams" gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	LoadBalancingPolicy  string          `json:"load_balancing_policy" gorm:"type:varchar(50);not null;default:round_robin"`
	TLSTerminate         bool            `json:"tls_terminate" gorm:"default:false;not null"`
	TLSPassthrough       bool            `json:"tls_passthrough" gorm:"default:true;not null"`
	ProxyProtocolVersion *string         `json:"proxy_protocol_version,omitempty" gorm:"type:varchar(5)"`
	CreatedAt            time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (*L4Route) TableName() string {
	return "l4_routes"
}

// L4Upstream represents an upstream server for L4 routing
type L4Upstream struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Weight *int   `json:"weight,omitempty"`
}

// L4UpstreamSlice is a custom type for storing L4Upstream slices as JSONB
type L4UpstreamSlice []L4Upstream

// Value implements the driver.Valuer interface for database storage
func (u *L4UpstreamSlice) Value() (driver.Value, error) {
	if u == nil || *u == nil {
		return "[]", nil
	}
	return json.Marshal(*u)
}

// Scan implements the sql.Scanner interface for database retrieval
func (u *L4UpstreamSlice) Scan(value interface{}) error {
	if value == nil {
		*u = []L4Upstream{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		*u = []L4Upstream{}
		return nil
	}

	if len(bytes) == 0 {
		*u = []L4Upstream{}
		return nil
	}

	return json.Unmarshal(bytes, u)
}

// L4ProxyListResponse is the response for listing L4 proxies
type L4ProxyListResponse struct {
	Items      []L4Proxy `json:"items"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	TotalPages int       `json:"total_pages"`
}

// L4ProxyStats represents statistics for L4 proxies
type L4ProxyStats struct {
	TotalProxies   int64 `json:"total_proxies"`
	ActiveProxies  int64 `json:"active_proxies"`
	TCPProxies     int64 `json:"tcp_proxies"`
	UDPProxies     int64 `json:"udp_proxies"`
	TotalRoutes    int64 `json:"total_routes"`
	TotalUpstreams int64 `json:"total_upstreams"`
}

// L4 Proxy validation errors
var (
	ErrL4ProxyNameRequired      = &ValidationError{Message: "L4 proxy name is required"}
	ErrL4ProxyNameTooLong       = &ValidationError{Message: "L4 proxy name must be at most 255 characters"}
	ErrL4ProxyInvalidProtocol   = &ValidationError{Message: "L4 proxy protocol must be 'tcp' or 'udp'"}
	ErrL4ProxyInvalidPort       = &ValidationError{Message: "L4 proxy port must be between 1 and 65535"}
	ErrL4RouteInvalidMatcher    = &ValidationError{Message: "invalid L4 route matcher type"}
	ErrL4RouteInvalidLBPolicy   = &ValidationError{Message: "invalid L4 route load balancing policy"}
	ErrL4RouteUpstreamsRequired = &ValidationError{Message: "at least one upstream is required for L4 route"}
	ErrL4UpstreamHostRequired   = &ValidationError{Message: "upstream host is required"}
	ErrL4UpstreamInvalidPort    = &ValidationError{Message: "upstream port must be between 1 and 65535"}
	ErrL4RouteTLSConflict       = &ValidationError{Message: "TLS terminate and TLS passthrough cannot both be enabled"}
	ErrL4RouteRegexRequired     = &ValidationError{Message: "regex pattern is required when matcher type is 'regexp'"}
	ErrL4RouteSNIRequired       = &ValidationError{Message: "SNI hostnames are required when matcher type is 'tls'"}
)

// Validate performs validation on the L4Proxy
func (p *L4Proxy) Validate() error {
	if p.Name == "" {
		return ErrL4ProxyNameRequired
	}

	if len(p.Name) > 255 {
		return ErrL4ProxyNameTooLong
	}

	if p.Protocol != L4ProtocolTCP && p.Protocol != L4ProtocolUDP {
		return ErrL4ProxyInvalidProtocol
	}

	if p.ListenPort < 1 || p.ListenPort > 65535 {
		return ErrL4ProxyInvalidPort
	}

	// Validate all routes
	for i := range p.Routes {
		if err := p.Routes[i].Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate performs validation on the L4Route
func (r *L4Route) Validate() error {
	// Validate matcher type
	validMatcherTypes := map[string]bool{
		L4MatcherAny:      true,
		L4MatcherTLS:      true,
		L4MatcherSSH:      true,
		L4MatcherPostgres: true,
		L4MatcherHTTP:     true,
		L4MatcherRDP:      true,
		L4MatcherSocks5:   true,
		L4MatcherRemoteIP: true,
		L4MatcherRegexp:   true,
	}
	if !validMatcherTypes[r.MatcherType] {
		return ErrL4RouteInvalidMatcher
	}

	// Validate load balancing policy
	validLBPolicies := map[string]bool{
		L4LoadBalancingRoundRobin: true,
		L4LoadBalancingLeastConn:  true,
		L4LoadBalancingRandom:     true,
		L4LoadBalancingFirst:      true,
		L4LoadBalancingIPHash:     true,
	}
	if !validLBPolicies[r.LoadBalancingPolicy] {
		return ErrL4RouteInvalidLBPolicy
	}

	// Validate upstreams
	if len(r.Upstreams) == 0 {
		return ErrL4RouteUpstreamsRequired
	}

	for i := range r.Upstreams {
		if err := r.Upstreams[i].Validate(); err != nil {
			return err
		}
	}

	// Validate TLS settings
	if r.TLSTerminate && r.TLSPassthrough {
		return ErrL4RouteTLSConflict
	}

	// Validate matcher-specific requirements
	if r.MatcherType == L4MatcherRegexp && (r.RegexPattern == nil || *r.RegexPattern == "") {
		return ErrL4RouteRegexRequired
	}

	if r.MatcherType == L4MatcherTLS && len(r.SNIHostnames) == 0 {
		return ErrL4RouteSNIRequired
	}

	return nil
}

// Validate performs validation on the L4Upstream
func (u *L4Upstream) Validate() error {
	if u.Host == "" {
		return ErrL4UpstreamHostRequired
	}

	if u.Port < 1 || u.Port > 65535 {
		return ErrL4UpstreamInvalidPort
	}

	return nil
}

// IsValidL4Protocol checks if the given protocol is a valid L4 protocol
func IsValidL4Protocol(protocol string) bool {
	return protocol == L4ProtocolTCP || protocol == L4ProtocolUDP
}

// IsValidL4MatcherType checks if the given matcher type is valid
func IsValidL4MatcherType(matcherType string) bool {
	validTypes := []string{
		L4MatcherAny,
		L4MatcherTLS,
		L4MatcherSSH,
		L4MatcherPostgres,
		L4MatcherHTTP,
		L4MatcherRDP,
		L4MatcherSocks5,
		L4MatcherRemoteIP,
		L4MatcherRegexp,
	}
	for _, t := range validTypes {
		if matcherType == t {
			return true
		}
	}
	return false
}

// IsValidL4LoadBalancingPolicy checks if the given load balancing policy is valid
func IsValidL4LoadBalancingPolicy(policy string) bool {
	validPolicies := []string{
		L4LoadBalancingRoundRobin,
		L4LoadBalancingLeastConn,
		L4LoadBalancingRandom,
		L4LoadBalancingFirst,
		L4LoadBalancingIPHash,
	}
	for _, p := range validPolicies {
		if policy == p {
			return true
		}
	}
	return false
}
