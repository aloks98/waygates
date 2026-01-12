// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"fmt"
	"net/url"
	"sort"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// ACLBuilder builds ACL routes for authentication and authorization.
type ACLBuilder struct {
	logger            *zap.Logger
	waygatesVerifyURL string
	waygatesLoginURL  string
}

// NewACLBuilder creates a new ACL builder.
func NewACLBuilder(logger *zap.Logger) *ACLBuilder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ACLBuilder{
		logger: logger,
	}
}

// SetWaygatesURLs sets the Waygates authentication URLs.
func (b *ACLBuilder) SetWaygatesURLs(verifyURL, loginURL string) *ACLBuilder {
	b.waygatesVerifyURL = verifyURL
	b.waygatesLoginURL = loginURL
	return b
}

// Default headers to copy from Waygates forward auth responses
var waygatesDefaultHeaders = []string{
	"X-Auth-User",
	"X-Auth-User-ID",
	"X-Auth-User-Email",
}

// Provider-specific default headers
var providerDefaultHeaders = map[string][]string{
	models.ACLProviderTypeAuthelia: {
		"Remote-User",
		"Remote-Groups",
		"Remote-Name",
		"Remote-Email",
	},
	models.ACLProviderTypeAuthentik: {
		"X-authentik-username",
		"X-authentik-groups",
		"X-authentik-email",
		"X-authentik-name",
		"X-authentik-uid",
	},
}

// BuildACLRoutes builds authentication routes for a proxy.
// Returns routes that should be placed BEFORE the main proxy route.
func (b *ACLBuilder) BuildACLRoutes(
	hostname string,
	pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
) ([]*HTTPRoute, error) {
	if group == nil {
		return nil, nil
	}

	// Analyze configured auth methods
	hasIPRules := len(group.IPRules) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0
	hasWaygatesAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasOAuthProviders := group.WaygatesAuth != nil && len(group.WaygatesAuth.AllowedProviders) > 0
	hasExternalProviders := len(group.ExternalProviders) > 0

	// If no auth methods configured, skip
	if !hasIPRules && !hasBasicAuth && !hasWaygatesAuth && !hasOAuthProviders && !hasExternalProviders {
		return nil, nil
	}

	// Build routes based on combination mode
	switch group.CombinationMode {
	case models.ACLCombinationModeAll:
		return b.buildAllModeRoutes(hostname, pathPattern, group, upstreamHandler)
	case models.ACLCombinationModeIPBypass:
		return b.buildIPBypassModeRoutes(hostname, pathPattern, group, upstreamHandler)
	default: // "any" mode is default
		return b.buildAnyModeRoutes(hostname, pathPattern, group, upstreamHandler)
	}
}

// buildAnyModeRoutes builds routes for ANY mode (OR logic).
// Any auth method can grant access.
func (b *ACLBuilder) buildAnyModeRoutes(
	hostname string,
	pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
) ([]*HTTPRoute, error) {
	var routes []*HTTPRoute

	// Categorize IP rules
	bypassIPs, allowIPs, denyIPs := categorizeIPRules(group.IPRules)

	// 1. IP deny route - highest priority
	if len(denyIPs) > 0 {
		route := b.buildIPDenyRoute(hostname, pathPattern, denyIPs)
		routes = append(routes, route)
	}

	// 2. IP bypass route - skip all auth
	if len(bypassIPs) > 0 {
		route := b.buildIPBypassRoute(hostname, pathPattern, bypassIPs, upstreamHandler)
		routes = append(routes, route)
	}

	// 3. IP allow route - grant access without auth
	if len(allowIPs) > 0 {
		route := b.buildIPAllowRoute(hostname, pathPattern, allowIPs, upstreamHandler)
		routes = append(routes, route)
	}

	// 4. Static assets bypass route
	route := b.buildStaticAssetsBypassRoute(hostname, upstreamHandler)
	routes = append(routes, route)

	// 5. Authentication route for remaining requests
	authRoute := b.buildAuthRoute(hostname, pathPattern, group, upstreamHandler, bypassIPs, allowIPs)
	if authRoute != nil {
		routes = append(routes, authRoute)
	}

	return routes, nil
}

