// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// DefaultServerName is the default HTTP server name.
const DefaultServerName = "srv0"

// Common port configurations.
const (
	PortHTTP  = 80
	PortHTTPS = 443
)

// NewHTTPApp creates a new HTTP application configuration.
func NewHTTPApp() *HTTPApp {
	return &HTTPApp{
		Servers: make(map[string]*HTTPServer),
	}
}

// NewHTTPServer creates a new HTTP server with the given listen addresses.
func NewHTTPServer(listen ...string) *HTTPServer {
	return &HTTPServer{
		Listen: listen,
		Routes: make([]*HTTPRoute, 0),
	}
}

// NewHTTPRoute creates a new HTTP route.
func NewHTTPRoute() *HTTPRoute {
	return &HTTPRoute{
		Match:  make([]MatcherSet, 0),
		Handle: make([]HTTPHandler, 0),
	}
}

// NewHTTPRouteWithMatch creates a new HTTP route with matchers.
func NewHTTPRouteWithMatch(matchers ...MatcherSet) *HTTPRoute {
	return &HTTPRoute{
		Match:  matchers,
		Handle: make([]HTTPHandler, 0),
	}
}

// AddServer adds a server to the HTTP app.
func (a *HTTPApp) AddServer(name string, server *HTTPServer) *HTTPApp {
	if a.Servers == nil {
		a.Servers = make(map[string]*HTTPServer)
	}
	a.Servers[name] = server
	return a
}

// AddRoute adds a route to the HTTP server.
func (s *HTTPServer) AddRoute(route *HTTPRoute) *HTTPServer {
	s.Routes = append(s.Routes, route)
	return s
}

// AddRoutes adds multiple routes to the HTTP server.
func (s *HTTPServer) AddRoutes(routes ...*HTTPRoute) *HTTPServer {
	s.Routes = append(s.Routes, routes...)
	return s
}

// WithAutoHTTPS configures automatic HTTPS for the server.
func (s *HTTPServer) WithAutoHTTPS(config *AutoHTTPSConfig) *HTTPServer {
	s.AutoHTTPS = config
	return s
}

// DisableAutoHTTPS disables automatic HTTPS for the server.
func (s *HTTPServer) DisableAutoHTTPS() *HTTPServer {
	s.AutoHTTPS = &AutoHTTPSConfig{
		Disabled: true,
	}
	return s
}

// AddMatch adds a matcher set to the route.
func (r *HTTPRoute) AddMatch(matchers ...MatcherSet) *HTTPRoute {
	r.Match = append(r.Match, matchers...)
	return r
}

// AddHandler adds a handler to the route.
func (r *HTTPRoute) AddHandler(handler HTTPHandler) *HTTPRoute {
	r.Handle = append(r.Handle, handler)
	return r
}

// SetTerminal marks the route as terminal.
func (r *HTTPRoute) SetTerminal(terminal bool) *HTTPRoute {
	r.Terminal = terminal
	return r
}

// NewReverseProxyRoute creates a route for reverse proxying to the given upstreams.
func NewReverseProxyRoute(hosts []string, upstreams []*Upstream) *HTTPRoute {
	handler := NewReverseProxyHandler(upstreams...)
	handler.Headers = &HeadersConfig{
		Request: StandardProxyHeaders(),
	}

	route := NewHTTPRoute()
	if len(hosts) > 0 {
		route.AddMatch(NewHostMatcher(hosts...))
	}
	route.AddHandler(ToHTTPHandler(handler))

	return route
}

// NewRedirectRoute creates a route for redirecting to a target URL.
func NewRedirectRoute(hosts []string, targetURL string, statusCode int) *HTTPRoute {
	handler := NewRedirectHandler(targetURL, statusCode)

	route := NewHTTPRoute()
	if len(hosts) > 0 {
		route.AddMatch(NewHostMatcher(hosts...))
	}
	route.AddHandler(ToHTTPHandler(handler))

	return route
}

// NewStaticFileRoute creates a route for serving static files.
func NewStaticFileRoute(hosts []string, rootPath string, indexNames []string) *HTTPRoute {
	handler := NewFileServerHandler(rootPath)
	if len(indexNames) > 0 {
		handler.WithIndexNames(indexNames...)
	}

	route := NewHTTPRoute()
	if len(hosts) > 0 {
		route.AddMatch(NewHostMatcher(hosts...))
	}
	route.AddHandler(ToHTTPHandler(handler))

	return route
}

// NewErrorRoute creates a route that responds with an error.
func NewErrorRoute(hosts []string, statusCode int, body string) *HTTPRoute {
	handler := NewStaticResponseHandler(statusCode, body)

	route := NewHTTPRoute()
	if len(hosts) > 0 {
		route.AddMatch(NewHostMatcher(hosts...))
	}
	route.AddHandler(ToHTTPHandler(handler))

	return route
}

// NewCatchAllRoute creates a catch-all route that responds with a 404.
func NewCatchAllRoute() *HTTPRoute {
	return &HTTPRoute{
		Handle: []HTTPHandler{
			ToHTTPHandler(NewStaticResponseHandler(404, "Not Found")),
		},
	}
}

// NewCatchAllRedirectRoute creates a catch-all route that redirects.
func NewCatchAllRedirectRoute(targetURL string) *HTTPRoute {
	return &HTTPRoute{
		Handle: []HTTPHandler{
			ToHTTPHandler(NewRedirectHandler(targetURL, 302)),
		},
	}
}

// BuildDefaultServer creates a default HTTP server with standard configuration.
func BuildDefaultServer(routes []*HTTPRoute) *HTTPServer {
	return &HTTPServer{
		Listen: []string{":443"},
		Routes: routes,
	}
}

// BuildHTTPOnlyServer creates an HTTP-only server (no TLS).
func BuildHTTPOnlyServer(routes []*HTTPRoute) *HTTPServer {
	return &HTTPServer{
		Listen: []string{":80"},
		Routes: routes,
		AutoHTTPS: &AutoHTTPSConfig{
			Disabled: true,
		},
	}
}

// ListenAddress formats a listen address with optional host and port.
func ListenAddress(host string, port int) string {
	if host == "" {
		return ":" + itoa(port)
	}
	return host + ":" + itoa(port)
}

// GroupRoutesByHost groups routes by their host matcher.
// Routes without a host matcher are placed in an empty string key.
func GroupRoutesByHost(routes []*HTTPRoute) map[string][]*HTTPRoute {
	grouped := make(map[string][]*HTTPRoute)

	for _, route := range routes {
		hosts := extractHosts(route)
		if len(hosts) == 0 {
			grouped[""] = append(grouped[""], route)
		} else {
			for _, host := range hosts {
				grouped[host] = append(grouped[host], route)
			}
		}
	}

	return grouped
}

// extractHosts extracts host names from a route's matchers.
func extractHosts(route *HTTPRoute) []string {
	var hosts []string
	for _, matchSet := range route.Match {
		if hostMatch, ok := matchSet["host"].(MatchHost); ok {
			hosts = append(hosts, hostMatch...)
		}
		if hostMatch, ok := matchSet["host"].([]string); ok {
			hosts = append(hosts, hostMatch...)
		}
	}
	return hosts
}

// CollectDomainsFromRoutes collects all unique domains from routes for TLS configuration.
func CollectDomainsFromRoutes(routes []*HTTPRoute) []string {
	domainSet := make(map[string]bool)
	for _, route := range routes {
		for _, host := range extractHosts(route) {
			domainSet[host] = true
		}
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}
	return domains
}
