// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// HTTPBuilder builds HTTP routes from proxy configurations.
type HTTPBuilder struct {
	logger *zap.Logger
}

// NewHTTPBuilder creates a new HTTP builder.
func NewHTTPBuilder(logger *zap.Logger) *HTTPBuilder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HTTPBuilder{
		logger: logger,
	}
}

// BuildReverseProxyRoutes builds routes for a reverse proxy.
func (b *HTTPBuilder) BuildReverseProxyRoutes(proxy *models.Proxy) ([]*HTTPRoute, error) {
	if proxy.Upstreams == nil {
		return nil, fmt.Errorf("reverse proxy requires at least one upstream")
	}

	upstreams, err := b.parseUpstreams(proxy.Upstreams)
	if err != nil {
		return nil, err
	}

	if len(upstreams) == 0 {
		return nil, fmt.Errorf("reverse proxy requires at least one upstream")
	}

	// Build the reverse proxy handler
	handler := b.buildReverseProxyHandler(proxy, upstreams)

	// Create the route
	route := NewHTTPRoute()
	route.AddMatch(NewHostMatcher(proxy.Hostname))
	route.AddHandler(handlerToMap(handler))
	route.SetTerminal(true)

	return []*HTTPRoute{route}, nil
}

// BuildReverseProxyRoutesWithACL builds routes for a reverse proxy with ACL protection.
func (b *HTTPBuilder) BuildReverseProxyRoutesWithACL(
	proxy *models.Proxy,
	assignments []models.ProxyACLAssignment,
	aclGroups map[int64]*models.ACLGroup,
	aclBuilder *ACLBuilder,
) ([]*HTTPRoute, error) {
	if proxy.Upstreams == nil {
		return nil, fmt.Errorf("reverse proxy requires at least one upstream")
	}

	upstreams, err := b.parseUpstreams(proxy.Upstreams)
	if err != nil {
		return nil, err
	}

	if len(upstreams) == 0 {
		return nil, fmt.Errorf("reverse proxy requires at least one upstream")
	}

	hostname := proxy.Hostname
	handler := b.buildReverseProxyHandler(proxy, upstreams)

	// Sort assignments by priority (lower number = higher priority)
	sortedAssignments := make([]models.ProxyACLAssignment, len(assignments))
	copy(sortedAssignments, assignments)
	sort.Slice(sortedAssignments, func(i, j int) bool {
		return sortedAssignments[i].Priority < sortedAssignments[j].Priority
	})

	// Build ACL routes
	var routes []*HTTPRoute
	for _, assignment := range sortedAssignments {
		group, ok := aclGroups[int64(assignment.ACLGroupID)]
		if !ok {
			b.logger.Warn("ACL group not found for assignment",
				zap.Int("assignment_id", assignment.ID),
				zap.Int("group_id", assignment.ACLGroupID),
			)
			continue
		}

		aclRoutes, err := aclBuilder.BuildACLRoutes(hostname, assignment.PathPattern, group, handler)
		if err != nil {
			b.logger.Warn("Failed to build ACL routes",
				zap.Int("assignment_id", assignment.ID),
				zap.Error(err),
			)
			continue
		}

		routes = append(routes, aclRoutes...)
	}

	// Add fallback route for paths not covered by ACL
	fallbackRoute := NewHTTPRoute()
	fallbackRoute.AddMatch(NewHostMatcher(hostname))
	fallbackRoute.AddHandler(handlerToMap(handler))
	fallbackRoute.SetTerminal(true)
	routes = append(routes, fallbackRoute)

	return routes, nil
}

