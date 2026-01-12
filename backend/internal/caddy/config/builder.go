// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"encoding/json"
	"fmt"

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

	// Configuration inputs
	settings    *Settings
	httpProxies []models.Proxy
	aclGroups   map[int64]*models.ACLGroup
	aclAssigns  map[int64][]models.ProxyACLAssignment
	notFound    *models.NotFoundSettings
}

// Settings holds the application settings for building the config.
type Settings struct {
	AdminEmail   string
	ACMEProvider string
	StoragePath  string

	// Waygates auth URLs
	WaygatesVerifyURL string
	WaygatesLoginURL  string

	// DNS provider credentials (loaded from environment)
	DNSCredentials map[string]string
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
	b.tlsBuilder.SetSettings(settings)
	if b.aclBuilder != nil && settings != nil {
		b.aclBuilder.SetWaygatesURLs(settings.WaygatesVerifyURL, settings.WaygatesLoginURL)
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
		httpApp.AddServer(DefaultServerName, server)
		config.Apps.HTTP = httpApp
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

	return config, nil
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

	for _, proxy := range b.httpProxies {
		if !proxy.IsActive {
			continue
		}

		proxyRoutes, err := b.buildProxyRoutes(&proxy)
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

	return routes, nil
}

// buildProxyRoutes builds routes for a single proxy.
func (b *Builder) buildProxyRoutes(proxy *models.Proxy) ([]*HTTPRoute, error) {
	// Check for ACL assignments
	assignments := b.aclAssigns[int64(proxy.ID)]
	hasACL := len(assignments) > 0 && b.aclBuilder != nil

	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		if hasACL {
			return b.httpBuilder.BuildReverseProxyRoutesWithACL(proxy, assignments, b.aclGroups, b.aclBuilder)
		}
		return b.httpBuilder.BuildReverseProxyRoutes(proxy)

	case models.ProxyTypeRedirect:
		return b.httpBuilder.BuildRedirectRoutes(proxy)

	case models.ProxyTypeStatic:
		return b.httpBuilder.BuildStaticRoutes(proxy)

	default:
		return nil, fmt.Errorf("unknown proxy type: %s", proxy.Type)
	}
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

	for _, proxy := range b.httpProxies {
		if !proxy.IsActive {
			continue
		}
		// Only collect domains for SSL-enabled proxies
		if proxy.SSLEnabled {
			domainSet[proxy.Hostname] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}

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
