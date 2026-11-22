// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"fmt"
	"net"
	"net/http"
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
		"X-Authentik-Username",
		"X-Authentik-Groups",
		"X-Authentik-Email",
		"X-Authentik-Name",
		"X-Authentik-Uid",
		"X-Authentik-Entitlements",
		"X-Authentik-Jwt",
		"X-Authentik-Meta-Jwks",
		"X-Authentik-Meta-Outpost",
		"X-Authentik-Meta-Provider",
		"X-Authentik-Meta-App",
		"X-Authentik-Meta-Version",
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

	// Validate stored configurations too; invalid legacy data must never produce
	// an unprotected fallback route or prevent unrelated sites from loading.
	for i := range group.ExternalProviders {
		if err := group.ExternalProviders[i].Validate(); err != nil {
			b.logger.Error("Invalid external ACL provider; failing closed", zap.Error(err))
			return []*HTTPRoute{aclUnavailableRoute(hostname, pathPattern)}, nil
		}
	}

	var routes []*HTTPRoute
	var err error
	switch group.CombinationMode {
	case models.ACLCombinationModeAll:
		routes, err = b.buildAllModeRoutes(hostname, pathPattern, group, upstreamHandler)
	case models.ACLCombinationModeIPBypass:
		routes, err = b.buildIPBypassModeRoutes(hostname, pathPattern, group, upstreamHandler)
	default:
		routes, err = b.buildAnyModeRoutes(hostname, pathPattern, group, upstreamHandler)
	}
	for _, route := range routes {
		route.Handle = append([]HTTPHandler{identityHeaderSanitizer(group)}, route.Handle...)
	}
	return routes, err
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

	// 4. Authentication route for remaining requests
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
	allAllowedIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	allAllowedIPs = append(allAllowedIPs, bypassIPs...)
	allAllowedIPs = append(allAllowedIPs, allowIPs...)
	if len(allAllowedIPs) > 0 {
		// Deny requests NOT from allowed IPs
		route := b.buildIPDenyNotInListRoute(hostname, pathPattern, allAllowedIPs)
		routes = append(routes, route)
	}

	// 3. Auth route for requests from allowed IPs
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

	excludeIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	excludeIPs = append(excludeIPs, bypassIPs...)
	excludeIPs = append(excludeIPs, allowIPs...)
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

// buildWaygatesForwardAuthHandler keeps Waygates' base-URL configuration and
// 401 login behavior, while sharing GET/body/TLS/identity handling with external auth.
func (b *ACLBuilder) buildWaygatesForwardAuthHandler() HTTPHandler {
	if b.waygatesVerifyURL == "" {
		return ToHTTPHandler(NewStaticResponseHandler(503,
			"Waygates forward-auth is not configured (ACL_WAYGATES_VERIFY_URL is empty)"))
	}
	u, err := models.ParseAuthURL(b.waygatesVerifyURL)
	if err != nil {
		return ToHTTPHandler(NewStaticResponseHandler(503, "Invalid Waygates authentication URL"))
	}
	// ACL_WAYGATES_VERIFY_URL has historically been a base URL.
	u.Path, u.RawPath, u.RawQuery, u.ForceQuery = "/api/auth/acl/verify", "", "", false
	handler := newForwardAuthHandler(u, waygatesDefaultHeaders)
	handler["headers"].(*HeadersConfig).Request.Set["X-Waygates-Client-IP"] = []string{"{http.vars.client_ip}"}

	var unauthorized HTTPHandler
	if b.waygatesLoginURL == "" {
		unauthorized = ToHTTPHandler(NewStaticResponseHandler(503,
			"Waygates forward-auth is not configured (ACL_WAYGATES_LOGIN_URL is empty)"))
	} else {
		unauthorized = ToHTTPHandler(NewRedirectHandler(
			fmt.Sprintf("%s?redirect={http.request.scheme}://{http.request.host}{http.request.uri}", b.waygatesLoginURL), 302))
	}
	handler["handle_response"] = append(handler["handle_response"].([]*HandleResponse),
		&HandleResponse{Match: &ResponseMatch{StatusCode: []int{401}}, Routes: []*HTTPRoute{{Handle: []HTTPHandler{unauthorized}}}},
		&HandleResponse{Match: &ResponseMatch{StatusCode: []int{403}}, Routes: []*HTTPRoute{{Handle: []HTTPHandler{ToHTTPHandler(NewStaticResponseHandler(403, "Access Denied"))}}}},
	)
	return handler
}

// buildExternalProviderHandler leaves non-2xx responses untouched: the provider
// owns its login redirects, cookies and denials. AuthRedirectURL is deprecated.
func (b *ACLBuilder) buildExternalProviderHandler(provider models.ACLExternalProvider) HTTPHandler {
	u, err := models.ParseAuthURL(provider.VerifyURL)
	if err != nil {
		return ToHTTPHandler(NewStaticResponseHandler(503, "Invalid external authentication URL"))
	}
	return newForwardAuthHandler(u, externalIdentityHeaders(provider))
}

func externalIdentityHeaders(provider models.ACLExternalProvider) []string {
	if len(provider.HeadersToCopy) > 0 {
		return provider.HeadersToCopy
	}
	return providerDefaultHeaders[provider.ProviderType]
}

func newForwardAuthHandler(endpoint *url.URL, headers []string) HTTPHandler {
	handler := authEndpointProxy(endpoint)
	// An explicit '?' also clears the application's query when the configured
	// endpoint has none. Caddy otherwise preserves the old query on a path rewrite.
	uri := endpoint.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	handler["rewrite"] = &RewriteHeaders{Method: http.MethodGet, URI: uri + "?" + endpoint.RawQuery}
	handler["handle_response"] = []*HandleResponse{forwardAuthSuccess(headers)}
	return handler
}