// BuildRedirectRoutes builds routes for a redirect proxy.
func (b *HTTPBuilder) BuildRedirectRoutes(proxy *models.Proxy) ([]*HTTPRoute, error) {
	redirectConfig, err := b.parseRedirectConfig(proxy.RedirectConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect config: %w", err)
	}

	targetURL := redirectConfig.Target
	// Use Caddy's full JSON placeholders, not the Caddyfile shorthands {uri}/
	// {query}: the shorthands are only expanded by the Caddyfile adapter and
	// resolve to empty strings in a raw JSON config.
	if redirectConfig.PreservePath {
		// {http.request.uri} is the full request URI (path and any query string).
		targetURL += "{http.request.uri}"
	} else if redirectConfig.PreserveQuery {
		// Prefix with '?' so the query is appended as a proper separator
		// (e.g. "target?foo=bar"); the query placeholder carries no leading '?'.
		targetURL += "?{http.request.uri.query}"
	}

	statusCode := redirectConfig.StatusCode
	if statusCode == 0 {
		statusCode = 302 // Default to temporary redirect
	}

	handler := NewRedirectHandler(targetURL, statusCode)

	route := NewHTTPRoute()
	route.AddMatch(NewHostMatcher(proxy.Hostname))
	route.AddHandler(ToHTTPHandler(handler))
	route.SetTerminal(true)

	return []*HTTPRoute{route}, nil
}

// BuildStaticRoutes builds routes for a static file server proxy.
func (b *HTTPBuilder) BuildStaticRoutes(proxy *models.Proxy) ([]*HTTPRoute, error) {
	staticConfig, err := b.parseStaticConfig(proxy.StaticConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid static config: %w", err)
	}

	handler := NewFileServerHandler(staticConfig.RootPath)

	if staticConfig.IndexFile != "" {
		handler.WithIndexNames(staticConfig.IndexFile)
	}

	if staticConfig.Browse {
		handler.WithBrowse("")
	}

	hostname := proxy.Hostname
	var routes []*HTTPRoute

	// If try_files is configured (for SPAs), add a rewrite route gated by a
	// file matcher. Caddy's file matcher tests each candidate against disk and
	// exposes the first existing one via {http.matchers.file.relative}; we
	// rewrite the request to that file. The matcher is required: without it the
	// rewrite fires for every request, so the file server returns the SPA
	// fallback (e.g. /index.html) even for real assets like /app.js.
	if len(staticConfig.TryFiles) > 0 {
		// Host and file must share one matcher set so both must match (AND).
		match := NewHostMatcher(hostname)
		match["file"] = &MatchFile{
			Root:     staticConfig.RootPath,
			TryFiles: normalizeTryFiles(staticConfig.TryFiles),
		}

		rewriteRoute := NewHTTPRoute()
		rewriteRoute.AddMatch(match)
		rewriteRoute.AddHandler(HTTPHandler{
			"handler": HandlerRewrite,
			"uri":     "{http.matchers.file.relative}",
		})
		routes = append(routes, rewriteRoute)
	}

	// Main file server route
	route := NewHTTPRoute()
	route.AddMatch(NewHostMatcher(hostname))
	route.AddHandler(ToHTTPHandler(handler))
	route.SetTerminal(true)
	routes = append(routes, route)

	return routes, nil
}

// buildReverseProxyHandler builds a reverse proxy handler with all configurations.
func (b *HTTPBuilder) buildReverseProxyHandler(proxy *models.Proxy, upstreams []*Upstream) *ReverseProxyHandler {
	handler := NewReverseProxyHandler(upstreams...)

	// Add standard proxy headers
	handler.Headers = &HeadersConfig{
		Request: StandardProxyHeaders(),
	}

	// Add custom request headers (forwarded to the upstream).
	for key, value := range proxy.CustomHeaders.Request {
		handler.Headers.Request.SetHeader(key, value)
	}
	// Add custom response headers (returned to the client).
	if len(proxy.CustomHeaders.Response) > 0 {
		if handler.Headers.Response == nil {
			handler.Headers.Response = &HeaderOps{}
		}
		for key, value := range proxy.CustomHeaders.Response {
			handler.Headers.Response.SetHeader(key, value)
		}
	}

	// Configure load balancing
	if len(proxy.LoadBalancing) > 0 {
		b.configureLoadBalancing(handler, proxy.LoadBalancing)
	}

	// Configure TLS transport if needed
	hasHTTPS := b.hasHTTPSUpstream(proxy.Upstreams)
	if hasHTTPS || proxy.TLSInsecureSkipVerify {
		handler.Transport = &HTTPTransport{
			Protocol: "http", // Required by Caddy to identify the transport module
			TLS: &TLSConfig{
				InsecureSkipVerify: proxy.TLSInsecureSkipVerify,
			},
		}
	}

	return handler
}