// buildAllModeRoutes builds routes for ALL mode (AND logic).
// All auth methods must pass.
func (b *ACLBuilder) buildAllModeRoutes(
	hostname string,
	pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
) ([]*HTTPRoute, error) {
	var routes []*HTTPRoute

	bypassIPs, allowIPs, denyIPs := categorizeIPRules(group.IPRules)

	// 1. IP deny route
	if len(denyIPs) > 0 {
		route := b.buildIPDenyRoute(hostname, pathPattern, denyIPs)
		routes = append(routes, route)
	}

	// 2. For ALL mode, combine bypass and allow IPs - requests must come from these IPs
	allAllowedIPs := append(bypassIPs, allowIPs...)
	if len(allAllowedIPs) > 0 {
		// Deny requests NOT from allowed IPs
		route := b.buildIPDenyNotInListRoute(hostname, pathPattern, allAllowedIPs)
		routes = append(routes, route)
	}

	// 3. Static assets bypass
	route := b.buildStaticAssetsBypassRoute(hostname, upstreamHandler)
	routes = append(routes, route)

	// 4. Auth route for requests from allowed IPs
	authRoute := b.buildAuthRouteWithIPRestriction(hostname, pathPattern, group, upstreamHandler, allAllowedIPs)
	if authRoute != nil {
		routes = append(routes, authRoute)
	}

	return routes, nil
}

// buildIPBypassModeRoutes builds routes for IP_BYPASS mode.
// Similar to ANY mode with specific IP bypass handling.
func (b *ACLBuilder) buildIPBypassModeRoutes(
	hostname string,
	pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
) ([]*HTTPRoute, error) {
	// IP bypass mode is functionally the same as ANY mode
	return b.buildAnyModeRoutes(hostname, pathPattern, group, upstreamHandler)
}

// buildIPDenyRoute creates a route that denies requests from specific IPs.
func (b *ACLBuilder) buildIPDenyRoute(hostname, pathPattern string, denyIPs []string) *HTTPRoute {
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}
	AddRemoteIPToMatcher(matcher, denyIPs...)

	handler := NewStaticResponseHandler(403, "Forbidden")

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   []HTTPHandler{ToHTTPHandler(handler)},
		Terminal: true,
	}
}

// buildIPDenyNotInListRoute creates a route that denies requests NOT from allowed IPs.
func (b *ACLBuilder) buildIPDenyNotInListRoute(hostname, pathPattern string, allowedIPs []string) *HTTPRoute {
	// Create a matcher for NOT in the allowed IP list
	notMatcher := NewNotMatcher(NewRemoteIPMatcher(allowedIPs...))
	matcher := CombineMatchers(NewHostMatcher(hostname), notMatcher)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}

	handler := NewStaticResponseHandler(403, "Forbidden")

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   []HTTPHandler{ToHTTPHandler(handler)},
		Terminal: true,
	}
}

// buildIPBypassRoute creates a route that bypasses auth for specific IPs.
func (b *ACLBuilder) buildIPBypassRoute(hostname, pathPattern string, bypassIPs []string, upstreamHandler *ReverseProxyHandler) *HTTPRoute {
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}
	AddRemoteIPToMatcher(matcher, bypassIPs...)

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   []HTTPHandler{handlerToMap(upstreamHandler)},
		Terminal: true,
	}
}

// buildIPAllowRoute creates a route that allows requests from specific IPs without auth.
func (b *ACLBuilder) buildIPAllowRoute(hostname, pathPattern string, allowIPs []string, upstreamHandler *ReverseProxyHandler) *HTTPRoute {
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}
	AddRemoteIPToMatcher(matcher, allowIPs...)

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   []HTTPHandler{handlerToMap(upstreamHandler)},
		Terminal: true,
	}
}

// buildStaticAssetsBypassRoute creates a route that bypasses auth for static assets.
func (b *ACLBuilder) buildStaticAssetsBypassRoute(hostname string, upstreamHandler *ReverseProxyHandler) *HTTPRoute {
	// Combine static asset paths and extensions
	paths := append(StaticAssetPaths(), StaticAssetExtensions()...)

	matcher := CombineMatchers(
		NewHostMatcher(hostname),
		NewPathMatcher(paths...),
	)

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   []HTTPHandler{handlerToMap(upstreamHandler)},
		Terminal: true,
	}
}

