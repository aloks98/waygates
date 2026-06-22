// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Builder orchestrates the generation of Caddy JSON configuration.
// It coordinates the HTTP, TLS, and ACL builders to produce a complete config.
type Builder struct {
	logger      *zap.Logger
	httpBuilder *HTTPBuilder
	tlsBuilder  *TLSBuilder
	aclBuilder  *ACLBuilder

	// Application settings (stored for use in Build).
	settings *Settings

	// Configuration inputs
	httpProxies []models.Proxy
	aclGroups   map[int64]*models.ACLGroup
	aclAssigns  map[int64][]models.ProxyACLAssignment
	notFound    *models.NotFoundSettings

	// Layer4 configuration (optional)
	layer4App *Layer4App

	// L4 SNI hostnames that need TLS certificates (for TLS termination)
	l4TLSHostnames []string

	// Trusted proxy configuration for real-client-IP resolution behind an
	// upstream proxy/tunnel (empty = Caddy treated as the edge).
	trustedProxies  []string
	clientIPHeaders []string
}

// Settings holds the application settings for building the config.
type Settings struct {
	AdminEmail   string
	ACMEProvider string
	StoragePath  string
	LogPath      string // Path for Caddy runtime + access log files (default: /var/log/caddy)

	// Waygates auth URLs
	WaygatesVerifyURL string
	WaygatesLoginURL  string

	// DNS provider credentials (loaded from environment)
	DNSCredentials map[string]string

	// TrustedProxies are CIDR ranges of upstream proxies/tunnels (e.g.
	// cloudflared, Pangolin) whose forwarded client-IP headers Caddy should
	// trust. Empty means Caddy is the edge. ClientIPHeaders lists the headers
	// to read the real client IP from (defaults to X-Forwarded-For in Caddy).
	TrustedProxies  []string
	ClientIPHeaders []string
}

// BuilderOption is a functional option for configuring the Builder.
type BuilderOption func(*Builder)

// NewBuilder creates a new configuration builder.
func NewBuilder(opts ...BuilderOption) *Builder {
	b := &Builder{
		logger:     zap.NewNop(),
		aclGroups:  make(map[int64]*models.ACLGroup),
		aclAssigns: make(map[int64][]models.ProxyACLAssignment),
	}

	for _, opt := range opts {
		opt(b)
	}

	// Initialize sub-builders
	b.httpBuilder = NewHTTPBuilder(b.logger)
	b.tlsBuilder = NewTLSBuilder(b.logger)

	return b
}

// WithLogger sets the logger for the builder.
func WithLogger(logger *zap.Logger) BuilderOption {
	return func(b *Builder) {
		if logger != nil {
			b.logger = logger
		}
	}
}

// WithACLBuilder sets the ACL builder.
func WithACLBuilder(aclBuilder *ACLBuilder) BuilderOption {
	return func(b *Builder) {
		b.aclBuilder = aclBuilder
	}
}

// SetSettings sets the application settings.
func (b *Builder) SetSettings(settings *Settings) *Builder {
	b.settings = settings
	b.tlsBuilder.SetSettings(settings)
	if settings != nil {
		if b.aclBuilder != nil {
			b.aclBuilder.SetWaygatesURLs(settings.WaygatesVerifyURL, settings.WaygatesLoginURL)
		}
		b.trustedProxies = settings.TrustedProxies
		b.clientIPHeaders = settings.ClientIPHeaders
	}
	return b
}

// SetHTTPProxies sets the HTTP proxies to include in the configuration.
func (b *Builder) SetHTTPProxies(proxies []models.Proxy) *Builder {
	b.httpProxies = proxies
	return b
}

// SetACLGroups sets the ACL groups for authentication configuration.
func (b *Builder) SetACLGroups(groups []models.ACLGroup) *Builder {
	b.aclGroups = make(map[int64]*models.ACLGroup)
	for i := range groups {
		b.aclGroups[int64(groups[i].ID)] = &groups[i]
	}
	return b
}

// SetACLAssignments sets the proxy ACL assignments.
func (b *Builder) SetACLAssignments(assignments []models.ProxyACLAssignment) *Builder {
	b.aclAssigns = make(map[int64][]models.ProxyACLAssignment)
	for _, a := range assignments {
		if a.Enabled {
			b.aclAssigns[int64(a.ProxyID)] = append(b.aclAssigns[int64(a.ProxyID)], a)
		}
	}
	return b
}

// SetNotFoundSettings sets the 404 response configuration.
func (b *Builder) SetNotFoundSettings(settings *models.NotFoundSettings) *Builder {
	b.notFound = settings
	return b
}

// SetLayer4App sets the Layer4 application configuration for TCP/UDP proxying.
func (b *Builder) SetLayer4App(layer4App *Layer4App) *Builder {
	b.layer4App = layer4App
	return b
}