// configureLoadBalancing configures load balancing on a reverse proxy handler.
func (b *HTTPBuilder) configureLoadBalancing(handler *ReverseProxyHandler, lb models.JSONField) {
	strategy, _ := lb["strategy"].(string)
	if strategy == "" {
		return
	}

	handler.LoadBalancing = &LoadBalancing{
		SelectionPolicy: &SelectionPolicy{
			Policy: mapLBStrategy(strategy),
		},
	}

	// Configure health checks if enabled
	if healthChecks, ok := lb["health_checks"].(map[string]interface{}); ok {
		if enabled, _ := healthChecks["enabled"].(bool); enabled {
			handler.HealthChecks = &HealthChecks{
				Active: &ActiveHealthCheck{},
			}

			if path, ok := healthChecks["path"].(string); ok && path != "" {
				handler.HealthChecks.Active.Path = path
			}
			if interval, ok := healthChecks["interval"].(string); ok && interval != "" {
				handler.HealthChecks.Active.Interval = Duration(interval)
			}
			if timeout, ok := healthChecks["timeout"].(string); ok && timeout != "" {
				handler.HealthChecks.Active.Timeout = Duration(timeout)
			}
		}
	}
}

// parseUpstreams parses upstream configuration from interface{}.
func (b *HTTPBuilder) parseUpstreams(upstreamsRaw interface{}) ([]*Upstream, error) {
	upstreamsList, ok := upstreamsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("upstreams must be an array")
	}

	var upstreams []*Upstream
	for _, up := range upstreamsList {
		upstreamMap, ok := up.(map[string]interface{})
		if !ok {
			continue
		}

		host, _ := upstreamMap["host"].(string)
		port, _ := upstreamMap["port"].(float64)
		scheme, _ := upstreamMap["scheme"].(string)

		if host == "" {
			continue
		}

		dial := host
		if port > 0 {
			dial = fmt.Sprintf("%s:%d", host, int(port))
		}

		// For HTTPS upstreams, Caddy needs to know to use TLS
		// This is handled at the transport level, not the dial address
		_ = scheme // Used to detect HTTPS for transport config

		upstreams = append(upstreams, &Upstream{
			Dial: dial,
		})
	}

	return upstreams, nil
}

// hasHTTPSUpstream checks if any upstream uses HTTPS.
func (b *HTTPBuilder) hasHTTPSUpstream(upstreamsRaw interface{}) bool {
	upstreamsList, ok := upstreamsRaw.([]interface{})
	if !ok {
		return false
	}

	for _, up := range upstreamsList {
		upstreamMap, ok := up.(map[string]interface{})
		if !ok {
			continue
		}
		scheme, _ := upstreamMap["scheme"].(string)
		if scheme == "https" {
			return true
		}
	}

	return false
}

// RedirectConfig represents redirect proxy configuration.
type RedirectConfig struct {
	Target        string `json:"target"`
	StatusCode    int    `json:"status_code"`
	PreservePath  bool   `json:"preserve_path"`
	PreserveQuery bool   `json:"preserve_query"`
}