// authEndpointProxy also serves the outpost's callback/logout routes. Without a
// rewrite, it preserves their method, URI, body and response exactly.
func authEndpointProxy(endpoint *url.URL) HTTPHandler {
	requestHeaders := &HeaderOps{Set: map[string][]string{
		"X-Forwarded-Method": {"{http.request.method}"},
		"X-Forwarded-Proto":  {"{http.request.scheme}"},
		"X-Forwarded-Host":   {"{http.request.host}"},
		"X-Forwarded-Uri":    {"{http.request.uri}"},
	}}
	transport := &HTTPTransport{Protocol: "http", DialTimeout: Duration("5s"), ResponseHeaderTimeout: Duration("10s")}
	if endpoint.Scheme == "https" {
		transport.TLS = &TLSConfig{ServerName: endpoint.Hostname()}
		requestHeaders.Set["Host"] = []string{endpoint.Host}
	}
	return HTTPHandler{
		"handler":   HandlerReverseProxy,
		"upstreams": []*Upstream{{Dial: extractDialAddress(endpoint.String())}},
		"transport": transport,
		"headers":   &HeadersConfig{Request: requestHeaders},
	}
}

// forwardAuthSuccess mirrors Caddy forward_auth's request-header copy semantics.
// Delete first, then copy only nonempty values, so missing claims cannot retain
// client input or turn into literal unresolved placeholders.
func forwardAuthSuccess(headers []string) *HandleResponse {
	routes := make([]*HTTPRoute, 0, 1+2*len(headers))
	routes = append(routes, &HTTPRoute{Handle: []HTTPHandler{
		ToHTTPHandler(NewCopyResponseHeadersHandler([]string{"Set-Cookie"})),
	}})
	for _, name := range headers {
		name = http.CanonicalHeaderKey(name)
		placeholder := "{http.reverse_proxy.header." + name + "}"
		routes = append(routes,
			&HTTPRoute{Handle: []HTTPHandler{ToHTTPHandler(NewHeadersHandler(&HeaderOps{Delete: []string{name}}, nil))}},
			&HTTPRoute{
				Match:  []MatcherSet{NewNotMatcher(MatcherSet{"vars": map[string][]string{placeholder: {""}}})},
				Handle: []HTTPHandler{ToHTTPHandler(NewHeadersHandler(&HeaderOps{Set: map[string][]string{name: {placeholder}}}, nil))},
			},
		)
	}
	return &HandleResponse{Match: &ResponseMatch{StatusCode: []int{2}}, Routes: routes}
}

// identityHeaderSanitizer runs before authentication AND on IP/API exceptions.
// Reserve known identity namespaces even when a custom copy list is configured.
func identityHeaderSanitizer(groups ...*models.ACLGroup) HTTPHandler {
	names := []string{"X-Authentik-*", "X-Auth-*", "Remote-User", "Remote-Groups", "Remote-Name", "Remote-Email", "X-Forwarded-User"}
	for _, group := range groups {
		if group == nil {
			continue
		}
		for i := range group.ExternalProviders {
			names = append(names, group.ExternalProviders[i].HeadersToCopy...)
		}
	}
	return ToHTTPHandler(NewHeadersHandler(&HeaderOps{Delete: names}, nil))
}

func aclUnavailableRoute(hostname, pathPattern string) *HTTPRoute {
	matcher := NewHostMatcher(hostname)
	if pathPattern != "" && pathPattern != "/*" {
		AddPathToMatcher(matcher, pathPattern)
	}
	return &HTTPRoute{Match: []MatcherSet{matcher}, Handle: []HTTPHandler{
		ToHTTPHandler(NewStaticResponseHandler(503, "ACL authentication is not configured correctly")),
	}, Terminal: true}
}

// authentikOutpostRoute reserves only Authentik's own namespace on the app host.
// A hostname can have only one outpost origin, even with path-scoped ACLs.
func authentikOutpostRoute(hostname string, groups []*models.ACLGroup) (*HTTPRoute, error) {
	var destination *url.URL
	for _, group := range groups {
		if group == nil || len(group.ExternalProviders) == 0 {
			continue
		}
		// Preserve the existing selection of the first external provider.
		provider := group.ExternalProviders[0]
		if provider.ProviderType != models.ACLProviderTypeAuthentik {
			continue
		}
		if err := provider.Validate(); err != nil {
			return nil, err
		}
		raw := provider.VerifyURL
		if provider.OutpostURL != nil && *provider.OutpostURL != "" {
			raw = *provider.OutpostURL
		}
		u, err := models.ParseAuthURL(raw)
		if err != nil {
			return nil, err
		}
		if destination != nil && (destination.Scheme != u.Scheme || destination.Host != u.Host) {
			return nil, fmt.Errorf("conflicting Authentik outpost origins on %s", hostname)
		}
		destination = u
	}
	if destination == nil {
		return nil, nil
	}
	return &HTTPRoute{
		Match:  []MatcherSet{NewHostPathMatcher(hostname, "/outpost.goauthentik.io", "/outpost.goauthentik.io/*")},
		Handle: []HTTPHandler{authEndpointProxy(destination)}, Terminal: true,
	}, nil
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

	return net.JoinHostPort(host, port)
}

// containsScheme checks if a string contains a URL scheme (e.g., "http://", "https://").
func containsScheme(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}