// SetL4TLSHostnames sets the SNI hostnames from L4 proxies that need TLS certificates.
// These hostnames will have HTTP routes created for ACME certificate provisioning.
func (b *Builder) SetL4TLSHostnames(hostnames []string) *Builder {
	b.l4TLSHostnames = hostnames
	return b
}

// Build generates the complete Caddy configuration.
func (b *Builder) Build() (*CaddyConfig, error) {
	config := &CaddyConfig{
		Admin: &AdminConfig{
			Listen: "localhost:2019",
		},
		Storage: &StorageConfig{
			Module: "file_system",
			Root:   "/data",
		},
		Apps: &AppsConfig{},
	}

	// Resolve log path (settings may be nil when builder is used stand-alone).
	logPath := "/var/log/caddy"
	if b.settings != nil && b.settings.LogPath != "" {
		logPath = b.settings.LogPath
	}
	roll := true

	// Always emit the runtime (default) log.
	config.Logging = &LoggingConfig{
		Logs: map[string]*LogConfig{
			"default": {
				Writer:  &LogWriter{Output: "file", Filename: filepath.Join(logPath, "runtime.log"), Roll: &roll, RollSizeMB: 10, RollKeep: 5, RollKeepDays: 7},
				Encoder: &LogEncoder{Format: "json"},
				Level:   "INFO",
			},
		},
	}

	// Build HTTP routes for all proxies
	routes, err := b.buildHTTPRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP routes: %w", err)
	}

	// Add catch-all route
	catchAllRoute := b.buildCatchAllRoute()
	if catchAllRoute != nil {
		routes = append(routes, catchAllRoute)
	}

	// Build HTTP app if we have routes
	if len(routes) > 0 {
		httpApp := NewHTTPApp()
		server := NewHTTPServer(":443", ":80")
		server.AddRoutes(routes...)
		b.applyTrustedProxies(server)
		server.Logs = &HTTPServerLogs{}
		httpApp.AddServer(DefaultServerName, server)
		config.Apps.HTTP = httpApp

		// Add HTTP access log config now that we have an HTTP server.
		config.Logging.Logs["access"] = &LogConfig{
			Include: []string{"http.log.access." + DefaultServerName},
			Writer:  &LogWriter{Output: "file", Filename: filepath.Join(logPath, "access.log"), Roll: &roll, RollSizeMB: 10, RollKeep: 5, RollKeepDays: 7},
			Encoder: &LogEncoder{Format: "json"},
		}
	}

	// Collect domains for TLS and build TLS app
	domains := b.collectTLSDomains()
	if len(domains) > 0 {
		tlsApp, err := b.tlsBuilder.Build(domains)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		config.Apps.TLS = tlsApp
	}

	// Set Layer4 app if configured
	if b.layer4App != nil && len(b.layer4App.Servers) > 0 {
		config.Apps.Layer4 = b.layer4App
	}

	return config, nil
}

// applyTrustedProxies configures the server to resolve the real client IP from
// trusted upstream proxies (e.g. a Cloudflare or Pangolin tunnel). It is a no-op
// when no trusted proxies are configured, leaving Caddy as the edge.
func (b *Builder) applyTrustedProxies(server *HTTPServer) {
	if len(b.trustedProxies) == 0 {
		return
	}
	server.TrustedProxies = &TrustedProxiesSource{
		Source: "static",
		Ranges: b.trustedProxies,
	}
	if len(b.clientIPHeaders) > 0 {
		server.ClientIPHeaders = b.clientIPHeaders
	}
}

// BuildJSON generates the Caddy configuration as formatted JSON bytes.
func (b *Builder) BuildJSON() ([]byte, error) {
	config, err := b.Build()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(config, "", "  ")
}

// BuildCompactJSON generates the Caddy configuration as compact JSON bytes.
func (b *Builder) BuildCompactJSON() ([]byte, error) {
	config, err := b.Build()
	if err != nil {
		return nil, err
	}
	return json.Marshal(config)
}

// buildHTTPRoutes builds routes for all HTTP proxies.
func (b *Builder) buildHTTPRoutes() ([]*HTTPRoute, error) {
	var routes []*HTTPRoute

	for i := range b.httpProxies {
		proxy := &b.httpProxies[i]
		if !proxy.IsActive {
			continue
		}

		proxyRoutes, err := b.buildProxyRoutes(proxy)
		if err != nil {
			b.logger.Warn("Failed to build routes for proxy",
				zap.Int("proxy_id", proxy.ID),
				zap.String("proxy_name", proxy.Name),
				zap.Error(err),
			)
			continue
		}

		routes = append(routes, proxyRoutes...)
	}

	// Add routes for L4 TLS hostnames (for ACME certificate provisioning)
	l4Routes := b.buildL4TLSRoutes()
	routes = append(routes, l4Routes...)

	return routes, nil
}