// parseRedirectConfig parses redirect configuration from JSONField.
func (b *HTTPBuilder) parseRedirectConfig(config models.JSONField) (*RedirectConfig, error) {
	if len(config) == 0 {
		return nil, fmt.Errorf("redirect config is required")
	}

	rc := &RedirectConfig{}

	if target, ok := config["target"].(string); ok {
		rc.Target = target
	} else {
		return nil, fmt.Errorf("redirect target is required")
	}

	if statusCode, ok := config["status_code"].(float64); ok {
		rc.StatusCode = int(statusCode)
	}

	if preservePath, ok := config["preserve_path"].(bool); ok {
		rc.PreservePath = preservePath
	}

	if preserveQuery, ok := config["preserve_query"].(bool); ok {
		rc.PreserveQuery = preserveQuery
	}

	return rc, nil
}

// StaticConfig represents static file server configuration.
type StaticConfig struct {
	RootPath          string   `json:"root_path"`
	IndexFile         string   `json:"index_file"`
	TryFiles          []string `json:"try_files"`
	TemplateRendering bool     `json:"template_rendering"`
	Browse            bool     `json:"browse"`
}

// tryFilesPlaceholderReplacer converts Caddyfile placeholder shorthands that
// may be stored in proxy config into the full placeholders Caddy's JSON config
// requires, so the file matcher resolves them correctly.
var tryFilesPlaceholderReplacer = strings.NewReplacer(
	"{path}", "{http.request.uri.path}",
	"{uri}", "{http.request.uri}",
)

// normalizeTryFiles rewrites Caddyfile placeholder shorthands in try_files
// entries to their JSON-config equivalents. Literal paths are left unchanged.
func normalizeTryFiles(tryFiles []string) []string {
	normalized := make([]string, len(tryFiles))
	for i, tf := range tryFiles {
		normalized[i] = tryFilesPlaceholderReplacer.Replace(tf)
	}
	return normalized
}

// parseStaticConfig parses static file server configuration from JSONField.
func (b *HTTPBuilder) parseStaticConfig(config models.JSONField) (*StaticConfig, error) {
	if len(config) == 0 {
		return nil, fmt.Errorf("static config is required")
	}

	sc := &StaticConfig{}

	if rootPath, ok := config["root_path"].(string); ok {
		sc.RootPath = rootPath
	} else {
		return nil, fmt.Errorf("root_path is required for static file server")
	}

	if indexFile, ok := config["index_file"].(string); ok {
		sc.IndexFile = indexFile
	}

	if tryFiles, ok := config["try_files"].([]interface{}); ok {
		for _, tf := range tryFiles {
			if s, ok := tf.(string); ok {
				sc.TryFiles = append(sc.TryFiles, s)
			}
		}
	}

	if templateRendering, ok := config["template_rendering"].(bool); ok {
		sc.TemplateRendering = templateRendering
	}

	if browse, ok := config["browse"].(bool); ok {
		sc.Browse = browse
	}

	return sc, nil
}

// mapLBStrategy maps strategy names to Caddy's policy names.
func mapLBStrategy(strategy string) string {
	switch strategy {
	case "round_robin":
		return "round_robin"
	case "least_conn":
		return "least_conn"
	case "random":
		return "random"
	case "first":
		return "first"
	case "ip_hash":
		return "ip_hash"
	case "uri_hash":
		return "uri_hash"
	case "header":
		return "header"
	default:
		return "round_robin"
	}
}

// handlerToMap converts a ReverseProxyHandler to HTTPHandler map.
func handlerToMap(h *ReverseProxyHandler) HTTPHandler {
	result := HTTPHandler{
		"handler":   h.Handler,
		"upstreams": h.Upstreams,
	}

	if h.LoadBalancing != nil {
		result["load_balancing"] = h.LoadBalancing
	}
	if h.HealthChecks != nil {
		result["health_checks"] = h.HealthChecks
	}
	if h.Transport != nil {
		result["transport"] = h.Transport
	}
	if h.Headers != nil {
		result["headers"] = h.Headers
	}

	return result
}
