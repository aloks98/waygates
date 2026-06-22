// Package config provides typed Go structs for generating Caddy JSON configuration.
// This replaces the string-based Caddyfile generation with type-safe JSON marshaling.
package config

import "encoding/json"

// CaddyConfig represents the root Caddy configuration structure.
// See: https://caddyserver.com/docs/json/
type CaddyConfig struct {
	Admin   *AdminConfig   `json:"admin,omitempty"`
	Logging *LoggingConfig `json:"logging,omitempty"`
	Storage *StorageConfig `json:"storage,omitempty"`
	Apps    *AppsConfig    `json:"apps,omitempty"`
}

// AdminConfig configures Caddy's admin API endpoint.
type AdminConfig struct {
	Disabled bool   `json:"disabled,omitempty"`
	Listen   string `json:"listen,omitempty"`
}

// LoggingConfig configures Caddy's logging subsystem.
type LoggingConfig struct {
	Logs map[string]*LogConfig `json:"logs,omitempty"`
}

// LogConfig represents a single log configuration.
type LogConfig struct {
	Writer  *LogWriter  `json:"writer,omitempty"`
	Encoder *LogEncoder `json:"encoder,omitempty"`
	Level   string      `json:"level,omitempty"`
	Include []string    `json:"include,omitempty"`
}

// LogWriter configures where logs are written.
type LogWriter struct {
	Output       string `json:"output,omitempty"`
	Filename     string `json:"filename,omitempty"`
	Roll         *bool  `json:"roll,omitempty"`
	RollSizeMB   int    `json:"roll_size_mb,omitempty"`
	RollKeep     int    `json:"roll_keep,omitempty"`
	RollKeepDays int    `json:"roll_keep_days,omitempty"` // days
}

// LogEncoder configures how logs are formatted.
type LogEncoder struct {
	Format string `json:"format,omitempty"`
}

// StorageConfig configures Caddy's storage module for certificates, etc.
type StorageConfig struct {
	Module string `json:"module,omitempty"`
	Root   string `json:"root,omitempty"`
}

// AppsConfig contains all Caddy application configurations.
type AppsConfig struct {
	TLS    *TLSApp    `json:"tls,omitempty"`
	HTTP   *HTTPApp   `json:"http,omitempty"`
	Layer4 *Layer4App `json:"layer4,omitempty"`
}

// TLSApp configures the TLS app for certificate management.
// Full implementation in tls.go
type TLSApp struct {
	Certificates *CertificatesConfig `json:"certificates,omitempty"`
	Automation   *AutomationConfig   `json:"automation,omitempty"`
}

// CertificatesConfig configures certificate sources.
type CertificatesConfig struct {
	Automate []string `json:"automate,omitempty"`
}

// AutomationConfig configures automatic certificate management.
type AutomationConfig struct {
	Policies []*AutomationPolicy `json:"policies,omitempty"`
}

// AutomationPolicy defines a certificate automation policy.
type AutomationPolicy struct {
	Subjects []string  `json:"subjects,omitempty"`
	Issuers  []*Issuer `json:"issuers,omitempty"`
}

// Issuer represents a certificate issuer (ACME provider).
type Issuer struct {
	Module     string            `json:"module,omitempty"`
	CA         string            `json:"ca,omitempty"`
	Email      string            `json:"email,omitempty"`
	Challenges *ChallengesConfig `json:"challenges,omitempty"`
}

// ChallengesConfig configures ACME challenges.
type ChallengesConfig struct {
	DNS *DNSChallengeConfig `json:"dns,omitempty"`
}

// DNSChallengeConfig configures DNS-01 challenge.
type DNSChallengeConfig struct {
	// Provider can be any of the DNS provider types defined in tls.go
	// (DNSProviderCloudflare, DNSProviderRoute53, etc.)
	Provider interface{} `json:"provider,omitempty"`
}