// buildAuthRoute creates an authentication route.
func (b *ACLBuilder) buildAuthRoute(
	hostname, pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
	bypassIPs, allowIPs []string,
) *HTTPRoute {
	// Determine auth type
	hasWaygatesAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasOAuthProviders := group.WaygatesAuth != nil && len(group.WaygatesAuth.AllowedProviders) > 0
	hasExternalProviders := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	hasSecureAuth := hasWaygatesAuth || hasOAuthProviders || hasExternalProviders

	// Build matcher excluding bypass and allow IPs
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}

	excludeIPs := append(bypassIPs, allowIPs...)
	if len(excludeIPs) > 0 {
		notMatcher := NewNotMatcher(NewRemoteIPMatcher(excludeIPs...))
		matcher = CombineMatchers(matcher, notMatcher)
	}

	var handlers []HTTPHandler

	if hasBasicAuth && !hasSecureAuth {
		// Only basic auth configured
		authHandler := b.buildBasicAuthHandler(group.BasicAuthUsers)
		handlers = append(handlers, ToHTTPHandler(authHandler))
	} else if hasSecureAuth {
		// Forward auth
		forwardAuthHandler := b.buildForwardAuthHandler(group)
		handlers = append(handlers, forwardAuthHandler)
	}

	// Add upstream handler
	handlers = append(handlers, handlerToMap(upstreamHandler))

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   handlers,
		Terminal: true,
	}
}

// buildAuthRouteWithIPRestriction creates an auth route that requires requests from specific IPs.
func (b *ACLBuilder) buildAuthRouteWithIPRestriction(
	hostname, pathPattern string,
	group *models.ACLGroup,
	upstreamHandler *ReverseProxyHandler,
	allowedIPs []string,
) *HTTPRoute {
	hasWaygatesAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasOAuthProviders := group.WaygatesAuth != nil && len(group.WaygatesAuth.AllowedProviders) > 0
	hasExternalProviders := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	hasSecureAuth := hasWaygatesAuth || hasOAuthProviders || hasExternalProviders

	// Build matcher for allowed IPs
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}
	if len(allowedIPs) > 0 {
		AddRemoteIPToMatcher(matcher, allowedIPs...)
	}

	var handlers []HTTPHandler

	if hasBasicAuth && !hasSecureAuth {
		authHandler := b.buildBasicAuthHandler(group.BasicAuthUsers)
		handlers = append(handlers, ToHTTPHandler(authHandler))
	} else if hasSecureAuth {
		forwardAuthHandler := b.buildForwardAuthHandler(group)
		handlers = append(handlers, forwardAuthHandler)
	}

	handlers = append(handlers, handlerToMap(upstreamHandler))

	return &HTTPRoute{
		Match:    []MatcherSet{matcher},
		Handle:   handlers,
		Terminal: true,
	}
}

// buildBasicAuthHandler creates a basic auth handler.
func (b *ACLBuilder) buildBasicAuthHandler(users []models.ACLBasicAuthUser) *AuthenticationHandler {
	accounts := make([]*BasicAuthAccount, len(users))
	for i, user := range users {
		accounts[i] = NewBasicAuthAccount(user.Username, user.PasswordHash)
	}

	return NewAuthenticationHandler(accounts, "Protected")
}

// buildForwardAuthHandler creates a forward auth handler for Waygates or external providers.
func (b *ACLBuilder) buildForwardAuthHandler(group *models.ACLGroup) HTTPHandler {
	// Determine which provider to use
	if len(group.ExternalProviders) > 0 {
		return b.buildExternalProviderHandler(group.ExternalProviders[0])
	}

	// Use Waygates forward auth
	return b.buildWaygatesForwardAuthHandler()
}