// buildL4TLSRoutes builds HTTP routes for L4 SNI hostnames that need TLS certificates.
// These routes serve a simple response to enable ACME certificate provisioning.
func (b *Builder) buildL4TLSRoutes() []*HTTPRoute {
	if len(b.l4TLSHostnames) == 0 {
		return nil
	}

	// Collect hostnames that don't already have an HTTP proxy
	httpHostnames := make(map[string]bool)
	for i := range b.httpProxies {
		if b.httpProxies[i].IsActive {
			httpHostnames[b.httpProxies[i].Hostname] = true
		}
	}

	var routes []*HTTPRoute
	for _, hostname := range b.l4TLSHostnames {
		// Skip if already handled by an HTTP proxy
		if httpHostnames[hostname] {
			continue
		}

		// Create a simple route that responds with "L4 Proxy" for ACME
		route := &HTTPRoute{
			Match: []MatcherSet{
				NewHostMatcher(hostname),
			},
			Handle: []HTTPHandler{
				{
					"handler": "static_response",
					"body":    "L4 Proxy - TLS Certificate Endpoint",
				},
			},
		}
		routes = append(routes, route)

		b.logger.Debug("Added L4 TLS certificate route",
			zap.String("hostname", hostname))
	}

	return routes
}

// buildProxyRoutes builds routes for a single proxy.
func (b *Builder) buildProxyRoutes(proxy *models.Proxy) ([]*HTTPRoute, error) {
	var routes []*HTTPRoute

	// Add security routes if BlockExploits is enabled
	if proxy.BlockExploits {
		securityRoutes := SecurityRoutesForHost(proxy.Hostname)
		routes = append(routes, securityRoutes...)
		b.logger.Debug("Added security routes for proxy",
			zap.String("hostname", proxy.Hostname),
			zap.Int("security_routes", len(securityRoutes)))
	}

	// Check for ACL assignments
	assignments := b.aclAssigns[int64(proxy.ID)]
	hasACL := len(assignments) > 0 && b.aclBuilder != nil

	var proxyRoutes []*HTTPRoute
	var err error

	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		if hasACL {
			proxyRoutes, err = b.httpBuilder.BuildReverseProxyRoutesWithACL(proxy, assignments, b.aclGroups, b.aclBuilder)
		} else {
			proxyRoutes, err = b.httpBuilder.BuildReverseProxyRoutes(proxy)
		}

	case models.ProxyTypeRedirect:
		proxyRoutes, err = b.httpBuilder.BuildRedirectRoutes(proxy)

	case models.ProxyTypeStatic:
		proxyRoutes, err = b.httpBuilder.BuildStaticRoutes(proxy)

	default:
		return nil, fmt.Errorf("unknown proxy type: %s", proxy.Type)
	}

	if err != nil {
		return nil, err
	}

	routes = append(routes, proxyRoutes...)
	return routes, nil
}

// buildCatchAllRoute builds the catch-all route for unmatched requests.
func (b *Builder) buildCatchAllRoute() *HTTPRoute {
	if b.notFound == nil {
		return NewCatchAllRoute()
	}

	if b.notFound.Mode == "redirect" && b.notFound.RedirectURL != "" {
		return NewCatchAllRedirectRoute(b.notFound.RedirectURL)
	}

	return NewCatchAllRoute()
}

// collectTLSDomains collects all domains that need TLS certificates.
func (b *Builder) collectTLSDomains() []string {
	domainSet := make(map[string]bool)

	for i := range b.httpProxies {
		proxy := &b.httpProxies[i]
		if !proxy.IsActive {
			continue
		}
		// Only collect domains for SSL-enabled proxies
		if proxy.SSLEnabled {
			domainSet[proxy.Hostname] = true
		}
	}

	// Add L4 TLS hostnames for certificate provisioning
	for _, hostname := range b.l4TLSHostnames {
		domainSet[hostname] = true
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}

	// Sort domains for deterministic JSON output
	sort.Strings(domains)

	return domains
}

// BuildSingleProxy generates JSON configuration for a single proxy.
// This is useful for validation or preview purposes.
func (b *Builder) BuildSingleProxy(proxy *models.Proxy) (*CaddyConfig, error) {
	routes, err := b.buildProxyRoutes(proxy)
	if err != nil {
		return nil, err
	}

	config := &CaddyConfig{
		Apps: &AppsConfig{
			HTTP: &HTTPApp{
				Servers: map[string]*HTTPServer{
					DefaultServerName: {
						Listen: []string{":443"},
						Routes: routes,
					},
				},
			},
		},
	}

	if proxy.SSLEnabled {
		config.Apps.TLS = NewTLSApp([]string{proxy.Hostname})
	}

	return config, nil
}