// HTTPApp configures the HTTP server application.
// Full implementation in http.go
type HTTPApp struct {
	Servers map[string]*HTTPServer `json:"servers,omitempty"`
}

// HTTPServerLogs enables access logging for an HTTP server.
type HTTPServerLogs struct {
	DefaultLoggerName string `json:"default_logger_name,omitempty"`
}

// HTTPServer configures an HTTP server instance.
type HTTPServer struct {
	Listen          []string              `json:"listen,omitempty"`
	Routes          []*HTTPRoute          `json:"routes,omitempty"`
	AutoHTTPS       *AutoHTTPSConfig      `json:"automatic_https,omitempty"`
	TLSConnPolicies []*TLSConnPolicy      `json:"tls_connection_policies,omitempty"`
	TrustedProxies  *TrustedProxiesSource `json:"trusted_proxies,omitempty"`
	ClientIPHeaders []string              `json:"client_ip_headers,omitempty"`
	Logs            *HTTPServerLogs       `json:"logs,omitempty"`
}

// TrustedProxiesSource configures which upstream proxies Caddy trusts when
// resolving the real client IP, using the "static" IP source with explicit
// CIDR ranges. When set, {http.vars.client_ip} is taken from
// client_ip_headers for requests arriving from these proxies.
// See: https://caddyserver.com/docs/json/apps/http/servers/trusted_proxies/
type TrustedProxiesSource struct {
	Source string   `json:"source"` // always "static"
	Ranges []string `json:"ranges,omitempty"`
}

// HTTPRoute defines a route for request matching and handling.
type HTTPRoute struct {
	Match    []MatcherSet  `json:"match,omitempty"`
	Handle   []HTTPHandler `json:"handle,omitempty"`
	Terminal bool          `json:"terminal,omitempty"`
}

// MatcherSet is a set of matchers that must all match for a route.
type MatcherSet map[string]interface{}

// HTTPHandler represents any HTTP handler configuration.
type HTTPHandler map[string]interface{}

// AutoHTTPSConfig configures automatic HTTPS behavior.
type AutoHTTPSConfig struct {
	Disabled          bool     `json:"disable,omitempty"`
	DisableRedirects  bool     `json:"disable_redirects,omitempty"`
	DisableCerts      bool     `json:"disable_certs,omitempty"`
	Skip              []string `json:"skip,omitempty"`
	SkipCerts         []string `json:"skip_certs,omitempty"`
	IgnoreLoadedCerts bool     `json:"ignore_loaded_certs,omitempty"`
}

// TLSConnPolicy configures TLS connection policies.
type TLSConnPolicy struct {
	Match                *TLSMatch   `json:"match,omitempty"`
	CertificateSelection interface{} `json:"certificate_selection,omitempty"`
	DefaultSNI           string      `json:"default_sni,omitempty"`
}

// TLSMatch defines matching criteria for TLS connections.
type TLSMatch struct {
	SNI []string `json:"sni,omitempty"`
}

// Layer4App configures the Layer 4 (TCP/UDP) proxy application.
// See: https://github.com/mholt/caddy-l4
type Layer4App struct {
	Servers map[string]*L4Server `json:"servers,omitempty"`
}

// L4Server configures an L4 server instance.
type L4Server struct {
	Listen []string   `json:"listen,omitempty"`
	Routes []*L4Route `json:"routes,omitempty"`
}

// L4Route defines a route for L4 connection matching and handling.
type L4Route struct {
	Match  []L4MatcherSet `json:"match,omitempty"`
	Handle []L4Handler    `json:"handle,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for CaddyConfig.
func (c *CaddyConfig) MarshalJSON() ([]byte, error) {
	type Alias CaddyConfig
	return json.Marshal((*Alias)(c))
}

// ToJSON marshals the config to formatted JSON bytes.
func (c *CaddyConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// ToCompactJSON marshals the config to compact JSON bytes.
func (c *CaddyConfig) ToCompactJSON() ([]byte, error) {
	return json.Marshal(c)
}