// buildWaygatesForwardAuthHandler creates a Waygates forward auth handler.
func (b *ACLBuilder) buildWaygatesForwardAuthHandler() HTTPHandler {
	// Extract host:port from URL for Caddy's Dial field
	dialAddr := extractDialAddress(b.waygatesVerifyURL)

	// Waygates forward auth is implemented as a reverse_proxy with specific configuration
	return HTTPHandler{
		"handler": HandlerReverseProxy,
		"upstreams": []*Upstream{
			{Dial: dialAddr},
		},
		"headers": &HeadersConfig{
			Request: &HeaderOps{
				Set: map[string][]string{
					"X-Forwarded-Method": {"{http.request.method}"},
					"X-Forwarded-Proto":  {"{http.request.scheme}"},
					"X-Forwarded-Host":   {"{http.request.host}"},
					"X-Forwarded-Uri":    {"{http.request.uri}"},
				},
			},
		},
		"rewrite": &RewriteHeaders{
			URI: "/api/auth/acl/verify",
		},
		"handle_response": []*HandleResponse{
			{
				Match: &ResponseMatch{
					StatusCode: []int{401},
				},
				Routes: []*HTTPRoute{
					{
						Handle: []HTTPHandler{
							ToHTTPHandler(NewRedirectHandler(
								fmt.Sprintf("%s?redirect={http.request.scheme}://{http.request.host}{http.request.uri}", b.waygatesLoginURL),
								302,
							)),
						},
					},
				},
			},
			{
				Match: &ResponseMatch{
					StatusCode: []int{403},
				},
				Routes: []*HTTPRoute{
					{
						Handle: []HTTPHandler{
							ToHTTPHandler(NewStaticResponseHandler(403, "Access Denied")),
						},
					},
				},
			},
		},
	}
}

// buildExternalProviderHandler creates an external provider forward auth handler.
func (b *ACLBuilder) buildExternalProviderHandler(provider models.ACLExternalProvider) HTTPHandler {
	// Get headers to copy
	headers := provider.HeadersToCopy
	if len(headers) == 0 {
		if defaultHeaders, ok := providerDefaultHeaders[provider.ProviderType]; ok {
			headers = defaultHeaders
		}
	}

	headerSet := make(map[string][]string)
	for _, h := range headers {
		headerSet[h] = []string{"{http.reverse_proxy.header." + h + "}"}
	}

	// Extract host:port from URL for Caddy's Dial field
	dialAddr := extractDialAddress(provider.VerifyURL)

	return HTTPHandler{
		"handler": HandlerReverseProxy,
		"upstreams": []*Upstream{
			{Dial: dialAddr},
		},
		"headers": &HeadersConfig{
			Request: &HeaderOps{
				Set: map[string][]string{
					"X-Forwarded-Method": {"{http.request.method}"},
					"X-Forwarded-Proto":  {"{http.request.scheme}"},
					"X-Forwarded-Host":   {"{http.request.host}"},
					"X-Forwarded-Uri":    {"{http.request.uri}"},
				},
			},
		},
	}
}

// categorizeIPRules separates IP rules by type.
func categorizeIPRules(rules []models.ACLIPRule) (bypass, allow, deny []string) {
	// Sort by priority first
	sorted := make([]models.ACLIPRule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	for _, rule := range sorted {
		switch rule.RuleType {
		case models.ACLIPRuleTypeBypass:
			bypass = append(bypass, rule.CIDR)
		case models.ACLIPRuleTypeAllow:
			allow = append(allow, rule.CIDR)
		case models.ACLIPRuleTypeDeny:
			deny = append(deny, rule.CIDR)
		}
	}
	return
}

// GetDefaultWaygatesHeaders returns the default headers copied from Waygates auth.
func GetDefaultWaygatesHeaders() []string {
	return waygatesDefaultHeaders
}

// GetProviderDefaultHeaders returns the default headers for a provider type.
func GetProviderDefaultHeaders(providerType string) []string {
	if headers, ok := providerDefaultHeaders[providerType]; ok {
		return headers
	}
	return nil
}

// extractDialAddress extracts the host:port from a URL for Caddy's Dial field.
// Input: "http://waygates:8080" or "https://auth.example.com:443/verify"
// Output: "waygates:8080" or "auth.example.com:443"
func extractDialAddress(rawURL string) string {
	// If it's already just host:port, return as-is
	if !containsScheme(rawURL) {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If parsing fails, return the original (let Caddy handle the error)
		return rawURL
	}

	host := parsed.Hostname()
	port := parsed.Port()

	// If no port specified, use default based on scheme
	if port == "" {
		switch parsed.Scheme {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}

	return fmt.Sprintf("%s:%s", host, port)
}

// containsScheme checks if a string contains a URL scheme (e.g., "http://", "https://").
func containsScheme(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}
